package infer

import (
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestDetectValidationGaps_EnumWithoutValues(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"prioridad": {Type: "enum", Source: "docs/.stem"},
		},
	}
	got := detectGapsForScope(stem, nil, nil)
	found := false
	for _, inf := range got {
		if inf.Type == "enum_without_values" && inf.Field == "prioridad" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected enum_without_values inference for 'prioridad'")
	}
}

func TestDetectValidationGaps_EnumWithValues_NoGap(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"draft", "active"}, Source: "docs/.stem"},
		},
	}
	got := detectGapsForScope(stem, nil, nil)
	for _, inf := range got {
		if inf.Type == "enum_without_values" && inf.Field == "estado" {
			t.Error("should not flag enum with values")
		}
	}
}

func TestDetectValidationGaps_UntypedField(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"mystery": {Source: "docs/.stem"},
		},
	}
	got := detectGapsForScope(stem, nil, nil)
	found := false
	for _, inf := range got {
		if inf.Type == "untyped_field" && inf.Field == "mystery" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected untyped_field inference for 'mystery'")
	}
}

func TestDetectValidationGaps_SequenceIncomplete(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"id": {Type: "sequence", Prefix: "T", Source: "docs/.stem"},
		},
	}
	got := detectGapsForScope(stem, nil, nil)
	found := false
	for _, inf := range got {
		if inf.Type == "sequence_incomplete" && inf.Field == "id" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected sequence_incomplete inference for 'id'")
	}
}

func TestDetectValidationGaps_RequiredUnderstatement(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Source: "docs/.stem"},
		},
	}
	records := make([]*extract.Record, 10)
	for i := range records {
		fm := map[string]any{}
		if i < 9 {
			fm["tipo"] = "feature"
		}
		records[i] = &extract.Record{Path: "doc.md", Frontmatter: fm}
	}
	got := detectGapsForScope(stem, records, nil)
	found := false
	for _, inf := range got {
		if inf.Type == "required_understatement" && inf.Field == "tipo" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected required_understatement inference for 'tipo'")
	}
}

func TestDetectValidationGaps_NilStem(t *testing.T) {
	got := detectGapsForScope(nil, nil, nil)
	if len(got) != 0 {
		t.Fatalf("expected 0 inferences for nil stem, got %d", len(got))
	}
}

func TestDetectValidationGaps_Deduplication(t *testing.T) {
	stem := &rules.StemFile{
		Path: "docs/.stem",
		Schema: map[string]rules.SchemaField{
			"prioridad": {Type: "enum", Source: "docs/.stem"},
		},
	}
	// Prior inference from enum_values detector covers this field
	prior := []Inference{{Type: "enum_values", Field: "prioridad"}}
	got := detectGapsForScope(stem, nil, prior)
	for _, inf := range got {
		if inf.Type == "enum_without_values" && inf.Field == "prioridad" {
			t.Error("should not flag field already covered by enum_values detector")
		}
	}
}

func TestDetectValidationGaps_MultiScope(t *testing.T) {
	conceptsStem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"status": {Type: "enum", Values: []string{"a"}, Source: "concepts/.stem"},
	}}
	// sources scope declares an enum WITHOUT values → a gap that root-only would miss.
	sourcesStem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"tipo": {Type: "enum", Source: "sources/.stem"},
	}}
	resolve := func(dir string) (*rules.StemFile, string) {
		if filepath.Base(dir) == "sources" {
			return sourcesStem, "sources/.stem"
		}
		return conceptsStem, "concepts/.stem"
	}
	records := []*extract.Record{{Path: "concepts/a.md"}, {Path: "sources/p.md"}}

	got := DetectValidationGaps(records, nil, ".", resolve)

	found := false
	for _, inf := range got {
		if inf.Type == "enum_without_values" && inf.Field == "tipo" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected enum_without_values for sources/.stem:tipo, got %+v", got)
	}
}

func TestDetectValidationGaps_LegacySectionParticipatesInRequiredUnderstatement(t *testing.T) {
	stem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"notes": {Type: "section", Source: "docs/.stem"},
	}}
	records := make([]*extract.Record, 5)
	for i := range records {
		records[i] = &extract.Record{Path: "doc.md", Frontmatter: map[string]any{"notes": "present"}}
	}
	got := detectGapsForScope(stem, records, nil)
	for _, inf := range got {
		if inf.Type == "required_understatement" && inf.Field == "notes" {
			return
		}
	}
	t.Fatalf("expected required_understatement for legacy section, got %+v", got)
}
