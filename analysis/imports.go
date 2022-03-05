package analysis

import (
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type ASTImport struct {
	Module     []string
	MajVersion int
	MinVersion int
	As         string

	// we use this to lint for unused imports
	Range PointRange
}

type URIImport struct {
	Path string
	As   string

	Range PointRange
}

func (i *ASTImport) ModuleString() string {
	return strings.Join(i.Module, ".")
}

func ExtractQualifiedIdentifier(node *sitter.Node, b []byte) []string {
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

func ExtractImports(root *sitter.Node, b []byte) ([]ASTImport, []URIImport) {
	var d []ASTImport
	var u []URIImport
	for i := 0; i < int(root.NamedChildCount()); i++ {
		child := root.NamedChild(i)
		if child.HasError() {
			continue
		}
		if child.Type() == "relative_import_statement" {
			var uri string
			var as string

			switch child.NamedChildCount() {
			case 2:
				as = child.NamedChild(1).NamedChild(0).Content(b)
				fallthrough
			case 1:
				uri = child.NamedChild(0).Content(b)
				uri = uri[1 : len(uri)-1]
			}

			u = append(u, URIImport{
				Path:  uri,
				As:    as,
				Range: FromNode(child),
			})
			continue
		}
		if child.Type() != "import_statement" {
			continue
		}
		switch child.NamedChildCount() {
		case 2:
			maj, min := extractVersionNumber(child.NamedChild(1), b)
			d = append(d, ASTImport{
				Module:     ExtractQualifiedIdentifier(child.NamedChild(0), b),
				MajVersion: maj,
				MinVersion: min,
				Range:      FromNode(child),
			})
		case 3:
			maj, min := extractVersionNumber(child.NamedChild(1), b)
			d = append(d, ASTImport{
				Module:     ExtractQualifiedIdentifier(child.NamedChild(0), b),
				MajVersion: maj,
				MinVersion: min,
				As:         child.NamedChild(2).NamedChild(0).Content(b),
				Range:      FromNode(child),
			})
		}
	}
	return d, u
}
