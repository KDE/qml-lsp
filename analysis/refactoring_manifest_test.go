package analysis

import (
	_ "embed"
	"testing"
)

//go:embed test.qmlrefactor
var data []byte

func TestManifest(t *testing.T) {
	manifest, err := LoadRefactorManifest("test.qmlrefactor", data)
	if err != nil {
		t.Fatal(err)
	}
	if len(manifest.PreferredAliases) != 2 {
		t.Fatalf("wrong count of aliases, expected 2, got %d", len(manifest.PreferredAliases))
	}
	if len(manifest.ReplaceUses) != 1 {
		t.Fatalf("wrong count of replace usages, expected 1, got %d", len(manifest.ReplaceUses))
	}
	if len(manifest.ReplaceVarWithLetAndConst) != 0 {
		t.Fatalf("wrong count of replace vars, expected 0, got %d", len(manifest.ReplaceVarWithLetAndConst))
	}
}
