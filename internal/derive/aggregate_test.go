package derive

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestAggregateAll_BasicEstado(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending", "tipo": "feature"}},
		{Path: "dir/T001-task.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed", "tipo": "software-module"}},
		{Path: "dir/T002-task.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed", "tipo": "software-module"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : estado`,
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)
	AggregateAll(context.Background(), records, "/root", resolver)

	if records[0].Derived["estado"] != "Completed" {
		t.Errorf("README estado = %v, want Completed", records[0].Derived["estado"])
	}
	// Non-index records should NOT have derived estado from aggregation.
	if _, hasEstado := records[1].Derived["estado"]; hasEstado {
		t.Errorf("task[0].Derived[estado] = %v, want absent", records[1].Derived["estado"])
	}
}

func TestAggregateAll_NotAllCompleted(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending", "tipo": "feature"}},
		{Path: "dir/T001-task.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed", "tipo": "software-module"}},
		{Path: "dir/T002-task.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Blocked", "tipo": "software-module"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : any(descendants, {.estado == "Blocked"}) ? "Blocked" : estado`,
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)
	AggregateAll(context.Background(), records, "/root", resolver)

	if records[0].Derived["estado"] != "Blocked" {
		t.Errorf("README estado = %v, want Blocked", records[0].Derived["estado"])
	}
}

func TestAggregateAll_MultiLevel(t *testing.T) {
	// Story README aggregates tasks, then Feature README aggregates from children.
	records := []*extract.Record{
		// Feature level
		{Path: "F01/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending", "tipo": "feature"}},
		// Story level
		{Path: "F01/S001/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Specified", "tipo": "historia"}},
		// Tasks
		{Path: "F01/S001/T001.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed", "tipo": "software-module"}},
		{Path: "F01/S001/T002.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed", "tipo": "software-module"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : estado`,
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)
	AggregateAll(context.Background(), records, "/root", resolver)

	// Story README: descendants = [T001, T002] → all Completed
	if records[1].Derived["estado"] != "Completed" {
		t.Errorf("Story README estado = %v, want Completed", records[1].Derived["estado"])
	}

	// Feature README: descendants = [T001, T002] → all Completed
	if records[0].Derived["estado"] != "Completed" {
		t.Errorf("Feature README estado = %v, want Completed", records[0].Derived["estado"])
	}
}

func TestAggregateAll_ChildrenVariable(t *testing.T) {
	// Test that children (sub-index records) are available with derived values.
	records := []*extract.Record{
		{Path: "E01/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "E01/F01/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Specified"}},
		{Path: "E01/F01/T001.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"child_count": "len(children)",
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)
	AggregateAll(context.Background(), records, "/root", resolver)

	// E01 README has 1 child index (F01/README.md)
	if records[0].Derived["child_count"] != 1 {
		t.Errorf("E01 child_count = %v, want 1", records[0].Derived["child_count"])
	}

	// F01 README has 0 child indexes
	if records[1].Derived["child_count"] != 0 {
		t.Errorf("F01 child_count = %v, want 0", records[1].Derived["child_count"])
	}
}

func TestAggregateAll_NoConfig(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "dir/T001.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed"}},
	}

	stem := &rules.StemFile{} // no aggregate
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)
	AggregateAll(context.Background(), records, "/root", resolver)

	if _, hasEstado := records[0].Derived["estado"]; hasEstado {
		t.Errorf("Derived[estado] = %v, want absent (no aggregate config)", records[0].Derived["estado"])
	}
}

func TestAggregateAll_NoIndexFiles(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/T001.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "dir/T002.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{"estado": `"Completed"`},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)
	AggregateAll(context.Background(), records, "/root", resolver)

	// No index files → no aggregation (only isIndex from enrichment)
	for _, rec := range records {
		if _, hasEstado := rec.Derived["estado"]; hasEstado {
			t.Errorf("%s.Derived[estado] = %v, want absent (no index files)", rec.Path, rec.Derived["estado"])
		}
	}
}

func TestAggregateAll_NilResolver(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{}},
	}
	// Should not panic.
	AggregateAll(context.Background(), records, "/root", nil)
	if records[0].Derived != nil {
		t.Error("expected nil Derived with nil resolver")
	}
}

func TestAggregateAll_EmptyDescendants(t *testing.T) {
	// Index file with no descendants — expression should use frontmatter estado.
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `len(descendants) == 0 ? estado : all(descendants, {.estado == "Completed"}) ? "Completed" : estado`,
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)
	AggregateAll(context.Background(), records, "/root", resolver)

	if records[0].Derived["estado"] != "Pending" {
		t.Errorf("estado = %v, want Pending (fallback)", records[0].Derived["estado"])
	}
}

func TestAggregateAll_RootReadme(t *testing.T) {
	// When scanned from a subdirectory, the root README.md has filepath.Dir = ".".
	// Descendants should still be collected correctly (not empty due to "./" prefix mismatch).
	records := []*extract.Record{
		{Path: "README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "In Progress"}},
		{Path: "F01/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "F01/T001-task.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "F02/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "F02/T001-task.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : "In Progress"`,
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)
	AggregateAll(context.Background(), records, "/root", resolver)

	// Root README should see all descendants including F02/T001 which is Pending.
	if records[0].Derived["estado"] != "In Progress" {
		t.Errorf("root README estado = %v, want 'In Progress' (not all descendants completed)", records[0].Derived["estado"])
	}
}

func TestAggregateAllSimple(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{}},
	}
	// No .stem in temp dir — should not panic.
	AggregateAllSimple(context.Background(), records, t.TempDir())
}
