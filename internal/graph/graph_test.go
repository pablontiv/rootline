package graph

import (
	"context"
	"path/filepath"
	"slices"
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

func TestBrokenLinks_TotalOrder(t *testing.T) {
	records := []*extract.Record{
		makeRecord("z-source.md", []extract.Link{
			{Target: "missing.md", Type: "reference", Line: 8},
		}),
		makeRecord("a-source.md", []extract.Link{
			{Target: "z-missing.md", Type: "reference", Line: 7},
			{Target: "a-missing.md", Type: "reference", Line: 9},
			{Target: "a-missing.md", Type: "blocks", Line: 3},
		}),
	}

	got := Build(context.Background(), records).BrokenLinks()
	want := []BrokenLink{
		{Source: "a-source.md", Target: "a-missing.md", Type: "blocks", Line: 3},
		{Source: "a-source.md", Target: "a-missing.md", Type: "reference", Line: 9},
		{Source: "a-source.md", Target: "z-missing.md", Type: "reference", Line: 7},
		{Source: "z-source.md", Target: "missing.md", Type: "reference", Line: 8},
	}
	if !slices.EqualFunc(got, want, func(a, b BrokenLink) bool {
		return a.Source == b.Source && a.Target == b.Target && a.Type == b.Type && a.Line == b.Line
	}) {
		t.Fatalf("BrokenLinks() = %+v, want %+v", got, want)
	}
}

func TestBrokenLinks_SuggestionTiesUseLexicalOrder(t *testing.T) {
	records := []*extract.Record{
		makeRecord("rat.md", nil),
		makeRecord("mat.md", nil),
		makeRecord("hat.md", nil),
		makeRecord("bat.md", nil),
		makeRecord("source.md", []extract.Link{
			{Target: "cat.md", Type: "reference", Line: 3},
		}),
	}
	g := Build(context.Background(), records)
	want := []string{"bat.md", "hat.md", "mat.md"}

	for run := 1; run <= 100; run++ {
		broken := g.BrokenLinks()
		if len(broken) != 1 {
			t.Fatalf("run %d: BrokenLinks() len = %d, want 1: %+v", run, len(broken), broken)
		}
		if !slices.Equal(broken[0].Suggestions, want) {
			t.Fatalf("run %d: suggestions = %v, want %v", run, broken[0].Suggestions, want)
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

func TestBuild_MarkdownTargetsUsedAsIs(t *testing.T) {
	records := []*extract.Record{
		{Path: filepath.Join("docs", "a.md"), Links: []extract.Link{
			{Target: filepath.Join("docs", "b.md"), Style: extract.StyleMarkdown, Line: 3},
		}},
		{Path: filepath.Join("docs", "b.md")},
	}
	g := Build(context.Background(), records)
	edges := g.Edges[filepath.Join("docs", "a.md")]
	if len(edges) != 1 || edges[0].Target != filepath.Join("docs", "b.md") {
		t.Fatalf("edges = %+v, want one edge to docs/b.md", edges)
	}
	if broken := g.BrokenLinks(); len(broken) != 0 {
		t.Errorf("broken = %+v, want none", broken)
	}
}

func TestBuild_MarkdownSkipsBasenameFallback(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Links: []extract.Link{
			{Target: "missing.md", Style: extract.StyleMarkdown, Line: 1},
		}},
		{Path: filepath.Join("other", "missing.md")},
	}
	g := Build(context.Background(), records)
	broken := g.BrokenLinks()
	if len(broken) != 1 || broken[0].Target != "missing.md" {
		t.Fatalf("broken = %+v, want unresolved markdown target reported verbatim", broken)
	}
}

func TestBuild_WikilinkBasenameFallbackUnchanged(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Links: []extract.Link{
			{Target: "missing.md", Style: extract.StyleWikilink, Line: 1},
		}},
		{Path: filepath.Join("other", "missing.md")},
	}
	g := Build(context.Background(), records)
	if broken := g.BrokenLinks(); len(broken) != 0 {
		t.Fatalf("broken = %+v, want basename fallback to resolve wikilink", broken)
	}
}

func TestSortedNodes_LexicalOrder(t *testing.T) {
	records := []*extract.Record{
		makeRecord("r2.md", nil),
		makeRecord("r1.md", nil),
		makeRecord("r6.md", nil),
		makeRecord("r3.md", nil),
	}
	g := Build(context.Background(), records)

	want := []string{"r1.md", "r2.md", "r3.md", "r6.md"}
	got := g.SortedNodes()
	if len(got) != len(want) {
		t.Fatalf("SortedNodes() = %v, want %v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SortedNodes() = %v, want %v", got, want)
		}
	}
}

