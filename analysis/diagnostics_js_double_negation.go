package analysis

import (
	"context"
	"fmt"
	"qml-lsp/lsp"

	sitter "github.com/smacker/go-tree-sitter"
)

type DiagnosticsJSDoubleNegation struct{}

func (DiagnosticsJSDoubleNegation) Analyze(ctx context.Context, fileURI string, fctx FileContext, engine *AnalysisEngine) (diags []Diagnostic) {
	data := fctx.Body

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(engine.Queries().DoubleNegation, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		diags = append(diags, Diagnostic{
			Diagnostic: lsp.Diagnostic{
				Range:    FromNode(match.Captures[0].Node).ToLSP(),
				Severity: lsp.SeverityInformation,
				Source:   `double negation lint`,
				Message:  fmt.Sprintf(`Many people find double negation hard to read. Consider using "Boolean(%s)" instead.`, match.Captures[1].Node.Content(data)),
			},
			ContextNode: match.Captures[0].Node,
		})
	}

	return diags
}
