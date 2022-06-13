package main

import (
	"bufio"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"qml-lsp/analysis"
	lspserver "qml-lsp/lsp-server"
	qml "qml-lsp/treesitter-qml"

	_ "qml-lsp/qt-libpaths"

	sitter "github.com/smacker/go-tree-sitter"
)

func main() {
	if len(os.Args) >= 3 {
		data, err := ioutil.ReadFile(os.Args[2])
		if err != nil {
			panic(err)
		}

		parser := analysis.QmlParser()
		tree := parser.Parse(nil, data)

		switch os.Args[1] {
		case "parse":
			println(tree.RootNode().String())
		case "query-repl":
			scanner := bufio.NewScanner(os.Stdin)

			for scanner.Scan() {
				var q *sitter.Query
				var e error
				if scanner.Text() == "r" {
					dat, _ := ioutil.ReadFile("query")
					q, e = sitter.NewQuery(dat, qml.GetLanguage())
				} else {
					q, e = sitter.NewQuery(scanner.Bytes(), qml.GetLanguage())
				}
				if e != nil {
					fmt.Printf("bad query: %s", e)
					continue
				}

				qc := sitter.NewQueryCursor()
				qc.Exec(q, tree.RootNode())

				for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
					for idx, cap := range match.Captures {
						println("capture", idx, cap.Node.String())
						println(cap.Node.Content(data))
					}
					if goNext {
						println("===")
					}
				}

				fmt.Printf("> ")
			}

			if err := scanner.Err(); err != nil {
				log.Fatal(err)
			}
		}
	} else {
		StartServer()
	}
}

func StartServer() {
	s := server{}
	a := lspserver.MethodMap{
		"initialize":                       lspserver.Zu(s.Initialize),
		"initialized":                      lspserver.Zu(s.Initialized),
		"textDocument/didOpen":             lspserver.Zu(s.DidOpen),
		"textDocument/didChange":           lspserver.Zu(s.DidChange),
		"textDocument/didClose":            lspserver.Zu(s.DidClose),
		"textDocument/completion":          lspserver.Zu(s.Completion),
		"workspace/didChangeWatchedFiles":  lspserver.Zu(s.DidChangeWatchedFiles),
		"textDocument/codeAction":          lspserver.Zu(s.CodeAction),
		"workspace/executeCommand":         lspserver.Zu(s.ExecuteCommand),
		"textDocument/documentLink":        lspserver.Zu(s.DocumentLink),
		"textDocument/codeLens":            lspserver.Zu(s.CodeLens),
		"textDocument/semanticTokens/full": lspserver.Zu(s.SemanticTokensFull),
		"textDocument/hover":               lspserver.Zu(s.Hover),
	}
	lspserver.StartServer(a)
}
