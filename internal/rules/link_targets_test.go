package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func writeLinkTargetFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveMarkdownTargets_RewritesResolvableTargets(t *testing.T) {
	root := t.TempDir()
	writeLinkTargetFixture(t, root, map[string]string{
		"README.md":          "# Root\n",
		"docs/a.md":          "# A\n",
		"docs/mi page.md":    "# Encoded\n",
		"docs/sub/README.md": "# Sub index\n",
	})

	rec := &extract.Record{
		Path: filepath.Join("docs", "a.md"),
		Links: []extract.Link{
			{Target: "mi%20page.md", Style: extract.StyleMarkdown},
			{Target: "sub", Style: extract.StyleMarkdown},
			{Target: "../README.md", Style: extract.StyleMarkdown},
			{Target: "/README.md", Style: extract.StyleMarkdown},
		},
	}
	ResolveMarkdownTargets([]*extract.Record{rec}, root)

	want := []string{
		filepath.Join("docs", "mi page.md"),
		filepath.Join("docs", "sub", "README.md"),
		"README.md",
		"README.md",
	}
	for i, w := range want {
		if rec.Links[i].Target != w {
			t.Errorf("link %d target = %q, want %q", i, rec.Links[i].Target, w)
		}
	}
}

func TestResolveMarkdownTargets_LeavesUnresolvableAndWikilinks(t *testing.T) {
	root := t.TempDir()
	writeLinkTargetFixture(t, root, map[string]string{
		"docs/a.md":    "# A\n",
		"docs/Page.md": "# Page\n",
	})

	rec := &extract.Record{
		Path: filepath.Join("docs", "a.md"),
		Links: []extract.Link{
			{Target: "page.md", Style: extract.StyleMarkdown},
			{Target: "no-existe.md", Style: extract.StyleMarkdown},
			{Target: "Page.md", Style: extract.StyleWikilink},
		},
	}
	ResolveMarkdownTargets([]*extract.Record{rec}, root)

	want := []string{"page.md", "no-existe.md", "Page.md"}
	for i, w := range want {
		if rec.Links[i].Target != w {
			t.Errorf("link %d target = %q, want %q", i, rec.Links[i].Target, w)
		}
	}
}
