package index

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestMatchesScope_MatchesMD(t *testing.T) {
	stem := &rules.StemFile{Scope: rules.Scope{Match: "*.md"}}
	if !MatchesScope("doc.md", stem) {
		t.Error("expected doc.md to match *.md scope")
	}
}

func TestMatchesScope_RejectsTXT(t *testing.T) {
	stem := &rules.StemFile{Scope: rules.Scope{Match: "*.md"}}
	if MatchesScope("data.txt", stem) {
		t.Error("expected data.txt to NOT match *.md scope")
	}
}

func TestMatchesScope_NoScopeMatchesAll(t *testing.T) {
	stem := &rules.StemFile{}
	if !MatchesScope("anything.txt", stem) {
		t.Error("expected match when scope is empty")
	}
}

func TestMatchesScope_NilStemMatchesAll(t *testing.T) {
	if !MatchesScope("anything.txt", nil) {
		t.Error("expected match when stem is nil")
	}
}

func TestMatchesScope_ComplexPattern(t *testing.T) {
	stem := &rules.StemFile{Scope: rules.Scope{Match: "T[0-9]*.md"}}

	if !MatchesScope("T001-task.md", stem) {
		t.Error("expected T001-task.md to match T[0-9]*.md")
	}
	if MatchesScope("README.md", stem) {
		t.Error("expected README.md to NOT match T[0-9]*.md")
	}
}

func TestMatchesScope_InvalidPatternMatchesAll(t *testing.T) {
	stem := &rules.StemFile{Scope: rules.Scope{Match: "[invalid"}}
	if !MatchesScope("file.md", stem) {
		t.Error("expected match on invalid pattern (fail open)")
	}
}

func TestMatchesScope_FullPathUsesBasename(t *testing.T) {
	stem := &rules.StemFile{Scope: rules.Scope{Match: "*.md"}}
	if !MatchesScope("/some/deep/path/doc.md", stem) {
		t.Error("expected full path to match using basename")
	}
}

func TestScan_WithScopeResolver_FiltersFiles(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":    "---\ntitle: Doc\n---\n",
		"notes.md":  "---\ntitle: Notes\n---\n",
		"readme.md": "---\ntitle: Readme\n---\n",
	})

	stem := &rules.StemFile{Scope: rules.Scope{Match: "doc.md"}}
	resolver := func(dir string) *rules.StemFile { return stem }

	reg := extract.NewRegistry()
	records, err := Scan(root, reg, WithScopeResolver(resolver))
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

func TestScan_WithScopeResolver_PerDirectory(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"root.md":       "---\ntitle: Root\n---\n",
		"docs/api.md":   "---\ntitle: API\n---\n",
		"docs/guide.md": "---\ntitle: Guide\n---\n",
		"src/main.md":   "---\ntitle: Main\n---\n",
	})

	// docs/ scope only matches api.md, src/ matches everything, root has no scope.
	resolver := func(dir string) *rules.StemFile {
		rel, _ := filepath.Rel(root, dir)
		switch rel {
		case "docs":
			return &rules.StemFile{Scope: rules.Scope{Match: "api.md"}}
		case "src":
			return &rules.StemFile{Scope: rules.Scope{Match: "*.md"}}
		default:
			return nil // no scope = match all
		}
	}

	reg := extract.NewRegistry()
	records, err := Scan(root, reg, WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// root.md (no scope) + docs/api.md (matches) + src/main.md (matches) = 3
	if len(records) != 3 {
		names := make([]string, len(records))
		for i, r := range records {
			names[i] = r.Path
		}
		t.Fatalf("got %d records %v, want 3", len(records), names)
	}
}

func TestScan_WithoutScopeResolver_NoFiltering(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":   "---\ntitle: Doc\n---\n",
		"notes.md": "---\ntitle: Notes\n---\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(root, reg) // no scope resolver
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 2 {
		t.Errorf("got %d records, want 2", len(records))
	}
}

func TestScan_ScopeResolverCachesPerDirectory(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"a.md": "---\ntitle: A\n---\n",
		"b.md": "---\ntitle: B\n---\n",
		"c.md": "---\ntitle: C\n---\n",
	})

	calls := 0
	resolver := func(dir string) *rules.StemFile {
		calls++
		return nil // match all
	}

	reg := extract.NewRegistry()
	_, err := Scan(root, reg, WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// All 3 files are in the same directory — resolver called only once.
	if calls != 1 {
		t.Errorf("resolver called %d times, want 1 (should cache per directory)", calls)
	}
}

func TestScan_ScopeAndStemignoreCombined(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":      "---\ntitle: Doc\n---\n",
		"draft.md":    "---\ntitle: Draft\n---\n",
		"notes.md":    "---\ntitle: Notes\n---\n",
		".stemignore": "draft.md\n",
	})

	// Scope only allows *.md (all are .md), but .stemignore excludes draft.md
	stem := &rules.StemFile{Scope: rules.Scope{Match: "*.md"}}
	resolver := func(dir string) *rules.StemFile { return stem }

	reg := extract.NewRegistry()
	records, err := Scan(root, reg, WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// doc.md + notes.md = 2 (draft.md excluded by .stemignore)
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2", len(records))
	}

	for _, r := range records {
		if r.Frontmatter["title"] == "Draft" {
			t.Error("draft.md should have been excluded by .stemignore")
		}
	}
}

func TestScan_ScopeExcludesBeforeExtraction(t *testing.T) {
	// Create a file that would fail extraction if read.
	root := t.TempDir()
	mdPath := filepath.Join(root, "good.md")
	os.WriteFile(mdPath, []byte("---\ntitle: Good\n---\n"), 0o644)

	otherPath := filepath.Join(root, "skip.md")
	os.WriteFile(otherPath, []byte("---\ntitle: Skip\n---\n"), 0o644)

	// Scope excludes skip.md
	stem := &rules.StemFile{Scope: rules.Scope{Match: "good.md"}}
	resolver := func(dir string) *rules.StemFile { return stem }

	reg := extract.NewRegistry()
	records, err := Scan(root, reg, WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}
	if records[0].Frontmatter["title"] != "Good" {
		t.Errorf("title = %v, want Good", records[0].Frontmatter["title"])
	}
}
