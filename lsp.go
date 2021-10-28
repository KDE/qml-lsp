package main

import (
	"context"
	_ "embed"
	"strings"
	"unicode"

	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type filecontext struct {
	modules []Module
}

type server struct {
	rootURI      string
	files        map[string]string
	filecontexts map[string]filecontext
}

func (s *server) Initialize(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.InitializeParams) (*lsp.InitializeResult, *lsp.InitializeError) {
	s.rootURI = string(params.RootURI)
	s.files = map[string]string{}
	s.filecontexts = map[string]filecontext{}

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

func (s *server) evaluate(ctx context.Context, conn jsonrpc2.JSONRPC2, uri lsp.DocumentURI, content string) {
	diags := lsp.PublishDiagnosticsParams{
		URI: uri,
	}

	s.filecontexts[strings.TrimPrefix(string(uri), s.rootURI)] = filecontext{modules: []Module{builtinM}}

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

	for _, it := range fcontext.modules {
		for _, component := range it.Components {
			for _, enum := range component.Enums {
				for mem := range enum.Values {
					if strings.HasPrefix(component.Name+"."+mem, w) {
						citems = append(citems, lsp.CompletionItem{
							Label:  component.Name + "." + mem,
							Kind:   lsp.CIKEnumMember,
							Detail: component.Name + "." + enum.Name,
						})
					}
				}
			}
		}
	}

	return &lsp.CompletionList{IsIncomplete: false, Items: citems}, nil
}
