package main

import (
	"context"
	"flag"
	"fmt"
	"io/ioutil"
	"log"
	"qml-lsp/analysis"
	"qml-lsp/qmltypes/qtquick"
	"strings"
)

func prepareString(in string) (out string) {
	lines := strings.Split(in, "\n")
	for idx := range lines {
		lines[idx] = strings.TrimLeft(lines[idx], " \t")
	}
	switch len(lines) {
	case 0:
		return ""
	case 1:
		return "\t" + lines[0]
	default:
		return fmt.Sprintf("\t%s\n\t...", lines[0])
	}
}

func main() {
	flag.Parse()

	eng := analysis.New(qtquick.BuiltinModule())
	eng.DoQMLPluginDump = false

	for _, file := range flag.Args() {
		data, err := ioutil.ReadFile(file)
		if err != nil {
			log.Fatalf("could not open file %s: %s", file, err)
		}

		err = eng.SetFileContext(file, data)
		if err != nil {
			log.Fatalf("could not analyse file %s: %s", file, err)
		}

		fctx, err := eng.GetFileContext(file)
		if err != nil {
			panic("err should never be non-nil: " + err.Error())
		}

		for _, diag := range analysis.DefaultDiagnostics {
			diags := diag.Analyze(context.Background(), file, fctx, eng)
			for _, out := range diags {
				fmt.Printf(
					"%d:%d - %d:%d\t%s\t%s (%s)\n",
					out.Range.Start.Line, out.Range.Start.Character,
					out.Range.End.Line, out.Range.End.Character,
					file,
					out.Message, out.Source,
				)
				if out.ContextNode != nil {
					fmt.Printf("\n%s\n", prepareString(out.ContextNode.Content(fctx.Body)))
				}
				fmt.Printf("\n")
			}
		}
	}
}