func TestSortedNodes_EmptyGraph(t *testing.T) {
	g := Build(context.Background(), nil)
	got := g.SortedNodes()
	if got == nil {
		t.Fatal("SortedNodes() = nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("SortedNodes() = %v, want empty", got)
	}
}

func TestSortedEdges_TotalOrder(t *testing.T) {
	// source.md links twice to target.md (lines 20 and 5) and once to other.md (line 15).
	// zsource.md sorts after source.md and proves cross-source ordering.
	records := []*extract.Record{
		makeRecord("zsource.md", []extract.Link{
			{Target: "target.md", Type: "reference", Line: 1},
		}),
		makeRecord("source.md", []extract.Link{
			{Target: "target.md", Type: "reference", Line: 20},
			{Target: "other.md", Type: "reference", Line: 15},
			{Target: "target.md", Type: "reference", Line: 5},
		}),
		makeRecord("target.md", nil),
		makeRecord("other.md", nil),
	}
	g := Build(context.Background(), records)

	want := []Edge{
		{Source: "source.md", Target: "other.md", Type: "reference", Line: 15},
		{Source: "source.md", Target: "target.md", Type: "reference", Line: 5},
		{Source: "source.md", Target: "target.md", Type: "reference", Line: 20},
		{Source: "zsource.md", Target: "target.md", Type: "reference", Line: 1},
	}
	got := g.SortedEdges()
	if len(got) != len(want) {
		t.Fatalf("SortedEdges() len = %d, want %d: %+v", len(got), len(want), got)
	}
	for i := range want {
		if got[i].Source != want[i].Source || got[i].Target != want[i].Target ||
			got[i].Line != want[i].Line || got[i].Type != want[i].Type {
			t.Fatalf("SortedEdges()[%d] = %+v, want %+v (full: %+v)", i, got[i], want[i], got)
		}
	}
}

func TestSortedEdges_TypeTiebreak(t *testing.T) {
	// Same source, target and line: link type breaks the tie lexically.
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{
			{Target: "b.md", Type: "reference", Line: 3},
			{Target: "b.md", Type: "blocks", Line: 3},
		}),
		makeRecord("b.md", nil),
	}
	g := Build(context.Background(), records)

	got := g.SortedEdges()
	if len(got) != 2 {
		t.Fatalf("SortedEdges() len = %d, want 2", len(got))
	}
	if got[0].Type != "blocks" || got[1].Type != "reference" {
		t.Fatalf("SortedEdges() types = [%q %q], want [blocks reference]", got[0].Type, got[1].Type)
	}
}

func TestSortedEdges_EmptyGraph(t *testing.T) {
	g := Build(context.Background(), []*extract.Record{makeRecord("a.md", nil)})
	got := g.SortedEdges()
	if got == nil {
		t.Fatal("SortedEdges() = nil, want empty non-nil slice")
	}
	if len(got) != 0 {
		t.Fatalf("SortedEdges() = %+v, want empty", got)
	}
}

// A prepared link is authoritative. rules.PrepareLinks already applied the
// basename fallback against the same record set, so Build must not re-run it
// and must trust the recorded outcome: an unresolved target is broken even if
// a same-named node exists, and a resolved one is not broken even though it
// names no node. That second case is the crux of issue #62 — an existing but
// ungoverned file resolved fine, and calling it broken is what made graph and
// validate disagree, because the schema declares what is governed, not what
// exists.
func TestBuild_PreparedResolutionIsAuthoritative(t *testing.T) {
	records := []*extract.Record{
		makeRecord("dir/missing.md", nil),
		makeRecord("a.md", []extract.Link{
			{Target: "missing.md", Style: extract.StyleWikilink, Line: 1, Resolution: extract.LinkUnresolved},
			{Target: "ungoverned.md", Style: extract.StyleMarkdown, Line: 2, Resolution: extract.LinkResolved},
		}),
	}
	g := Build(context.Background(), records)
	broken := g.BrokenLinks()
	if len(broken) != 1 || broken[0].Target != "missing.md" {
		t.Fatalf("broken = %+v, want only the unresolved target, left verbatim", broken)
	}
}

// --- Deterministic cycle enumeration (issue #114) ---

// twoDisjointCycles is the fixture from issue #114: a↔b and c↔d, two cycles
// that share no node, so the outer ordering is decided purely by which node the
// DFS happens to root at first.
func twoDisjointCycles() []*extract.Record {
	return []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "b.md", Type: "reference", Line: 1}}),
		makeRecord("b.md", []extract.Link{{Target: "a.md", Type: "reference", Line: 1}}),
		makeRecord("c.md", []extract.Link{{Target: "d.md", Type: "reference", Line: 1}}),
		makeRecord("d.md", []extract.Link{{Target: "c.md", Type: "reference", Line: 1}}),
	}
}

// TestDetectCycles_RepeatedRunsAreIdentical is the regression test for issue
// #114. Go randomizes map iteration per range statement, so a handful of runs
// can agree by luck; the issue's own repro needed ~100 CLI invocations to show
// all four variants. 200 in-process runs make a false green vanishingly
// unlikely while staying fast.
func TestDetectCycles_RepeatedRunsAreIdentical(t *testing.T) {
	var first [][]string
	for run := 1; run <= 200; run++ {
		got := Build(context.Background(), twoDisjointCycles()).DetectCycles()
		if run == 1 {
			first = got
			continue
		}
		if !cyclesEqual(got, first) {
			t.Fatalf("run %d differs from run 1:\n run 1: %v\n run %d: %v", run, first, run, got)
		}
	}
}

