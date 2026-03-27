package graph

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func makeRecord(path string, links []extract.Link) *extract.Record {
	return &extract.Record{
		Path:        path,
		Type:        "markdown",
		Frontmatter: map[string]any{},
		Links:       links,
	}
}

func TestBuild_ThreeRecordsWithLinks(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "b.md", Type: "reference", Line: 1}}),
		makeRecord("b.md", []extract.Link{{Target: "c.md", Type: "blocks", Line: 1}}),
		makeRecord("c.md", nil),
	}

	g := Build(context.Background(), records)
	if len(g.Nodes) != 3 {
		t.Errorf("nodes = %d, want 3", len(g.Nodes))
	}
	if len(g.Edges["a.md"]) != 1 || g.Edges["a.md"][0].Target != "b.md" {
		t.Errorf("edges[a.md] = %+v", g.Edges["a.md"])
	}
}

func TestDetectCycles_NoCycles(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "b.md", Type: "reference", Line: 1}}),
		makeRecord("b.md", []extract.Link{{Target: "c.md", Type: "reference", Line: 1}}),
		makeRecord("c.md", nil),
	}

	g := Build(context.Background(), records)
	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("got %d cycles, want 0: %v", len(cycles), cycles)
	}
}

func TestDetectCycles_ABC_Cycle(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "b.md", Type: "reference", Line: 1}}),
		makeRecord("b.md", []extract.Link{{Target: "c.md", Type: "reference", Line: 1}}),
		makeRecord("c.md", []extract.Link{{Target: "a.md", Type: "reference", Line: 1}}),
	}

	g := Build(context.Background(), records)
	cycles := g.DetectCycles()
	if len(cycles) != 1 {
		t.Fatalf("got %d cycles, want 1: %v", len(cycles), cycles)
	}
	cycle := cycles[0]
	if len(cycle) < 3 {
		t.Errorf("cycle too short: %v", cycle)
	}
	// Cycle must close: first == last.
	if cycle[0] != cycle[len(cycle)-1] {
		t.Errorf("cycle should close (first == last): %v", cycle)
	}
	// All three nodes should appear.
	nodes := map[string]bool{}
	for _, n := range cycle[:len(cycle)-1] {
		nodes[n] = true
	}
	if !nodes["a.md"] || !nodes["b.md"] || !nodes["c.md"] {
		t.Errorf("cycle should contain a.md, b.md, c.md: %v", cycle)
	}
}

func TestDetectCycles_SelfReference(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "a.md", Type: "reference", Line: 1}}),
	}

	g := Build(context.Background(), records)
	cycles := g.DetectCycles()
	if len(cycles) != 1 {
		t.Fatalf("got %d cycles, want 1: %v", len(cycles), cycles)
	}
	if cycles[0][0] != "a.md" || cycles[0][1] != "a.md" {
		t.Errorf("expected self-cycle [a.md, a.md], got %v", cycles[0])
	}
}

func TestBrokenLinks_NonexistentTarget(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "nonexistent.md", Type: "reference", Line: 5}}),
	}

	g := Build(context.Background(), records)
	broken := g.BrokenLinks()
	if len(broken) != 1 {
		t.Fatalf("got %d broken links, want 1", len(broken))
	}
	if broken[0].Target != "nonexistent.md" || broken[0].Source != "a.md" || broken[0].Line != 5 {
		t.Errorf("broken link = %+v", broken[0])
	}
}

func TestBrokenLinks_AllValid(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "b.md", Type: "reference", Line: 1}}),
		makeRecord("b.md", nil),
	}

	g := Build(context.Background(), records)
	broken := g.BrokenLinks()
	if len(broken) != 0 {
		t.Errorf("got %d broken links, want 0: %v", len(broken), broken)
	}
}

func TestResolveTarget_BasenameFallback(t *testing.T) {
	records := []*extract.Record{
		makeRecord("subdir/T001-task.md", nil),
		makeRecord("other/source.md", []extract.Link{
			{Target: "T001-task", Type: "blocks", Line: 3},
		}),
	}

	g := Build(context.Background(), records)
	broken := g.BrokenLinks()
	if len(broken) != 0 {
		t.Errorf("got %d broken links, want 0: %v", len(broken), broken)
	}
	edges := g.Edges["other/source.md"]
	if len(edges) != 1 || edges[0].Target != "subdir/T001-task.md" {
		t.Errorf("edge target = %q, want %q", edges[0].Target, "subdir/T001-task.md")
	}
}

func TestResolveTarget_BasenameFallback_WithExtension(t *testing.T) {
	records := []*extract.Record{
		makeRecord("subdir/T001-task.md", nil),
		makeRecord("other/source.md", []extract.Link{
			{Target: "T001-task.md", Type: "blocks", Line: 3},
		}),
	}

	g := Build(context.Background(), records)
	broken := g.BrokenLinks()
	if len(broken) != 0 {
		t.Errorf("got %d broken links, want 0: %v", len(broken), broken)
	}
	edges := g.Edges["other/source.md"]
	if len(edges) != 1 || edges[0].Target != "subdir/T001-task.md" {
		t.Errorf("edge target = %q, want %q", edges[0].Target, "subdir/T001-task.md")
	}
}

func TestResolveTarget_BasenameFallback_Ambiguous(t *testing.T) {
	records := []*extract.Record{
		makeRecord("dir1/T001-task.md", nil),
		makeRecord("dir2/T001-task.md", nil),
		makeRecord("source.md", []extract.Link{
			{Target: "T001-task", Type: "blocks", Line: 1},
		}),
	}

	g := Build(context.Background(), records)
	broken := g.BrokenLinks()
	if len(broken) != 1 {
		t.Errorf("got %d broken links, want 1 (ambiguous): %v", len(broken), broken)
	}
}

