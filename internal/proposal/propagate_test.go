package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestDetectPropagateAggregate_Stale(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "S001/README.md",
			Frontmatter: map[string]any{"estado": "Pending"},
			Derived:     map[string]any{"estado": "Completed"},
		},
		{
			Path:        "S001/T001.md",
			Frontmatter: map[string]any{"estado": "Completed"},
		},
		{
			Path:        "S001/T002.md",
			Frontmatter: map[string]any{"estado": "Completed"},
		},
	}
	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"`,
		},
	}

	proposals := DetectPropagateAggregate(records, stem)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	p := proposals[0]
	if p.Type != PropagateAggregate {
		t.Errorf("type = %q, want propagate_aggregate", p.Type)
	}
	if p.From != "Pending" || p.To != "Completed" {
		t.Errorf("from=%q to=%q, want Pending->Completed", p.From, p.To)
	}
}

func TestDetectPropagateAggregate_Consistent(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "S001/README.md",
			Frontmatter: map[string]any{"estado": "Completed"},
			Derived:     map[string]any{"estado": "Completed"},
		},
		{
			Path:        "S001/T001.md",
			Frontmatter: map[string]any{"estado": "Completed"},
		},
	}
	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"`,
		},
	}

	proposals := DetectPropagateAggregate(records, stem)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (values match)", len(proposals))
	}
}

func TestDetectPropagateAggregateForRecordsLimitsTargetsButKeepsPopulation(t *testing.T) {
	first := &extract.Record{
		Path:        "S001/README.md",
		Frontmatter: map[string]any{"estado": "Pending"},
		Derived:     map[string]any{"estado": "Completed"},
	}
	second := &extract.Record{
		Path:        "S002/README.md",
		Frontmatter: map[string]any{"estado": "Pending"},
		Derived:     map[string]any{"estado": "Completed"},
	}
	records := []*extract.Record{
		first,
		{Path: "S001/T001.md", Frontmatter: map[string]any{"estado": "Completed"}},
		second,
		{Path: "S002/T001.md", Frontmatter: map[string]any{"estado": "Completed"}},
	}
	stem := &rules.StemFile{Aggregate: map[string]any{
		"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"`,
	}}

	proposals := DetectPropagateAggregateForRecords(records, []*extract.Record{first}, stem)
	if len(proposals) != 1 || len(proposals[0].Paths) != 1 || proposals[0].Paths[0] != first.Path {
		t.Fatalf("proposals = %#v, want only %s", proposals, first.Path)
	}
}

func TestDetectPropagateAggregate_MissingFrontmatter(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "S001/README.md",
			Frontmatter: map[string]any{},
			Derived:     map[string]any{"estado": "Completed"},
		},
		{
			Path:        "S001/T001.md",
			Frontmatter: map[string]any{"estado": "Completed"},
		},
	}
	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"`,
		},
	}

	proposals := DetectPropagateAggregate(records, stem)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (missing frontmatter handled by AddField)", len(proposals))
	}
}

func TestDetectPropagateAggregate_NonIndex(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "S001/T001.md",
			Frontmatter: map[string]any{"estado": "Pending"},
			Derived:     map[string]any{"estado": "Completed"},
		},
	}
	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"`,
		},
	}

	proposals := DetectPropagateAggregate(records, stem)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (non-index file)", len(proposals))
	}
}

func TestDetectPropagateAggregate_NoAggregate(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "S001/README.md",
			Frontmatter: map[string]any{"estado": "Pending"},
			Derived:     map[string]any{"estado": "Completed"},
		},
	}
	stem := &rules.StemFile{
		Aggregate: nil,
	}

	proposals := DetectPropagateAggregate(records, stem)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (no aggregate in stem)", len(proposals))
	}
}

func TestDetectPropagateAggregate_Mixed(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "S001/README.md",
			Frontmatter: map[string]any{"estado": "Pending", "tipo": "historia"},
			Derived:     map[string]any{"estado": "Completed", "tipo": "historia"},
		},
		{
			Path:        "S001/T001.md",
			Frontmatter: map[string]any{"estado": "Completed", "tipo": "historia"},
		},
	}
	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"`,
			"tipo":   `"historia"`,
		},
	}

	proposals := DetectPropagateAggregate(records, stem)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1 (only estado is stale)", len(proposals))
	}
	if proposals[0].Field != "estado" {
		t.Errorf("field = %q, want estado", proposals[0].Field)
	}
}

func TestDetectPropagateAggregate_UncoveredEnumValue(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "S001/README.md",
			Frontmatter: map[string]any{"estado": "Pending"},
			Derived:     map[string]any{"estado": "Pending"},
		},
		{
			Path:        "S001/T001.md",
			Frontmatter: map[string]any{"estado": "Completed"},
		},
		{
			Path:        "S001/T002.md",
			Frontmatter: map[string]any{"estado": "Obsolete"},
		},
	}
	// Formula only references "Completed" and "Pending" — "Obsolete" is uncovered.
	stem := &rules.StemFile{
		Aggregate: map[string]any{
			"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"`,
		},
	}

	proposals := DetectPropagateAggregate(records, stem)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (formula doesn't cover 'Obsolete')", len(proposals))
	}
}
