package main

import (
	"errors"
	"fmt"
	"io/ioutil"
	"log"
	"os"
	"path"
	"path/filepath"
	"qml-lsp/analysis"
	"qml-lsp/qmltypes/qtquick"
	qml "qml-lsp/treesitter-qml"
	"qml-lsp/tsutils"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type queries struct {
	Properties          *sitter.Query `((comment)? @comment . (property_declaration) @property)`
	PropertyIdentifier  *sitter.Query `(property_identifier) @property`
	PropertyType        *sitter.Query `(property_type) @type`
	Import              *sitter.Query `(import_statement (qualified_identifier) @ident)`
	QualifiedIdentifier *sitter.Query `(qualified_identifier) @qual`
	ClassName           *sitter.Query `(program (object_declaration (qualified_identifier) @ident)) @prog`
	ClassComment        *sitter.Query `(program (comment) @comment . (object_declaration (qualified_identifier)))`
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

func extractSuperclass(q *queries, tree *sitter.Tree, eng *analysis.AnalysisEngine, data []byte) (as, name, full string) {
	qc := sitter.NewQueryCursor()
	qc.Exec(q.ClassName, tree.RootNode())
	match, _ := qc.NextMatch()

	node := match.Captures[1].Node

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
		return as, name, match.Captures[1].Node.Content(data)
	}

	return as, name, fmt.Sprintf("%s::%s", strings.Join(split, "::"), comp.SaneName())
}

func dumpProperties(tree *sitter.Tree, q *queries, data []byte) {
	qc := sitter.NewQueryCursor()
	qc.Exec(q.Properties, tree.RootNode())

	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		var (
			comment, kind, name string
			propIdx             int
		)

		switch len(match.Captures) {
		case 1:
			propIdx = 0
		case 2:
			comment = match.Captures[0].Node.Content(data)
			propIdx = 1
		}

		pname := sitter.NewQueryCursor()
		pname.Exec(q.PropertyIdentifier, match.Captures[propIdx].Node)
		n, _ := pname.NextMatch()
		name = n.Captures[0].Node.Content(data)

		ptype := sitter.NewQueryCursor()
		ptype.Exec(q.PropertyType, match.Captures[propIdx].Node)
		t, _ := ptype.NextMatch()
		kind = t.Captures[0].Node.Content(data)

		fmt.Printf("%s\nQ_PROPERTY(%s %s READ dummyGetter_%s_ignore)\n", comment, kind, name, name)
	}
}

func main() {
	data, err := ioutil.ReadFile(os.Args[1])
	if err != nil {
		log.Fatalf("could not open file %s: %s", os.Args[1], err)
	}

	eng := analysis.New(qtquick.BuiltinModule())
	eng.DoQMLPluginDump = false
	eng.SetFileContext(os.Args[1], data)
	fctx, _ := eng.GetFileContext(os.Args[1])

	parser := qmlParser()
	tree := parser.Parse(nil, data)

	q := queries{}
	err = tsutils.InitQueriesStructure(&q)
	if err != nil {
		log.Fatalf("failed to prepare queries: %s", err)
	}

	from, classNameOnly, className := extractSuperclass(&q, tree, eng, data)

	var classComment string
	var splat bool
	var fromSplat string
	{
		qc := sitter.NewQueryCursor()
		qc.Exec(q.ClassComment, tree.RootNode())
		match, _ := qc.NextMatch()
		if match == nil {
			goto brk
		}

		classComment = match.Captures[0].Node.Content(data)
		splat = strings.Contains(classComment, "@splat")
		if splat {
			classComment = strings.ReplaceAll(classComment, "@splat", "")
		}
	}

brk:
	for _, impt := range fctx.Imports {
		if impt.As == from && impt.URI.IsRelativePath {
			if strings.Contains(impt.URI.Path, "template") ||
				strings.Contains(impt.URI.Path, "impl") ||
				strings.Contains(impt.URI.Path, "private") {
				splat = true
			}
			fromSplat = path.Join(impt.URI.Path, classNameOnly+".qml")
			break
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

	fmt.Println(classComment)

	var splatTree *sitter.Tree
	var splatData []byte

	if splat {
		target := fromSplat
		if target == "" {
			dir := path.Dir(os.Args[1])
			it := path.Join(dir, fmt.Sprintf("%s.qml", classNameOnly))

			list, err := filepath.Glob(it)
			if err != nil {
				println("splat failed: " + err.Error())
				goto brk2
			}

			fmt.Fprintf(os.Stderr, "%s %+v", it, list)
			for _, item := range list {
				_, err := os.Stat(item)
				if err == nil {
					target = item
					break
				}
				if err != nil && !os.IsNotExist(err) {
					println("splat failed: " + err.Error())
					goto brk2
				}
			}
		}
		_, err := os.Stat(target)
		if err != nil && !os.IsNotExist(err) {
			println("splat failed: " + err.Error())
			goto brk2
		}

		splatData, err = ioutil.ReadFile(target)
		if err != nil {
			log.Fatalf("could not open splat file %s: %s", target, err)
		}
		eng.SetFileContext(target, splatData)
		splatTree = parser.Parse(nil, splatData)

		_, _, className = extractSuperclass(&q, splatTree, eng, splatData)
	}
brk2:

	fmt.Printf("class %s : public %s {\npublic:\n", resolveName(), className)

	// properties
	if splatTree != nil {
		dumpProperties(splatTree, &q, splatData)
	}

	dumpProperties(tree, &q, data)

	fmt.Printf("};\n")
}
