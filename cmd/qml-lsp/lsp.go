package main

import (
	"context"
	_ "embed"
	"fmt"
	"qml-lsp/analysis"
	"qml-lsp/qmltypes/qtquick"
	"strings"
	"unicode"

	"qml-lsp/lsp"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/sourcegraph/jsonrpc2"
)

type server struct {
	rootURI              string
	analysis             *analysis.AnalysisEngine
	possibleImports      []analysis.ImportRelAndMaj
	possibleImportMinors map[analysis.ImportRelAndMaj]int
}

func (s *server) Initialize(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.InitializeParams) (*lsp.InitializeResult, *lsp.InitializeError) {
	s.rootURI = string(params.RootURI)
	s.analysis = analysis.New(qtquick.BuiltinModule())
	s.possibleImports, s.possibleImportMinors = analysis.PossibleImports()

	return &lsp.InitializeResult{
		Capabilities: lsp.ServerCapabilities{
			TextDocumentSync: &lsp.TextDocumentSyncOptions{
				OpenClose: true,
				Change:    lsp.Full,
			},
			CodeActionProvider: true,
			ExecuteCommandProvider: lsp.ExecuteCommandOptions{
				Commands: []string{codeActionExtractInlineComponent},
			},
			CompletionProvider: lsp.CompletionOptions{
				TriggerCharacters: []string{"."},
			},
			SemanticTokensProvider: lsp.SemanticTokensOptions{
				Legend: lsp.SemanticTokensLegend{
					TokenTypes: []string{
						SemanticTokenTypeNamespace,
						SemanticTokenTypeType,
						SemanticTokenTypeClass,
						SemanticTokenTypeEnum,
						SemanticTokenTypeInterface,
						SemanticTokenTypeStruct,
						SemanticTokenTypeTypeParameter,
						SemanticTokenTypeParameter,
						SemanticTokenTypeVariable,
						SemanticTokenTypeProperty,
						SemanticTokenTypeEnumMember,
						SemanticTokenTypeEvent,
						SemanticTokenTypeFunction,
						SemanticTokenTypeMethod,
						SemanticTokenTypeMacro,
						SemanticTokenTypeKeyword,
						SemanticTokenTypeModifier,
						SemanticTokenTypeComment,
						SemanticTokenTypeString,
						SemanticTokenTypeNumber,
						SemanticTokenTypeRegexp,
						SemanticTokenTypeOperator,
					},
					TokenModifiers: []string{
						SemanticTokenTypeDeclaration,
						SemanticTokenTypeDefinition,
						SemanticTokenTypeReadonly,
						SemanticTokenTypeStatic,
						SemanticTokenTypeDeprecated,
						SemanticTokenTypeAbstract,
						SemanticTokenTypeAsync,
						SemanticTokenTypeModification,
						SemanticTokenTypeDocumentation,
						SemanticTokenTypeDefaultLibrary,
					},
				},
				Range: false,
				Full:  true,
			},
		},
	}, nil
}

func traverseTree(n *sitter.Node, f func(*sitter.Node)) {
	f(n)
	for i := 0; i < int(n.ChildCount()); i++ {
		traverseTree(n.Child(i), f)
	}
}

