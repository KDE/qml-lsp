package flow

import (
	"fmt"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type FlowNode interface {
	isFlowNode()
	GetID() int
	Debug() string
}

// FlowUnreachable represents a flow node that's unreachable
type FlowUnreachable struct {
	ID         int
	Reportable bool
}

func (f *FlowUnreachable) isFlowNode() {}
func (f *FlowUnreachable) GetID() int {
	return f.ID
}
func (f *FlowUnreachable) Debug() string {
	return "unreachable"
}

// FlowStart represents a flow node that's the first one
type FlowStart struct {
	ID int
}

func (f *FlowStart) isFlowNode() {}
func (f *FlowStart) GetID() int {
	return f.ID
}
func (f *FlowStart) Debug() string {
	return "start"
}

// FlowJoin represents a flow node where multiple nodes can join into
// one single node
type FlowJoin struct {
	ID          int
	Antecedents []FlowNode
}

func (f *FlowJoin) isFlowNode() {}
func (f *FlowJoin) GetID() int {
	return f.ID
}
func (f *FlowJoin) addAntecedent(to FlowNode) {
	f.Antecedents = append(f.Antecedents, to)
}
func (f *FlowJoin) Debug() string {
	var ids []string
	for _, a := range f.Antecedents {
		ids = append(ids, strconv.Itoa(a.GetID()))
	}
	return fmt.Sprintf("join from [%s]", strings.Join(ids, ", "))
}

var _ FlowNode = &FlowJoin{}

// FlowAssignment represents a flow node where a value might be assigned
// to one or more identifiers
type FlowAssignment struct {
	ID         int
	Antecedent FlowNode

	Node *sitter.Node
}

func (f *FlowAssignment) isFlowNode() {}
func (f *FlowAssignment) GetID() int {
	return f.ID
}
func (f *FlowAssignment) Debug() string {
	return fmt.Sprintf("assign from %d", f.Antecedent.GetID())
}

// FlowCondition represents a conditional node
type FlowCondition struct {
	ID         int
	Antecedent FlowNode

	Node       *sitter.Node
	AssumeTrue bool
}

func (f *FlowCondition) isFlowNode() {}
func (f *FlowCondition) GetID() int {
	return f.ID
}
func (f *FlowCondition) Debug() string {
	return fmt.Sprintf("condition from %d", f.Antecedent.GetID())
}
