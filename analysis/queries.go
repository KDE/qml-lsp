package analysis

import (
	qml "qml-lsp/treesitter-qml"

	sitter "github.com/smacker/go-tree-sitter"
)

type Queries struct {
	PropertyTypes                           *sitter.Query
	ObjectDeclarationTypes                  *sitter.Query
	WithStatements                          *sitter.Query
	ParentObjectChildPropertySets           *sitter.Query
	StatementBlocksWithVariableDeclarations *sitter.Query
	VariableAssignments                     *sitter.Query
	DoubleNegation                          *sitter.Query
	InlineComponents                        *sitter.Query
	CoercingEquality                        *sitter.Query
	CoercingInequality                      *sitter.Query
	AssignmentInCondition                   *sitter.Query
	JSInsideQML                             *sitter.Query
	Identifier                              *sitter.Query
	StrongScopes                            *sitter.Query
	ObjectDeclarations                      *sitter.Query
}

func (q *Queries) Init() error {
	var err error
	q.PropertyTypes, err = sitter.NewQuery([]byte("(property_declarator (property_type) @ident)"), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.ObjectDeclarationTypes, err = sitter.NewQuery([]byte("(object_declaration (qualified_identifier) @ident)"), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.ObjectDeclarations, err = sitter.NewQuery([]byte("(object_declaration) @ident"), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.WithStatements, err = sitter.NewQuery([]byte(`(with_statement "with" @bad)`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.ParentObjectChildPropertySets, err = sitter.NewQuery([]byte(`(object_declaration
		(qualified_identifier) @outer
		(object_block
			(object_declaration
				(object_block
					(property_set (qualified_identifier) @prop)))))`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.StatementBlocksWithVariableDeclarations, err = sitter.NewQuery([]byte(`
	(statement_block
		(variable_declaration
			"var" @keyword
			(variable_declarator name: (identifier) @name))
		(_)* @following)
`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.VariableAssignments, err = sitter.NewQuery([]byte(`
(assignment_expression left: (identifier) @ident)
	`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.DoubleNegation, err = sitter.NewQuery([]byte(`
(unary_expression operator: "!" argument: (unary_expression operator: "!" argument: (_) @arg)) @outer
	`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.InlineComponents, err = sitter.NewQuery([]byte(`
(inline_type_declaration
	(identifier) @name
	(qualified_identifier) @superclass
	(object_block) @body)
	`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.CoercingEquality, err = sitter.NewQuery([]byte(`
(binary_expression "==") @expression
	`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.CoercingInequality, err = sitter.NewQuery([]byte(`
(binary_expression "!=") @expression
	`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.AssignmentInCondition, err = sitter.NewQuery([]byte(`
(_ condition: (parenthesized_expression

	[
		(augmented_assignment_expression)
		(assignment_expression)
	] @assignment

))
	`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.JSInsideQML, err = sitter.NewQuery([]byte(`
[
	(normal_property value: [(expression) (script_statement)] @value )
	(readonly_property value: [(expression) (script_statement)] @value )
	(property_set value: [(expression) (script_statement)] @value )
]
`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.Identifier, err = sitter.NewQuery([]byte(`
(identifier) @identifier
`), qml.GetLanguage())
	if err != nil {
		return err
	}
	q.StrongScopes, err = sitter.NewQuery([]byte(`
[
	(statement_block) @scope
	(script_statement) @scope
	(program) @scope
	(inline_type_declaration) @scope
]
`), qml.GetLanguage())
	if err != nil {
		return err
	}
	return nil
}
