package analysis

import (
	"io/fs"
	"io/ioutil"
	"path/filepath"
	"qml-lsp/qmltypes"
	"strconv"
	"strings"
)

type ImportRelAndMaj struct {
	URL   string
	Major int
}

func extractExport(export string) (url string, component string, vmaj, vmin int, ok bool) {
	slash := strings.Index(export, "/")
	space := strings.Index(export, " ")
	if slash == -1 || space == -1 {
		return "", "", -1, -1, false
	}

	url = export[0:slash]
	component = export[slash+1 : space]

	ver := export[space+1:]
	it := strings.Split(ver, ".")
	if len(it) != 2 {
		return url, component, 1, 0, true
	}

	vmaj64, err := strconv.ParseInt(it[0], 10, 64)
	if err != nil {
		return url, component, 1, 0, true
	}

	vmin64, err := strconv.ParseInt(it[1], 10, 64)
	if err != nil {
		return url, component, int(vmaj64), 0, true
	}

	return url, component, int(vmaj64), int(vmin64), true
}

func PossibleImports() ([]ImportRelAndMaj, map[ImportRelAndMaj]int) {
	imports := []ImportRelAndMaj{}
	importMinors := map[ImportRelAndMaj]int{}

	for _, qmlPath := range paths {
		filepath.WalkDir(qmlPath, func(path string, dentry fs.DirEntry, err error) error {
			if dentry.Name() != "qmldir" {
				return nil
			}

			pluginsTypesPath := filepath.Join(filepath.Dir(path), "plugins.qmltypes")

			data, err := ioutil.ReadFile(pluginsTypesPath)
			if err != nil {
				return nil
			}

			var d qmltypes.QMLTypesFile
			err = qmltypes.Parser.ParseBytes(pluginsTypesPath, data, &d)
			if err != nil {
				return nil
			}

			var m Module
			err = qmltypes.Unmarshal(qmltypes.Value{Object: &d.Main}, &m)
			if err != nil {
				return nil
			}

			for _, component := range m.Components {
				for _, export := range component.Exports {
					url, _, maj, min, ok := extractExport(export)
					if !ok {
						continue
					}
					key := ImportRelAndMaj{url, maj}
					if v, ok := importMinors[key]; ok {
						if min > v {
							importMinors[key] = min
						}
					} else {
						imports = append(imports, key)
						importMinors[key] = min
					}
				}
			}

			return err
		})
	}

	return imports, importMinors
}
