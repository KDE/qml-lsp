package analysis

import (
	"fmt"
	"qml-lsp/qmltypes"
	"strings"
)

type Refactoring struct {
	PreferredAliases          []RefactorPreferredAlias            `qml:"@PreferredAlias"`
	ReplaceUses               []RefactorReplaceUses               `qml:"@ReplaceUses"`
	ReplaceVarWithLetAndConst []RefactorReplaceVarWithLetAndConst `qml:"@ReplaceVarWithLetAndConst"`
}
type RefactorComponent struct {
	URI          string `qml:"uri"`
	MajorVersion int    `qml:"majorVersion"`
	MinorVersion int    `qml:"minorVersion"`
	Name         string `qml:"name"`
}
type RefactorPreferredAlias struct {
	For          string `qml:"for"`
	MajorVersion int    `qml:"majorVersion"`
	Alias        string `qml:"alias"`
}

func (c RefactorComponent) Import(alias string) string {
	var b strings.Builder

	b.WriteString("import ")
	b.WriteString(fmt.Sprintf(`%s %d.%d`, c.URI, c.MajorVersion, c.MinorVersion))

	if alias != "" {
		b.WriteString(" as ")
		b.WriteString(alias)
	}

	return b.String()
}

func LoadRefactorManifest(file string, b []byte) (*Refactoring, error) {
	var document qmltypes.QMLTypesFile

	err := qmltypes.Parser.ParseBytes(file, b, &document)
	if err != nil {
		return nil, fmt.Errorf("Failed to parse file %s: %+w", file, err)
	}

	var refactoring Refactoring
	err = qmltypes.Unmarshal(qmltypes.Value{Object: &document.Main}, &refactoring)

	if err != nil {
		return nil, fmt.Errorf("Failed to unmarshal file %s: %+w", file, err)
	}

	return &refactoring, nil
}

func (r *Refactoring) Execute(uri string, engine *AnalysisEngine) error {
	for _, it := range r.ReplaceUses {
		feh := it.Execute(r, uri, engine)
		if feh != nil {
			return fmt.Errorf("refactoring failed: %+w", feh)
		}
	}
	for _, it := range r.ReplaceVarWithLetAndConst {
		feh := it.Execute(r, uri, engine)
		if feh != nil {
			return fmt.Errorf("refactoring failed: %+w", feh)
		}
	}
	return nil
}
