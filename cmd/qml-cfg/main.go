package main

import (
	"fmt"
	"io/ioutil"
	"os"
	"qml-lsp/analysis"
	"qml-lsp/analysis/flow"
	"qml-lsp/qmltypes/qtquick"
	"regexp"
	"strconv"
	"strings"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	sitter "github.com/smacker/go-tree-sitter"
)

func recurseCreateNodes(it flow.FlowNode, g *cgraph.Graph, visited map[int]bool, content []byte) {
	if v, _ := visited[it.GetID()]; v {
		return
	}
	visited[it.GetID()] = true

	switch k := it.(type) {
	case *flow.FlowStart:
		e, err := g.CreateNode(strconv.Itoa(k.ID))
		if err != nil {
			panic(err)
		}
		e.SetLabel("Start")
	case *flow.FlowJoin:
		e, err := g.CreateNode(strconv.Itoa(k.ID))
		if err != nil {
			panic(err)
		}
		e.SetLabel("Join")
		for _, antecedent := range k.Antecedents {
			recurseCreateNodes(antecedent, g, visited, content)
		}
	case *flow.FlowAssignment:
		e, err := g.CreateNode(strconv.Itoa(k.ID))
		if err != nil {
			panic(err)
		}
		e.SetLabel(k.Node.Content(content))
		recurseCreateNodes(k.Antecedent, g, visited, content)
	case *flow.FlowCondition:
		e, err := g.CreateNode(strconv.Itoa(k.ID))
		if err != nil {
			panic(err)
		}
		e.SetLabel(fmt.Sprintf("%s is %t", k.Node.Content(content), k.AssumeTrue))
		recurseCreateNodes(k.Antecedent, g, visited, content)
	}
}

func recurseCreateEdges(it flow.FlowNode, g *cgraph.Graph, visited map[int]bool, content []byte) {
	if v, _ := visited[it.GetID()]; v {
		return
	}
	visited[it.GetID()] = true

	switch k := it.(type) {
	case *flow.FlowStart:
		// nothing
	case *flow.FlowJoin:
		for _, antecedent := range k.Antecedents {
			from, err := g.Node(strconv.Itoa(antecedent.GetID()))
			if err != nil {
				panic(err)
			}
			to, err := g.Node(strconv.Itoa(k.GetID()))
			if err != nil {
				panic(err)
			}
			g.CreateEdge(
				fmt.Sprintf("%d -> %d", antecedent.GetID(), k.GetID()),
				from,
				to,
			)
			recurseCreateEdges(antecedent, g, visited, content)
		}
	case *flow.FlowAssignment:
		from, err := g.Node(strconv.Itoa(k.Antecedent.GetID()))
		if err != nil {
			panic(err)
		}
		to, err := g.Node(strconv.Itoa(k.GetID()))
		if err != nil {
			panic(err)
		}
		g.CreateEdge(
			fmt.Sprintf("%d -> %d", k.Antecedent.GetID(), k.GetID()),
			from,
			to,
		)
		recurseCreateEdges(k.Antecedent, g, visited, content)
	case *flow.FlowCondition:
		from, err := g.Node(strconv.Itoa(k.Antecedent.GetID()))
		if err != nil {
			panic(err)
		}
		to, err := g.Node(strconv.Itoa(k.GetID()))
		if err != nil {
			panic(err)
		}
		g.CreateEdge(
			fmt.Sprintf("%d -> %d", k.Antecedent.GetID(), k.GetID()),
			from,
			to,
		)
		recurseCreateEdges(k.Antecedent, g, visited, content)
	}
}

func flowToDot(builder *flow.Builder, node *sitter.Node, toWhere string, content []byte) {
	g := graphviz.New()
	graph, err := g.Graph()
	if err != nil {
		panic(err)
	}

	defer func() {
		if err := graph.Close(); err != nil {
			panic(err)
		}
		g.Close()
	}()

	gnode, err := graph.CreateNode("title")
	if err != nil {
		panic(err)
	}
	gnode.SetShape(cgraph.SquareShape)
	gnode.SetLabel(Dedent(node.Content(content)))

	visited := map[int]bool{}
	recurseCreateNodes(builder.FlowNodes[node], graph, visited, content)
	visited = map[int]bool{}
	recurseCreateEdges(builder.FlowNodes[node], graph, visited, content)

	theNode, err := graph.Node(strconv.Itoa(builder.FlowNodes[node].GetID()))
	if err != nil {
		panic(err)
	}
	theNode.SetShape(cgraph.HexagonShape)

	if err := g.RenderFilename(graph, graphviz.PNG, toWhere); err != nil {
		panic(err)
	}
}

var (
	whitespaceOnly    = regexp.MustCompile("(?m)^[ \t]+$")
	leadingWhitespace = regexp.MustCompile("(?m)(^[ \t]*)(?:[^ \t\n])")
)

// Dedent removes any common leading whitespace from every line in text.
//
// This can be used to make multiline strings to line up with the left edge of
// the display, while still presenting them in the source code in indented
// form.
func Dedent(text string) string {
	var margin string

	text = whitespaceOnly.ReplaceAllString(text, "")
	indents := leadingWhitespace.FindAllStringSubmatch(text, -1)

	// Look for the longest leading string of spaces and tabs common to all
	// lines.
	for i, indent := range indents {
		if i == 0 {
			margin = indent[1]
		} else if strings.HasPrefix(indent[1], margin) {
			// Current line more deeply indented than previous winner:
			// no change (previous winner is still on top).
			continue
		} else if strings.HasPrefix(margin, indent[1]) {
			// Current line consistent with and no deeper than previous winner:
			// it's the new winner.
			margin = indent[1]
		} else {
			// Current line and previous winner have no common whitespace:
			// there is no margin.
			margin = ""
			break
		}
	}

	if margin != "" {
		text = regexp.MustCompile("(?m)^"+margin).ReplaceAllString(text, "")
	}
	return text
}

func main() {
	eng := analysis.New(qtquick.BuiltinModule())
	file, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		panic(err)
	}
	err = eng.SetFileContext(os.Args[1], file)
	if err != nil {
		panic(err)
	}
	ctx, _ := eng.GetFileContext(os.Args[1])

	qc := sitter.NewQueryCursor()
	qc.Exec(eng.Queries().JSInsideQML, ctx.Tree.RootNode())
	i := int64(0)
	for match, goNext := qc.NextMatch(); //
	goNext;                              //
	match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			builder := flow.New()
			builder.Build(cap.Node)

			for node := range builder.FlowNodes {
				flowToDot(builder, node, os.Args[2]+strconv.FormatInt(i, 10)+".png", file)
				i++
			}
		}
	}
}
