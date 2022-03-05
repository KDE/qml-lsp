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

func refactor(ctx *cli.Context) error {
	if ctx.Args().Len() < 1 {
		println("I need you to give me a refactor manifest.")
	}

	manifestPath := ctx.Args().Get(0)
	manifestData, err := ioutil.ReadFile(manifestPath)
	if err != nil {
		return fmt.Errorf("failed to read file %s while loading refactor manifest: %+w", manifestPath, err)
	}

	eng := analysis.New(qtquick.BuiltinModule())
	eng.DoQMLPluginDump = ctx.Bool("use-qmlplugindump")

	refactoring, err := analysis.LoadRefactorManifest(manifestPath, manifestData)
	if err != nil {
		return fmt.Errorf("failed to load refactoring manifest: %+w", err)
	}

	walkQmlFiles(".", func(path string, d fs.DirEntry, err error) error {
		data, err := ioutil.ReadFile(path)
		if err != nil {
			return fmt.Errorf("failed to read file %s while analysing: %+w", path, err)
		}

		eng.SetFileContext(path, data)
		_, err = eng.GetFileContext(path)
		if err != nil {
			return fmt.Errorf("failed to analyse file %s: %+w", path, err)
		}

		err = refactoring.Execute(path, eng)
		if err != nil {
			return fmt.Errorf("failed to refactor file %s: %+w", path, err)
		}

		fctx, err := eng.GetFileContext(path)
		if err != nil {
			return fmt.Errorf("failed to analyse refactored file %s: %+w", path, err)
		}

		err = ioutil.WriteFile(path, fctx.Body, os.ModePerm)
		if err != nil {
			return fmt.Errorf("failed to write refactored file %s: %+w", path, err)
		}

		return nil
	})

	return nil
}

func walkQmlFiles(from string, walk fs.WalkDirFunc) error {
	return filepath.WalkDir(from, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}

		if d.IsDir() {
			return nil
		}
		if !strings.HasSuffix(d.Name(), ".qml") {
			return nil
		}

		return walk(path, d, err)
	})
}

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
	err = walkQmlFiles(".", func(path string, d fs.DirEntry, err error) error {
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

		types, err := eng.TypeReferences(path, node)
		if err != nil {
			return fmt.Errorf("failed to get references to types: %+w", err)
		}

		// we've gathered all our types, now we try to match them to imports
	outerLoop:
		for _, usage := range types {
			usageKind := usage.Content(data)
			location := usage.StartPoint()
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
		ExitErrHandler: func(context *cli.Context, err error) {
			println(err.Error())
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
			{
				Name:   "refactor",
				Action: refactor,
			},
		},
	}
	app.Run(os.Args)
}
