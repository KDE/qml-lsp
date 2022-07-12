package flow

import (
	sitter "github.com/smacker/go-tree-sitter"
)

type Builder struct {
	currentFlow    FlowNode
	breakTarget    *FlowJoin
	continueTarget *FlowJoin

	FlowNodes    map[*sitter.Node]FlowNode
	AllFlowNodes []FlowNode

	id int
}

func New() *Builder {
	var b Builder

	b.currentFlow = b.note(&FlowStart{b.nextID()})
	b.breakTarget = nil
	b.continueTarget = nil

	b.FlowNodes = map[*sitter.Node]FlowNode{}

	return &b
}

func (b *Builder) CurrentFlow() FlowNode {
	return b.currentFlow
}

func (b *Builder) nextID() int {
	b.id++
	return b.id
}

func (b *Builder) note(f FlowNode) FlowNode {
	b.AllFlowNodes = append(b.AllFlowNodes, f)
	return f
}
func (b *Builder) newFlowCondition(
	antecedent FlowNode,
	expression *sitter.Node,
	assumeTrue bool,
) FlowNode {
	if expression == nil {
		if assumeTrue {
			return antecedent
		} else {
			panic("unreachable")
		}
	}

	if expression.Type() == "true" && !assumeTrue || expression.Type() == "false" && assumeTrue {
		panic("unreachable")
	}

	if !isNarrowingExpression(expression) {
		return antecedent
	}

	return b.note(&FlowCondition{b.nextID(), antecedent, expression, assumeTrue})
}

func (b *Builder) newFlowUnreachable() FlowNode {
	return b.note(&FlowUnreachable{b.nextID(), false})
}

func (b *Builder) newFlowJoin() *FlowJoin {
	return b.note(&FlowJoin{b.nextID(), nil}).(*FlowJoin)
}

func (b *Builder) newFlowAssignment(antecedent FlowNode, node *sitter.Node) FlowNode {
	switch node.Type() {
	case "variable_declarator":
		break
	default:
		panic("bad flow assignment " + node.String())
	}
	return b.note(&FlowAssignment{b.nextID(), antecedent, node})
}

func isNarrowingExpression(
	expression *sitter.Node,
) bool {
	switch expression.Type() {
	case "identifier", "this", "member_expression":
		return true
	case "call_expression":
		return true
	case "parenthesized_expression":
		return isNarrowingExpression(expression.Child(1))
	case "binary_expression":
		return isNarrowingBinaryExpression(expression)
	default:
		return false
	}
}

func isNarrowingBinaryExpression(
	expression *sitter.Node,
) bool {
	left := expression.ChildByFieldName("left")
	right := expression.ChildByFieldName("right")

	switch expression.ChildByFieldName("operator").Type() {
	case "==", "!=", "===", "!===":
		return isNarrowingExpression(left) &&
			(right.Type() == "null" || right.Type() == "identifier")
	case "&&", "||":
		return isNarrowingExpression(left) || isNarrowingExpression(right)
	case "instanceof":
		return isNarrowingExpression(left)
	default:
		return false
	}
}

func (b *Builder) finishFlow(flow FlowNode) FlowNode {
outer:
	for {
		switch flw := flow.(type) {
		case *FlowJoin:
			if len(flw.Antecedents) == 0 {
				return b.newFlowUnreachable()
			}
			if len(flw.Antecedents) > 1 {
				break outer
			}
			flow = flw.Antecedents[0]
		default:
			break outer
		}
	}
	return flow
}

func (b *Builder) Build(node *sitter.Node) {
	b.buildChildren(node)
}

func (b *Builder) forEachBuild(node *sitter.Node) {
	for i := 0; i < int(node.NamedChildCount()); i++ {
		b.buildChildren(node.NamedChild(i))
	}
}

func (b *Builder) buildChildren(node *sitter.Node) {
	b.FlowNodes[node] = b.currentFlow
	switch node.Type() {
	case "while_statement":
		b.buildWhileStatement(node)
	case "lexical_declaration":
		b.buildLexicalDeclaration(node)
	case "variable_declarator":
		b.buildVariableDeclarator(node)
	case "if_statement":
		b.buildIfStatement(node)
	case "break_statement", "continue_statement":
		b.buildBreakOrContinueStatement(node)
	default:
		b.forEachBuild(node)
	}
}

func (b *Builder) buildLexicalDeclaration(node *sitter.Node) {
	b.forEachBuild(node)
}

func (b *Builder) buildVariableDeclarator(node *sitter.Node) {
	b.forEachBuild(node)
	if value := node.ChildByFieldName("value"); value != nil {
		b.currentFlow = b.newFlowAssignment(b.currentFlow, node)
	}
}

func (b *Builder) buildBreakOrContinueStatement(node *sitter.Node) {
	makeFlow := func(node *sitter.Node, breakTarget *FlowJoin, continueTarget *FlowJoin) {
		label := (*FlowJoin)(nil)
		if node.Type() == "break_statement" {
			label = breakTarget
		} else {
			label = continueTarget
		}
		if label != nil {
			label.addAntecedent(b.currentFlow)
			b.currentFlow = b.newFlowUnreachable()
		}
	}
	label := node.ChildByFieldName("label")
	if label != nil {
		b.Build(label)

		panic("flow for label not supported")
	} else {
		makeFlow(node, b.breakTarget, b.continueTarget)
	}
}

func (b *Builder) buildIfStatement(node *sitter.Node) {
	postIf := b.newFlowJoin()

	b.Build(node.ChildByFieldName("condition"))

	postConditionFlow := b.currentFlow
	b.currentFlow = b.newFlowCondition(b.currentFlow, node.ChildByFieldName("condition"), true)

	b.Build(node.ChildByFieldName("consequence"))
	postIf.addAntecedent(b.currentFlow)

	b.currentFlow = b.newFlowCondition(postConditionFlow, node.ChildByFieldName("condition"), false)

	if alternative := node.ChildByFieldName("alternative"); alternative != nil {
		b.Build(alternative)
	}

	postIf.addAntecedent(b.currentFlow)
	b.currentFlow = b.finishFlow(postIf)
}

func (b *Builder) buildWhileStatement(node *sitter.Node) {
	preWhile := b.newFlowJoin()
	postWhile := b.newFlowJoin()

	preWhile.addAntecedent(b.currentFlow)
	b.currentFlow = preWhile

	condition := node.ChildByFieldName("condition")

	b.Build(condition)
	postWhile.addAntecedent(b.newFlowCondition(b.currentFlow, condition, false))

	b.currentFlow = b.newFlowCondition(b.currentFlow, condition, true)

	saveBreak := b.breakTarget
	saveContinue := b.continueTarget

	b.breakTarget = postWhile
	b.continueTarget = preWhile

	body := node.ChildByFieldName("body")
	b.Build(body)

	b.breakTarget = saveBreak
	b.continueTarget = saveContinue

	preWhile.addAntecedent(b.currentFlow)
	b.currentFlow = b.finishFlow(postWhile)
}
