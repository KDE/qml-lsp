package analysis

import (
	"qml-lsp/lsp"

	sitter "github.com/smacker/go-tree-sitter"
)

type PointRange struct {
	StartPoint sitter.Point
	EndPoint   sitter.Point
	StartByte  uint32
	EndByte    uint32
}

func FromNode(n *sitter.Node) PointRange {
	return PointRange{
		StartPoint: n.StartPoint(),
		EndPoint:   n.EndPoint(),
		StartByte:  n.StartByte(),
		EndByte:    n.EndByte(),
	}
}

func (p PointRange) ToLSP() lsp.Range {
	return lsp.Range{
		Start: lsp.Position{Line: uint32(p.StartPoint.Row), Character: uint32(p.StartPoint.Column)},
		End:   lsp.Position{Line: uint32(p.EndPoint.Row), Character: uint32(p.EndPoint.Column)}}
}
