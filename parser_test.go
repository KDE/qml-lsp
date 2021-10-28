package main

import (
	_ "embed"
	"testing"
)

//go:embed test/builtins.qmltypes
var builtins string

//go:embed test/QtWebEngine.qmltypes
var webengine string

func TestRawParser(t *testing.T) {
	var (
		document QMLTypesFile
		err      error
	)

	var files = map[string]string{
		"test/builtins.qmltypes":    builtins,
		"test/QtWebEngine.qmltypes": webengine,
	}

	for file, content := range files {
		err = parser.ParseString(file, content, &document)
		if err != nil {
			t.Fatalf("Failed to parse file %s: %s", file, err)
		}
	}
}
