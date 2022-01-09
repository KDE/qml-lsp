package analysis

import (
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"path"
	"qml-lsp/qmltypes"
	libpaths "qml-lsp/qt-libpaths"
)

var paths = libpaths.Paths()

type importVersion int

const (
	fullyVersioned importVersion = iota
	majorlyVersioned
	notVersioned
)

func versionString(vmaj, vmin int, version importVersion) string {
	switch version {
	case fullyVersioned:
		return fmt.Sprintf(".%d.%d", vmaj, vmin)
	case majorlyVersioned:
		return fmt.Sprintf(".%d", vmaj)
	default:
		return ""
	}
}

func potentialQmlPaths(parts, basePaths []string, vmaj, vmin int) []string {
	var retPaths []string
	for _, impVer := range []importVersion{fullyVersioned, majorlyVersioned, notVersioned} {
		ver := versionString(vmaj, vmin, impVer)

		for _, basePath := range basePaths {
			dir := basePath

			it := path.Join(append([]string{dir}, parts...)...) + ver
			retPaths = append(retPaths, it)

			if impVer != notVersioned {
				for index := len(parts) - 2; index >= 0; index-- {
					first := path.Join(append([]string{dir}, parts[0:index+1]...)...) + ver
					parts := append([]string{first}, parts[index+1:]...)
					it := path.Join(parts...)
					retPaths = append(retPaths, it)
				}
			}
		}
	}
	return retPaths
}

var (
	errQmlTypesNotFound  = errors.New(".qmltypes not found in any of the potential paths")
	errComponentNotFound = errors.New("component not found")
)

func actualQmlPath(s []string, vmaj, vmin int) (string, error) {
	potentialPaths := potentialQmlPaths(s, paths, vmaj, vmin)
	for _, it := range potentialPaths {
		pluginsQmltypes := path.Join(it, "plugins.qmltypes")
		if _, err := os.Stat(pluginsQmltypes); err == nil {
			return it, nil
		} else if errors.Is(err, os.ErrNotExist) {
			continue
		} else {
			return "", fmt.Errorf("failed to determine actual qml path: %+w", err)
		}
	}
	return "", errQmlTypesNotFound
}

func loadPluginTypes(typesPath string, data []byte) (Module, error) {
	var d qmltypes.QMLTypesFile
	err := qmltypes.Parser.ParseBytes(typesPath, data, &d)
	if err != nil {
		return Module{}, fmt.Errorf("failed to parse qmltypes file at %s: %+w", typesPath, err)
	}

	var m Module
	err = qmltypes.Unmarshal(qmltypes.Value{Object: &d.Main}, &m)
	if err != nil {
		return Module{}, fmt.Errorf("failed to unmarshal qmltypes file at %s: %+w", typesPath, err)
	}

	for idx, cmp := range m.Components {
		cmp.GetActualName()
		m.Components[idx] = cmp
	}

	return m, nil
}

func loadPluginTypesFile(qmlPath string) (Module, error) {
	typesPath := path.Join(qmlPath, "plugins.qmltypes")
	data, err := ioutil.ReadFile(typesPath)
	if err != nil {
		return Module{}, fmt.Errorf("failed to read qmltypes file at %s: %+w", typesPath, err)
	}

	return loadPluginTypes(typesPath, data)
}
