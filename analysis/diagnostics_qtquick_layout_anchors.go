package analysis

import (
	"context"
	"qml-lsp/lsp"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type DiagnosticsQtQuickLayoutAnchors struct{}

var anchorsInLayoutWarnings = map[string]string{
	"anchors.alignWhenCentered":      `Don't use anchors.alignWhenCentered in a {{kind}}. Layouts always pixel-align their items, so tihs is unneccesary.`,
	"anchors.baseline":               `Don't use anchors.baseline in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignBaseline"`,
	"anchors.baselineOffset":         `Don't use anchors.baselineOffset in a {{kind}}. Instead, consider setting the "{{pfx}}Layout.bottomMargin".`,
	"anchors.bottom":                 `Don't use anchors.bottom in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignBottom"`,
	"anchors.bottomMargin":           `Don't use anchors.bottomMargin in a {{kind}}. Instead, consider setting the "{{pfx}}Layout.bottomMargin"`,
	"anchors.centerIn":               `Don't use anchors.centerIn in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignVCenter | Qt.AlignHCenter"`,
	"anchors.fill":                   `Don't use anchors.fill in a {{kind}}. Instead, consider using "{{pfx}}Layout.fillWidth: true" and "{{pfx}}Layout.fillHeight: true"`,
	"anchors.horizontalCenter":       `Don't use anchors.horizontalCenter in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignHCenter"`,
	"anchors.horizontalCenterOffset": `Don't use anchors.horizontalCenterOffset in a {{kind}}. Instead, consider using "{{pfx}}Layout.leftMargin" or "{{pfx}}Layout.rightMargin"`,
	"anchors.left":                   `Don't use anchors.left in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignLeft"`,
	"anchors.leftMargin":             `Don't use anchors.leftMargin in a {{kind}}. Instead, consider using "{{pfx}}Layout.leftMargin"`,
	"anchors.margins":                `Don't use anchors.margins in a {{kind}}. Instead, consider using "{{pfx}}Layout.margins"`,
	"anchors.right":                  `Don't use anchors.right in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignRight"`,
	"anchors.rightMargin":            `Don't use anchors.rightMargin in a {{kind}}. Instead, consider using "{{pfx}}Layout.rightMargin"`,
	"anchors.top":                    `Don't use anchors.top in a {{kind}}. Instead, consider using "{{pfx}}Layout.alignment: Qt.AlignTop"`,
	"anchors.topMargin":              `Don't use anchors.topMargin in a {{kind}}. Instead, consider using "{{pfx}}Layout.topMargin"`,
	"anchors.verticalCenter":         `Don't use anchors.verticalCenter in a {{kind}}. Instead, consider using "{{pfx}}Layout.horizontalAlignment: Qt.AlignHCenter"`,
	"anchors.verticalCenterOffset":   `Don't use anchors.verticalCenterOffset in a {{kind}}. Instead, consider using "{{pfx}}Layout.topMargin" or "{{pfx}}Layout.bottomMargin"`,
}

func (DiagnosticsQtQuickLayoutAnchors) Analyze(ctx context.Context, fileURI string, fctx FileContext, engine *AnalysisEngine) (diags []lsp.Diagnostic) {
	data := fctx.Body
	imports := fctx.Imports

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(engine.Queries().ParentObjectChildPropertySets, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		parentType := match.Captures[0].Node.Content(data)
		childProperty := match.Captures[1].Node.Content(data)

		if !strings.HasPrefix(childProperty, "anchors") {
			continue
		}

		for _, item := range imports {
			if item.URI.Path != "QtQuick.Layouts" {
				continue
			}

			pfx := ""
			if item.As != "" {
				pfx = item.As + "."
			}

			for _, comp := range item.Module.Components {
				if pfx+comp.SaneName() != parentType {
					continue
				}

				v, ok := anchorsInLayoutWarnings[childProperty]
				if !ok {
					continue
				}

				diags = append(diags, lsp.Diagnostic{
					Range:    FromNode(match.Captures[1].Node).ToLSP(),
					Severity: lsp.SeverityError,
					Source:   "anchors in layouts lint",
					Message:  strings.ReplaceAll(strings.ReplaceAll(v, "{{kind}}", pfx+comp.SaneName()), "{{pfx}}", pfx),
				})
			}
		}
	}

	return diags
}
