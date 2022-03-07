package analysis

import sitter "github.com/smacker/go-tree-sitter"

type typingVisitor struct {
	*AnalysisEngine
	DefaultVisitor
}

func (s *AnalysisEngine) typeVariables(uri string, fctx *FileContext) {
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	k := typingVisitor{}
	k.DefaultAnswer = true

	Accept(fctx.Tree.RootNode(), &k)
}

func (s *AnalysisEngine) analyseFile(uri string, fctx *FileContext) {
	s.typeVariables(uri, fctx)
}
