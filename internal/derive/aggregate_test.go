package derive

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestAggregateAll_BasicEstado(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending", "tipo": "feature"}},
		{Path: "dir/T001-task.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completado", "tipo": "software-module"}},
		{Path: "dir/T002-task.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completado", "tipo": "software-module"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completado"}) ? "Completado" : estado`,
		},
	}
	resolver := func(dir string) *rules.StemFile { return stem }

	AggregateAll(records, "/root", resolver)

	if records[0].Derived["estado"] != "Completado" {
		t.Errorf("README estado = %v, want Completado", records[0].Derived["estado"])
	}
	// Non-index records should NOT have derived estado.
	if records[1].Derived != nil {
		t.Errorf("task[0].Derived = %v, want nil", records[1].Derived)
	}
}

func TestAggregateAll_NotAllCompleted(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending", "tipo": "feature"}},
		{Path: "dir/T001-task.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completado", "tipo": "software-module"}},
		{Path: "dir/T002-task.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Bloqueada", "tipo": "software-module"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completado"}) ? "Completado" : any(descendants, {.estado == "Bloqueada"}) ? "Bloqueada" : estado`,
		},
	}
	resolver := func(dir string) *rules.StemFile { return stem }

	AggregateAll(records, "/root", resolver)

	if records[0].Derived["estado"] != "Bloqueada" {
		t.Errorf("README estado = %v, want Bloqueada", records[0].Derived["estado"])
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
		{Path: "F01/S001/T001.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completado", "tipo": "software-module"}},
		{Path: "F01/S001/T002.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completado", "tipo": "software-module"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completado"}) ? "Completado" : estado`,
		},
	}
	resolver := func(dir string) *rules.StemFile { return stem }

	AggregateAll(records, "/root", resolver)

	// Story README: descendants = [T001, T002] → all Completado
	if records[1].Derived["estado"] != "Completado" {
		t.Errorf("Story README estado = %v, want Completado", records[1].Derived["estado"])
	}

	// Feature README: descendants = [T001, T002] → all Completado
	if records[0].Derived["estado"] != "Completado" {
		t.Errorf("Feature README estado = %v, want Completado", records[0].Derived["estado"])
	}
}

func TestAggregateAll_ChildrenVariable(t *testing.T) {
	// Test that children (sub-index records) are available with derived values.
	records := []*extract.Record{
		{Path: "E01/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "E01/F01/README.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Specified"}},
		{Path: "E01/F01/T001.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completado"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"child_count": "len(children)",
		},
	}
	resolver := func(dir string) *rules.StemFile { return stem }

	AggregateAll(records, "/root", resolver)

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
		{Path: "dir/T001.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completado"}},
	}

	stem := &rules.StemFile{} // no aggregate
	resolver := func(dir string) *rules.StemFile { return stem }

	AggregateAll(records, "/root", resolver)

	if records[0].Derived != nil {
		t.Errorf("Derived = %v, want nil", records[0].Derived)
	}
}

func TestAggregateAll_NoIndexFiles(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/T001.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completado"}},
		{Path: "dir/T002.md", Type: "markdown", Frontmatter: map[string]any{"estado": "Completado"}},
	}

	stem := &rules.StemFile{
		Aggregate: map[string]any{"estado": `"Completado"`},
	}
	resolver := func(dir string) *rules.StemFile { return stem }

	AggregateAll(records, "/root", resolver)

	// No index files → no aggregation
	for _, rec := range records {
		if rec.Derived != nil {
			t.Errorf("%s.Derived = %v, want nil", rec.Path, rec.Derived)
		}
	}
}

func TestAggregateAll_NilResolver(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{}},
	}
	// Should not panic.
	AggregateAll(records, "/root", nil)
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
			"estado": `len(descendants) == 0 ? estado : all(descendants, {.estado == "Completado"}) ? "Completado" : estado`,
		},
	}
	resolver := func(dir string) *rules.StemFile { return stem }

	AggregateAll(records, "/root", resolver)

	if records[0].Derived["estado"] != "Pending" {
		t.Errorf("estado = %v, want Pending (fallback)", records[0].Derived["estado"])
	}
}
