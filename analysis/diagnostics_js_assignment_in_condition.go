package analysis

import (
	"context"
	"fmt"
	"qml-lsp/lsp"

	sitter "github.com/smacker/go-tree-sitter"
)

type DiagnosticsJSAssignmentInCondition struct{}

func (DiagnosticsJSAssignmentInCondition) Analyze(ctx context.Context, fileURI string, fctx FileContext, engine *AnalysisEngine) (diags []Diagnostic) {
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(engine.Queries().AssignmentInCondition, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		diags = append(diags, Diagnostic{
			Diagnostic: lsp.Diagnostic{
				Range:    FromNode(match.Captures[0].Node).ToLSP(),
				Severity: lsp.SeverityWarning,
				Source:   `condition assignment`,
				Message:  fmt.Sprintf(`Avoid assigning to variables in conditions.`),
			},
			ContextNode: match.Captures[0].Node.Parent().Parent(),
		})
	}

	return diags
}
