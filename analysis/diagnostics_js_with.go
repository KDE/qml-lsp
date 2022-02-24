package analysis

import (
	"context"
	"qml-lsp/lsp"

	sitter "github.com/smacker/go-tree-sitter"
)

type DiagnosticsJSWith struct{}

func (DiagnosticsJSWith) Analyze(ctx context.Context, fileURI string, fctx FileContext, engine *AnalysisEngine) (diags []lsp.Diagnostic) {
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(engine.Queries().WithStatements, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			diags = append(diags, lsp.Diagnostic{
				Range:    FromNode(cap.Node).ToLSP(),
				Severity: lsp.SeverityWarning,
				Source:   "with lint",
				Message:  "Don't use with statements in modern JavaScript",
			})
		}
	}

	return diags
}
