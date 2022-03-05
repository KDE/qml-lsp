package analysis

import (
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

type RefactorReplaceVarWithLetAndConst struct {
}

func (d *RefactorReplaceVarWithLetAndConst) Execute(r *Refactoring, uri string, engine *AnalysisEngine) error {
	fctx, err := engine.GetFileContext(uri)
	if err != nil {
		return fmt.Errorf("failed to refactor var -> let/const: %+w", err)
	}
	data := fctx.Body

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	ic := sitter.NewQueryCursor()
	defer ic.Close()

	replacements := replacementlist{}

	qc.Exec(engine.Queries().StatementBlocksWithVariableDeclarations, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		vname := match.Captures[1].Node.Content(data)
		keyword := match.Captures[0].Node
		remaining := match.Captures[2:]

		var isSet bool

	outer:
		for _, cap := range remaining {
			ic.Exec(engine.Queries().VariableAssignments, cap.Node)
			for imatch, igoNext := ic.NextMatch(); igoNext; imatch, igoNext = ic.NextMatch() {
				iname := imatch.Captures[0].Node.Content(data)

				if vname == iname {
					isSet = true
					break outer
				}
			}
		}

		if isSet {
			replacements = append(replacements, replaceSpan{
				start: keyword.StartByte(),
				end:   keyword.EndByte(),
				with:  "let",
			})
		} else {
			replacements = append(replacements, replaceSpan{
				start: keyword.StartByte(),
				end:   keyword.EndByte(),
				with:  "const",
			})
		}
	}

	engine.SetFileContext(uri, []byte(replacements.applyTo(string(data))))

	return nil
}
