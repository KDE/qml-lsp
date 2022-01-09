package main

import (
	"context"
	_ "embed"
	"fmt"
	"log"
	"qml-lsp/analysis"
	"qml-lsp/qmltypes/qtquick"
	qml "qml-lsp/treesitter-qml"
	"strings"
	"unicode"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/sourcegraph/go-lsp"
	"github.com/sourcegraph/jsonrpc2"
)

type queries struct {
	propertyTypes                           *sitter.Query
	objectDeclarationTypes                  *sitter.Query
	withStatements                          *sitter.Query
	parentObjectChildPropertySets           *sitter.Query
	statementBlocksWithVariableDeclarations *sitter.Query
	variableAssignments                     *sitter.Query
	doubleNegation                          *sitter.Query
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
	q.parentObjectChildPropertySets, err = sitter.NewQuery([]byte(`(object_declaration
		(qualified_identifier) @outer
		(object_block
			(object_declaration
				(object_block
					(property_set (qualified_identifier) @prop)))))`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.statementBlocksWithVariableDeclarations, err = sitter.NewQuery([]byte(`
	(statement_block
		(variable_declaration
			"var" @keyword
			(variable_declarator name: (identifier) @name))
		(_)* @following)
`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.variableAssignments, err = sitter.NewQuery([]byte(`
(assignment_expression left: (identifier) @ident)
	`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.doubleNegation, err = sitter.NewQuery([]byte(`
(unary_expression operator: "!" argument: (unary_expression operator: "!" argument: (_) @arg)) @outer
	`), qml.GetLanguage())
	if err != nil {
		return err
	}
	return nil
}

type server struct {
	rootURI  string
	analysis *analysis.AnalysisEngine

	q queries
}

func (s *server) Initialize(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.InitializeParams) (*lsp.InitializeResult, *lsp.InitializeError) {
	s.rootURI = string(params.RootURI)
	s.analysis = analysis.New(qtquick.BuiltinModule())

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

func (s *server) lintUnusedImports(ctx context.Context, fileURI string, diags *lsp.PublishDiagnosticsParams) {
	fctx, _ := s.analysis.GetFileContext(fileURI)
	data := fctx.Body
	imports := fctx.Imports
	used := make([]bool, len(imports))

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	types := map[string]struct{}{}

	// gather all the refernces to types in the documents

	// uses in property declarations, such as
	// property -> Kirigami.AboutPage <- aboutPage: ...
	qc.Exec(s.q.propertyTypes, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			types[cap.Node.Content(data)] = struct{}{}
		}
	}

	// uses in object blocks, such as
	// -> Kirigami.AboutPage <- { }
	qc.Exec(s.q.objectDeclarationTypes, fctx.Tree.RootNode())
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
				if prefix+component.SaneName() == kind {
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
	fctx, _ := s.analysis.GetFileContext(fileURI)

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(s.q.withStatements, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			diags.Diagnostics = append(diags.Diagnostics, lsp.Diagnostic{
				Range:    analysis.FromNode(cap.Node).ToLSP(),
				Severity: lsp.Warning,
				Source:   "with lint",
				Message:  "Don't use with statements in modern JavaScript",
			})
		}
	}
}

func (s *server) lintAlias(ctx context.Context, fileURI string, diags *lsp.PublishDiagnosticsParams) {
	fctx, _ := s.analysis.GetFileContext(fileURI)
	data := fctx.Body

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(s.q.propertyTypes, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			if cap.Node.Content(data) != "alias" {
				continue
			}
			diags.Diagnostics = append(diags.Diagnostics, lsp.Diagnostic{
				Range:    analysis.FromNode(cap.Node).ToLSP(),
				Severity: lsp.Warning,
				Source:   "alias lint",
				Message:  "Don't use property alias. Instead, consider binding the aliased property to a property of the concrete type on this type.",
			})
		}
	}
}

var anchorsInLayoutWarnings = map[string]string{
	"anchors.alignWhenCentered":      `Don't use anchors.alignWhenCentered in a {{kind}}. Layouts always pixel-align their items, so tihs is unneccesary.`,
	"anchors.baseline":               `Don't use anchors.baseline in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignBaseline"`,
	"anchors.baselineOffset":         `Don't use anchors.baselineOffset in a {{kind}}. Instead, consider setting the "{{pfx}}Layout.bottomMargin".`,
	"anchors.bottom":                 `Don't use anchors.bottom in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignBottom"`,
	"anchors.bottomMargin":           `Don't use anchors.bottomMargin in a {{kind}}. Instead, consider setting the "{{pfx}}Layout.bottomMargin"`,
	"anchors.centerIn":               `Don't use anchors.centerIn in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignVCenter | Qt.AlignHCenter"`,
	"anchors.fill":                   `Don't use anchors.fill in a {{kind}}. Instead, consider using "{{pfx}}Layout.fillWidth: true" and "{{pfx}}Layout.fillHeight: true"`,
	"anchors.horizontalCenter":       `Don't use anchors.horizontalCenter in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignHCenter"`,
	"anchors.horizontalCenterOffset": `Don't use anchors.horizontalCenterOffset in a {{kind}}. Instead, consider using "{{pfx}}Layout.leftMargin" or "{{pfx}}Layout.rightMargin"`,
	"anchors.left":                   `Don't use anchors.left in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignLeft"`,
	"anchors.leftMargin":             `Don't use anchors.leftMargin in a {{kind}}. Instead, consider using "{{pfx}}Layout.leftMargin"`,
	"anchors.margins":                `Don't use anchors.margins in a {{kind}}. Instead, consider using "{{pfx}}Layout.margins"`,
	"anchors.right":                  `Don't use anchors.right in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignRight"`,
	"anchors.rightMargin":            `Don't use anchors.rightMargin in a {{kind}}. Instead, consider using "{{pfx}}Layout.rightMargin"`,
	"anchors.top":                    `Don't use anchors.top in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignTop"`,
	"anchors.topMargin":              `Don't use anchors.topMargin in a {{kind}}. Instead, consider using "{{pfx}}Layout.topMargin"`,
	"anchors.verticalCenter":         `Don't use anchors.verticalCenter in a {{kind}}. Instead, consider using "{{pfx}}Layout.horizontalAlignment: Qt.AlignHCenter"`,
	"anchors.verticalCenterOffset":   `Don't use anchors.verticalCenterOffset in a {{kind}}. Instead, consider using "{{pfx}}Layout.topMargin" or "{{pfx}}Layout.bottomMargin"`,
}

func (s *server) lintVar(ctx context.Context, fileURI string, diags *lsp.PublishDiagnosticsParams) {
	fctx, _ := s.analysis.GetFileContext(fileURI)
	data := fctx.Body

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	ic := sitter.NewQueryCursor()
	defer ic.Close()

	qc.Exec(s.q.statementBlocksWithVariableDeclarations, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		vname := match.Captures[1].Node.Content(data)
		keyword := match.Captures[0].Node
		remaining := match.Captures[2:]

		var isSet bool

	outer:
		for _, cap := range remaining {
			ic.Exec(s.q.variableAssignments, cap.Node)
			for imatch, igoNext := ic.NextMatch(); igoNext; imatch, igoNext = ic.NextMatch() {
				iname := imatch.Captures[0].Node.Content(data)

				if vname == iname {
					isSet = true
					break outer
				}
			}
		}

		if isSet {
			diags.Diagnostics = append(diags.Diagnostics, lsp.Diagnostic{
				Range:    analysis.FromNode(keyword).ToLSP(),
				Severity: lsp.Warning,
				Source:   "var lint",
				Message:  `Don't use var in modern JavaScript. Consider using "let" here instead.`,
			})
		} else {
			diags.Diagnostics = append(diags.Diagnostics, lsp.Diagnostic{
				Range:    analysis.FromNode(keyword).ToLSP(),
				Severity: lsp.Warning,
				Source:   "var lint",
				Message:  `Don't use var in modern JavaScript. Consider using "const" here instead.`,
			})
		}
	}
}

func (s *server) lintLayoutAnchors(ctx context.Context, fileURI string, diags *lsp.PublishDiagnosticsParams) {
	fctx, _ := s.analysis.GetFileContext(fileURI)
	data := fctx.Body
	imports := fctx.Imports

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(s.q.parentObjectChildPropertySets, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		parentType := match.Captures[0].Node.Content(data)
		childProperty := match.Captures[1].Node.Content(data)

		if !strings.HasPrefix(childProperty, "anchors") {
			continue
		}

		for _, item := range imports {
			if item.URI.Path != "QtQuick.Layouts" {
				continue
			}

			pfx := ""
			if item.As != "" {
				pfx = item.As + "."
			}

			for _, comp := range item.Module.Components {
				if pfx+comp.SaneName() != parentType {
					continue
				}

				v, ok := anchorsInLayoutWarnings[childProperty]
				if !ok {
					continue
				}

				diags.Diagnostics = append(diags.Diagnostics, lsp.Diagnostic{
					Range:    analysis.FromNode(match.Captures[1].Node).ToLSP(),
					Severity: lsp.Error,
					Source:   "anchors in layouts lint",
					Message:  strings.ReplaceAll(strings.ReplaceAll(v, "{{kind}}", pfx+comp.SaneName()), "{{pfx}}", pfx),
				})
			}
		}
	}
}

func (s *server) lintDoubleNegation(ctx context.Context, fileURI string, diags *lsp.PublishDiagnosticsParams) {
	fctx, _ := s.analysis.GetFileContext(fileURI)
	data := fctx.Body

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(s.q.doubleNegation, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		diags.Diagnostics = append(diags.Diagnostics, lsp.Diagnostic{
			Range:    analysis.FromNode(match.Captures[0].Node).ToLSP(),
			Severity: lsp.Info,
			Source:   `double negation lint`,
			Message:  fmt.Sprintf(`Many people find double negation hard to read. Consider using "Boolean(%s)" instead.`, match.Captures[1].Node.Content(data)),
		})
	}
}

func (s *server) doLints(ctx context.Context, fileURI string, diags *lsp.PublishDiagnosticsParams) {
	s.lintUnusedImports(ctx, fileURI, diags)
	s.lintAlias(ctx, fileURI, diags)
	s.lintWith(ctx, fileURI, diags)
	s.lintLayoutAnchors(ctx, fileURI, diags)
	s.lintVar(ctx, fileURI, diags)
	s.lintDoubleNegation(ctx, fileURI, diags)
}

func (s *server) evaluate(ctx context.Context, conn jsonrpc2.JSONRPC2, uri lsp.DocumentURI, content string) {
	diags := lsp.PublishDiagnosticsParams{
		URI: uri,
	}

	fileURI := strings.TrimPrefix(string(uri), s.rootURI)

	s.analysis.SetFileContext(fileURI, []byte(content))

	s.doLints(ctx, fileURI, &diags)

	conn.Notify(ctx, "textDocument/publishDiagnostics", diags)
}

func (s *server) Initialized(ctx context.Context, conn jsonrpc2.JSONRPC2, params struct{}) {

}

func (s *server) DidOpen(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.DidOpenTextDocumentParams) {
	go s.evaluate(ctx, conn, params.TextDocument.URI, params.TextDocument.Text)
}

func (s *server) DidChange(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.DidChangeTextDocumentParams) {
	go s.evaluate(ctx, conn, params.TextDocument.URI, params.ContentChanges[0].Text)
}

func (s *server) DidChangeWatchedFiles(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.DidChangeWatchedFilesParams) {

}

func (s *server) DidClose(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.DidCloseTextDocumentParams) {
	s.analysis.DeleteFileContext(strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI))
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

func locateEnclosingComponent(n *sitter.Node, b []byte) string {
	if n.Parent() == nil {
		return ""
	}

	if n.Type() == "object_declaration" {
		return strings.Join(analysis.ExtractQualifiedIdentifier(n.Child(0), b), ".")
	}

	return locateEnclosingComponent(n.Parent(), b)
}

func (s *server) Completion(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.CompletionParams) (*lsp.CompletionList, error) {
	uri := strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI)
	fctx, _ := s.analysis.GetFileContext(uri)
	document := string(fctx.Body)

	idx := posToIdx(document, params.Position)
	if idx == -1 {
		return &lsp.CompletionList{IsIncomplete: false}, nil
	}

	w := wordAtPos(document, idx)
	m := findNearestMatchingNode(fctx.Tree.RootNode(), params.Position)
	enclosing := locateEnclosingComponent(m, []byte(document))

	citems := []lsp.CompletionItem{}
	println(w)

	doComponents := func(prefix string, components []analysis.Component) {
		for _, component := range components {
			if strings.HasPrefix(prefix+component.SaneName(), w) {
				citems = append(citems, lsp.CompletionItem{
					Label:      prefix + component.SaneName(),
					Kind:       lsp.CIKClass,
					InsertText: strings.TrimPrefix(prefix+component.SaneName(), w),
				})
			}
			if prefix+component.SaneName() == enclosing {
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
					if strings.HasPrefix(prefix+component.SaneName()+"."+mem, w) {
						citems = append(citems, lsp.CompletionItem{
							Label:      prefix + component.SaneName() + "." + mem,
							Kind:       lsp.CIKEnumMember,
							Detail:     prefix + component.SaneName() + "." + enum.Name,
							InsertText: strings.TrimPrefix(prefix+component.SaneName()+"."+mem, w),
						})
					}
				}
			}
			if component.AttachedType == "" {
				continue
			}
			for _, comp := range components {
				if comp.Name != component.AttachedType {
					continue
				}

				for _, prop := range comp.Properties {
					fullName := prefix + component.SaneName() + "." + prop.Name
					println(fullName, w)
					if !strings.HasPrefix(fullName, w) {
						continue
					}

					citems = append(citems, lsp.CompletionItem{
						Label:      fullName,
						Kind:       lsp.CIKProperty,
						Detail:     fmt.Sprintf("attached %s", prefix+component.SaneName()),
						InsertText: strings.TrimPrefix(fullName+": ", w),
					})
				}
			}
		}
	}

	doComponents("", s.analysis.BuiltinModule().Components)

	for _, module := range fctx.Imports {
		if module.As == "" {
			doComponents("", module.Module.Components)
		} else {
			doComponents(module.As+".", module.Module.Components)
		}
	}

	return &lsp.CompletionList{IsIncomplete: false, Items: citems}, nil
}
