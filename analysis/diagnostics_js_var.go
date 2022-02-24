package analysis

import (
	"context"
	"qml-lsp/lsp"

	sitter "github.com/smacker/go-tree-sitter"
)

type DiagnosticsJSVar struct{}

func (DiagnosticsJSVar) Analyze(ctx context.Context, fileURI string, fctx FileContext, engine *AnalysisEngine) (diags []lsp.Diagnostic) {
	data := fctx.Body

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	ic := sitter.NewQueryCursor()
	defer ic.Close()

	qc.Exec(engine.Queries().StatementBlocksWithVariableDeclarations, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		vname := match.Captures[1].Node.Content(data)
		keyword := match.Captures[0].Node
		remaining := match.Captures[2:]

		var isSet bool

	outer:
		for _, cap := range remaining {
			ic.Exec(engine.Queries().VariableAssignments, cap.Node)
			for imatch, igoNext := ic.NextMatch(); igoNext; imatch, igoNext = ic.NextMatch() {
				iname := imatch.Captures[0].Node.Content(data)

				if vname == iname {
					isSet = true
					break outer
				}
			}
		}

		if isSet {
			diags = append(diags, lsp.Diagnostic{
				Range:    FromNode(keyword).ToLSP(),
				Severity: lsp.SeverityWarning,
				Source:   "var lint",
				Message:  `Don't use var in modern JavaScript. Consider using "let" here instead.`,
			})
		} else {
			diags = append(diags, lsp.Diagnostic{
				Range:    FromNode(keyword).ToLSP(),
				Severity: lsp.SeverityWarning,
				Source:   "var lint",
				Message:  `Don't use var in modern JavaScript. Consider using "const" here instead.`,
			})
		}
	}

	return diags
}
