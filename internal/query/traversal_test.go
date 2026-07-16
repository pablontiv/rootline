package query

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
)

func subWhere(s string) *string { return &s }

// traversalFixture builds a small linked corpus:
//
//	sources/witness-a.md (verification: corroborated) --supports--> entities/tool-a.md
//	sources/witness-b.md (verification: unverified)   --supports--> entities/tool-b.md
//	entities/tool-c.md   (no inbound links)
func traversalFixture() []*extract.Record {
	return []*extract.Record{
		{
			Path:        "entities/tool-a.md",
			Frontmatter: map[string]any{"kind": "tool"},
		},
		{
			Path:        "entities/tool-b.md",
			Frontmatter: map[string]any{"kind": "tool"},
		},
		{
			Path:        "entities/tool-c.md",
			Frontmatter: map[string]any{"kind": "tool"},
		},
		{
			Path:        "sources/witness-a.md",
			Frontmatter: map[string]any{"verification": "corroborated"},
			Links: []extract.Link{
				{Target: "tool-a.md", Type: "supports", Style: extract.StyleWikilink},
			},
		},
		{
			Path:        "sources/witness-b.md",
			Frontmatter: map[string]any{"verification": "unverified"},
			Links: []extract.Link{
				{Target: "tool-b.md", Type: "supports", Style: extract.StyleWikilink},
			},
		},
	}
}

func recordsByPath(records []*extract.Record, paths ...string) []*extract.Record {
	byPath := make(map[string]*extract.Record, len(records))
	for _, r := range records {
		byPath[r.Path] = r
	}
	var out []*extract.Record
	for _, p := range paths {
		out = append(out, byPath[p])
	}
	return out
}

func pathsOf(records []*extract.Record) []string {
	var out []string
	for _, r := range records {
		out = append(out, r.Path)
	}
	return out
}

func TestFilterByTraversal_OutboundSubWhere(t *testing.T) {
	records := traversalFixture()
	g := graph.Build(context.Background(), records)
	candidates := recordsByPath(records, "sources/witness-a.md", "sources/witness-b.md")

	got, err := FilterByTraversal(context.Background(), candidates, g, TraversalOptions{
		HasOutbound: subWhere("kind == 'tool'"),
	})
	if err != nil {
		t.Fatalf("FilterByTraversal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v, want both witnesses (their targets are tools)", pathsOf(got))
	}

	got, err = FilterByTraversal(context.Background(), candidates, g, TraversalOptions{
		HasOutbound: subWhere("kind == 'concept'"),
	})
	if err != nil {
		t.Fatalf("FilterByTraversal: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want no matches for kind concept", pathsOf(got))
	}
}

func TestFilterByTraversal_TypeFilter(t *testing.T) {
	records := traversalFixture()
	g := graph.Build(context.Background(), records)
	candidates := recordsByPath(records, "entities/tool-a.md", "entities/tool-b.md", "entities/tool-c.md")

	got, err := FilterByTraversal(context.Background(), candidates, g, TraversalOptions{
		HasInbound:  subWhere(""),
		InboundType: "supports",
	})
	if err != nil {
		t.Fatalf("FilterByTraversal: %v", err)
	}
	if len(got) != 2 {
		t.Fatalf("got %v, want tool-a and tool-b (typed inbound)", pathsOf(got))
	}

	got, err = FilterByTraversal(context.Background(), candidates, g, TraversalOptions{
		HasInbound:  subWhere(""),
		InboundType: "refutes",
	})
	if err != nil {
		t.Fatalf("FilterByTraversal: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want no matches for type refutes", pathsOf(got))
	}
}

func TestFilterByTraversal_EmptySubWhereMeansExistence(t *testing.T) {
	records := traversalFixture()
	g := graph.Build(context.Background(), records)
	candidates := recordsByPath(records, "entities/tool-a.md", "entities/tool-c.md")

	got, err := FilterByTraversal(context.Background(), candidates, g, TraversalOptions{
		HasInbound: subWhere(""),
	})
	if err != nil {
		t.Fatalf("FilterByTraversal: %v", err)
	}
	if len(got) != 1 || got[0].Path != "entities/tool-a.md" {
		t.Errorf("got %v, want [entities/tool-a.md] (tool-c has no inbound)", pathsOf(got))
	}
}

func TestFilterByTraversal_BrokenLinkNeverMatches(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "sources/witness.md",
			Frontmatter: map[string]any{"verification": "corroborated"},
			Links: []extract.Link{
				{Target: "missing-tool.md", Type: "supports", Style: extract.StyleWikilink},
			},
		},
	}
	g := graph.Build(context.Background(), records)

	got, err := FilterByTraversal(context.Background(), records, g, TraversalOptions{
		HasOutbound: subWhere(""),
	})
	if err != nil {
		t.Fatalf("FilterByTraversal: %v", err)
	}
	if len(got) != 0 {
		t.Errorf("got %v, want empty (target does not exist)", pathsOf(got))
	}
}

func TestFilterByTraversal_SubWhereAgainstDerivedField(t *testing.T) {
	records := []*extract.Record{
		{Path: "entities/tool-a.md", Frontmatter: map[string]any{"kind": "tool"}},
		{
			Path:        "sources/witness.md",
			Frontmatter: map[string]any{"verification": "unverified"},
			Derived:     map[string]any{"effective_status": "corroborated"},
			Links: []extract.Link{
				{Target: "tool-a.md", Type: "supports", Style: extract.StyleWikilink},
			},
		},
	}
	g := graph.Build(context.Background(), records)
	candidates := recordsByPath(records, "entities/tool-a.md")

	got, err := FilterByTraversal(context.Background(), candidates, g, TraversalOptions{
		HasInbound: subWhere("effective_status == 'corroborated'"),
	})
	if err != nil {
		t.Fatalf("FilterByTraversal: %v", err)
	}
	if len(got) != 1 || got[0].Path != "entities/tool-a.md" {
		t.Errorf("got %v, want [entities/tool-a.md] (derived field on witness)", pathsOf(got))
	}
}

func TestFilterByTraversal_InvalidSubWhereErrors(t *testing.T) {
	records := traversalFixture()
	g := graph.Build(context.Background(), records)

	_, err := FilterByTraversal(context.Background(), records, g, TraversalOptions{
		HasInbound: subWhere("kind =="),
	})
	if err == nil {
		t.Fatal("expected compile error for malformed sub-where")
	}
}

func TestFilterByTraversal_InboundSubWhere(t *testing.T) {
	records := traversalFixture()
	g := graph.Build(context.Background(), records)
	candidates := recordsByPath(records, "entities/tool-a.md", "entities/tool-b.md", "entities/tool-c.md")

	got, err := FilterByTraversal(context.Background(), candidates, g, TraversalOptions{
		HasInbound: subWhere("verification == 'corroborated'"),
	})
	if err != nil {
		t.Fatalf("FilterByTraversal: %v", err)
	}
	if len(got) != 1 || got[0].Path != "entities/tool-a.md" {
		t.Errorf("got %v, want [entities/tool-a.md]", pathsOf(got))
	}
}
