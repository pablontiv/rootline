package migrate

import (
	"os"
	"path/filepath"
	"testing"
)

func TestFindStemFiles(t *testing.T) {
	dir := t.TempDir()

	// Create directory structure with .stem files.
	sub1 := filepath.Join(dir, "sub1")
	sub2 := filepath.Join(dir, "sub2")
	hidden := filepath.Join(dir, ".hidden")
	if err := os.MkdirAll(sub1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(sub2, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(hidden, 0755); err != nil {
		t.Fatal(err)
	}

	// Write .stem files.
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub1, ".stem"), []byte("version: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sub2, ".stem"), []byte("version: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	// .stem inside hidden dir should be skipped.
	if err := os.WriteFile(filepath.Join(hidden, ".stem"), []byte("version: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	stems, err := FindStemFiles(dir)
	if err != nil {
		t.Fatalf("FindStemFiles failed: %v", err)
	}

	if len(stems) != 3 {
		t.Errorf("expected 3 .stem files (root, sub1, sub2), got %d: %v", len(stems), stems)
	}

	// Verify hidden directory .stem is not included.
	for _, s := range stems {
		if filepath.Base(filepath.Dir(s)) == ".hidden" {
			t.Error("should not include .stem from hidden directory")
		}
	}
}

func TestFindStemFilesEmpty(t *testing.T) {
	dir := t.TempDir()

	stems, err := FindStemFiles(dir)
	if err != nil {
		t.Fatalf("FindStemFiles failed: %v", err)
	}
	if len(stems) != 0 {
		t.Errorf("expected 0 .stem files, got %d", len(stems))
	}
}

func TestFindGitRoot(t *testing.T) {
	dir := t.TempDir()

	// Create .git directory.
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// Create nested subdirectory.
	sub := filepath.Join(dir, "a", "b", "c")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}

	root, err := findGitRoot(sub)
	if err != nil {
		t.Fatalf("findGitRoot failed: %v", err)
	}
	if root != dir {
		t.Errorf("expected %s, got %s", dir, root)
	}
}

func TestFindGitRootNotFound(t *testing.T) {
	dir := t.TempDir()

	_, err := findGitRoot(dir)
	if err == nil {
		t.Fatal("expected error for missing .git")
	}
}

func TestLoadPreviousStemFromFile(t *testing.T) {
	dir := t.TempDir()

	stemContent := `version: 2
schema:
  estado:
    type: string
    required: true
`
	fromPath := filepath.Join(dir, "previous.stem")
	if err := os.WriteFile(fromPath, []byte(stemContent), 0644); err != nil {
		t.Fatal(err)
	}

	stem, err := LoadPreviousStem("current.stem", fromPath)
	if err != nil {
		t.Fatalf("LoadPreviousStem failed: %v", err)
	}
	if stem == nil {
		t.Fatal("expected non-nil stem")
	}
	if _, ok := stem.Schema["estado"]; !ok {
		t.Error("expected estado in schema")
	}
}

func TestLoadPreviousStemFromInvalidFile(t *testing.T) {
	dir := t.TempDir()
	fromPath := filepath.Join(dir, "nonexistent.stem")

	_, err := LoadPreviousStem("current.stem", fromPath)
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestLoadPreviousStemFromGitFails(t *testing.T) {
	// When no --from is specified and we're not in a real git repo,
	// loadFromGit should fail.
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := LoadPreviousStem(stemPath, "")
	if err == nil {
		t.Fatal("expected error when git is not available")
	}
}
