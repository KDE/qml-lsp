package main

import (
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

func extractQualifiedIdentifier(node *sitter.Node, b []byte) []string {
	var s []string
	for i := 0; i < int(node.ChildCount()); i++ {
		child := node.Child(i)
		if child.Type() != "identifier" {
			continue
		}
		s = append(s, child.Content(b))
	}
	return s
}

func mustInt(s string) int {
	i, _ := strconv.Atoi(s)
	return i
}

func extractVersionNumber(node *sitter.Node, b []byte) (int, int) {
	it := strings.Split(node.Content(b), ".")
	return mustInt(it[0]), mustInt(it[1])
}

func extractImports(root *sitter.Node, b []byte) []importData {
	var d []importData
	for i := 0; i < int(root.ChildCount()); i++ {
		child := root.Child(i)
		if child.Type() != "import_statement" {
			continue
		}
		switch child.ChildCount() {
		case 3:
			maj, min := extractVersionNumber(child.Child(2), b)
			d = append(d, importData{
				Module:     extractQualifiedIdentifier(child.Child(1), b),
				MajVersion: maj,
				MinVersion: min,
			})
		case 4:
			maj, min := extractVersionNumber(child.Child(2), b)
			d = append(d, importData{
				Module:     extractQualifiedIdentifier(child.Child(1), b),
				MajVersion: maj,
				MinVersion: min,
				As:         child.Child(3).Child(1).Content(b),
			})
		}
	}
	return d
}

type importData struct {
	Module     []string
	MajVersion int
	MinVersion int
	As         string
}

func (i *importData) moduleString() string {
	return strings.Join(i.Module, ".")
}
