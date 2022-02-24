package analysis

import (
	"context"
	"qml-lsp/lsp"

	sitter "github.com/smacker/go-tree-sitter"
)

type DiagnosticsQMLAlias struct{}

func (DiagnosticsQMLAlias) Analyze(ctx context.Context, fileURI string, fctx FileContext, engine *AnalysisEngine) (diags []Diagnostic) {
	data := fctx.Body

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(engine.Queries().PropertyTypes, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			if cap.Node.Content(data) != "alias" {
				continue
			}
			diags = append(diags, Diagnostic{
				Diagnostic: lsp.Diagnostic{
					Range:    FromNode(cap.Node).ToLSP(),
					Severity: lsp.SeverityWarning,
					Source:   "alias lint",
					Message:  "Don't use property alias. Instead, consider binding the aliased property to a property of the concrete type on this type.",
				},
				ContextNode: cap.Node.Parent(),
			})
		}
	}

	return diags
}
