package analysis

import (
	"context"
	"qml-lsp/lsp"
)

type DiagnosticsQMLUnusedImports struct{}

func (DiagnosticsQMLUnusedImports) Analyze(ctx context.Context, fileURI string, fctx FileContext, engine *AnalysisEngine) (diags []lsp.Diagnostic) {
	imports := fctx.Imports
	used, err := engine.UsedImports(fileURI, fctx.Tree.RootNode())
	if err != nil {
		return nil
	}

	// now let's go through our imports and raise warnings for any unused imports
	for idx, importData := range imports {
		isUsed := used[idx]

		if isUsed {
			continue
		}

		// oops, this import isn't used! let's raise a diagnostic...
		diags = append(diags, lsp.Diagnostic{
			Range:    importData.Range.ToLSP(),
			Severity: lsp.SeverityWarning,
			Source:   "import lint",
			Message:  "Unused import",
		})
	}

	return diags
}
