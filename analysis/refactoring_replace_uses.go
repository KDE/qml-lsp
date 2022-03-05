package analysis

import (
	"fmt"
	"strings"
)

type RefactorReplaceUses struct {
	Of   RefactorComponent `qml:"of"`
	With RefactorComponent `qml:"with"`
}

// TODO: rewrite minor versions when needed
func (d *RefactorReplaceUses) Execute(r *Refactoring, uri string, eng *AnalysisEngine) error {
	fctx, err := eng.GetFileContext(uri)
	if err != nil {
		return fmt.Errorf("failed to get file context while performing replace uses refactor: %+w", err)
	}

	data := fctx.Body
	imports := fctx.Imports
	destinationFound := false
	destinationPrefix := ""
	sourcePrefix := ""
	lastImport := ImportData{}

	for _, impt := range imports {
		if impt.URI.Path == d.With.URI && impt.URI.MajorVersion == d.With.MajorVersion {
			destinationPrefix = impt.As + "."
			destinationFound = true
			break
		}
		lastImport = impt
	}
	for _, impt := range imports {
		if impt.URI.Path == d.Of.URI && impt.URI.MajorVersion == d.Of.MajorVersion {
			if impt.As != "" {
				sourcePrefix = impt.As + "."
			}
			break
		}
	}

	replacements := replacementlist{}

	if !destinationFound {
		destinationAlias := ""
		for _, it := range r.PreferredAliases {
			if it.For == d.With.URI && it.MajorVersion == d.With.MajorVersion {
				destinationAlias = it.Alias
				destinationPrefix = it.Alias + "."
			}
		}
		endByte := lastImport.Range.EndByte

		replacements = append(replacements, replaceSpan{
			start: endByte,
			end:   endByte,
			with:  "\n" + d.With.Import(destinationAlias),
		})
	}

	types, err := eng.TypeReferences(uri, fctx.Tree.RootNode())
	if err != nil {
		return fmt.Errorf("failed to get references to types: %+w", err)
	}

	replaced := false

	// we've gathered all the references to types/components
outerLoop:
	for _, usage := range types {
		usageKind := usage.Content(data)
		if usageKind != sourcePrefix+d.Of.Name {
			continue
		}

		// location := usage.StartPoint()
		for idx := range imports {
			importData := imports[idx]
			if importData.URI.Path != d.Of.URI || importData.URI.MajorVersion != d.Of.MajorVersion {
				continue
			}

			// handle stuff like "import org.kde.kirigami 2.10 as Kirigami"
			// Kirigami.AboutData vs AboutData.
			prefix := ""
			if importData.As != "" {
				prefix = importData.As + "."
			}

			for _, component := range importData.Module.Components {
				if prefix+component.SaneName() == usageKind {
					replacements = append(replacements, replaceSpan{
						start: usage.StartByte(),
						end:   usage.EndByte(),
						with:  destinationPrefix + d.With.Name,
					})
					replaced = true
					continue outerLoop
				}
			}
			if prefix != "" && strings.HasPrefix(usageKind, prefix) && strings.HasSuffix(usageKind, d.Of.Name) {
				replacements = append(replacements, replaceSpan{
					start: usage.StartByte(),
					end:   usage.EndByte(),
					with:  destinationPrefix + d.With.Name,
				})
				replaced = true
				continue outerLoop
			}
		}
	}

	if !replaced {
		return nil
	}

	eng.SetFileContext(uri, []byte(replacements.applyTo(string(data))))

	return nil
}
