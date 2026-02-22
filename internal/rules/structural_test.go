package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func mustMkdir(t *testing.T, path string) {
	t.Helper()
	if err := os.Mkdir(path, 0755); err != nil {
		t.Fatal(err)
	}
}

func mustWriteTestFile(t *testing.T, path string, data []byte) {
	t.Helper()
	if err := os.WriteFile(path, data, 0644); err != nil {
		t.Fatal(err)
	}
}

func TestValidateDirectory_RequireIndex(t *testing.T) {
	dir := t.TempDir()
	// Create 3 subdirs, only 2 have README.md.
	for _, name := range []string{"sub1", "sub2", "sub3"} {
		mustMkdir(t, filepath.Join(dir, name))
	}
	mustWriteTestFile(t, filepath.Join(dir, "sub1", "README.md"), []byte("# Sub1"))
	mustWriteTestFile(t, filepath.Join(dir, "sub2", "README.md"), []byte("# Sub2"))
	// sub3 has no README.md

	stem := &StemFile{
		Path: "test/.stem",
		Structural: StructuralRules{
			Subdirs: SubdirRules{RequireIndex: "README.md", Severity: "warn"},
		},
	}

	errs := ValidateDirectory(dir, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if errs[0].Rule != "require_index" {
		t.Errorf("rule = %q, want require_index", errs[0].Rule)
	}
	if errs[0].Severity != "warn" {
		t.Errorf("severity = %q, want warn", errs[0].Severity)
	}
}

func TestValidateDirectory_MinChildren(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "only-one"))

	stem := &StemFile{
		Path: "test/.stem",
		Structural: StructuralRules{
			Subdirs: SubdirRules{MinChildren: 2},
		},
	}

	errs := ValidateDirectory(dir, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if errs[0].Rule != "min_children" {
		t.Errorf("rule = %q, want min_children", errs[0].Rule)
	}
}

func TestValidateDirectory_MaxChildren(t *testing.T) {
	dir := t.TempDir()
	for i := 0; i < 5; i++ {
		mustMkdir(t, filepath.Join(dir, filepath.Base(t.Name())+string(rune('a'+i))))
	}

	stem := &StemFile{
		Path: "test/.stem",
		Structural: StructuralRules{
			Subdirs: SubdirRules{MaxChildren: 3},
		},
	}

	errs := ValidateDirectory(dir, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1: %v", len(errs), errs)
	}
	if errs[0].Rule != "max_children" {
		t.Errorf("rule = %q, want max_children", errs[0].Rule)
	}
}

func TestValidateDirectory_NoStructural(t *testing.T) {
	dir := t.TempDir()
	stem := &StemFile{Path: "test/.stem"}

	errs := ValidateDirectory(dir, stem)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (no structural rules)", len(errs))
	}
}

func TestValidateDirectory_NilStem(t *testing.T) {
	dir := t.TempDir()
	errs := ValidateDirectory(dir, nil)
	if len(errs) != 0 {
		t.Errorf("got %d errors, want 0 (nil stem)", len(errs))
	}
}

func TestValidateDirectory_DefaultSeverity(t *testing.T) {
	dir := t.TempDir()
	mustMkdir(t, filepath.Join(dir, "sub"))

	stem := &StemFile{
		Path: "test/.stem",
		Structural: StructuralRules{
			Subdirs: SubdirRules{RequireIndex: "README.md"},
		},
	}

	errs := ValidateDirectory(dir, stem)
	if len(errs) != 1 {
		t.Fatalf("got %d errors, want 1", len(errs))
	}
	if errs[0].Severity != "error" {
		t.Errorf("severity = %q, want error (default)", errs[0].Severity)
	}
}
