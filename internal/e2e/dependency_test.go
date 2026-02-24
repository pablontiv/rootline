package e2e

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/query"
)

// stemWithDeriveAndLinks returns a .stem YAML string that configures:
// - scope: *.md
// - link type "blocks" with field "blocked_by"
// - derive expression for estado based on blocked_by
//
// The derive expression logic:
//   - If no blockers (blocked_by == nil): keep original estado
//   - If all blockers are Completed: "Pending" (unblocked, ready to work)
//   - Otherwise: "Blocked" (blocked by incomplete dependencies)
const stemWithDeriveAndLinks = `version: 1
scope:
  match: "*.md"
schema:
  estado:
    type: string
links:
  blocks:
    target: "*.md"
    field: blocked_by
derive:
  estado: 'blocked_by == nil ? estado : (all(blocked_by, {# == "Completed"}) ? "Pending" : "Blocked")'
`

// TestDependencyChain_LinearChain tests a linear dependency chain:
// C (Completed, no deps) -> B blocks C (should derive Pending) -> A blocks B (should derive Blocked)
//
// Chain: A --blocks--> B --blocks--> C
// C has estado=Completed and no blockers.
// B has estado=Pending and blocks C (C is Completed, so B stays Pending).
// A has estado=Pending and blocks B (B frontmatter is Pending but *not* Completed, so A is Blocked).
//
// Note: The derive engine reads from Frontmatter["estado"] of targets, not
// from Derived["estado"]. This is a single-pass design — no transitive
// propagation of derived values.
func TestDependencyChain_LinearChain(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": stemWithDeriveAndLinks,

		// C: leaf node, already Completed, no blockers
		"tasks/C.md": "---\nestado: Completed\n---\n# Task C\n",

		// B: blocks C (C is Completed, so all blockers done -> Pending)
		"tasks/B.md": "---\nestado: Pending\n---\n# Task B\n\n[[blocks:C.md]]\n",

		// A: blocks B (B frontmatter is Pending, not Completed -> Blocked)
		"tasks/A.md": "---\nestado: Pending\n---\n# Task A\n\n[[blocks:B.md]]\n",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(context.Background(), root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(records) != 3 {
		t.Fatalf("got %d records, want 3", len(records))
	}

	// Run derivation on all records.
	derive.DeriveAll(context.Background(), records, root, derive.DefaultResolver())

	// Build a map for easy lookup.
	byPath := make(map[string]*extract.Record)
	for _, r := range records {
		byPath[r.Path] = r
	}

	// C: no blockers -> estado preserved as "Completed"
	recC := byPath["tasks/C.md"]
	if recC == nil {
		t.Fatal("missing tasks/C.md")
	}
	estadoC, ok := recC.EffectiveField("estado")
	if !ok {
		t.Fatal("C: missing estado")
	}
	if estadoC != "Completed" {
		t.Errorf("C: estado = %v, want Completed", estadoC)
	}

	// B: blocks C (Completed) -> all blockers done -> derived "Pending"
	recB := byPath["tasks/B.md"]
	if recB == nil {
		t.Fatal("missing tasks/B.md")
	}
	estadoB, ok := recB.EffectiveField("estado")
	if !ok {
		t.Fatal("B: missing estado")
	}
	if estadoB != "Pending" {
		t.Errorf("B: estado = %v, want Pending", estadoB)
	}

	// A: blocks B (B frontmatter is "Pending", not Completed) -> derived "Blocked"
	recA := byPath["tasks/A.md"]
	if recA == nil {
		t.Fatal("missing tasks/A.md")
	}
	estadoA, ok := recA.EffectiveField("estado")
	if !ok {
		t.Fatal("A: missing estado")
	}
	if estadoA != "Blocked" {
		t.Errorf("A: estado = %v, want Blocked", estadoA)
	}
}

// TestDependencyChain_MultipleBlockers tests a record blocked by multiple dependencies.
// A blocks B and C. Both B and C must be Completed for A to be unblocked.
func TestDependencyChain_MultipleBlockers(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": stemWithDeriveAndLinks,

		"tasks/B.md": "---\nestado: Completed\n---\n# Task B\n",
		"tasks/C.md": "---\nestado: Pending\n---\n# Task C\n",

		// A blocks both B and C
		"tasks/A.md": "---\nestado: Pending\n---\n# Task A\n\n[[blocks:B.md]]\n[[blocks:C.md]]\n",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(context.Background(), root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	derive.DeriveAll(context.Background(), records, root, derive.DefaultResolver())

	byPath := make(map[string]*extract.Record)
	for _, r := range records {
		byPath[r.Path] = r
	}

	// A: B is Completed but C is Pending -> not all done -> "Blocked"
	recA := byPath["tasks/A.md"]
	if recA == nil {
		t.Fatal("missing tasks/A.md")
	}
	estadoA, ok := recA.EffectiveField("estado")
	if !ok {
		t.Fatal("A: missing estado")
	}
	if estadoA != "Blocked" {
		t.Errorf("A: estado = %v, want Blocked (C is still Pending)", estadoA)
	}
}

// TestDependencyChain_MultipleBlockersAllDone tests that when all blockers
// are Completed, the record is derived as Pending (unblocked).
func TestDependencyChain_MultipleBlockersAllDone(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": stemWithDeriveAndLinks,

		"tasks/B.md": "---\nestado: Completed\n---\n# Task B\n",
		"tasks/C.md": "---\nestado: Completed\n---\n# Task C\n",

		// A blocks both B and C — both Completed
		"tasks/A.md": "---\nestado: Pending\n---\n# Task A\n\n[[blocks:B.md]]\n[[blocks:C.md]]\n",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(context.Background(), root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	derive.DeriveAll(context.Background(), records, root, derive.DefaultResolver())

	byPath := make(map[string]*extract.Record)
	for _, r := range records {
		byPath[r.Path] = r
	}

	// A: both B and C are Completed -> all blockers done -> "Pending"
	recA := byPath["tasks/A.md"]
	if recA == nil {
		t.Fatal("missing tasks/A.md")
	}
	estadoA, ok := recA.EffectiveField("estado")
	if !ok {
		t.Fatal("A: missing estado")
	}
	if estadoA != "Pending" {
		t.Errorf("A: estado = %v, want Pending (all blockers Completed)", estadoA)
	}
}

// TestDependencyChain_Cycle tests that a circular dependency (A blocks B,
// B blocks A) does not cause an infinite loop. The derive engine is single-pass
// and reads from Frontmatter, not Derived, so cycles resolve naturally without
// hanging. Both records should derive "Blocked" because neither target has
// estado=Completed in frontmatter.
func TestDependencyChain_Cycle(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": stemWithDeriveAndLinks,

		// A blocks B, B blocks A -> cycle
		"tasks/A.md": "---\nestado: Pending\n---\n# Task A\n\n[[blocks:B.md]]\n",
		"tasks/B.md": "---\nestado: Pending\n---\n# Task B\n\n[[blocks:A.md]]\n",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(context.Background(), root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// This must complete without hanging (no infinite loop).
	derive.DeriveAll(context.Background(), records, root, derive.DefaultResolver())

	byPath := make(map[string]*extract.Record)
	for _, r := range records {
		byPath[r.Path] = r
	}

	// Both A and B reference each other. Since neither has estado=Completed
	// in frontmatter, both should derive "Blocked".
	recA := byPath["tasks/A.md"]
	if recA == nil {
		t.Fatal("missing tasks/A.md")
	}
	estadoA, ok := recA.EffectiveField("estado")
	if !ok {
		t.Fatal("A: missing estado")
	}
	if estadoA != "Blocked" {
		t.Errorf("A: estado = %v, want Blocked (cycle: B is Pending)", estadoA)
	}

	recB := byPath["tasks/B.md"]
	if recB == nil {
		t.Fatal("missing tasks/B.md")
	}
	estadoB, ok := recB.EffectiveField("estado")
	if !ok {
		t.Fatal("B: missing estado")
	}
	if estadoB != "Blocked" {
		t.Errorf("B: estado = %v, want Blocked (cycle: A is Pending)", estadoB)
	}
}

// TestDependencyChain_MissingTarget tests that a link to a non-existent
// target is silently skipped — the record's estado is preserved as-is
// (blocked_by will be nil since no target resolves).
func TestDependencyChain_MissingTarget(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": stemWithDeriveAndLinks,

		// A blocks NonExistent.md — target does not exist
		"tasks/A.md": "---\nestado: Pending\n---\n# Task A\n\n[[blocks:NonExistent.md]]\n",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(context.Background(), root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	derive.DeriveAll(context.Background(), records, root, derive.DefaultResolver())

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	rec := records[0]
	estado, ok := rec.EffectiveField("estado")
	if !ok {
		t.Fatal("missing estado")
	}
	// blocked_by is nil (target not found) -> derive preserves original estado
	if estado != "Pending" {
		t.Errorf("estado = %v, want Pending (non-existent target skipped)", estado)
	}
}

// TestDependencyChain_NoLinks tests that a document without any blocks links
// preserves its original estado. The derive expression should evaluate
// blocked_by == nil as true and return the original estado.
func TestDependencyChain_NoLinks(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": stemWithDeriveAndLinks,

		// A has no links at all
		"tasks/A.md": "---\nestado: Completed\n---\n# Task A (no deps)\n",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(context.Background(), root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	derive.DeriveAll(context.Background(), records, root, derive.DefaultResolver())

	if len(records) != 1 {
		t.Fatalf("got %d records, want 1", len(records))
	}

	rec := records[0]
	estado, ok := rec.EffectiveField("estado")
	if !ok {
		t.Fatal("missing estado")
	}
	if estado != "Completed" {
		t.Errorf("estado = %v, want Completed (no links, preserved)", estado)
	}
}

// TestDependencyChain_QueryAfterDerive tests the full extract -> derive -> query
// pipeline. The query engine operates on frontmatter fields, so we verify
// that frontmatter-based queries still work alongside derived values, and
// that EffectiveField correctly returns derived estado for each record.
func TestDependencyChain_QueryAfterDerive(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": stemWithDeriveAndLinks,

		"tasks/C.md": "---\nestado: Completed\n---\n# Task C\n",
		"tasks/B.md": "---\nestado: Pending\n---\n# Task B\n\n[[blocks:C.md]]\n",
		"tasks/A.md": "---\nestado: Pending\n---\n# Task A\n\n[[blocks:B.md]]\n",
	})

	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(context.Background(), root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	derive.DeriveAll(context.Background(), records, root, derive.DefaultResolver())

	// Query on frontmatter estado (not derived) — both A and B have frontmatter Pending.
	result, err := query.Execute(records, &query.Query{
		Where: &query.Condition{Op: query.OpEq, Field: "estado", Value: "Pending"},
	})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	qr := result.(*query.QueryResult)
	if qr.Meta.Count != 2 {
		t.Errorf("frontmatter Pending count = %d, want 2 (A and B)", qr.Meta.Count)
	}

	// Query on frontmatter Completed — only C.
	result2, err := query.Execute(records, &query.Query{
		Where: &query.Condition{Op: query.OpEq, Field: "estado", Value: "Completed"},
	})
	if err != nil {
		t.Fatalf("query error: %v", err)
	}
	qr2 := result2.(*query.QueryResult)
	if qr2.Meta.Count != 1 {
		t.Errorf("frontmatter Completed count = %d, want 1 (C)", qr2.Meta.Count)
	}

	// Verify EffectiveField returns derived estado (takes precedence over frontmatter).
	byPath := make(map[string]*extract.Record)
	for _, r := range records {
		byPath[r.Path] = r
	}

	tests := []struct {
		path    string
		want    string
		comment string
	}{
		{"tasks/A.md", "Blocked", "A blocked by B (Pending)"},
		{"tasks/B.md", "Pending", "B unblocked (C is Completed)"},
		{"tasks/C.md", "Completed", "C has no deps, preserved"},
	}
	for _, tt := range tests {
		rec := byPath[tt.path]
		if rec == nil {
			t.Fatalf("missing %s", tt.path)
		}
		got, ok := rec.EffectiveField("estado")
		if !ok {
			t.Fatalf("%s: missing effective estado", tt.path)
		}
		if got != tt.want {
			t.Errorf("%s: effective estado = %v, want %s (%s)", tt.path, got, tt.want, tt.comment)
		}
	}
}
