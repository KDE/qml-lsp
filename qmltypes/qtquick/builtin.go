package qtquick

import (
	_ "embed"
	"qml-lsp/analysis"
	"qml-lsp/qmltypes"
)

//go:embed builtins.qmltypes
var builtin string

var builtinM analysis.Module

func init() {
	var d qmltypes.QMLTypesFile

	err := qmltypes.Parser.ParseString("builtin", builtin, &d)
	if err != nil {
		panic(err)
	}

	err = qmltypes.Unmarshal(qmltypes.Value{Object: &d.Main}, &builtinM)
	if err != nil {
		panic(err)
	}
}

func BuiltinModule() analysis.Module {
	return builtinM
}
