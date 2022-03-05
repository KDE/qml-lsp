package main

import (
	"fmt"
	"io/fs"
	"io/ioutil"
	"os"
	"path/filepath"
	"qml-lsp/analysis"
	"qml-lsp/qmltypes/qtquick"
	"strconv"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
	"github.com/urfave/cli/v2"
)

func whatUses(ctx *cli.Context) error {
	if ctx.Args().Len() < 2 {
		println("I need you to tell me the [package] and [major-version] to search for.")
	}

	pkg := ctx.Args().Get(0)
	_ver := ctx.Args().Get(1)
	__ver, err := strconv.ParseInt(_ver, 10, 64)
	if err != nil {
		return fmt.Errorf("i didn't understand your major version: %+w", err)
	}
	ver := int(__ver)
	searchingComponent := ctx.String("component")

	eng := analysis.New(qtquick.BuiltinModule())
	eng.DoQMLPluginDump = ctx.Bool("use-qmlplugindump")
	err = filepath.WalkDir(".", func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".qml") {
			return nil
		}

		data, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s while analysing: %+w", path, err)
		}

		eng.SetFileContext(path, data)
		fctx, err := eng.GetFileContext(path)
		if err != nil {
			return fmt.Errorf("failed to analyse file %s: %+w", path, err)
		}

		data = fctx.Body
		imports := fctx.Imports
		node := fctx.Tree.RootNode()

		qc := sitter.NewQueryCursor()
		defer qc.Close()

		types := map[string]sitter.Point{}

		// gather all the refernces to types in the documents

		// uses in property declarations, such as
		// property -> Kirigami.AboutPage <- aboutPage: ...
		qc.Exec(eng.Queries().PropertyTypes, node)
		for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
			for _, cap := range match.Captures {
				types[cap.Node.Content(data)] = cap.Node.StartPoint()
			}
		}

		// uses in object blocks, such as
		// -> Kirigami.AboutPage <- { }
		qc.Exec(eng.Queries().ObjectDeclarationTypes, node)
		for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
			for _, cap := range match.Captures {
				types[cap.Node.Content(data)] = cap.Node.StartPoint()
			}
		}

		// we've gathered all our types, now we try to match them to imports
	outerLoop:
		for usageKind, location := range types {
			for idx := range imports {
				importData := imports[idx]
				if importData.URI.Path != pkg || importData.URI.MajorVersion != ver {
					continue
				}

				// handle stuff like "import org.kde.kirigami 2.10 as Kirigami"
				// Kirigami.AboutData vs AboutData.
				prefix := ""
				if importData.As != "" {
					prefix = importData.As + "."
				}

				for _, component := range importData.Module.Components {
					if searchingComponent != "" && searchingComponent != component.SaneName() {
						continue
					}
					if prefix+component.SaneName() == usageKind {
						fmt.Printf("%s:%d:%d uses %s\n", path, location.Column, location.Row, component.SaneName())
						continue outerLoop
					}
				}
				if prefix != "" && strings.HasPrefix(usageKind, prefix) {
					fmt.Printf("%s:%d:%d uses %s (weak match, type not found in qmltypes)\n", path, location.Column, location.Row, usageKind)
					continue outerLoop
				}
			}
		}

		return nil
	})

	return err
}

func main() {
	app := cli.App{
		Flags: []cli.Flag{
			&cli.BoolFlag{
				Name: "use-qmlplugindump",
			},
		},
		Commands: []*cli.Command{
			{
				Name:   "what-uses",
				Action: whatUses,
				Flags: []cli.Flag{
					&cli.StringFlag{
						Name:  "component",
						Usage: "search for specific usages of this component",
					},
				},
			},
		},
	}
	app.Run(os.Args)
}
