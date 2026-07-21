package rules

import (
	"os"
	"path/filepath"
	"strings"
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

// Slice 2: Cross-repository warning test
// TestChainHasNoDeclaredBoundary verifies detection of chains that terminated
// at the filesystem root without a root: true marker.
func TestChainHasNoDeclaredBoundary(t *testing.T) {
	t.Run("chain with marker has declared boundary", func(t *testing.T) {
		root := t.TempDir()
		stemPath := filepath.Join(root, ".stem")
		if err := os.WriteFile(stemPath, []byte("version: 2\nroot: true\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		entries, err := WalkUp(root)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have one entry with root: true
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if !entries[0].Stem.Root {
			t.Error("expected Root to be true")
		}

		// Chain has declared boundary
		if ChainHasNoDeclaredBoundary(entries) {
			t.Error("chain with root: true should have declared boundary")
		}
	})

	t.Run("chain without marker has no declared boundary", func(t *testing.T) {
		tmpdir := t.TempDir()
		// Create a .stem without root: true marker
		stemPath := filepath.Join(tmpdir, ".stem")
		if err := os.WriteFile(stemPath, []byte("version: 2\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		entries, err := WalkUp(tmpdir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Should have one entry without root: true (reached filesystem root)
		if len(entries) != 1 {
			t.Fatalf("got %d entries, want 1", len(entries))
		}
		if entries[0].Stem.Root {
			t.Error("expected Root to be false")
		}

		// Chain does NOT have declared boundary
		if !ChainHasNoDeclaredBoundary(entries) {
			t.Error("chain without root: true should have no declared boundary")
		}
	})
}

// TestProposeRootDirectory verifies that ProposeRootDirectory returns the directory
// that should be marked as root.
func TestProposeRootDirectory(t *testing.T) {
	t.Run("proposes the root of the chain", func(t *testing.T) {
		tmpdir := t.TempDir()
		// Create .stem in tmpdir
		stemPath := filepath.Join(tmpdir, ".stem")
		if err := os.WriteFile(stemPath, []byte("version: 2\n"), 0o644); err != nil {
			t.Fatal(err)
		}

		entries, err := WalkUp(tmpdir)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		proposed := ProposeRootDirectory(entries, tmpdir)
		if proposed != tmpdir {
			t.Errorf("expected %q, got %q", tmpdir, proposed)
		}
	})

	t.Run("returns empty string for empty entries", func(t *testing.T) {
		proposed := ProposeRootDirectory(nil, "")
		if proposed != "" {
			t.Errorf("expected empty string for nil entries, got %q", proposed)
		}
	})
}

// TestApplyRootMarker verifies that the root marker can be applied to a .stem file.
func TestApplyRootMarker(t *testing.T) {
	t.Run("adds root: true to .stem file", func(t *testing.T) {
		tmpdir := t.TempDir()
		stemPath := filepath.Join(tmpdir, ".stem")

		// Create a .stem without root: true
		content := []byte("version: 2\nscope:\n  match: \"*.md\"\n")
		if err := os.WriteFile(stemPath, content, 0o644); err != nil {
			t.Fatal(err)
		}

		// Apply root marker
		err := ApplyRootMarker(stemPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Read back and verify
		data, err := os.ReadFile(stemPath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		if !strings.Contains(string(data), "root: true") {
			t.Errorf("root: true not found in file content: %s", string(data))
		}
	})

	t.Run("is idempotent", func(t *testing.T) {
		tmpdir := t.TempDir()
		stemPath := filepath.Join(tmpdir, ".stem")

		// Create a .stem with root: true
		content := []byte("version: 2\nroot: true\nscope:\n  match: \"*.md\"\n")
		if err := os.WriteFile(stemPath, content, 0o644); err != nil {
			t.Fatal(err)
		}

		// Apply root marker again
		err := ApplyRootMarker(stemPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Read back and verify it's still valid
		data, err := os.ReadFile(stemPath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		// Should still contain root: true only once
		count := strings.Count(string(data), "root: true")
		if count != 1 {
			t.Errorf("root: true appears %d times, expected 1", count)
		}
	})

	t.Run("handles .stem with no version line", func(t *testing.T) {
		tmpdir := t.TempDir()
		stemPath := filepath.Join(tmpdir, ".stem")

		// Create a minimal .stem without version
		content := []byte("scope:\n  match: \"*.md\"\n")
		if err := os.WriteFile(stemPath, content, 0o644); err != nil {
			t.Fatal(err)
		}

		// Apply root marker
		err := ApplyRootMarker(stemPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Read back and verify
		data, err := os.ReadFile(stemPath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		if !strings.Contains(string(data), "root: true") {
			t.Errorf("root: true not found in file")
		}
	})

	t.Run("preserves file content when adding root marker", func(t *testing.T) {
		tmpdir := t.TempDir()
		stemPath := filepath.Join(tmpdir, ".stem")

		originalContent := []byte("version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: string\n")
		if err := os.WriteFile(stemPath, originalContent, 0o644); err != nil {
			t.Fatal(err)
		}

		// Apply root marker
		err := ApplyRootMarker(stemPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		// Read back and verify all original content is still there
		data, err := os.ReadFile(stemPath)
		if err != nil {
			t.Fatalf("failed to read file: %v", err)
		}

		content := string(data)
		if !strings.Contains(content, "version: 2") {
			t.Error("version line lost")
		}
		if !strings.Contains(content, "scope:") {
			t.Error("scope section lost")
		}
		if !strings.Contains(content, "schema:") {
			t.Error("schema section lost")
		}
		if !strings.Contains(content, "root: true") {
			t.Error("root: true not added")
		}
	})
}

// TestComposeNoDeclaredBoundaryError verifies the error message for non-interactive mode.
func TestComposeNoDeclaredBoundaryError(t *testing.T) {
	tmpdir := t.TempDir()
	stemPath := filepath.Join(tmpdir, ".stem")

	// Create a .stem without root: true
	if err := os.WriteFile(stemPath, []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	proposed := tmpdir

	msg := ComposeNoDeclaredBoundaryError(stemPath, proposed)

	// Error message must contain:
	// 1. What happened (no boundary)
	// 2. Which stray .stem was picked up
	// 3. Which directory should own the marker
	// 4. The exact line to add
	if msg == "" {
		t.Error("error message must not be empty")
	}
	if !strings.Contains(msg, stemPath) {
		t.Errorf("error message must mention stray .stem: %s", msg)
	}
	if !strings.Contains(msg, "root: true") {
		t.Errorf("error message must include the exact line to add: %s", msg)
	}
}

// TestAttemptRootMarkerMigration_NoTTY verifies that migration fails with a clear
// error when no TTY is available (non-interactive environment like CI).
func TestAttemptRootMarkerMigration_NoTTY(t *testing.T) {
	tmpdir := t.TempDir()
	stemPath := filepath.Join(tmpdir, ".stem")

	// Create a .stem without root: true
	if err := os.WriteFile(stemPath, []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Build entries as if from WalkUp
	entries, err := WalkUp(tmpdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// No TTY: hasStdin = false
	result := AttemptRootMarkerMigration(entries, tmpdir, false)

	// Should not apply
	if result.Applied {
		t.Error("migration should not apply without TTY")
	}

	// Should have error message
	if result.Error == "" {
		t.Error("migration should return error message without TTY")
	}

	// Error should be actionable
	if !strings.Contains(result.Error, "root: true") {
		t.Errorf("error should include remediation: %s", result.Error)
	}

	// File should not be modified
	data, _ := os.ReadFile(stemPath)
	if strings.Contains(string(data), "root: true") {
		t.Error("file should not have been modified")
	}
}

// TestAttemptRootMarkerMigration_WithMarker verifies that migration skips
// when a marker already exists.
func TestAttemptRootMarkerMigration_WithMarker(t *testing.T) {
	tmpdir := t.TempDir()
	stemPath := filepath.Join(tmpdir, ".stem")

	// Create a .stem WITH root: true
	if err := os.WriteFile(stemPath, []byte("version: 2\nroot: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := WalkUp(tmpdir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should not attempt migration when marker exists
	result := AttemptRootMarkerMigration(entries, tmpdir, true)

	if result.Applied {
		t.Error("migration should not apply when marker already exists")
	}
	if result.Error != "" {
		t.Errorf("migration should not error when marker exists: %s", result.Error)
	}
}

// TestStemCache_WalkUp verifies that the StemCache correctly caches results
// from WalkUp, reducing filesystem I/O on repeated calls.
func TestStemCache_WalkUp(t *testing.T) {
	root := setupTree(t,
		[]string{"a/.stem", "a/b/.stem", "a/b/c/.stem"},
		"a/.stem",
	)

	target := filepath.Join(root, "a", "b", "c")
	cache := NewStemCache()

	// First call: cache miss
	entries1, err1 := cache.WalkUp(target)
	if err1 != nil {
		t.Fatalf("first WalkUp failed: %v", err1)
	}

	// Second call: cache hit
	entries2, err2 := cache.WalkUp(target)
	if err2 != nil {
		t.Fatalf("second WalkUp failed: %v", err2)
	}

	// Results should be identical
	if len(entries1) != len(entries2) {
		t.Errorf("cache returned different length: %d vs %d", len(entries1), len(entries2))
	}

	// Cache should have at least 3 hits (for the 3 .stem files)
	hits := cache.Hits()
	if hits < 1 {
		t.Errorf("cache hits should be > 0, got %d", hits)
	}
}

// TestWalkUp_EmptyEntries_ChainHasNoDeclaredBoundary verifies edge case
// where entries slice is nil or empty.
func TestWalkUp_EmptyEntries_ChainHasNoDeclaredBoundary(t *testing.T) {
	result := ChainHasNoDeclaredBoundary(nil)
	if !result {
		t.Error("empty entries should report no declared boundary")
	}

	result = ChainHasNoDeclaredBoundary([]StemEntry{})
	if !result {
		t.Error("zero-length entries should report no declared boundary")
	}
}

// TestProposeRootDirectory_WithGitHint verifies that ProposeRootDirectory
// uses git repository as a hint for boundary placement.
func TestProposeRootDirectory_WithGitHint(t *testing.T) {
	tmpdir := t.TempDir()

	// Create .git marker (used as boundary hint)
	gitDir := filepath.Join(tmpdir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create nested .stem structure
	level1 := filepath.Join(tmpdir, "level1")
	level2 := filepath.Join(level1, "level2")
	if err := os.MkdirAll(level2, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create .stem files
	for _, dir := range []string{tmpdir, level1, level2} {
		stemPath := filepath.Join(dir, ".stem")
		if err := os.WriteFile(stemPath, []byte("version: 2\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	// Load entries from level2
	entries, err := WalkUp(level2)
	if err != nil {
		t.Fatalf("WalkUp failed: %v", err)
	}

	// ProposeRootDirectory should prefer a location inside .git boundary
	proposed := ProposeRootDirectory(entries, level2)
	if proposed == "" {
		t.Fatalf("ProposeRootDirectory returned empty string")
	}

	// Proposed directory should be at or inside tmpdir (the .git root)
	rel, err := filepath.Rel(tmpdir, proposed)
	if err != nil || strings.HasPrefix(rel, "..") {
		t.Errorf("proposed dir %q should be inside git root %q", proposed, tmpdir)
	}
}

// TestIsAtOrBelow_EdgeCases verifies path containment logic.
func TestIsAtOrBelow_EdgeCases(t *testing.T) {
	tmpdir := t.TempDir()

	// Same path
	if !isAtOrBelow(tmpdir, tmpdir) {
		t.Error("isAtOrBelow should return true for same path")
	}

	// Child path
	child := filepath.Join(tmpdir, "child")
	if !isAtOrBelow(child, tmpdir) {
		t.Error("isAtOrBelow should return true for child path")
	}

	// Parent path
	if isAtOrBelow(tmpdir, child) {
		t.Error("isAtOrBelow should return false for parent path")
	}

	// Unrelated path
	tmpdir2 := t.TempDir()
	if isAtOrBelow(tmpdir, tmpdir2) {
		t.Error("isAtOrBelow should return false for unrelated paths")
	}
}

// TestProjectBoundaryHint_NoGit verifies that projectBoundaryHint returns
// empty string when no .git is found.
func TestProjectBoundaryHint_NoGit(t *testing.T) {
	tmpdir := t.TempDir()
	// Create a directory with no .git anywhere above it
	targetDir := filepath.Join(tmpdir, "project")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := projectBoundaryHint(targetDir)
	if result != "" {
		t.Errorf("projectBoundaryHint should return empty when no .git, got %q", result)
	}
}

// TestProjectBoundaryHint_WithGit verifies that projectBoundaryHint finds
// the .git directory.
func TestProjectBoundaryHint_WithGit(t *testing.T) {
	tmpdir := t.TempDir()

	// Create .git marker
	gitDir := filepath.Join(tmpdir, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create nested target
	targetDir := filepath.Join(tmpdir, "level1", "level2")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	result := projectBoundaryHint(targetDir)
	if result != tmpdir {
		t.Errorf("projectBoundaryHint should return %q, got %q", tmpdir, result)
	}
}

// TestWalkUp_AbsPathConversion verifies that WalkUp correctly handles
// relative paths by converting them to absolute.
func TestWalkUp_AbsPathConversion(t *testing.T) {
	root := setupTree(t,
		[]string{"project/.stem"},
		"project/.stem",
	)

	projectDir := filepath.Join(root, "project")
	targetDir := filepath.Join(projectDir, "subdir")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a file for testing
	testFile := filepath.Join(targetDir, "test.md")
	if err := os.WriteFile(testFile, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Save current directory to restore later
	oldwd, err := os.Getwd()
	if err != nil {
		t.Fatalf("failed to get wd: %v", err)
	}
	defer func() { _ = os.Chdir(oldwd) }()

	// Change to the root to test relative path resolution
	if err := os.Chdir(root); err != nil {
		t.Fatalf("failed to chdir: %v", err)
	}

	// Resolve using relative path
	relPath := filepath.Join("project", "subdir", "test.md")
	entries, err := WalkUp(relPath)
	if err != nil {
		t.Fatalf("WalkUp with relative path failed: %v", err)
	}

	// Should find the .stem file
	if len(entries) == 0 {
		t.Fatal("expected to find .stem with relative path")
	}
}
