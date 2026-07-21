package index

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// helper to create a temp tree with files and optional .stemignore.
func setupScanTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for relPath, content := range files {
		absPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

func TestScan_FindsMarkdownRecursively(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":          "---\ntitle: Root\n---\n# Root",
		"sub/readme.md":   "---\ntitle: Sub\n---\n# Sub",
		"sub/deep/api.md": "---\ntitle: API\n---\n# API",
	})

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 3 {
		t.Errorf("got %d records, want 3", len(records))
	}
}

func TestScan_ExcludesStemignore(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":      "---\ntitle: Keep\n---\n",
		"draft.md":    "---\ntitle: Draft\n---\n",
		".stemignore": "draft.md\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Frontmatter["title"] != "Keep" {
		t.Errorf("title = %v, want Keep", records[0].Frontmatter["title"])
	}
}

func TestScan_StemignoreGlobPattern(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":      "---\ntitle: Keep\n---\n",
		"notes.log":   "not markdown",
		"data.txt":    "not markdown",
		".stemignore": "*.log\n*.txt\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only doc.md should be found (others have no extractor anyway,
	// but .stemignore should also exclude them).
	if len(records) != 1 {
		t.Errorf("got %d records, want 1", len(records))
	}
}

func TestScan_StemignoreInSubdirectory(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"root.md":         "---\ntitle: Root\n---\n",
		"sub/keep.md":     "---\ntitle: Keep\n---\n",
		"sub/ignored.md":  "---\ntitle: Ignored\n---\n",
		"sub/.stemignore": "ignored.md\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// root.md + sub/keep.md = 2 (sub/ignored.md excluded).
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}

	titles := map[string]bool{}
	for _, r := range records {
		if t, ok := r.Frontmatter["title"].(string); ok {
			titles[t] = true
		}
	}
	if titles["Ignored"] {
		t.Error("sub/ignored.md should have been excluded by sub/.stemignore")
	}
}

func TestScan_StemignoreComments(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":      "---\ntitle: Doc\n---\n",
		"draft.md":    "---\ntitle: Draft\n---\n",
		".stemignore": "# This is a comment\n\ndraft.md\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Errorf("got %d records, want 1", len(records))
	}
}

func TestScan_ExcludesGitDir(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":              "---\ntitle: Doc\n---\n",
		".git/config":         "gitconfig",
		".git/objects/abc.md": "---\ntitle: Git\n---\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Frontmatter["title"] != "Doc" {
		t.Errorf("title = %v, want Doc", records[0].Frontmatter["title"])
	}
}

func TestScan_EmptyDirectory(t *testing.T) {
	root := t.TempDir()

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 0 {
		t.Errorf("got %d records, want 0", len(records))
	}
}

func TestScan_IgnoresUnknownExtensions(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":     "---\ntitle: Doc\n---\n",
		"data.json":  `{"key": "value"}`,
		"config.yml": "key: value",
		"script.sh":  "#!/bin/bash",
	})

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Only .md has a registered extractor.
	if len(records) != 1 {
		t.Errorf("got %d records, want 1", len(records))
	}
}

func TestScan_DelegatesExtractionCorrectly(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md": "---\ntitle: Hello\nstatus: draft\n---\n# Content\n\nBody text.",
	})

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	rec := records[0]
	if rec.Type != "markdown" {
		t.Errorf("type = %q, want markdown", rec.Type)
	}
	if rec.Frontmatter["title"] != "Hello" {
		t.Errorf("title = %v, want Hello", rec.Frontmatter["title"])
	}
	if rec.Frontmatter["status"] != "draft" {
		t.Errorf("status = %v, want draft", rec.Frontmatter["status"])
	}
	if rec.Body != "# Content\n\nBody text." {
		t.Errorf("body = %q", rec.Body)
	}
}

func TestScan_StemignoreDoesNotApplyToSiblingPrefix(t *testing.T) {
	// Regression: a .stemignore in "sub/" must NOT apply to "sub-extra/"
	// because "sub-extra" starts with "sub" as a string prefix.
	root := setupScanTree(t, map[string]string{
		"sub/keep.md":       "---\ntitle: Keep\n---\n",
		"sub/ignored.md":    "---\ntitle: Ignored\n---\n",
		"sub/.stemignore":   "ignored.md\n",
		"sub-extra/keep.md": "---\ntitle: SiblingKeep\n---\n",
		"sub-extra/also.md": "---\ntitle: SiblingAlso\n---\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// sub/keep.md + sub-extra/keep.md + sub-extra/also.md = 3
	// sub/ignored.md is excluded by sub/.stemignore.
	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}

	titles := map[string]bool{}
	for _, r := range records {
		if title, ok := r.Frontmatter["title"].(string); ok {
			titles[title] = true
		}
	}
	if titles["Ignored"] {
		t.Error("sub/ignored.md should be excluded by sub/.stemignore")
	}
	if !titles["SiblingKeep"] || !titles["SiblingAlso"] {
		t.Error("sub-extra/ files should NOT be excluded by sub/.stemignore")
	}
}

func TestParseStemignore(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, ".stemignore")
	content := "# comment\n\n*.log\ndraft.md\n"
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}

	patterns, err := parseStemignore(path)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(patterns) != 2 {
		t.Fatalf("got %d patterns, want 2", len(patterns))
	}
	if patterns[0] != "*.log" || patterns[1] != "draft.md" {
		t.Errorf("patterns = %v, want [*.log draft.md]", patterns)
	}
}

