package qmltypes_test

import (
	"qml-lsp/analysis"
	"qml-lsp/qmltypes"
	"testing"
)

type testDep struct {
	file    string
	content string
}

func TestUnmarshalLarge(t *testing.T) {
	var deps = []testDep{
		{
			file:    "test/builtins.qmltypes",
			content: builtins,
		},
		{
			file:    "test/QtWebEngine.qmltypes",
			content: webengine,
		},
	}
	for _, it := range deps {
		var document qmltypes.QMLTypesFile

		err := qmltypes.Parser.ParseString(it.file, it.content, &document)
		if err != nil {
			t.Fatalf("Failed to parse file %s: %s", it.file, err)
		}

		var modu analysis.Module
		err = qmltypes.Unmarshal(qmltypes.Value{Object: &document.Main}, &modu)

		if err != nil {
			t.Fatalf("Failed to unmarshal file %s: %s", it.file, err)
		}
	}
}
