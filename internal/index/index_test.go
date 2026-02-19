package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
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
	records, err := Scan(root, reg)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 3 {
		t.Errorf("got %d records, want 3", len(records))
	}
}

func TestScan_ExcludesStemignore(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":       "---\ntitle: Keep\n---\n",
		"draft.md":     "---\ntitle: Draft\n---\n",
		".stemignore":  "draft.md\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(root, reg)
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
		"doc.md":       "---\ntitle: Keep\n---\n",
		"notes.log":    "not markdown",
		"data.txt":     "not markdown",
		".stemignore":  "*.log\n*.txt\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(root, reg)
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
		"root.md":          "---\ntitle: Root\n---\n",
		"sub/keep.md":      "---\ntitle: Keep\n---\n",
		"sub/ignored.md":   "---\ntitle: Ignored\n---\n",
		"sub/.stemignore":  "ignored.md\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(root, reg)
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
		"doc.md":       "---\ntitle: Doc\n---\n",
		"draft.md":     "---\ntitle: Draft\n---\n",
		".stemignore":  "# This is a comment\n\ndraft.md\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(root, reg)
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
	records, err := Scan(root, reg)
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
	records, err := Scan(root, reg)
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
	records, err := Scan(root, reg)
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
	records, err := Scan(root, reg)
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

func TestParseStemignore_NotFound(t *testing.T) {
	_, err := parseStemignore("/nonexistent/.stemignore")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}
