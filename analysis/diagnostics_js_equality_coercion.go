package analysis

import (
	"context"
	"fmt"
	"qml-lsp/lsp"

	sitter "github.com/smacker/go-tree-sitter"
)

type DiagnosticsJSEqualityCoercion struct{}

func (DiagnosticsJSEqualityCoercion) Analyze(ctx context.Context, fileURI string, fctx FileContext, engine *AnalysisEngine) (diags []lsp.Diagnostic) {
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(engine.Queries().CoercingEquality, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		diags = append(diags, lsp.Diagnostic{
			Range:    FromNode(match.Captures[0].Node).ToLSP(),
			Severity: lsp.SeverityInformation,
			Source:   `equality coercion`,
			Message:  fmt.Sprintf(`== may perform type coercion, leading to unexpected results. Consider using === instead.`),
		})
	}
	qc.Exec(engine.Queries().CoercingInequality, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		diags = append(diags, lsp.Diagnostic{
			Range:    FromNode(match.Captures[0].Node).ToLSP(),
			Severity: lsp.SeverityInformation,
			Source:   `inequality coercion`,
			Message:  fmt.Sprintf(`!= may perform type coercion, leading to unexpected results. Consider using !== instead.`),
		})
	}

	return diags
}
