package graph

import (
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

	g := Build(records)
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

	g := Build(records)
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

	g := Build(records)
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

	g := Build(records)
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

	g := Build(records)
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

	g := Build(records)
	broken := g.BrokenLinks()
	if len(broken) != 0 {
		t.Errorf("got %d broken links, want 0: %v", len(broken), broken)
	}
}

func TestBuild_NoLinks(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", nil),
		makeRecord("b.md", nil),
	}

	g := Build(records)
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