// TestDetectCycles_IndependentOfRecordOrder pins the stronger half of the
// contract: the answer is a function of the graph, not of the order records
// were scanned. Sorted DFS roots alone would not give this — the adjacency
// lists have to be canonical too.
func TestDetectCycles_IndependentOfRecordOrder(t *testing.T) {
	forward := twoDisjointCycles()
	reversed := twoDisjointCycles()
	for i, j := 0, len(reversed)-1; i < j; i, j = i+1, j-1 {
		reversed[i], reversed[j] = reversed[j], reversed[i]
	}

	a := Build(context.Background(), forward).DetectCycles()
	b := Build(context.Background(), reversed).DetectCycles()
	if !cyclesEqual(a, b) {
		t.Fatalf("record order changed the result:\n forward: %v\n reversed: %v", a, b)
	}
}

// TestDetectCycles_CanonicalOuterOrder asserts the exact documented answer for
// the issue fixture, not merely that two runs agree.
func TestDetectCycles_CanonicalOuterOrder(t *testing.T) {
	want := [][]string{
		{"a.md", "b.md", "a.md"},
		{"c.md", "d.md", "c.md"},
	}
	for run := 1; run <= 50; run++ {
		got := Build(context.Background(), twoDisjointCycles()).DetectCycles()
		if !cyclesEqual(got, want) {
			t.Fatalf("run %d: DetectCycles() = %v, want %v", run, got, want)
		}
	}
}

// TestDetectCycles_CanonicalRotation covers the rotation half of the defect.
// Every node of a 3-cycle is a legal DFS entry point, so without canonical
// rotation the same cycle prints three different ways. The documented answer
// starts at the lexicographically smallest member.
func TestDetectCycles_CanonicalRotation(t *testing.T) {
	records := []*extract.Record{
		makeRecord("b.md", []extract.Link{{Target: "c.md", Type: "reference", Line: 1}}),
		makeRecord("c.md", []extract.Link{{Target: "a.md", Type: "reference", Line: 1}}),
		makeRecord("a.md", []extract.Link{{Target: "b.md", Type: "reference", Line: 1}}),
	}
	want := [][]string{{"a.md", "b.md", "c.md", "a.md"}}
	for run := 1; run <= 50; run++ {
		got := Build(context.Background(), records).DetectCycles()
		if !cyclesEqual(got, want) {
			t.Fatalf("run %d: DetectCycles() = %v, want %v", run, got, want)
		}
	}
}

// TestDetectCycles_DuplicateParallelLinksReportOneCycle: b.md links to a.md
// twice, so the same back edge fires twice and the identical path is emitted
// twice. Two byte-identical entries carry no information a consumer can act on.
func TestDetectCycles_DuplicateParallelLinksReportOneCycle(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "b.md", Type: "reference", Line: 1}}),
		makeRecord("b.md", []extract.Link{
			{Target: "a.md", Type: "reference", Line: 1},
			{Target: "a.md", Type: "reference", Line: 2},
		}),
	}
	want := [][]string{{"a.md", "b.md", "a.md"}}
	got := Build(context.Background(), records).DetectCycles()
	if !cyclesEqual(got, want) {
		t.Fatalf("DetectCycles() = %v, want %v", got, want)
	}
}

// TestDetectCycles_OverlappingCyclesAreStable exercises the case the issue
// flagged as unproven at scale: cycles sharing nodes, where the reported set is
// a function of the DFS forest. Pinning the forest must pin the set.
func TestDetectCycles_OverlappingCyclesAreStable(t *testing.T) {
	records := []*extract.Record{
		makeRecord("a.md", []extract.Link{{Target: "b.md", Type: "reference", Line: 1}}),
		makeRecord("b.md", []extract.Link{
			{Target: "c.md", Type: "reference", Line: 1},
			{Target: "a.md", Type: "reference", Line: 2},
		}),
		makeRecord("c.md", []extract.Link{{Target: "a.md", Type: "reference", Line: 1}}),
	}

	var first [][]string
	for run := 1; run <= 200; run++ {
		got := Build(context.Background(), records).DetectCycles()
		if run == 1 {
			first = got
			continue
		}
		if !cyclesEqual(got, first) {
			t.Fatalf("run %d differs from run 1:\n run 1: %v\n run %d: %v", run, first, run, got)
		}
	}
	if len(first) != 2 {
		t.Fatalf("got %d cycles, want 2: %v", len(first), first)
	}
}

// cyclesEqual compares two cycle lists element by element.
func cyclesEqual(a, b [][]string) bool {
	return slices.EqualFunc(a, b, slices.Equal[[]string])
}
