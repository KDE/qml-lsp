package qmltypes_test

import (
	_ "embed"
	"qml-lsp/qmltypes"
	"testing"
)

//go:embed qtquick/builtins.qmltypes
var builtins string

//go:embed test/QtWebEngine.qmltypes
var webengine string

func TestRawParser(t *testing.T) {
	var (
		document qmltypes.QMLTypesFile
		err      error
	)

	var files = map[string]string{
		"test/builtins.qmltypes":    builtins,
		"test/QtWebEngine.qmltypes": webengine,
	}

	for file, content := range files {
		err = qmltypes.Parser.ParseString(file, content, &document)
		if err != nil {
			t.Fatalf("Failed to parse file %s: %s", file, err)
		}
	}
}
