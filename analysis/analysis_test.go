package analysis_test

import (
	_ "embed"
	"qml-lsp/analysis"
	"qml-lsp/qmltypes/qtquick"
	"testing"
)

//go:embed test.qml
var testQML []byte

func TestVisitor(t *testing.T) {
	eng := analysis.New(qtquick.BuiltinModule())
	err := eng.SetFileContext("test.qml", testQML)
	if err != nil {
		t.Fatal(err)
	}
	fctx, err := eng.GetFileContext("test.qml")
	if err != nil {
		t.Fatal(err)
	}
	t.Fatal(fctx)
}
