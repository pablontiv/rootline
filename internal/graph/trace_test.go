package graph

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestTrace_ForwardChain(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "backlog/B039.md",
			Frontmatter: map[string]any{"estado": "remediado"},
			Links:       []extract.Link{{Target: "backup-design", Type: "reference", Source: "frontmatter:spec"}},
		},
		{
			Path:        "specs/backup-design.md",
			Frontmatter: map[string]any{"estado": "implementado"},
		},
		{
			Path:        "plans/backup-plan.md",
			Frontmatter: map[string]any{"estado": "implementado"},
			Links:       []extract.Link{{Target: "backup-design", Type: "reference", Source: "frontmatter:spec"}},
		},
	}
	g := Build(context.Background(), records)
	result := g.Trace("backlog/B039.md", TraceOptions{})

	if result.Start != "backlog/B039.md" {
		t.Errorf("expected start 'backlog/B039.md', got %q", result.Start)
	}
	if len(result.Nodes) < 1 {
		t.Fatalf("expected at least 1 node, got %d", len(result.Nodes))
	}
}

func TestTrace_Reverse(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "specs/my-design.md",
			Frontmatter: map[string]any{"estado": "implementado"},
		},
		{
			Path:        "plans/my-plan.md",
			Frontmatter: map[string]any{"estado": "implementado"},
			Links:       []extract.Link{{Target: "my-design", Type: "reference", Source: "frontmatter:spec"}},
		},
		{
			Path:        "backlog/B001.md",
			Frontmatter: map[string]any{"estado": "remediado"},
			Links:       []extract.Link{{Target: "my-design", Type: "reference", Source: "frontmatter:spec"}},
		},
	}
	g := Build(context.Background(), records)
	result := g.Trace("specs/my-design.md", TraceOptions{Reverse: true})

	if len(result.Nodes) < 2 {
		t.Fatalf("expected at least 2 nodes (plan + backlog), got %d", len(result.Nodes))
	}
}

func TestTrace_DepthLimit(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Links: []extract.Link{{Target: "b", Type: "reference", Source: "frontmatter:ref"}}},
		{Path: "b.md", Links: []extract.Link{{Target: "c", Type: "reference", Source: "frontmatter:ref"}}},
		{Path: "c.md", Links: []extract.Link{{Target: "d", Type: "reference", Source: "frontmatter:ref"}}},
		{Path: "d.md"},
	}
	g := Build(context.Background(), records)
	result := g.Trace("a.md", TraceOptions{MaxDepth: 1})

	// Should only reach b.md at depth 1, not c.md or d.md
	for _, n := range result.Nodes {
		if n.Depth > 1 {
			t.Errorf("node %q at depth %d exceeds limit 1", n.Path, n.Depth)
		}
	}
}

func TestTrace_NonexistentStart(t *testing.T) {
	g := &Graph{Nodes: map[string]*extract.Record{}, Edges: map[string][]Edge{}}
	result := g.Trace("nonexistent.md", TraceOptions{})
	if len(result.Nodes) != 0 {
		t.Errorf("expected 0 nodes for nonexistent start, got %d", len(result.Nodes))
	}
}

func TestTrace_CycleDetection(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Links: []extract.Link{{Target: "b", Type: "reference", Source: "frontmatter:ref"}}},
		{Path: "b.md", Links: []extract.Link{{Target: "a", Type: "reference", Source: "frontmatter:ref"}}},
	}
	g := Build(context.Background(), records)
	result := g.Trace("a.md", TraceOptions{})

	// Should not infinite loop — should visit each node once
	if len(result.Nodes) > 2 {
		t.Errorf("expected at most 2 nodes with cycle, got %d", len(result.Nodes))
	}
}
