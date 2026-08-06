package index

import (
	"context"
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
	resolver := func(dir string) (*rules.StemFile, error) { return stem, nil }

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg, WithScopeResolver(resolver))
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

	// docs/ scope only matches api.md, src/ matches everything, root has no stem.
	resolver := func(dir string) (*rules.StemFile, error) {
		rel, _ := filepath.Rel(root, dir)
		switch rel {
		case "docs":
			return &rules.StemFile{Scope: rules.Scope{Match: "api.md"}}, nil
		case "src":
			return &rules.StemFile{Scope: rules.Scope{Match: "*.md"}}, nil
		default:
			return nil, rules.ErrNoSchemaFound // no stem = exclude
		}
	}

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg, WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// root.md excluded (ErrNoSchemaFound) + docs/api.md (matches) + src/main.md (matches) = 2
	if len(records) != 2 {
		names := make([]string, len(records))
		for i, r := range records {
			names[i] = r.Path
		}
		t.Fatalf("got %d records %v, want 2", len(records), names)
	}
}

func TestScan_WithoutScopeResolver_NoFiltering(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		"doc.md":   "---\ntitle: Doc\n---\n",
		"notes.md": "---\ntitle: Notes\n---\n",
	})

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg) // no scope resolver
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
	resolver := func(dir string) (*rules.StemFile, error) {
		calls++
		return &rules.StemFile{}, nil // empty stem = match all
	}

	reg := extract.NewRegistry()
	_, err := Scan(context.Background(), root, reg, WithScopeResolver(resolver))
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
	resolver := func(dir string) (*rules.StemFile, error) { return stem, nil }

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg, WithScopeResolver(resolver))
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
	root := setupScanTree(t, map[string]string{
		"good.md": "---\ntitle: Good\n---\n",
		"skip.md": "---\ntitle: Skip\n---\n",
	})

	// Scope excludes skip.md
	stem := &rules.StemFile{Scope: rules.Scope{Match: "good.md"}}
	resolver := func(dir string) (*rules.StemFile, error) { return stem, nil }

	reg := extract.NewRegistry()
	records, err := Scan(context.Background(), root, reg, WithScopeResolver(resolver))
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

func TestIsIgnored_AppliesRootAndNestedPatterns(t *testing.T) {
	root := setupScanTree(t, map[string]string{
		".stemignore":        "*.tmp\n",
		"root.tmp":           "",
		"nested/.stemignore": "draft.md\n",
		"nested/draft.md":    "",
		"nested/keep.md":     "",
	})

	tests := []struct {
		path string
		want bool
	}{
		{path: "root.tmp", want: true},
		{path: "nested/draft.md", want: true},
		{path: "nested/keep.md", want: false},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			got := IsIgnored(root, filepath.Join(root, tt.path))
			if got != tt.want {
				t.Fatalf("IsIgnored(%q) = %v, want %v", tt.path, got, tt.want)
			}
		})
	}
}