func (s *server) SemanticTokensFull(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.SemanticTokensParams) (*lsp.SemanticTokens, error) {
	fileURI := strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI)
	fctx, err := s.analysis.GetFileContext(fileURI)
	if err != nil {
		return nil, fmt.Errorf("failed to get file URI for semantic tokens: %+w", err)
	}

	node := fctx.Tree.RootNode()

	toks := []uint32{}

	previousLine := uint32(0)
	previousStartChar := uint32(0)

	addToken := func(n *sitter.Node, tokenKind NumSemanticTokenKind, tokenModifiers NumSemanticTokenModifier) {
		line := n.StartPoint().Row
		startChar := n.StartPoint().Column
		length := n.EndByte() - n.StartByte()

		deltaLine := line - uint32(previousLine)
		var deltaStart uint32
		if line == previousLine {
			deltaStart = startChar - previousStartChar
		} else {
			deltaStart = startChar
		}
		previousLine = line
		previousStartChar = startChar

		toks = append(toks, deltaLine, deltaStart, length, uint32(tokenKind), uint32(tokenModifiers))
	}
	traverseTree(node, func(n *sitter.Node) {
		switch {
		case n.Type() == "identifier" && n.Parent().Type() == "inline_type_declaration":
			addToken(n, NumSemanticTokenTypeType, 0)
		case n.Type() == "qualified_identifier" && n.Parent().Type() == "inline_type_declaration":
			addToken(n, NumSemanticTokenTypeType, 0)

		case n.Type() == "qualified_identifier" && n.Parent().Type() == "import_statement":
			addToken(n, NumSemanticTokenTypeNamespace, 0)
		case n.Type() == "identifier" && n.Parent().Type() == "qualified_identifier" && n.Parent().Parent().Type() == "import_statement":
			return
		case n.Type() == "named_import" && n.Parent().Type() == "import_statement":
			addToken(n, NumSemanticTokenTypeNamespace, 0)

		case n.Type() == "type_identifier" && n.Parent().Type() == "property_type" && n.EndPoint() == n.Parent().EndPoint():
			addToken(n, NumSemanticTokenTypeClass, 0)
		case n.Type() == "type_identifier" && n.Parent().Type() == "property_type":
			addToken(n, NumSemanticTokenTypeNamespace, 0)
		case n.Parent() != nil && n.Parent().Type() == "property_type":
			addToken(n, NumSemanticTokenTypeType, 0)
		case n.Type() == "property_identifier":
			addToken(n, NumSemanticTokenTypeProperty, 0)

		case n.Type() == "identifier" && n.Parent().Type() == "qualified_identifier" && n.Parent().Parent().Type() == "property_set":
			addToken(n, NumSemanticTokenTypeProperty, 0)

		case n.Type() == "type_identifier" && n.Parent().Type() == "enum":
			addToken(n, NumSemanticTokenTypeEnum, 0)
		case n.Type() == "enum_member":
			addToken(n, NumSemanticTokenTypeEnumMember, 0)

		case n.Type() == "identifier" && n.Parent().Type() == "qualified_identifier" && n.EndPoint() == n.Parent().EndPoint():
			addToken(n, NumSemanticTokenTypeClass, 0)
		case n.Type() == "identifier" && n.Parent().Type() == "qualified_identifier":
			addToken(n, NumSemanticTokenTypeNamespace, 0)

		case n.Type() == "string":
			addToken(n, NumSemanticTokenTypeString, 0)
		case n.Type() == "regex":
			addToken(n, NumSemanticTokenTypeRegexp, 0)
		case n.Type() == "number":
			addToken(n, NumSemanticTokenTypeNumber, 0)
		case n.Type() == "comment":
			addToken(n, NumSemanticTokenTypeComment, 0)

		case n.Type() == "as",
			n.Type() == "import",
			n.Type() == "enum" && n.String() == "enum",
			n.Type() == "component":
			addToken(n, NumSemanticTokenTypeKeyword, NumSemanticTokenTypeDeprecated)
		}
	})

	return &lsp.SemanticTokens{
		Data: toks,
	}, nil
}

func (s *server) CodeLens(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.CodeLensParams) ([]lsp.CodeLens, error) {
	return []lsp.CodeLens{}, nil
}

func (s *server) DocumentLink(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.DocumentLinkParams) ([]lsp.DocumentLink, error) {
	return []lsp.DocumentLink{}, nil
}

