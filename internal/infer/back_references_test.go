package infer

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
)

func buildGraph(records []*extract.Record) *graph.Graph {
	return graph.Build(context.Background(), records)
}

func TestDetectMissingBackRefsNone(t *testing.T) {
	// A↔B bidirectional — no missing back-references
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{}, Links: []extract.Link{
			{Target: "b.md", Type: "reference", Line: 1},
		}},
		{Path: "b.md", Frontmatter: map[string]any{}, Links: []extract.Link{
			{Target: "a.md", Type: "reference", Line: 1},
		}},
	}
	g := buildGraph(records)
	got := DetectMissingBackReferences(g)
	if len(got) != 0 {
		t.Errorf("expected no inferences for bidirectional links, got %d", len(got))
	}
}

func TestDetectMissingBackRefFound(t *testing.T) {
	// A→B but B does not reference A
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{}, Links: []extract.Link{
			{Target: "b.md", Type: "reference", Line: 1},
		}},
		{Path: "b.md", Frontmatter: map[string]any{}, Links: nil},
	}
	g := buildGraph(records)
	got := DetectMissingBackReferences(g)
	if len(got) != 1 {
		t.Fatalf("expected 1 missing back-reference, got %d", len(got))
	}
	if got[0].Type != "missing_back_reference" {
		t.Errorf("expected type missing_back_reference, got %s", got[0].Type)
	}
	if got[0].Source != "b.md" {
		t.Errorf("expected source b.md, got %s", got[0].Source)
	}
	if got[0].Value != "a.md" {
		t.Errorf("expected value a.md, got %s", got[0].Value)
	}
}

func TestDetectMissingBackRefSkipsBlocks(t *testing.T) {
	// A blocks B — unidirectional, should not report
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{}, Links: []extract.Link{
			{Target: "b.md", Type: "blocks", Line: 1},
		}},
		{Path: "b.md", Frontmatter: map[string]any{}, Links: nil},
	}
	g := buildGraph(records)
	got := DetectMissingBackReferences(g)
	if len(got) != 0 {
		t.Errorf("expected no inferences for unidirectional 'blocks' link, got %d", len(got))
	}
}

func TestDetectMissingBackRefSkipsBrokenLinks(t *testing.T) {
	// A→nonexistent — target not in graph, should not report
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{}, Links: []extract.Link{
			{Target: "nonexistent.md", Type: "reference", Line: 1},
		}},
	}
	g := buildGraph(records)
	got := DetectMissingBackReferences(g)
	if len(got) != 0 {
		t.Errorf("expected no inferences for broken links, got %d", len(got))
	}
}

func TestDetectMissingBackRefDedup(t *testing.T) {
	// A references B twice — should only report one missing back-reference
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{}, Links: []extract.Link{
			{Target: "b.md", Type: "reference", Line: 1},
			{Target: "b.md", Type: "relates", Line: 2},
		}},
		{Path: "b.md", Frontmatter: map[string]any{}, Links: nil},
	}
	g := buildGraph(records)
	got := DetectMissingBackReferences(g)
	if len(got) != 1 {
		t.Errorf("expected 1 deduped inference, got %d", len(got))
	}
}
