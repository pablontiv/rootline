package rules

import (
	"path/filepath"
	"testing"
)

func TestFieldDeclarationChecksAreSortedByStemPathAndFieldName(t *testing.T) {
	root := t.TempDir()
	parsedStems := map[string]*StemFile{
		filepath.Join(root, "z", ".stem"): {
			Schema: map[string]SchemaField{
				"zeta":  {Type: "number"},
				"alpha": {Type: "number"},
			},
		},
		filepath.Join(root, "a", ".stem"): {
			Schema: map[string]SchemaField{
				"omega": {Type: "number"},
				"beta":  {Type: "number"},
			},
		},
	}

	checks := fieldDeclarationChecks(root, parsedStems)

	got := make([]string, 0, len(checks))
	for _, check := range checks {
		got = append(got, check.Path+":"+check.Field)
	}
	want := []string{
		filepath.Join("a", ".stem") + ":beta",
		filepath.Join("a", ".stem") + ":omega",
		filepath.Join("z", ".stem") + ":alpha",
		filepath.Join("z", ".stem") + ":zeta",
	}
	if len(got) != len(want) {
		t.Fatalf("got %v want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got %v want %v", got, want)
		}
	}
}
