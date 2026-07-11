package e2e

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestE2E_LinkStyles_GraphIncludesMarkdownWhenDeclared(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":            "version: 2\nlinks:\n  styles: [markdown]\n",
		"README.md":        "# Root\n\n[overview](docs/overview.md)\n",
		"docs/overview.md": "# Overview\n\n[back](../README.md)\n",
	})
	ctx := context.Background()
	records, err := index.Scan(ctx, root, extract.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	rules.FilterLinksByStyles(records, root)
	rules.ResolveMarkdownTargets(records, root)
	g := graph.Build(ctx, records)

	if len(g.Edges["README.md"]) != 1 {
		t.Fatalf("README edges = %+v, want 1 markdown edge", g.Edges["README.md"])
	}
	if g.Edges["README.md"][0].Target != filepath.Join("docs", "overview.md") {
		t.Errorf("edge target = %q", g.Edges["README.md"][0].Target)
	}
	back := g.Edges[filepath.Join("docs", "overview.md")]
	if len(back) != 1 || back[0].Target != "README.md" {
		t.Errorf("back edges = %+v, want one edge to README.md", back)
	}
	if broken := g.BrokenLinks(); len(broken) != 0 {
		t.Errorf("broken = %+v, want none", broken)
	}
}

func TestE2E_LinkStyles_WikilinkRepoUnaffected(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": "version: 2\nlinks:\n  allowed: [reference]\n",
		"a.md":  "[[b]]\n\nAlso a [markdown link](c.md) that must stay invisible.\n",
		"b.md":  "# B\n",
	})
	ctx := context.Background()
	records, err := index.Scan(ctx, root, extract.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}

	// Validation: the markdown link to a NONEXISTENT c.md produces no errors.
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		effective, err := rules.ResolveForRecord(filepath.Dir(absPath), rec.Path)
		if err != nil {
			t.Fatal(err)
		}
		errs := rules.Validate(ctx, rec, effective)
		errs = append(errs, rules.CheckLinks(rec.Links, effective.Links, absPath, nil)...)
		if len(errs) != 0 {
			t.Errorf("%s: unexpected errors %+v", rec.Path, errs)
		}
	}

	// Graph: only the wikilink edge exists.
	rules.FilterLinksByStyles(records, root)
	g := graph.Build(ctx, records)
	edges := g.Edges["a.md"]
	if len(edges) != 1 || edges[0].Type != "reference" {
		t.Fatalf("edges = %+v, want single wikilink edge to b", edges)
	}
	if edges[0].Target != "b.md" {
		t.Errorf("edge target = %q, want b.md", edges[0].Target)
	}
	if edges[0].Target == "c.md" {
		t.Error("markdown link leaked into wikilink-only graph")
	}
}

func TestE2E_LinkStyles_GraphMixedStyles(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":     "version: 2\nlinks:\n  styles: [wikilink, markdown]\n",
		"README.md": "# Root\n\n[[a.md]]\n[b](docs/b.md)\n",
		"a.md":      "# A\n",
		"docs/b.md": "# B\n",
	})
	ctx := context.Background()
	records, err := index.Scan(ctx, root, extract.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	rules.FilterLinksByStyles(records, root)
	rules.ResolveMarkdownTargets(records, root)
	g := graph.Build(ctx, records)

	if len(g.Edges["README.md"]) != 2 {
		t.Fatalf("README edges = %+v, want wikilink + markdown edges", g.Edges["README.md"])
	}
	if broken := g.BrokenLinks(); len(broken) != 0 {
		t.Errorf("broken = %+v, want none", broken)
	}
}