func TestScan_ParallelProducesSameResults(t *testing.T) {
	files := make(map[string]string, 50)
	for i := 0; i < 50; i++ {
		path := fmt.Sprintf("docs/doc%03d.md", i)
		files[path] = fmt.Sprintf("---\ntitle: Doc %d\n---\n# Doc %d\n\nBody of doc %d.", i, i, i)
	}

	root := setupScanTree(t, files)
	reg := extract.NewRegistry()

	records, err := Scan(context.Background(), root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 50 {
		t.Fatalf("got %d records, want 50", len(records))
	}

	// Verify all titles are present.
	titles := make(map[string]bool, 50)
	for _, r := range records {
		if title, ok := r.Frontmatter["title"].(string); ok {
			titles[title] = true
		}
	}
	for i := 0; i < 50; i++ {
		expected := fmt.Sprintf("Doc %d", i)
		if !titles[expected] {
			t.Errorf("missing record with title %q", expected)
		}
	}
}

func TestParseStemignore_NotFound(t *testing.T) {
	_, err := parseStemignore("/nonexistent/.stemignore")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

// TestScan_ResolverHardErrorAbortsScan tests that a hard error from the
// resolver (parse error, IO error) aborts the entire scan.
func TestScan_ResolverHardErrorAbortsScan(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md": "---\ntitle: Doc\n---\n",
	})

	reg := extract.NewRegistry()

	// Create a resolver that returns a hard error.
	resolver := func(dir string) (*rules.StemFile, error) {
		return nil, fmt.Errorf("simulated read error")
	}

	records, err := Scan(context.Background(), root, reg, WithScopeResolver(resolver))
	if err == nil {
		t.Fatal("expected an error from resolver hard failure, got success")
	}
	if records != nil {
		t.Errorf("expected nil records on hard error, got %v", records)
	}
	if !errors.Is(err, fmt.Errorf("simulated read error")) && err.Error() != "simulated read error" {
		t.Errorf("error should be the hard error, got: %v", err)
	}
}

// TestScan_ResolverErrNoSchemaFoundContinuesScan tests that when a resolver
// returns ErrNoSchemaFound for a directory, the scan skips that directory and
// continues scanning other files.
func TestScan_ResolverErrNoSchemaFoundContinuesScan(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":      "---\ntitle: Doc\n---\n",
		"sub/file.md": "---\ntitle: File\n---\n",
	})

	reg := extract.NewRegistry()

	// Create a resolver that returns ErrNoSchemaFound for "sub" but success for root.
	resolver := func(dir string) (*rules.StemFile, error) {
		if filepath.Base(dir) == "sub" {
			return nil, rules.ErrNoSchemaFound
		}
		// Return a minimal stem for root.
		return &rules.StemFile{}, nil
	}

	records, err := Scan(context.Background(), root, reg, WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("expected scan to continue after ErrNoSchemaFound, got error: %v", err)
	}

	// Should have found only doc.md (at root where resolver succeeds).
	// sub/file.md should have been skipped (resolver returned ErrNoSchemaFound for sub).
	if len(records) != 1 {
		paths := make([]string, len(records))
		for i, r := range records {
			paths[i] = r.Path
		}
		t.Errorf("expected 1 record (only root doc.md), got %d: %v", len(records), paths)
	}
	if records[0].Path != "doc.md" {
		t.Errorf("expected record path doc.md, got %q", records[0].Path)
	}
}

// TestScan_NoResolverResultReturnsErrNoSchemaFound tests that when a scan
// finds no files that can be resolved (all directories have no schema),
// the scan returns ErrNoSchemaFound instead of (nil, nil).
func TestScan_NoResolverResultReturnsErrNoSchemaFound(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md": "---\ntitle: Doc\n---\n",
	})

	reg := extract.NewRegistry()

	// Create a resolver that always returns ErrNoSchemaFound.
	resolver := func(dir string) (*rules.StemFile, error) {
		return nil, rules.ErrNoSchemaFound
	}

	records, err := Scan(context.Background(), root, reg, WithScopeResolver(resolver))
	if err == nil {
		t.Fatal("expected ErrNoSchemaFound when no files resolve, got success")
	}
	if !errors.Is(err, rules.ErrNoSchemaFound) {
		t.Fatalf("expected ErrNoSchemaFound, got: %v", err)
	}
	if records != nil {
		t.Errorf("expected nil records when returning ErrNoSchemaFound, got %v", records)
	}
}
