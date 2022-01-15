package analysis

import (
	"errors"
	"fmt"
	qml "qml-lsp/treesitter-qml"
	"strings"

	sitter "github.com/smacker/go-tree-sitter"
)

type ImportData struct {
	URI    ImportName
	Module *Module
	As     string
	Range  PointRange
}

type FileContext struct {
	Imports []ImportData
	Body    []byte
	Tree    *sitter.Tree
}

type resultSting struct {
	s string
	e error
}

type resultModule struct {
	m *Module
	e error
}

func fromRaw(s []string, vmaj, vmin int) ImportName {
	return ImportName{strings.Join(s, "."), vmaj, vmin, false}
}

type ImportName struct {
	Path           string
	MajorVersion   int
	MinorVersion   int
	IsRelativePath bool
}

type AnalysisEngine struct {
	DoQMLPluginDump bool

	fileContexts map[string]FileContext

	importNamesToResolvedPaths map[ImportName]resultSting
	resolvedPathsToModules     map[string]resultModule

	builtinModule Module
}

func QmlParser() *sitter.Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(qml.GetLanguage())

	return parser
}

func New(builtin Module) *AnalysisEngine {
	return &AnalysisEngine{
		DoQMLPluginDump:            true,
		fileContexts:               map[string]FileContext{},
		importNamesToResolvedPaths: map[ImportName]resultSting{},
		resolvedPathsToModules:     map[string]resultModule{},
		builtinModule:              builtin,
	}
}

func (s *AnalysisEngine) DeleteFileContext(uri string) {
	delete(s.fileContexts, uri)
}

func (s *AnalysisEngine) BuiltinModule() Module {
	return s.builtinModule
}

func (s *AnalysisEngine) SetFileContext(uri string, content []byte) error {
	fctx := FileContext{}

	it := QmlParser()
	fctx.Tree = it.Parse(nil, content)
	fctx.Body = content

	importData, relativeData := ExtractImports(fctx.Tree.RootNode(), content)
	for _, it := range importData {
		m, err := s.GetModule(it.Module, it.MajVersion, it.MinVersion)
		if err != nil {
			// println(err.Error())
			continue
		}
		fctx.Imports = append(fctx.Imports, ImportData{
			Module: m,
			URI:    fromRaw(it.Module, it.MajVersion, it.MinVersion),
			As:     it.As,
			Range:  it.Range,
		})
	}
	for _, it := range relativeData {
		fctx.Imports = append(fctx.Imports, ImportData{
			Module: &Module{},
			URI:    ImportName{Path: it.Path, IsRelativePath: true},
			As:     it.As,
			Range:  it.Range,
		})
	}

	s.fileContexts[uri] = fctx

	return nil
}

func (s *AnalysisEngine) GetFileContext(uri string) (FileContext, error) {
	return s.fileContexts[uri], nil
}

func (s *AnalysisEngine) ResolveComponent(as, name string, inURI string) (Component, ImportName, *Module, error) {
	fctx := s.fileContexts[inURI]

	for _, imp := range fctx.Imports {
		if as != "" && imp.As != as {
			continue
		}

		for _, comp := range imp.Module.Components {
			if comp.SaneName() == name {
				return comp, imp.URI, imp.Module, nil
			}
		}
	}

	return Component{}, ImportName{}, nil, errComponentNotFound
}

func (s *AnalysisEngine) GetModule(uri []string, vmaj, vmin int) (*Module, error) {
	imported := fromRaw(uri, vmaj, vmin)

	display := fmt.Sprintf("%s %d.%d", strings.Join(uri, "."), vmaj, vmin)

	var (
		resolved string
		err      error
		module   Module
	)

	if v, ok := s.importNamesToResolvedPaths[imported]; ok {
		if v.e != nil {
			return nil, fmt.Errorf("failed to get module %s: %+w", display, v.e)
		}

		if vv, ok := s.resolvedPathsToModules[v.s]; ok {
			if vv.e != nil {
				return nil, fmt.Errorf("failed to get module %s: %+w", display, vv.e)
			}

			return vv.m, nil
		} else {
			goto resolvedToModule
		}
	} else {
		goto importNameToResolved
	}

importNameToResolved:
	resolved, err = actualQmlPath(uri, vmaj, vmin)
	s.importNamesToResolvedPaths[imported] = resultSting{resolved, err}
	if err != nil {
		if errors.Is(err, errQmlTypesNotFound) {
			if !s.DoQMLPluginDump {
				return nil, fmt.Errorf("failed to resolve import to file, and qmlplugindump is disabled, preventing using it to resolve data: %+w", err)
			}

			var inMem = fmt.Sprintf(`/\inmem:%s/\%d.%d`, strings.Join(uri, "."), vmaj, vmin)
			data, err := qmlPluginDump(uri, vmaj, vmin)
			if err != nil {
				return nil, fmt.Errorf("failed to resolve import to file, and qmlplugindump failed: %+w", err)
			}
			module, err = loadPluginTypes("inmemory", data)
			s.resolvedPathsToModules[inMem] = resultModule{&module, err}
			if err != nil {
				return nil, fmt.Errorf("failed to load module types generated from qmlplugindump: %+w", err)
			}
			s.importNamesToResolvedPaths[imported] = resultSting{inMem, nil}
			return &module, nil
		}
		return nil, fmt.Errorf("failed to resolve import to file: %+w", err)
	}

resolvedToModule:
	module, err = loadPluginTypesFile(resolved)
	s.resolvedPathsToModules[resolved] = resultModule{&module, err}
	if err != nil {
		return nil, fmt.Errorf("failed to load module types: %+w", err)
	}

	return &module, nil
}
