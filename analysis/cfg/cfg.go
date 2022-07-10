package cfg

import sitter "github.com/smacker/go-tree-sitter"

type NodeID int
type NodeType int
type EdgeType int
type EdgeID int

const (
	StartNode NodeType = iota
	GoodEndNode
	BadEndNode

	BlockOpenNode
	BlockCloseNode

	// a statement
	BodyNode

	// an expression
	ForkNode

	// no ast
	JoinNode
)

const (
	HaltEdge EdgeType = iota
	TruthConditionalEdge
	FalseConditionalEdge
	JoinEdge

	BodyEdge
)

type Graph struct {
	currentID     NodeID
	currentEdgeID EdgeID

	startNode   NodeID
	goodEndNode NodeID
	badEndNode  NodeID

	Nodes []Node
	Edges []Edge
}

func (g *Graph) getNextID() NodeID {
	g.currentID++
	return g.currentID
}

func (g *Graph) getNextEdgeID() EdgeID {
	g.currentEdgeID++
	return g.currentEdgeID
}

func (g *Graph) NodeByID(id NodeID) Node {
	for _, node := range g.Nodes {
		if node.ID == id {
			return node
		}
	}
	return Node{}
}

func (g *Graph) EdgeByID(id EdgeID) Edge {
	for _, node := range g.Edges {
		if node.ID == id {
			return node
		}
	}
	return Edge{}
}

func (g *Graph) IncomingEdges(id NodeID) (ret []Edge) {
	for _, edge := range g.Edges {
		if edge.To == id {
			ret = append(ret, edge)
		}
	}
	return ret
}

func (g *Graph) OutgoingEdges(id NodeID) (ret []Edge) {
	for _, edge := range g.Edges {
		if edge.From == id {
			ret = append(ret, edge)
		}
	}
	return ret
}

type Node struct {
	ID   NodeID
	Type NodeType
	AST  *sitter.Node
}

func (n *Node) String() string {
	if n.AST != nil && n.Type != BlockOpenNode && n.Type != BlockCloseNode {
		return n.Type.String() + "\n" + n.AST.String()
	}
	return n.Type.String()
}

func (n NodeType) String() string {
	switch n {
	case StartNode:
		return "start"
	case GoodEndNode:
		return "good end"
	case BadEndNode:
		return "bad end"
	case BodyNode:
		return "body"
	case ForkNode:
		return "fork"
	case JoinNode:
		return "join"
	case BlockOpenNode:
		return "block open"
	case BlockCloseNode:
		return "block close"
	default:
		panic("bad nodetype")
	}
}

func (e EdgeType) String() string {
	switch e {
	case HaltEdge:
		return "halt"
	case TruthConditionalEdge:
		return "on true"
	case FalseConditionalEdge:
		return "on false"
	case BodyEdge:
		return "body"
	case JoinEdge:
		return "join"
	default:
		panic("bad edge")
	}
}

type Edge struct {
	From NodeID
	To   NodeID
	Type EdgeType
	ID   EdgeID
}

func handleStatementBlock(tree *sitter.Node, graph *Graph, closer NodeID) (in NodeID, out NodeID) {
	var innerJoin NodeID
	open := graph.newNode(BlockOpenNode, tree)
	close := graph.newNode(BlockCloseNode, tree)

	for i := 0; i < int(tree.NamedChildCount()); i++ {
		child := tree.NamedChild(i)
		cometo, gofrom := handleStatement(child, graph, close)

		if i == 0 {
			in = cometo
		}

		if innerJoin != 0 {
			graph.connect(innerJoin, cometo, BodyEdge)
		}

		innerJoin = gofrom
		out = gofrom
	}

	if in != 0 {
		graph.connect(open, in, BodyEdge)
		in = open
	}
	if out != 0 {
		graph.connect(out, close, BodyEdge)
		out = close
	}

	return
}

func (g *Graph) connect(from NodeID, to NodeID, kind EdgeType) {
	if from == 0 {
		panic("connecting from nil node")
	} else if to == 0 {
		panic("connecting to nil node")
	} else if from == to {
		panic("self-connect")
	}
	g.Edges = append(g.Edges, Edge{
		From: from,
		To:   to,
		Type: kind,
		ID:   g.getNextEdgeID(),
	})
}

func (g *Graph) newNode(kind NodeType, tree *sitter.Node) NodeID {
	g.Nodes = append(g.Nodes, Node{
		ID:   g.getNextID(),
		Type: kind,
		AST:  tree,
	})
	return g.currentID
}

