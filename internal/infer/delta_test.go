package infer

import (
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func singleScope(stem *rules.StemFile) StemResolver {
	return func(dir string) (*rules.StemFile, string) { return stem, "/.stem" }
}

func TestFilterCoveredInferences_RequiredCovered(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Required: true, Values: []string{"Pending", "Done"}},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	inferences := []Inference{
		{Type: "required_field", Field: "estado", Message: "estado is required"},
	}

	deltas := FilterCoveredInferences(inferences, records, ".", singleScope(stem))
	if len(deltas) != 0 {
		t.Errorf("expected inference to be filtered (covered by stem), got %v", deltas)
	}
}

func TestFilterCoveredInferences_RequiredNotCovered(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string"},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	inferences := []Inference{
		{Type: "required_field", Field: "estado", Message: "estado is required"},
	}

	deltas := FilterCoveredInferences(inferences, records, ".", singleScope(stem))
	if len(deltas) != 1 {
		t.Errorf("expected 1 uncovered inference, got %d", len(deltas))
	}
}

func TestFilterCoveredInferences_FieldTypeCovered(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum"},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "draft"}},
	}

	inferences := []Inference{
		{Type: "field_type", Field: "estado", Value: "enum"},
	}

	deltas := FilterCoveredInferences(inferences, records, ".", singleScope(stem))
	if len(deltas) != 0 {
		t.Errorf("expected field_type to be covered, got %v", deltas)
	}
}

func TestFilterCoveredInferences_EnumCovered(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"task", "feature"}},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"tipo": "task"}},
	}

	inferences := []Inference{
		{Type: "enum_values", Field: "tipo", Value: "[task feature]"},
	}

	deltas := FilterCoveredInferences(inferences, records, ".", singleScope(stem))
	if len(deltas) != 0 {
		t.Errorf("expected enum_values to be covered, got %v", deltas)
	}
}

func TestFilterCoveredInferences_ConstantCovered(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "string", Required: true},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	inferences := []Inference{
		{Type: "constant_field", Field: "estado", Value: "Pending"},
	}

	deltas := FilterCoveredInferences(inferences, records, ".", singleScope(stem))
	if len(deltas) != 0 {
		t.Errorf("expected constant_field to be covered, got %v", deltas)
	}
}

func TestFilterCoveredInferences_NilStem(t *testing.T) {
	nilResolver := func(dir string) (*rules.StemFile, string) { return nil, "" }

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "draft"}},
	}

	inferences := []Inference{
		{Type: "required_field", Field: "estado"},
	}

	deltas := FilterCoveredInferences(inferences, records, ".", nilResolver)
	if len(deltas) != 1 {
		t.Errorf("expected all inferences with nil stem, got %d", len(deltas))
	}
}

func TestFilterCoveredInferences_UnknownTypePassesThrough(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "string", Required: true},
		},
	}

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"Contexto": "body"}},
	}

	inferences := []Inference{
		{Type: "section_patterns", Field: "Contexto", Value: "1.00"},
	}

	deltas := FilterCoveredInferences(inferences, records, ".", singleScope(stem))
	if len(deltas) != 1 {
		t.Errorf("expected unknown type to pass through, got %d", len(deltas))
	}
}

func TestFilterCoveredInferences_MultiScope(t *testing.T) {
	conceptsStem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"status": {Type: "enum", Values: []string{"a", "b"}},
	}}
	sourcesStem := &rules.StemFile{Schema: map[string]rules.SchemaField{}}
	resolve := func(dir string) (*rules.StemFile, string) {
		if filepath.Base(dir) == "sources" {
			return sourcesStem, "sources/.stem"
		}
		return conceptsStem, "concepts/.stem"
	}
	records := []*extract.Record{
		{Path: "concepts/a.md", Frontmatter: map[string]any{"status": "a"}},
		{Path: "sources/p.md", Frontmatter: map[string]any{"ref": "x"}},
	}
	inferences := []Inference{
		{Type: "enum_values", Field: "status"},              // covered in concepts scope
		{Type: "field_type", Field: "ref", Value: "string"}, // not covered anywhere
	}
	got := FilterCoveredInferences(inferences, records, ".", resolve)
	if len(got) != 1 || got[0].Field != "ref" {
		t.Errorf("expected only 'ref' to survive, got %+v", got)
	}
}

