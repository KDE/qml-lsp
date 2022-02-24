package analysis

import (
	"context"
	"qml-lsp/lsp"

	sitter "github.com/smacker/go-tree-sitter"
)

type Diagnostics interface {
	Analyze(ctx context.Context, fileURI string, fctx FileContext, engine *AnalysisEngine) (diags []Diagnostic)
}

type Diagnostic struct {
	lsp.Diagnostic

	ContextNode *sitter.Node
}
