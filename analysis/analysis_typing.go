package analysis

import (
	"errors"
	"fmt"

	sitter "github.com/smacker/go-tree-sitter"
)

func (s *AnalysisEngine) lookupEnv(fctx *FileContext, node *sitter.Node, name string) (TypeURI, bool) {
	for n := node; n != nil; n = n.Parent() {
		d := fctx.Tree.Data[n]
		if d.Types == nil {
			continue
		}
		v, found := d.Types[name]
		if found {
			return v, true
		}
	}
	return TypeURI{}, false
}

func (s *AnalysisEngine) setEnv(fctx *FileContext, node *sitter.Node, name string, value TypeURI) {
	for n := node; n != nil; n = n.Parent() {
		d := fctx.Tree.Data[n]
		if !d.IsStrongScope || d.Types == nil {
			continue
		}
		d.Types[name] = value
		return
	}
	panic("couldn't set an env")
}

func (s *AnalysisEngine) setEnvWeak(fctx *FileContext, node *sitter.Node, name string, value TypeURI) {
	for n := node; n != nil; n = n.Parent() {
		d := fctx.Tree.Data[n]
		if !(d.IsWeakScope || d.IsStrongScope) || d.Types == nil {
			continue
		}
		d.Types[name] = value
		return
	}
	panic("couldn't set a weak env")
}

func (s *AnalysisEngine) typeOfExpression(uri string, fctx *FileContext, node *sitter.Node) (turi TypeURI, terr error) {
	defer func() {
		if terr == nil {
			v := fctx.Tree.Data[node]
			v.Kind = turi
			fctx.Tree.Data[node] = v
		}
	}()
	if v := fctx.Tree.Data[node].Kind; v != (TypeURI{}) {
		return v, nil
	}
	switch node.Type() {
	case "number":
		return NumberURI, nil
	case "string":
		return StringURI, nil
	case "true":
		return BooleanURI, nil
	case "false":
		return BooleanURI, nil
	case "identifier":
		var_, found := s.lookupEnv(fctx, node, node.Content(fctx.Body))
		if !found {
			return TypeURI{}, fmt.Errorf("variable %s not found", node.Content(fctx.Body))
		}
		return var_, nil
	case "ternary_expression":
		mid, err := s.typeOfExpression(uri, fctx, node.ChildByFieldName("condition"))
		if err != nil {
			return TypeURI{}, fmt.Errorf("failed to type ternary because of an error in the condition: %w", err)
		}
		lhs, err := s.typeOfExpression(uri, fctx, node.ChildByFieldName("consequence"))
		if err != nil {
			return TypeURI{}, fmt.Errorf("failed to type ternary because of an error in the left-hand side: %w", err)
		}
		rhs, err := s.typeOfExpression(uri, fctx, node.ChildByFieldName("alternative"))
		if err != nil {
			return TypeURI{}, fmt.Errorf("failed to type ternary because of an error in the right-hand side: %w", err)
		}
		if mid != BooleanURI {
			// TODO: flag an error
		}
		if lhs != rhs {
			// TODO: flag an error
		}
		return lhs, nil
	case "parenthesized_expression":
		return s.typeOfExpression(uri, fctx, node.Child(1))
	default:
		return TypeURI{}, errors.New("typing this expression isn't implemented yet: " + node.Type())
	}
}

func (s *AnalysisEngine) typeVariablesInner(uri string, fctx *FileContext, node *sitter.Node) {
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(s.queries.Identifier, node)
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		if match.Captures[0].Node.Parent().Type() == "variable_declarator" {
			name := match.Captures[0].Node.Content(fctx.Body)
			value := match.Captures[0].Node.Parent().ChildByFieldName("value")
			if value == nil {
				// TODO: flag an issue
				continue
			}

			k, err := s.typeOfExpression(uri, fctx, value)
			if err != nil {
				println(err.Error())
				// TODO: flag an issue
				continue
			}

			s.setEnv(fctx, node, name, k)
		}
		identNode := match.Captures[0].Node
		data := fctx.Tree.Data[identNode]
		var_, found := s.lookupEnv(fctx, identNode, identNode.Content(fctx.Body))
		if found {
			data.Kind = var_
		}
		fctx.Tree.Data[identNode] = data
	}
}

func (s *AnalysisEngine) typeVariables(uri string, fctx *FileContext) {
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(s.queries.JSInsideQML, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		s.typeVariablesInner(uri, fctx, match.Captures[0].Node)
	}
}

func (s *AnalysisEngine) markScopes(uri string, fctx *FileContext) {
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(s.queries.StrongScopes, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			k := fctx.Tree.Data[cap.Node]
			if k.Types == nil {
				k.Types = map[string]TypeURI{}
			}
			k.IsStrongScope = true
			fctx.Tree.Data[cap.Node] = k
		}
	}
}

func (s *AnalysisEngine) typeObjects(uri string, fctx *FileContext) {
	qc := sitter.NewQueryCursor()
	defer qc.Close()

	qc.Exec(s.queries.ObjectDeclarations, fctx.Tree.RootNode())
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			k := fctx.Tree.Data[cap.Node]
			if k.Types == nil {
				k.Types = map[string]TypeURI{}
			}
			k.IsWeakScope = true
			fctx.Tree.Data[cap.Node] = k

			name := cap.Node.NamedChild(0).Content(fctx.Body)

			doComponents := func(prefix string, components []Component) {
				for _, component := range components {
					if prefix+component.SaneName() == name {
						for _, prop := range component.Properties {
							// TODO: handle non-primitives
							s.setEnvWeak(fctx, cap.Node, prop.Name, TypeURI{
								Path:         "",
								MajorVersion: 0,
								Name:         prop.Type,
								ReactiveList: prop.IsList,
							})
						}
					}
					if component.AttachedType == "" {
						continue
					}
					// TODO: make attached properties type

					// for _, comp := range components {
					// 	if comp.Name != component.AttachedType {
					// 		continue
					// 	}

					// 	for _, prop := range comp.Properties {
					// 		fullName := prefix + component.SaneName() + "." + prop.Name
					// 		println(fullName, w)
					// 		if !strings.HasPrefix(fullName, w) {
					// 			continue
					// 		}

					// 		citems = append(citems, lsp.CompletionItem{
					// 			Label:      fullName,
					// 			Kind:       lsp.PropertyCompletion,
					// 			Detail:     fmt.Sprintf("attached %s", prefix+component.SaneName()),
					// 			InsertText: strings.TrimPrefix(fullName+": ", w),
					// 		})
					// 	}
					// }
				}
			}

			doComponents("", s.BuiltinModule().Components)

			for _, module := range fctx.Imports {
				if module.As == "" {
					doComponents("", module.Module.Components)
				} else {
					doComponents(module.As+".", module.Module.Components)
				}
			}
		}
	}
}

func (s *AnalysisEngine) analyseFile(uri string, fctx *FileContext) {
	s.markScopes(uri, fctx)
	s.typeObjects(uri, fctx)
	s.typeVariables(uri, fctx)
}
