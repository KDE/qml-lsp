package main

import (
	"context"
	_ "embed"
	"fmt"
	"strings"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type filecontextimportdata struct {
	Module *Module
	As     string
}

type filecontext struct {
	imports []filecontextimportdata
	tree    *sitter.Tree
}

type importName struct {
	path string
	vmaj int
	vmin int
}

type resultSting struct {
	s string
	e error
}

type resultModule struct {
	m *Module
	e error
}

func fromRaw(s []string, vmaj, vmin int) importName {
	return importName{strings.Join(s, "."), vmaj, vmin}
}

type server struct {
	rootURI      string
	files        map[string]string
	filecontexts map[string]filecontext

	importNamesToResolvedPaths map[importName]resultSting
	resolvedPathsToModules     map[string]resultModule

	builtinModule Module
}

func (s *server) Initialize(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.InitializeParams) (*lsp.InitializeResult, *lsp.InitializeError) {
	s.rootURI = string(params.RootURI)
	s.files = map[string]string{}
	s.filecontexts = map[string]filecontext{}

	s.importNamesToResolvedPaths = map[importName]resultSting{}
	s.resolvedPathsToModules = map[string]resultModule{}

	s.builtinModule = builtinM

	return &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptionsOrKind{
				Options: &lsp.TextDocumentSyncOptions{
					OpenClose: true,
					Change:    lsp.TDSKFull,
				},
			},
			CompletionProvider: &lsp.CompletionOptions{
				TriggerCharacters: []string{"."},
			},
		},
	}, nil
}

//go:embed test/builtins.qmltypes
var builtin string

var builtinM Module

func init() {
	var d QMLTypesFile

	err := parser.ParseString("builtin", builtin, &d)
	if err != nil {
		panic(err)
	}

	err = unmarshal(Value{Object: &d.Main}, &builtinM)
	if err != nil {
		panic(err)
	}
}

func (s *server) getModule(uri []string, vmaj, vmin int) (*Module, error) {
	imported := fromRaw(uri, vmaj, vmin)

	var (
		resolved string
		err      error
		module   Module
	)

	if v, ok := s.importNamesToResolvedPaths[imported]; ok {
		if v.e != nil {
			return nil, fmt.Errorf("failed to get module: %+w", v.e)
		}

		if vv, ok := s.resolvedPathsToModules[v.s]; ok {
			if vv.e != nil {
				return nil, fmt.Errorf("failed to get module: %+w", vv.e)
			}

			return vv.m, nil
		} else {
			goto resolvedToModule
		}
	} else {
		goto importNameToResolved
	}

importNameToResolved:
	resolved, err = actualQmlPath(uri, vmaj, vmin)
	s.importNamesToResolvedPaths[imported] = resultSting{resolved, err}
	if err != nil {
		return nil, fmt.Errorf("failed to resolve import to file: %+w", err)
	}

resolvedToModule:
	module, err = loadPluginTypes(resolved)
	s.resolvedPathsToModules[resolved] = resultModule{&module, err}
	if err != nil {
		return nil, fmt.Errorf("failed to load module types: %+w", err)
	}

	return &module, nil
}

func (s *server) evaluate(ctx context.Context, conn jsonrpc2.JSONRPC2, uri lsp.DocumentURI, content string) {
	diags := lsp.PublishDiagnosticsParams{
		URI: uri,
	}

	fileURI := strings.TrimPrefix(string(uri), s.rootURI)
	cont := []byte(s.files[fileURI])

	fctx := filecontext{}

	it := qmlParser()
	fctx.tree = it.Parse(nil, cont)

	importData := extractImports(fctx.tree.RootNode(), cont)
	for _, it := range importData {
		m, err := s.getModule(it.Module, it.MajVersion, it.MinVersion)
		if err != nil {
			println(err.Error())
			continue
		}
		fctx.imports = append(fctx.imports, filecontextimportdata{
			Module: m,
			As:     it.As,
		})
	}

	s.filecontexts[fileURI] = fctx

	conn.Notify(ctx, "textDocument/publishDiagnostics", diags)
}

func (s *server) Initialized(ctx context.Context, conn jsonrpc2.JSONRPC2, params struct{}) {

}

func (s *server) DidOpen(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.DidOpenTextDocumentParams) {
	s.files[strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI)] = params.TextDocument.Text
	go s.evaluate(ctx, conn, params.TextDocument.URI, params.TextDocument.Text)
}

func (s *server) DidChange(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.DidChangeTextDocumentParams) {
	s.files[strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI)] = params.ContentChanges[0].Text
	go s.evaluate(ctx, conn, params.TextDocument.URI, params.ContentChanges[0].Text)
}

func (s *server) DidClose(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.DidCloseTextDocumentParams) {
	delete(s.files, strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI))
	delete(s.filecontexts, strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI))
}

func posToIdx(str string, pos lsp.Position) int {
	col := 0
	line := 0
	for i, c := range str {
		if c == '\n' {
			col = 0
			line++
		} else {
			col++
		}

		if col == pos.Character && line == pos.Line {
			return i
		}
	}
	return -1
}

func wordAtPos(str string, idx int) string {
	var s strings.Builder

	for i, c := range str {
		if unicode.IsSpace(c) {
			s.Reset()
		} else {
			s.WriteRune(c)
		}

		if i == idx {
			for _, cc := range str[i+1:] {
				if unicode.IsSpace(cc) {
					return s.String()
				} else {
					s.WriteRune(cc)
				}
			}
		}
	}

	return ""
}

func (s *server) Completion(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.CompletionParams) (*lsp.CompletionList, error) {
	uri := strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI)
	document := s.files[uri]
	fcontext := s.filecontexts[uri]

	idx := posToIdx(document, params.Position)
	if idx == -1 {
		return &lsp.CompletionList{IsIncomplete: false}, nil
	}

	w := wordAtPos(document, idx)
	println(w)

	citems := []lsp.CompletionItem{}

	doComponents := func(prefix string, components []Component) {
		for _, component := range components {
			for _, enum := range component.Enums {
				for mem := range enum.Values {
					println(prefix+component.Name+"."+mem, w)
					if strings.HasPrefix(prefix+component.Name+"."+mem, w) {
						citems = append(citems, lsp.CompletionItem{
							Label:  prefix + component.Name + "." + mem,
							Kind:   lsp.CIKEnumMember,
							Detail: prefix + component.Name + "." + enum.Name,
						})
					}
				}
			}
		}
	}

	doComponents("", s.builtinModule.Components)

	for _, module := range fcontext.imports {
		if module.As == "" {
			doComponents("", module.Module.Components)
		} else {
			doComponents(module.As+".", module.Module.Components)
		}
	}

	return &lsp.CompletionList{IsIncomplete: false, Items: citems}, nil
}
