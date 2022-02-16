package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"path"
	"strings"
	"time"

	"qml-lsp/analysis"
	"qml-lsp/lsp"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/sourcegraph/jsonrpc2"
)

const (
	codeActionExtractInlineComponent = "qml-lsp.extract_inline_component"
)

const example = `component Yupekosi : Yourmom`

func Abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func Min(x int, y int) int {
	if x < y {
		return x
	}
	return y
}

func leadingSpaces(line string) int {
	return len(line) - len(strings.TrimLeft(line, " "))
}

func leadingTabs(line string) int {
	return len(line) - len(strings.TrimLeft(line, "\t"))
}

func standalonify(code string) string {
	lines := strings.Split(code, "\n")
	lines = lines[1:]

	spaces := math.MaxInt
	tabs := math.MaxInt

	for _, line := range lines {
		spaces = Min(spaces, leadingSpaces(line))
		tabs = Min(tabs, leadingTabs(line))
	}

	processedLines := strings.Split(code, "\n")
	for idx, line := range processedLines {
		line = strings.TrimPrefix(line, strings.Repeat(" ", spaces))
		line = strings.TrimPrefix(line, strings.Repeat("\t", tabs))
		processedLines[idx] = line
	}

	return strings.Join(processedLines, "\n")
}

func (s *server) extractInlineComponent(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.CodeActionParams) (interface{}, error) {
	uri := strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI)
	dirname := strings.TrimSuffix(string(params.TextDocument.URI), path.Base(uri))
	fctx, err := s.analysis.GetFileContext(uri)
	if err != nil {
		return nil, err
	}

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	var name *sitter.Node
	var superclass *sitter.Node
	var body *sitter.Node
	_ = name
	_ = superclass
	_ = body

	qc.Exec(s.analysis.Queries().InlineComponents, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		capName := match.Captures[0]
		if matches(capName.Node, params.Range.Start) {
			name = match.Captures[0].Node
			superclass = match.Captures[1].Node
			body = match.Captures[2].Node
			goto yoinky
		}
	}

	return nil, errors.New("node not found")

yoinky:

	var builder strings.Builder

	imports := fctx.Imports
	impl := standalonify(body.Content(fctx.Body))
	kind := superclass.Content(fctx.Body)

	used, err := s.analysis.UsedImports(uri, body)
	if err != nil {
		return nil, err
	}

	for idx := range used {
		if !used[idx] {
			continue
		}

		item := imports[idx]
		builder.WriteString(item.ToSourceString())
		builder.WriteString("\n")
	}
	builder.WriteString("\n")
	builder.WriteString(kind)
	builder.WriteString(" ")
	builder.WriteString(impl)

	newURI := lsp.DocumentURI(fmt.Sprintf("%s/%s.qml", dirname, name.Content(fctx.Body)))
	edits := lsp.ApplyWorkspaceEditParams{
		Edit: lsp.WorkspaceEdit{
			DocumentChanges: []interface{}{
				lsp.TextDocumentEdit{
					TextDocument: lsp.OptionalVersionedTextDocumentIdentifier{
						TextDocumentIdentifier: lsp.TextDocumentIdentifier{
							URI: params.TextDocument.URI,
						},
					},
					Edits: []lsp.TextEdit{
						{
							Range:   analysis.FromNode(body.Parent()).ToLSP(),
							NewText: "",
						},
					},
				},
				lsp.CreateFile{
					Kind: "create",
					URI:  newURI,
				},
				lsp.TextDocumentEdit{
					TextDocument: lsp.OptionalVersionedTextDocumentIdentifier{
						TextDocumentIdentifier: lsp.TextDocumentIdentifier{
							URI: newURI,
						},
					},
					Edits: []lsp.TextEdit{
						{
							Range:   lsp.Range{},
							NewText: builder.String(),
						},
					},
				},
			},
			Changes: map[lsp.DocumentURI][]lsp.TextEdit{},
		},
	}

	dl, _ := context.WithDeadline(ctx, time.Now().Add(time.Second*5))
	var r lsp.ApplyWorkspaceEditResult
	err = conn.Call(dl, "workspace/applyEdit", edits, &r)

	if err != nil {
		return nil, err
	}

	return 0, nil
}

func (s *server) canExtractInlineComponent(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.CodeActionParams) (string, bool, error) {
	uri := strings.TrimPrefix(string(params.TextDocument.URI), s.rootURI)
	fctx, err := s.analysis.GetFileContext(uri)
	if err != nil {
		return "", false, err
	}

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(s.analysis.Queries().InlineComponents, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		name := match.Captures[0]
		if matches(name.Node, params.Range.Start) {
			return name.Node.Content(fctx.Body), true, nil
		}
	}

	return "", false, nil
}

func unmarshalIface(out interface{}, in interface{}) error {
	data, err := json.Marshal(in)
	if err != nil {
		return err
	}
	return json.Unmarshal(data, out)
}

func (s *server) ExecuteCommand(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.ExecuteCommandParams) (interface{}, error) {
	switch params.Command {
	case codeActionExtractInlineComponent:
		var p lsp.CodeActionParams
		err := unmarshalIface(&p, params.Arguments[0])
		if err != nil {
			return nil, err
		}
		go s.extractInlineComponent(ctx, conn, p)
		return 0, nil
	}

	return nil, errors.New("Unsupported command")
}

func (s *server) CodeAction(ctx context.Context, conn jsonrpc2.JSONRPC2, params lsp.CodeActionParams) ([]lsp.Command, error) {
	k := []lsp.Command{}

	name, can, err := s.canExtractInlineComponent(ctx, conn, params)
	if err != nil {
		return nil, err
	}
	if can {
		marshalled, _ := json.Marshal(params)
		k = append(k, lsp.Command{
			Title:     fmt.Sprintf("Extract inline component into %s.qml", name),
			Command:   codeActionExtractInlineComponent,
			Arguments: []json.RawMessage{marshalled},
		})
	}

	return k, nil
}
