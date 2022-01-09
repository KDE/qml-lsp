package analysis

import (
	sitter "github.com/smacker/go-tree-sitter"
	"github.com/sourcegraph/go-lsp"
)

type PointRange struct {
	StartPoint sitter.Point
	EndPoint   sitter.Point
}

func FromNode(n *sitter.Node) PointRange {
	return PointRange{
		StartPoint: n.StartPoint(),
		EndPoint:   n.EndPoint(),
	}
}

func (p PointRange) ToLSP() lsp.Range {
	return lsp.Range{
		Start: lsp.Position{Line: int(p.StartPoint.Row), Character: int(p.StartPoint.Column)},
		End:   lsp.Position{Line: int(p.EndPoint.Row), Character: int(p.EndPoint.Column)}}
}
