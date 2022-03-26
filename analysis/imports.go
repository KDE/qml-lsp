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
	if node == nil {
		return -1, -1
	}
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

			uri = child.ChildByFieldName("uri").Content(b)
			uri = uri[1 : len(uri)-1]

			if field := child.ChildByFieldName("alias"); field != nil {
				as = field.Content(b)
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

		maj, min := extractVersionNumber(child.ChildByFieldName("number"), b)
		if maj == -1 || min == -1 {
			continue
		}
		import_ := ASTImport{
			Module:     ExtractQualifiedIdentifier(child.ChildByFieldName("uri"), b),
			MajVersion: maj,
			MinVersion: min,
			Range:      FromNode(child),
		}
		if alias := child.ChildByFieldName("alias"); alias != nil {
			import_.As = alias.ChildByFieldName("aliasName").Content(b)
		}

		d = append(d, import_)
	}
	return d, u
}
