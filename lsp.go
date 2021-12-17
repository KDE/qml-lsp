package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	qml "qml-lsp/treesitter-qml"
	"strings"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type filecontextimportdata struct {
	Module *Module
	As     string
	Range  PointRange
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

type queries struct {
	propertyTypes          *sitter.Query
	objectDeclarationTypes *sitter.Query
	withStatements         *sitter.Query
}

func (q *queries) init() error {
	var err error
	q.propertyTypes, err = sitter.NewQuery([]byte("(property_declarator (property_type) @ident)"), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.objectDeclarationTypes, err = sitter.NewQuery([]byte("(object_declaration (qualified_identifier) @ident)"), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.withStatements, err = sitter.NewQuery([]byte(`(with_statement "with" @bad)`), qml.GetLanguage())
	if err != nil {
		return err
	}
	return nil
}

type server struct {
	rootURI      string
	files        map[string]string
	filecontexts map[string]filecontext

	importNamesToResolvedPaths map[importName]resultSting
	resolvedPathsToModules     map[string]resultModule

	builtinModule Module

	q queries
}

func (s *server) Initialize(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.InitializeParams) (*lsp.InitializeResult, *lsp.InitializeError) {
	s.rootURI = string(params.RootURI)
	s.files = map[string]string{}
	s.filecontexts = map[string]filecontext{}

	s.importNamesToResolvedPaths = map[importName]resultSting{}
	s.resolvedPathsToModules = map[string]resultModule{}

	s.builtinModule = builtinM

	err := s.q.init()
	if err != nil {
		log.Printf("error initting queries: %s", err)
		return nil, &lsp.InitializeError{Retry: false}
	}

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

func (s *server) lintUnusedImports(ctx context.Context, fileURI string, diags *lsp.PublishDiagnosticsParams) {
	fctx := s.filecontexts[fileURI]
	data := []byte(s.files[fileURI])
	imports := fctx.imports
	used := make([]bool, len(imports))

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	types := map[string]struct{}{}

	// gather all the refernces to types in the documents

	// uses in property declarations, such as
	// property -> Kirigami.AboutPage <- aboutPage: ...
	qc.Exec(s.q.propertyTypes, fctx.tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			types[cap.Node.Content(data)] = struct{}{}
		}
	}

	// uses in object blocks, such as
	// -> Kirigami.AboutPage <- { }
	qc.Exec(s.q.objectDeclarationTypes, fctx.tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			types[cap.Node.Content(data)] = struct{}{}
		}
	}

	// we've gathered all our types, now we try to match them to imports
outerLoop:
	for kind := range types {
		for idx := range imports {
			importData := imports[idx]
			isUsed := used[idx]

			// if this import is already known used, we don't need to waste time
			// checking if it's used again
			if isUsed {
				continue
			}

			// handle stuff like "import org.kde.kirigami 2.10 as Kirigami"
			// Kirigami.AboutData vs AboutData.
			prefix := ""
			if importData.As != "" {
				prefix = importData.As + "."
			}

			for _, component := range importData.Module.Components {
				if prefix+saneify(component.Name) == kind {
					used[idx] = true
					continue outerLoop
				}
			}
		}
	}

	// now let's go through our imports and raise warnings for any unused imports
	for idx, importData := range imports {
		isUsed := used[idx]

		if isUsed {
			continue
		}

		// oops, this import isn't used! let's raise a diagnostic...
		diags.Diagnostics = append(diags.Diagnostics, lsp.Diagnostic{
			Range:    importData.Range.ToLSP(),
			Severity: lsp.Warning,
			Source:   "import lint",
			Message:  "Unused import",
		})
	}
}

func (s *server) lintWith(ctx context.Context, fileURI string, diags *lsp.PublishDiagnosticsParams) {
	fctx := s.filecontexts[fileURI]

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(s.q.withStatements, fctx.tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			diags.Diagnostics = append(diags.Diagnostics, lsp.Diagnostic{
				Range:    FromNode(cap.Node).ToLSP(),
				Severity: lsp.Warning,
				Source:   "with lint",
				Message:  "Don't use with statements in modern JavaScript",
			})
		}
	}
}

func (s *server) doLints(ctx context.Context, fileURI string, diags *lsp.PublishDiagnosticsParams) {
	s.lintUnusedImports(ctx, fileURI, diags)
	s.lintWith(ctx, fileURI, diags)
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
			Range:  it.Range,
		})
	}

	s.filecontexts[fileURI] = fctx

	s.doLints(ctx, fileURI, &diags)

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

func matches(node *sitter.Node, p lsp.Position) bool {
	p.Character++
	p.Line++
	if node.StartPoint().Row <= uint32(p.Line) && uint32(p.Line) <= node.EndPoint().Row {
		if node.StartPoint().Row == node.EndPoint().Row {
			if node.StartPoint().Column <= uint32(p.Character) && uint32(p.Character) <= node.EndPoint().Column {
				return true
			}
		} else {
			return true
		}
	}
	return false
}

func findNearestMatchingNode(n *sitter.Node, p lsp.Position) *sitter.Node {
	for i := 0; i < int(n.ChildCount()); i++ {
		child := n.Child(i)

		if matches(child, p) {
			return findNearestMatchingNode(child, p)
		}
	}

	return n
}

func parentTypes(n *sitter.Node) []string {
	if n.Parent() == nil {
		return []string{n.Type()}
	} else {
		return append([]string{n.Type()}, parentTypes(n.Parent())...)
	}
}

func locateEnclosingComponent(n *sitter.Node, b []byte) string {
	if n.Parent() == nil {
		return ""
	}

	if n.Type() == "object_declaration" {
		return strings.Join(extractQualifiedIdentifier(n.Child(0), b), ".")
	}

	return locateEnclosingComponent(n.Parent(), b)
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
	m := findNearestMatchingNode(fcontext.tree.RootNode(), params.Position)
	enclosing := locateEnclosingComponent(m, []byte(document))

	citems := []lsp.CompletionItem{}

	doComponents := func(prefix string, components []Component) {
		for _, component := range components {
			component.Name = saneify(component.Name)
			if strings.HasPrefix(prefix+component.Name, w) {
				citems = append(citems, lsp.CompletionItem{
					Label:      prefix + component.Name,
					Kind:       lsp.CIKClass,
					InsertText: strings.TrimPrefix(prefix+component.Name, w),
				})
			}
			if prefix+component.Name == enclosing {
				for _, prop := range component.Properties {
					if strings.HasPrefix(prop.Name, w) {
						citems = append(citems, lsp.CompletionItem{
							Label:      prop.Name,
							Kind:       lsp.CIKProperty,
							Detail:     prop.Type,
							InsertText: strings.TrimPrefix(prop.Name, w),
						})
					}
				}
			}
			for _, enum := range component.Enums {
				for mem := range enum.Values {
					if strings.HasPrefix(prefix+component.Name+"."+mem, w) {
						citems = append(citems, lsp.CompletionItem{
							Label:      prefix + component.Name + "." + mem,
							Kind:       lsp.CIKEnumMember,
							Detail:     prefix + component.Name + "." + enum.Name,
							InsertText: strings.TrimPrefix(prefix+component.Name+"."+mem, w),
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
