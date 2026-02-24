package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// helper creates a temp directory tree with .stem files and a .git marker.
func setupTree(t *testing.T, stems []string, gitAt string) string {
	t.Helper()
	root := t.TempDir()

	// Create .git marker.
	if gitAt != "" {
		if err := os.MkdirAll(filepath.Join(root, gitAt, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}
	}

	// Create .stem files.
	for _, s := range stems {
		dir := filepath.Join(root, filepath.Dir(s))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := []byte("version: 1\nscope:\n  match: \"*.md\"\n")
		if err := os.WriteFile(filepath.Join(root, s), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestWalkUp_MultiLevel(t *testing.T) {
	root := setupTree(t,
		[]string{"a/.stem", "a/b/c/.stem"},
		"a",
	)
	// Also create the target dir.
	target := filepath.Join(root, "a", "b", "c", "d")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := WalkUp(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	// Root-to-leaf order: a/.stem first, then a/b/c/.stem.
	if filepath.Base(filepath.Dir(entries[0].Path)) != "a" {
		t.Errorf("entries[0] dir = %q, want a/", filepath.Dir(entries[0].Path))
	}
	if filepath.Base(filepath.Dir(entries[1].Path)) != "c" {
		t.Errorf("entries[1] dir = %q, want c/", filepath.Dir(entries[1].Path))
	}
}

func TestWalkUp_StopsAtGit(t *testing.T) {
	root := setupTree(t,
		[]string{"above/.stem", "above/repo/.stem", "above/repo/sub/.stem"},
		"above/repo",
	)
	target := filepath.Join(root, "above", "repo", "sub")

	entries, err := WalkUp(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find repo/.stem and repo/sub/.stem, NOT above/.stem.
	for _, e := range entries {
		rel, _ := filepath.Rel(root, e.Path)
		if rel == filepath.Join("above", ".stem") {
			t.Error("found above/.stem — should have stopped at .git boundary")
		}
	}
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2", len(entries))
	}
}

func TestWalkUp_SkipsDirsWithoutStem(t *testing.T) {
	root := setupTree(t,
		[]string{"a/.stem", "a/b/c/.stem"},
		"a",
	)
	// a/b/ has no .stem — should be skipped.
	target := filepath.Join(root, "a", "b", "c")

	entries, err := WalkUp(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2 (skipping b/)", len(entries))
	}
}

func TestWalkUp_NoStemFiles(t *testing.T) {
	root := setupTree(t, nil, "repo")
	target := filepath.Join(root, "repo")

	entries, err := WalkUp(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 0 {
		t.Errorf("got %d entries, want 0", len(entries))
	}
}

func TestWalkUp_NoGitReturnsError(t *testing.T) {
	// No .git anywhere — WalkUp must return an error.
	root := t.TempDir()
	sub := filepath.Join(root, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}

	_, err := WalkUp(sub)
	if err == nil {
		t.Fatal("expected error when no .git is found, got nil")
	}
}

func TestWalkUp_TargetIsFile(t *testing.T) {
	root := setupTree(t,
		[]string{"repo/.stem"},
		"repo",
	)
	// Create a file (not a directory) as target.
	filePath := filepath.Join(root, "repo", "doc.md")
	if err := os.WriteFile(filePath, []byte("hello"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := WalkUp(filePath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 1 {
		t.Fatalf("got %d entries, want 1", len(entries))
	}
}

func TestParseStemFile_Success(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	content := []byte("version: 1\nscope:\n  match: \"*.md\"\n")
	if err := os.WriteFile(stemPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	stem, err := ParseStemFile(stemPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stem.Version != 1 {
		t.Errorf("version = %d, want 1", stem.Version)
	}
	if stem.Scope.Match != "*.md" {
		t.Errorf("scope.match = %q, want %q", stem.Scope.Match, "*.md")
	}
}