func (s *server) doLints(ctx context.Context, fileURI string, diags *lsp.PublishDiagnosticsParams) {
	for _, diag := range analysis.DefaultDiagnostics {
		fctx, _ := s.analysis.GetFileContext(fileURI)
		adiags := diag.Analyze(ctx, fileURI, fctx, s.analysis)
		for _, d := range adiags {
			diags.Diagnostics = append(diags.Diagnostics, d.Diagnostic)
		}
	}
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
	col := uint32(0)
	line := uint32(0)
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

/**
 * A code action represents a change that can be performed in code, e.g. to fix
 * a problem or to refactor code.
 *
 * A CodeAction must set either `edit` and/or a `command`. If both are supplied,
 * the `edit` is applied first, then the `command` is executed.
 */
type CodeAction struct {
	/**
	 * A short, human-readable, title for this code action.
	 */
	Title string `json:"title"`

	/**
	 * The kind of the code action.
	 *
	 * Used to filter code actions.
	 */
	Kind lsp.CodeActionKind `json:"kind,omitempty"`

	/**
	 * The diagnostics that this code action resolves.
	 */
	Diagnostics []lsp.Diagnostic `json:"diagnostic"`

	/**
	 * Marks this as a preferred action. Preferred actions are used by the
	 * `auto fix` command and can be targeted by keybindings.
	 *
	 * A quick fix should be marked preferred if it properly addresses the
	 * underlying error. A refactoring should be marked preferred if it is the
	 * most reasonable choice of actions to take.
	 *
	 * @since 3.15.0
	 */
	IsPreferred bool `json:"isPreferred,omitempty"`

	/**
	 * Marks that the code action cannot currently be applied.
	 *
	 * Clients should follow the following guidelines regarding disabled code
	 * actions:
	 *
	 * - Disabled code actions are not shown in automatic lightbulbs code
	 *   action menus.
	 *
	 * - Disabled actions are shown as faded out in the code action menu when
	 *   the user request a more specific type of code action, such as
	 *   refactorings.
	 *
	 * - If the user has a keybinding that auto applies a code action and only
	 *   a disabled code actions are returned, the client should show the user
	 *   an error message with `reason` in the editor.
	 *
	 * @since 3.16.0
	 */
	Disabled *struct {

		/**
		 * Human readable description of why the code action is currently
		 * disabled.
		 *
		 * This is displayed in the code actions UI.
		 */
		Reason string `json:"reason"`
	} `json:"disabled,omitempty"`

	/**
	 * The workspace edit this code action performs.
	 */
	Edit *lsp.WorkspaceEdit `json:"edit,omitempty"`

	/**
	 * A command this code action executes. If a code action
	 * provides an edit and a command, first the edit is
	 * executed and then the command.
	 */
	Command *lsp.Command `json:"command,omitempty"`

	/**
	 * A data entry field that is preserved on a code action between
	 * a `textDocument/codeAction` and a `codeAction/resolve` request.
	 *
	 * @since 3.16.0
	 */
	Data interface{} `json:"data"`
}

func (s *server) completeComponentContext(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.CompletionParams, word string, node *sitter.Node) (*lsp.CompletionList, error) {
	uri := strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI)
	fctx, _ := s.analysis.GetFileContext(uri)
	document := string(fctx.Body)

	enclosing := locateEnclosingComponent(node, []byte(document))

	citems := []lsp.CompletionItem{}

	doComponents := func(prefix string, components []analysis.Component) {
		for _, component := range components {
			if strings.HasPrefix(prefix+component.SaneName(), word) {
				citems = append(citems, lsp.CompletionItem{
					Label:      prefix + component.SaneName(),
					Kind:       lsp.ClassCompletion,
					InsertText: strings.TrimPrefix(prefix+component.SaneName(), word),
				})
			}
			if prefix+component.SaneName() == enclosing {
				for _, prop := range component.Properties {
					if strings.HasPrefix(prop.Name, word) {
						citems = append(citems, lsp.CompletionItem{
							Label:      prop.Name,
							Kind:       lsp.PropertyCompletion,
							Detail:     prop.Type,
							InsertText: strings.TrimPrefix(prop.Name, word),
						})
					}
				}
			}
			for _, enum := range component.Enums {
				for mem := range enum.Values {
					if strings.HasPrefix(prefix+component.SaneName()+"."+mem, word) {
						citems = append(citems, lsp.CompletionItem{
							Label:      prefix + component.SaneName() + "." + mem,
							Kind:       lsp.EnumMemberCompletion,
							Detail:     prefix + component.SaneName() + "." + enum.Name,
							InsertText: strings.TrimPrefix(prefix+component.SaneName()+"."+mem, word),
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
					if !strings.HasPrefix(fullName, word) {
						continue
					}

					citems = append(citems, lsp.CompletionItem{
						Label:      fullName,
						Kind:       lsp.PropertyCompletion,
						Detail:     fmt.Sprintf("attached %s", prefix+component.SaneName()),
						InsertText: strings.TrimPrefix(fullName+": ", word),
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

func (s *server) completeProgramContext(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.CompletionParams, word string, node *sitter.Node) (*lsp.CompletionList, error) {
	citems := []lsp.CompletionItem{}

	for _, impt := range s.possibleImports {
		if strings.HasPrefix(impt.URL, word) {
			rest := fmt.Sprintf("%s %d.%d", impt.URL, impt.Major, s.possibleImportMinors[impt])
			citems = append(citems, lsp.CompletionItem{
				Label:      rest,
				Kind:       lsp.ModuleCompletion,
				InsertText: strings.TrimPrefix(rest, word),
			})
		}
	}

	return &lsp.CompletionList{IsIncomplete: false, Items: citems}, nil
}

type cursorContext int

const (
	program cursorContext = iota
	objectBlock
)

func classifyNodeContext(n *sitter.Node) cursorContext {
	switch n.Type() {
	case "program":
		return program
	case "object_block":
		return objectBlock
	default:
		if n.Parent() == nil {
			panic("unreachable")
		}
		return classifyNodeContext(n.Parent())
	}
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

	switch classifyNodeContext(m) {
	case program:
		return s.completeProgramContext(ctx, conn, params, w, m)
	case objectBlock:
		return s.completeComponentContext(ctx, conn, params, w, m)
	default:
		return &lsp.CompletionList{IsIncomplete: false, Items: []lsp.CompletionItem{}}, nil
	}
}
