package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"qml-lsp/analysis"
	"qml-lsp/qmltypes/qtquick"
	qml "qml-lsp/treesitter-qml"
	"qml-lsp/tsutils"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type queries struct {
	Properties          *sitter.Query `((comment) @comment . (property_declaration) @property)`
	PropertyIdentifier  *sitter.Query `(property_identifier) @property`
	PropertyType        *sitter.Query `(property_type) @type`
	Import              *sitter.Query `(import_statement (qualified_identifier) @ident)`
	QualifiedIdentifier *sitter.Query `(qualified_identifier) @qual`
	ClassName           *sitter.Query `(program (object_declaration (qualified_identifier) @ident)) @prog`
}

func qmlParser() *sitter.Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(qml.GetLanguage())

	return parser
}

func resolveName() string {
	base := path.Base(os.Args[1])

	qmldirLoc := path.Join(path.Dir(os.Args[1]), "qmldir")
	qmldir, err := ioutil.ReadFile(qmldirLoc)
	if err != nil && !errors.Is(err, os.ErrNotExist) {
		log.Fatalf("failed to read qmldir file: %s", err)
	}

	if len(qmldir) == 0 {
		return strings.TrimSuffix(base, path.Ext(base))
	}

	lines := strings.Split(string(qmldir), "\n")

	for _, line := range lines {
		if !strings.Contains(line, base) {
			continue
		}

		k := strings.Fields(line)

		switch len(k) {
		case 3:
			return k[0]
		case 4:
			return k[1]
		default:
			log.Fatalf("malformed qmldir line: '%s'", line)
		}
	}

	return strings.TrimSuffix(base, path.Ext(base))
}

func main() {
	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("could not open file %s: %s", os.Args[1], err)
	}

	eng := analysis.New(qtquick.BuiltinModule())
	eng.DoQMLPluginDump = false
	eng.SetFileContext(os.Args[1], data)

	parser := qmlParser()
	tree := parser.Parse(nil, data)

	q := queries{}
	err = tsutils.InitQueriesStructure(&q)
	if err != nil {
		log.Fatalf("failed to prepare queries: %s", err)
	}

	var className string
	{
		qc := sitter.NewQueryCursor()
		qc.Exec(q.ClassName, tree.RootNode())
		match, _ := qc.NextMatch()

		node := match.Captures[1].Node

		as := ""
		name := ""

		switch node.NamedChildCount() {
		case 1:
			name = node.NamedChild(0).Content(data)
		case 2:
			as = node.NamedChild(0).Content(data)
			name = node.NamedChild(1).Content(data)
		}

		comp, uri, _, err := eng.ResolveComponent(as, name, os.Args[1])
		split := strings.Split(uri.Path, ".")

		if err != nil {
			className = match.Captures[1].Node.Content(data)
		} else {
			className = fmt.Sprintf("%s::%s", strings.Join(split, "::"), comp.SaneName())
		}
	}

	// imports
	{
		qc := sitter.NewQueryCursor()
		qc.Exec(q.Import, tree.RootNode())

		for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
			node := match.Captures[0].Node
			s := make([]string, node.NamedChildCount())

			for i := uint32(0); i < node.NamedChildCount(); i++ {
				s[i] = node.NamedChild(int(i)).Content(data)
			}

			fmt.Printf("using namespace %s;\n", strings.Join(s, "::"))
		}
	}

	fmt.Printf("class %s : public %s {\npublic:\n", resolveName(), className)

	// properties
	{
		qc := sitter.NewQueryCursor()
		qc.Exec(q.Properties, tree.RootNode())

		for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
			comment := match.Captures[0].Node.Content(data)

			pname := sitter.NewQueryCursor()
			pname.Exec(q.PropertyIdentifier, match.Captures[1].Node)
			n, _ := pname.NextMatch()
			name := n.Captures[0].Node.Content(data)

			ptype := sitter.NewQueryCursor()
			ptype.Exec(q.PropertyType, match.Captures[1].Node)
			t, _ := ptype.NextMatch()
			kind := t.Captures[0].Node.Content(data)

			fmt.Printf("%s\nQ_PROPERTY(%s %s READ dummyGetter_%s_ignore)\n", comment, kind, name, name)
		}
	}

	fmt.Printf("};\n")
}
