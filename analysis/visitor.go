package analysis

import sitter "github.com/smacker/go-tree-sitter"

type ASTVisitor interface {
	Visit(*sitter.Node) bool
	EndVisit(*sitter.Node)
}

type DefaultVisitor struct{ DefaultAnswer bool }

var _ ASTVisitor = &DefaultVisitor{}

func (d *DefaultVisitor) Visit(*sitter.Node) bool {
	return d.DefaultAnswer
}

func (d *DefaultVisitor) EndVisit(*sitter.Node) {

}

func Walk(k *sitter.Node, v ASTVisitor) {
	if !v.Visit(k) {
		return
	}
	defer v.EndVisit(k)

	switch k.Type() {
	case "program", "object_block", "property_value":
		for i := 0; i < int(k.NamedChildCount()); i++ {
			Walk(k.NamedChild(i), v)
		}
	case "pragma_statement", "import_statement", "relative_import_statement":
		break
	case "object_declaration":
		Walk(k.NamedChild(1), v)
	case "property_set":
		Walk(k.ChildByFieldName("value").NextNamedSibling(), v)
	default:
		panic("unhandled visit " + k.Type() + " in " + k.Parent().String())
	}
}
