package analysis

import sitter "github.com/smacker/go-tree-sitter"

type ASTVisitor interface {
	VisitProgram(*sitter.Node) bool
	EndVisitProgram(*sitter.Node)

	VisitPragma(*sitter.Node) bool
	EndVisitPragma(*sitter.Node)

	VisitImport(*sitter.Node) bool
	EndVisitImport(*sitter.Node)

	VisitRelativeImport(*sitter.Node) bool
	EndVisitRelativeImport(*sitter.Node)

	VisitObjectDeclaration(*sitter.Node) bool
	EndVisitObjectDeclaration(*sitter.Node)

	VisitObjectBlock(*sitter.Node) bool
	EndVisitObjectBlock(*sitter.Node)

	VisitPropertySet(*sitter.Node) bool
	EndVisitPropertySet(*sitter.Node)

	VisitPropertyValue(*sitter.Node) bool
	EndVisitPropertyValue(*sitter.Node)
}

type DefaultVisitor struct{ DefaultAnswer bool }

var _ ASTVisitor = &DefaultVisitor{}

func (d *DefaultVisitor) VisitProgram(*sitter.Node) bool {
	return d.DefaultAnswer
}

func (d *DefaultVisitor) EndVisitProgram(*sitter.Node) {

}

func (d *DefaultVisitor) VisitPragma(*sitter.Node) bool {
	return d.DefaultAnswer
}

func (d *DefaultVisitor) EndVisitPragma(*sitter.Node) {

}

func (d *DefaultVisitor) VisitImport(*sitter.Node) bool {
	return d.DefaultAnswer
}

func (d *DefaultVisitor) EndVisitImport(*sitter.Node) {

}

func (d *DefaultVisitor) VisitRelativeImport(*sitter.Node) bool {
	return d.DefaultAnswer
}

func (d *DefaultVisitor) EndVisitRelativeImport(*sitter.Node) {

}

func (d *DefaultVisitor) VisitObjectDeclaration(*sitter.Node) bool {
	return d.DefaultAnswer
}

func (d *DefaultVisitor) EndVisitObjectDeclaration(*sitter.Node) {

}

func (d *DefaultVisitor) VisitObjectBlock(*sitter.Node) bool {
	return d.DefaultAnswer
}

func (d *DefaultVisitor) EndVisitObjectBlock(*sitter.Node) {

}

func (d *DefaultVisitor) VisitPropertySet(*sitter.Node) bool {
	return d.DefaultAnswer
}

func (d *DefaultVisitor) EndVisitPropertySet(*sitter.Node) {

}

func (d *DefaultVisitor) VisitPropertyValue(*sitter.Node) bool {
	return d.DefaultAnswer
}

func (d *DefaultVisitor) EndVisitPropertyValue(*sitter.Node) {

}

func Accept(k *sitter.Node, v ASTVisitor) {
	switch k.Type() {
	case "program":
		if v.VisitProgram(k) {
			for i := 0; i < int(k.NamedChildCount()); i++ {
				Accept(k.NamedChild(i), v)
			}
		}
		v.EndVisitProgram(k)
	case "pragma_statement":
		v.VisitPragma(k)
		v.EndVisitPragma(k)
	case "import_statement":
		v.VisitImport(k)
		v.EndVisitImport(k)
	case "relative_import_statement":
		v.VisitRelativeImport(k)
		v.EndVisitRelativeImport(k)
	case "object_declaration":
		if v.VisitObjectDeclaration(k) {
			Accept(k.NamedChild(1), v)
		}
		v.EndVisitObjectDeclaration(k)
	case "object_block":
		if v.VisitObjectBlock(k) {
			for i := 0; i < int(k.NamedChildCount()); i++ {
				Accept(k.NamedChild(i), v)
			}
		}
		v.EndVisitObjectBlock(k)
	case "property_set":
		if v.VisitPropertySet(k) {
			println(k.ChildByFieldName("value").NextNamedSibling().String())
			Accept(k.ChildByFieldName("value").NextNamedSibling(), v)
		}
		v.EndVisitPropertySet(k)
	case "property_value":
		if v.VisitPropertyValue(k) {
			for i := 0; i < int(k.NamedChildCount()); i++ {
				Accept(k.NamedChild(i), v)
			}
		}
		v.EndVisitPropertyValue(k)
	default:
		panic("unhandled visit " + k.Type() + " in " + k.Parent().String())
	}
}
