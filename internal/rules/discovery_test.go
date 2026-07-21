package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// helper creates a temp directory tree with .stem files and a root marker.
func setupTree(t *testing.T, stems []string, rootMarkerAt string) string {
	t.Helper()
	root := t.TempDir()

	// Create .stem files.
	for _, s := range stems {
		dir := filepath.Join(root, filepath.Dir(s))
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
		content := []byte("version: 2\nscope:\n  match: \"*.md\"\n")
		// Add root: true if this is the marker location
		if filepath.Clean(s) == filepath.Clean(rootMarkerAt) {
			content = []byte("version: 2\nroot: true\nscope:\n  match: \"*.md\"\n")
		}
		if err := os.WriteFile(filepath.Join(root, s), content, 0o644); err != nil {
			t.Fatal(err)
		}
	}

	return root
}

func TestWalkUp_MultiLevel(t *testing.T) {
	root := setupTree(t,
		[]string{"a/.stem", "a/b/c/.stem"},
		"a/.stem",
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

func TestWalkUp_StopsAtRootMarker(t *testing.T) {
	root := t.TempDir()

	// Create /repo/.stem with root: true
	repoDir := filepath.Join(root, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	repoStem := filepath.Join(repoDir, ".stem")
	if err := os.WriteFile(repoStem, []byte("version: 2\nroot: true\nscope:\n  match: \"*.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create /repo/docs/.stem without root marker
	docsDir := filepath.Join(repoDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	docsStem := filepath.Join(docsDir, ".stem")
	if err := os.WriteFile(docsStem, []byte("version: 2\nscope:\n  match: \"*.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create /repo/docs/task.md
	taskFile := filepath.Join(docsDir, "task.md")
	if err := os.WriteFile(taskFile, []byte("# Task"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := WalkUp(taskFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	// entries[0] should be the root marker at /repo/.stem
	if !entries[0].Stem.Root {
		t.Error("entries[0] should have Root: true")
	}

	// entries[1] should be /repo/docs/.stem
	if filepath.Base(filepath.Dir(entries[1].Path)) != "docs" {
		t.Errorf("entries[1] dir = %q, want docs", filepath.Dir(entries[1].Path))
	}
}

func TestWalkUp_SkipsDirsWithoutStem(t *testing.T) {
	root := t.TempDir()

	// Create /a/.stem with root: true
	aDir := filepath.Join(root, "a")
	if err := os.MkdirAll(aDir, 0o755); err != nil {
		t.Fatal(err)
	}
	aStem := filepath.Join(aDir, ".stem")
	if err := os.WriteFile(aStem, []byte("version: 2\nroot: true\nscope:\n  match: \"*.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create /a/b/c/.stem (no .stem in a/b/)
	bcDir := filepath.Join(aDir, "b", "c")
	if err := os.MkdirAll(bcDir, 0o755); err != nil {
		t.Fatal(err)
	}
	bcStem := filepath.Join(bcDir, ".stem")
	if err := os.WriteFile(bcStem, []byte("version: 2\nscope:\n  match: \"*.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

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
	root := t.TempDir()
	target := filepath.Join(root, "repo")
	if err := os.MkdirAll(target, 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := WalkUp(target)
	// Should return ErrNoSchemaFound when no .stem is found anywhere
	if err != ErrNoSchemaFound {
		t.Fatalf("expected ErrNoSchemaFound, got %v", err)
	}

	if len(entries) > 0 {
		t.Errorf("expected no entries with ErrNoSchemaFound, got %d", len(entries))
	}
}

func TestWalkUp_ReturnsErrNoSchemaFoundWhenZeroStems(t *testing.T) {
	// Create orphan directory with no .stem anywhere in the chain
	root := t.TempDir()
	orphanDir := filepath.Join(root, "orphan", "docs")
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatal(err)
	}

	entries, err := WalkUp(orphanDir)
	if err == nil {
		t.Fatal("expected ErrNoSchemaFound when no .stem is found, got nil")
	}

	if err != ErrNoSchemaFound {
		t.Errorf("expected ErrNoSchemaFound, got %v", err)
	}

	if len(entries) > 0 {
		t.Error("entries should be empty when ErrNoSchemaFound is returned")
	}
}

func TestWalkUp_StableRegardlessOfWorkingDirectory(t *testing.T) {
	root := t.TempDir()

	// Create /project/.stem with root: true
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}
	projectStem := filepath.Join(projectDir, ".stem")
	if err := os.WriteFile(projectStem, []byte("version: 2\nroot: true\nscope:\n  match: \"*.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create /project/docs/.stem
	docsDir := filepath.Join(projectDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	docsStem := filepath.Join(docsDir, ".stem")
	if err := os.WriteFile(docsStem, []byte("version: 2\nscope:\n  match: \"*.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create /project/docs/task.md
	taskFile := filepath.Join(docsDir, "task.md")
	if err := os.WriteFile(taskFile, []byte("# Task"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Call WalkUp twice; both should return identical results
	entries1, err1 := WalkUp(taskFile)
	entries2, err2 := WalkUp(taskFile)

	if err1 != nil || err2 != nil {
		t.Fatalf("unexpected error: err1=%v, err2=%v", err1, err2)
	}

	if len(entries1) != len(entries2) {
		t.Errorf("got different lengths: %d vs %d", len(entries1), len(entries2))
	}

	if entries1[0].Path != entries2[0].Path {
		t.Errorf("entries[0] paths differ: %q vs %q", entries1[0].Path, entries2[0].Path)
	}

	if len(entries1) > 1 && entries1[1].Path != entries2[1].Path {
		t.Errorf("entries[1] paths differ: %q vs %q", entries1[1].Path, entries2[1].Path)
	}
}

func TestWalkUp_GapsNeverStopWalk(t *testing.T) {
	root := t.TempDir()

	// Create /repo/.stem with root: true
	repoDir := filepath.Join(root, "repo")
	if err := os.MkdirAll(repoDir, 0o755); err != nil {
		t.Fatal(err)
	}
	repoStem := filepath.Join(repoDir, ".stem")
	if err := os.WriteFile(repoStem, []byte("version: 2\nroot: true\nscope:\n  match: \"*.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create /repo/docs/.stem
	docsDir := filepath.Join(repoDir, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	docsStem := filepath.Join(docsDir, ".stem")
	if err := os.WriteFile(docsStem, []byte("version: 2\nscope:\n  match: \"*.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create /repo/docs/O01/ with NO .stem (gap)
	gapDir := filepath.Join(docsDir, "O01")
	if err := os.MkdirAll(gapDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create /repo/docs/O01/F01.md
	recordFile := filepath.Join(gapDir, "F01.md")
	if err := os.WriteFile(recordFile, []byte("# Record"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := WalkUp(recordFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should find both /repo/.stem and /repo/docs/.stem; gap at O01 did NOT stop walk
	if len(entries) < 2 {
		t.Errorf("got %d entries, want at least 2 (gap at O01 should not stop walk)", len(entries))
	}

	// First entry should be root marker
	if !entries[0].Stem.Root {
		t.Error("entries[0] should have Root: true")
	}
}

func TestWalkUp_TargetIsFile(t *testing.T) {
	root := setupTree(t,
		[]string{"repo/.stem"},
		"repo/.stem",
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
	content := []byte("version: 2\nscope:\n  match: \"*.md\"\n")
	if err := os.WriteFile(stemPath, content, 0o644); err != nil {
		t.Fatal(err)
	}

	stem, err := ParseStemFile(stemPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stem.Version != 2 {
		t.Errorf("version = %d, want 2", stem.Version)
	}
	if stem.Scope.Match != "*.md" {
		t.Errorf("scope.match = %q, want %q", stem.Scope.Match, "*.md")
	}
}
