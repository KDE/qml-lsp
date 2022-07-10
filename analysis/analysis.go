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
	Node   *sitter.Node
	Error  error
}

func (i ImportData) ToSourceString() string {
	var b strings.Builder

	b.WriteString("import ")
	if i.URI.IsRelativePath {
		// TODO: escaped strings
		b.WriteString(fmt.Sprintf(`"%s"`, i.URI.Path))
	} else {
		b.WriteString(fmt.Sprintf(`%s %d.%d`, i.URI.Path, i.URI.MajorVersion, i.URI.MinorVersion))
	}

	if i.As != "" {
		b.WriteString(" as ")
		b.WriteString(i.As)
	}

	return b.String()
}

type FileContext struct {
	Imports []ImportData
	Body    []byte
	Tree    FileTree
}

type FileTree struct {
	*sitter.Tree
	Data map[*sitter.Node]NodeData
}

type NodeData struct {
	IsStrongScope bool
	IsWeakScope   bool
	Types         map[string]TypeURI
	Kind          TypeURI
}

type TypeURI struct {
	Path         string
	MajorVersion int
	Name         string
	ReactiveList bool
	Pointer      bool
}

func (t *TypeURI) String() string {
	v := ""
	if t.Path == "" {
		v = t.Name
	} else {
		v = fmt.Sprintf("%s/%d %s", t.Path, t.MajorVersion, t.Name)
	}

	if t.ReactiveList {
		return fmt.Sprintf("list<%s> (reactive)", v)
	} else {
		return v
	}
}

var NumberURI = TypeURI{"", 0, "number", false, false}
var BooleanURI = TypeURI{"", 0, "bool", false, false}
var StringURI = TypeURI{"", 0, "string", false, false}
var ComplexURI = TypeURI{"", 0, "complexType", false, false}

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

	queries Queries
}

func QmlParser() *sitter.Parser {
	parser := sitter.NewParser()
	parser.SetLanguage(qml.GetLanguage())

	return parser
}

func New(builtin Module) *AnalysisEngine {
	k := &AnalysisEngine{
		DoQMLPluginDump:            true,
		fileContexts:               map[string]FileContext{},
		importNamesToResolvedPaths: map[ImportName]resultSting{},
		resolvedPathsToModules:     map[string]resultModule{},
		builtinModule:              builtin,
		queries:                    Queries{},
	}
	if err := k.queries.Init(); err != nil {
		panic(err)
	}
	return k
}

func (s *AnalysisEngine) Queries() Queries {
	return s.queries
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
	fctx.Tree = FileTree{it.Parse(nil, content), map[*sitter.Node]NodeData{}}
	fctx.Body = content

	importData, relativeData := ExtractImports(fctx.Tree.RootNode(), content)
	for _, it := range importData {
		m, err := s.GetModule(it.Module, it.MajVersion, it.MinVersion)
		if err != nil {
			fctx.Imports = append(fctx.Imports, ImportData{
				Module: &Module{},
				URI:    fromRaw(it.Module, it.MajVersion, it.MinVersion),
				As:     it.As,
				Range:  it.Range,
				Error:  err,
			})
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

	s.analyseFile(uri, &fctx)

	s.fileContexts[uri] = fctx

	return nil
}

func (s *AnalysisEngine) GetFileContext(uri string) (FileContext, error) {
	k, ok := s.fileContexts[uri]
	if !ok {
		return FileContext{}, errors.New("file context not found")
	}
	return k, nil
}

func (s *AnalysisEngine) TypeReferences(inURI string, node *sitter.Node) ([]*sitter.Node, error) {
	_, err := s.GetFileContext(inURI)
	if err != nil {
		return nil, err
	}

	types := []*sitter.Node{}

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	// gather all the refernces to types in the documents

	// uses in property declarations, such as
	// property -> Kirigami.AboutPage <- aboutPage: ...
	qc.Exec(s.queries.PropertyTypes, node)
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			types = append(types, cap.Node)
		}
	}

	// uses in object blocks, such as
	// -> Kirigami.AboutPage <- { }
	qc.Exec(s.queries.ObjectDeclarationTypes, node)
	for match, goNext := qc.NextMatch(); goNext; match, goNext = qc.NextMatch() {
		for _, cap := range match.Captures {
			types = append(types, cap.Node)
		}
	}

	return types, nil
}

func (s *AnalysisEngine) UsedImports(inURI string, node *sitter.Node) ([]bool, error) {
	fctx, err := s.GetFileContext(inURI)
	if err != nil {
		return nil, err
	}

	data := fctx.Body
	imports := fctx.Imports
	used := make([]bool, len(imports))

	qc := sitter.NewQueryCursor()
	defer qc.Close()

	types, err := s.TypeReferences(inURI, node)
	if err != nil {
		return nil, fmt.Errorf("failed to get references to types: %+w", err)
	}

	// we've gathered all our types, now we try to match them to imports
outerLoop:
	for _, tkind := range types {
		kind := tkind.Content(data)
		for idx := range imports {
			importData := imports[idx]
			isUsed := used[idx]

			// if this import is already known used, we don't need to waste time
			// checking if it's used again
			if isUsed {
				continue
			}

			// handle stuff like "import org.kde.kirigami 2.10 as Kirigami"
			// Kirigami.AboutData vs AboutData.
			prefix := ""
			if importData.As != "" {
				prefix = importData.As + "."
			}

			for _, component := range importData.Module.Components {
				if prefix+component.SaneName() == kind {
					used[idx] = true
					continue outerLoop
				}
			}
		}
	}

	return used, nil
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