func handleStatement(tree *sitter.Node, graph *Graph, closer NodeID) (cometo NodeID, gofrom NodeID) {
	switch tree.Type() {
	case "lexical_declaration":
		graph.Nodes = append(graph.Nodes, Node{
			graph.getNextID(),
			BodyNode,
			tree,
		})
		return graph.currentID, graph.currentID
	case "expression_statement":
		graph.Nodes = append(graph.Nodes, Node{
			graph.getNextID(),
			BodyNode,
			tree,
		})
		return graph.currentID, graph.currentID
	case "return_statement":
		graph.Nodes = append(graph.Nodes, Node{
			graph.getNextID(),
			BodyNode,
			tree,
		})
		if closer == 0 {
			graph.connect(graph.currentID, graph.goodEndNode, HaltEdge)
		} else {
			graph.connect(graph.currentID, closer, BodyEdge)
			graph.connect(closer, graph.goodEndNode, HaltEdge)
		}
		return graph.currentID, 0
	case "throw_statement":
		graph.Nodes = append(graph.Nodes, Node{
			graph.getNextID(),
			BodyNode,
			tree,
		})
		if closer == 0 {
			graph.connect(graph.currentID, graph.badEndNode, HaltEdge)
		} else {
			graph.connect(graph.currentID, closer, BodyEdge)
			graph.connect(closer, graph.badEndNode, HaltEdge)
		}
		return graph.currentID, 0
	case "if_statement":
		graph.Nodes = append(graph.Nodes, Node{
			graph.getNextID(),
			ForkNode,
			tree.ChildByFieldName("condition"),
		})
		forkNode := graph.currentID

		graph.Nodes = append(graph.Nodes, Node{
			graph.getNextID(),
			JoinNode,
			nil,
		})
		joinNode := graph.currentID

		consequence := tree.ChildByFieldName("consequence")
		cometoCons, gofromCons := handleStatement(consequence, graph, closer)
		if cometoCons == 0 {
			graph.connect(forkNode, joinNode, TruthConditionalEdge)
		} else {
			graph.connect(forkNode, cometoCons, TruthConditionalEdge)
		}
		if gofromCons != 0 {
			graph.connect(gofromCons, joinNode, JoinEdge)
		}

		alternative := tree.ChildByFieldName("alternative")
		if alternative == nil {
			graph.connect(forkNode, joinNode, FalseConditionalEdge)
		} else {
			cometoAlt, gofromAlt := handleStatement(alternative.NamedChild(0), graph, closer)
			if cometoAlt == 0 {
				graph.connect(forkNode, joinNode, FalseConditionalEdge)
			} else {
				graph.connect(forkNode, cometoAlt, FalseConditionalEdge)
			}
			if gofromAlt != 0 {
				graph.connect(gofromAlt, joinNode, JoinEdge)
			}
		}

		return forkNode, joinNode
	case "while_statement":
		graph.Nodes = append(graph.Nodes, Node{
			graph.getNextID(),
			ForkNode,
			tree.ChildByFieldName("condition"),
		})
		forkNode := graph.currentID

		graph.Nodes = append(graph.Nodes, Node{
			graph.getNextID(),
			JoinNode,
			nil,
		})
		joinNode := graph.currentID

		body := tree.ChildByFieldName("body")
		cometoBody, gofromBody := handleStatement(body, graph, closer)
		graph.connect(forkNode, joinNode, FalseConditionalEdge)

		if cometoBody == 0 {
			graph.connect(forkNode, joinNode, TruthConditionalEdge)
		} else {
			graph.connect(forkNode, cometoBody, TruthConditionalEdge)
		}
		if gofromBody != 0 {
			graph.connect(gofromBody, forkNode, JoinEdge)
		}

		return forkNode, joinNode
	case "statement_block":
		return handleStatementBlock(tree, graph, closer)
	default:
		panic("unhandled statement type " + tree.Type() + " " + tree.String())
	}
}

func handleExpression(tree *sitter.Node, graph *Graph) {
	switch tree.Type() {
	default:
		panic("unhandled expression type " + tree.Type() + " " + tree.String())
	}
}

func handleRoot(tree *sitter.Node, graph *Graph) {
	if tree.Type() == "script_statement" {
		in, out := handleStatementBlock(tree.NamedChild(0), graph, 0)
		graph.connect(graph.startNode, in, BodyEdge)
		if out != 0 {
			graph.connect(out, graph.goodEndNode, BodyEdge)
		}
	} else {
		handleExpression(tree.NamedChild(0), graph)
	}
}

func (g *Graph) StartNode() NodeID {
	return g.startNode
}

func From(tree *sitter.Node) *Graph {
	ret := &Graph{}

	ret.Nodes = append(ret.Nodes, Node{ret.getNextID(), StartNode, nil})
	ret.startNode = ret.currentID
	ret.Nodes = append(ret.Nodes, Node{ret.getNextID(), GoodEndNode, nil})
	ret.goodEndNode = ret.currentID
	ret.Nodes = append(ret.Nodes, Node{ret.getNextID(), BadEndNode, nil})
	ret.badEndNode = ret.currentID

	handleRoot(tree, ret)

	return ret
}
