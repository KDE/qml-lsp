package main

import (
	"io/ioutil"
	"os"
	"qml-lsp/analysis"
	"qml-lsp/analysis/cfg"
	"qml-lsp/qmltypes/qtquick"
	"strconv"

	"github.com/goccy/go-graphviz"
	"github.com/goccy/go-graphviz/cgraph"
	sitter "github.com/smacker/go-tree-sitter"
)

func graphToDot(cg *cfg.Graph, toWhere string) {
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

	nodes := map[cfg.NodeID]*cgraph.Node{}

	for _, node := range cg.Nodes {
		n, err := graph.CreateNode(strconv.FormatInt(int64(node.ID), 10))
		n.SetLabel(node.String())
		if err != nil {
			panic(err)
		}
		nodes[node.ID] = n
	}
	for _, edge := range cg.Edges {
		e, err := graph.CreateEdge(strconv.FormatInt(int64(edge.ID), 10), nodes[edge.From], nodes[edge.To])
		e.SetLabel(edge.Type.String())
		if err != nil {
			panic(err)
		}
	}

	if err := g.RenderFilename(graph, graphviz.PNG, toWhere); err != nil {
		panic(err)
	}
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
			graph := cfg.From(cap.Node)
			graphToDot(graph, os.Args[2]+strconv.FormatInt(i, 10)+".png")
			// println(cap.Node.Content(file))
			// println(cap.Node.String())
		}
	}
}
