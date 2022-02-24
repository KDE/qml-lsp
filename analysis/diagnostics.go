package analysis

import (
	"context"
	"qml-lsp/lsp"
)

type Diagnostics interface {
	Analyze(ctx context.Context, fileURI string, fctx FileContext, engine *AnalysisEngine) (diags []lsp.Diagnostic)
}