func TestResolveTarget_BasenameFallback_NoMatch(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", nil),
		makeRecord("source.md", []extract.Link{
			{Target: "nonexistent-task", Type: "blocks", Line: 1},
		}),
	}

	g := Build(context.Background(), records)
	broken := g.BrokenLinks()
	if len(broken) != 1 {
		t.Errorf("got %d broken links, want 1: %v", len(broken), broken)
	}
}

func TestDetectCycles_FourNodeCycle(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "b.md", Type: "reference", Line: 1}}),
		makeRecord("b.md", []extract.Link{{Target: "c.md", Type: "reference", Line: 1}}),
		makeRecord("c.md", []extract.Link{{Target: "d.md", Type: "reference", Line: 1}}),
		makeRecord("d.md", []extract.Link{{Target: "a.md", Type: "reference", Line: 1}}),
	}

	g := Build(context.Background(), records)
	cycles := g.DetectCycles()
	if len(cycles) != 1 {
		t.Fatalf("got %d cycles, want 1: %v", len(cycles), cycles)
	}
	cycle := cycles[0]
	// Cycle must close and contain 4 unique nodes.
	if cycle[0] != cycle[len(cycle)-1] {
		t.Errorf("cycle should close: %v", cycle)
	}
	nodes := map[string]bool{}
	for _, n := range cycle[:len(cycle)-1] {
		nodes[n] = true
	}
	if len(nodes) != 4 {
		t.Errorf("expected 4 unique nodes in cycle, got %d: %v", len(nodes), cycle)
	}
}

func TestDetectCycles_MultipleDisjoint(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "b.md", Type: "reference", Line: 1}}),
		makeRecord("b.md", []extract.Link{{Target: "a.md", Type: "reference", Line: 1}}),
		makeRecord("c.md", []extract.Link{{Target: "d.md", Type: "reference", Line: 1}}),
		makeRecord("d.md", []extract.Link{{Target: "c.md", Type: "reference", Line: 1}}),
	}

	g := Build(context.Background(), records)
	cycles := g.DetectCycles()
	if len(cycles) != 2 {
		t.Fatalf("got %d cycles, want 2: %v", len(cycles), cycles)
	}
}

func TestBrokenLinks_MultipleFromSameSource(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{
			{Target: "x.md", Type: "reference", Line: 1},
			{Target: "y.md", Type: "reference", Line: 2},
			{Target: "z.md", Type: "reference", Line: 3},
		}),
		makeRecord("z.md", nil), // z exists, x and y don't
	}

	g := Build(context.Background(), records)
	broken := g.BrokenLinks()
	if len(broken) != 2 {
		t.Fatalf("got %d broken links, want 2: %v", len(broken), broken)
	}
	for _, b := range broken {
		if b.Source != "a.md" {
			t.Errorf("broken link source = %q, want a.md", b.Source)
		}
	}
}

func TestBuild_EmptyGraph(t *testing.T) {
	g := Build(context.Background(), []*extract.Record{})
	if len(g.Nodes) != 0 {
		t.Errorf("nodes = %d, want 0", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Errorf("edges = %d, want 0", len(g.Edges))
	}
	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}
	broken := g.BrokenLinks()
	if len(broken) != 0 {
		t.Errorf("broken links = %d, want 0", len(broken))
	}
}

func TestBuild_FrontmatterLinks(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "plans/my-plan.md",
			Frontmatter: map[string]any{"tipo": "plan"},
			Links: []extract.Link{
				{Target: "my-design", Type: "reference", Source: "frontmatter:spec"},
			},
		},
		{
			Path:        "specs/my-design.md",
			Frontmatter: map[string]any{"tipo": "spec"},
			Links: []extract.Link{
				{Target: "B039-velero", Type: "reference", Source: "frontmatter:backlog_ids"},
			},
		},
		{
			Path:        "B039-velero.md",
			Frontmatter: map[string]any{"tipo": "deferred-item"},
		},
	}
	g := Build(context.Background(), records)

	// plans/my-plan.md should have edge to specs/my-design.md (via basename fallback)
	edges := g.Edges["plans/my-plan.md"]
	if len(edges) != 1 {
		t.Fatalf("expected 1 edge from plan, got %d", len(edges))
	}
	if edges[0].Target != "specs/my-design.md" {
		t.Errorf("expected target 'specs/my-design.md', got %q", edges[0].Target)
	}

	// specs/my-design.md should have edge to B039-velero.md
	edges2 := g.Edges["specs/my-design.md"]
	if len(edges2) != 1 {
		t.Fatalf("expected 1 edge from spec, got %d", len(edges2))
	}

	// No broken links
	broken := g.BrokenLinks()
	if len(broken) != 0 {
		t.Errorf("expected 0 broken links, got %d: %+v", len(broken), broken)
	}
}

func TestBuild_NoLinks(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", nil),
		makeRecord("b.md", nil),
	}

	g := Build(context.Background(), records)
	if len(g.Nodes) != 2 {
		t.Errorf("nodes = %d, want 2", len(g.Nodes))
	}
	if len(g.Edges) != 0 {
		t.Errorf("edges = %d, want 0", len(g.Edges))
	}
	cycles := g.DetectCycles()
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}
}