func TestIsCovered_SectionSourceAndRequiredness(t *testing.T) {
	tests := []struct {
		name  string
		inf   Inference
		field rules.SchemaField
		want  bool
	}{
		{
			name:  "required section covered by same required source",
			inf:   Inference{Type: "required_section", Field: "notes", SourceDirective: `body.section["## Notes"]`},
			field: rules.SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`},
			want:  true,
		},
		{
			name:  "optional section covered by same optional source",
			inf:   Inference{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Notes"]`},
			field: rules.SchemaField{Type: "string", Extract: `body.section["## Notes"]`},
			want:  true,
		},
		{
			name:  "optional section covered by stronger required source",
			inf:   Inference{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Notes"]`},
			field: rules.SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`},
			want:  true,
		},
		{
			name:  "required section not covered by optional source",
			inf:   Inference{Type: "required_section", Field: "notes", SourceDirective: `body.section["## Notes"]`},
			field: rules.SchemaField{Type: "string", Extract: `body.section["## Notes"]`},
			want:  false,
		},
		{
			name:  "different source is not covered",
			inf:   Inference{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Context"]`},
			field: rules.SchemaField{Type: "string", Extract: `body.section["## Notes"]`},
			want:  false,
		},
		{
			name:  "missing existing source is not covered",
			inf:   Inference{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Notes"]`},
			field: rules.SchemaField{Type: "string"},
			want:  false,
		},
		{
			name:  "missing inferred source is not covered",
			inf:   Inference{Type: "optional_section", Field: "notes"},
			field: rules.SchemaField{Type: "string", Extract: `body.section["## Notes"]`},
			want:  false,
		},
		{
			name:  "matching source non-string field is not covered",
			inf:   Inference{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Notes"]`},
			field: rules.SchemaField{Type: "section", Extract: `body.section["## Notes"]`},
			want:  false,
		},
		{
			name:  "legacy heading-only field is not covered",
			inf:   Inference{Type: "optional_section", Field: "notes", SourceDirective: `body.section["## Notes"]`},
			field: rules.SchemaField{Heading: "Notes"},
			want:  false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stem := &rules.StemFile{Schema: map[string]rules.SchemaField{"notes": tt.field}}
			if got := isCovered(tt.inf, stem); got != tt.want {
				t.Fatalf("isCovered() = %v, want %v", got, tt.want)
			}
		})
	}
}

func TestFilterCoveredInferences_SectionSourceDelta(t *testing.T) {
	stem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"notes": {Type: "string", Required: true, Extract: `body.section["## Notes"]`},
	}}
	records := []*extract.Record{{Path: "a.md", Frontmatter: map[string]any{"notes": "present"}}}
	inferences := []Inference{
		{Type: "required_section", Field: "notes", SourceDirective: `body.section["## Notes"]`},
		{Type: "required_section", Field: "notes", SourceDirective: `body.section["## Context"]`},
	}

	got := FilterCoveredInferences(inferences, records, ".", singleScope(stem))
	if len(got) != 1 || got[0].SourceDirective != `body.section["## Context"]` {
		t.Fatalf("expected only changed-source section delta, got %+v", got)
	}
}

func TestIsCovered_EnumWithoutValues(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"draft", "active"}},
		},
	}
	if !isCovered(Inference{Type: "enum_without_values", Field: "estado"}, stem) {
		t.Error("enum_without_values should be covered when field has values")
	}
}

func TestIsCovered_UntypedField(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"titulo": {Type: "string"},
		},
	}
	if !isCovered(Inference{Type: "untyped_field", Field: "titulo"}, stem) {
		t.Error("untyped_field should be covered when field has type")
	}
}

func TestIsCovered_SequenceIncomplete(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"id": {Type: "sequence", Prefix: "T", Digits: 3},
		},
	}
	if !isCovered(Inference{Type: "sequence_incomplete", Field: "id"}, stem) {
		t.Error("sequence_incomplete should be covered when prefix and digits set")
	}
}

func TestIsCovered_RequiredUnderstatement(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Required: true},
		},
	}
	if !isCovered(Inference{Type: "required_understatement", Field: "tipo"}, stem) {
		t.Error("required_understatement should be covered when field is required")
	}
}
