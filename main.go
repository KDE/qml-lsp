package main

import (
	"io/ioutil"
	"os"
	qml "qml-lsp/treesitter-qml"

	_ "qml-lsp/qt-libpaths"

	sitter "github.com/smacker/go-tree-sitter"
)

func qmlParser() *sitter.Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(qml.GetLanguage())

	return parser
}

func main() {
	if len(os.Args) >= 3 && os.Args[1] == "parse" {
		data, err := ioutil.ReadFile(os.Args[2])
		if err != nil {
			panic(err)
		}

		parser := qmlParser()
		tree := parser.Parse(nil, data)
		println(tree.RootNode().String())
	} else {
		StartServer()
	}
}
