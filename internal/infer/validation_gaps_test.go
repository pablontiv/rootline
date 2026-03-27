package infer

import (
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
	got := DetectValidationGaps(stem, nil, nil)
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
	got := DetectValidationGaps(stem, nil, nil)
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
	got := DetectValidationGaps(stem, nil, nil)
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
	got := DetectValidationGaps(stem, nil, nil)
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
	got := DetectValidationGaps(stem, records, nil)
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
	got := DetectValidationGaps(nil, nil, nil)
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
	got := DetectValidationGaps(stem, nil, prior)
	for _, inf := range got {
		if inf.Type == "enum_without_values" && inf.Field == "prioridad" {
			t.Error("should not flag field already covered by enum_values detector")
		}
	}
}
