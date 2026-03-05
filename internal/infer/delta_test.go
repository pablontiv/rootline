package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestFilterCoveredInferences_RequiredCovered(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Required: true, Values: []string{"Pending", "Done"}},
		},
	}

	inferences := []Inference{
		{Type: "required_field", Field: "estado", Message: "estado is required"},
	}

	deltas := FilterCoveredInferences(inferences, stem)
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

	inferences := []Inference{
		{Type: "required_field", Field: "estado", Message: "estado is required"},
	}

	deltas := FilterCoveredInferences(inferences, stem)
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

	inferences := []Inference{
		{Type: "field_type", Field: "estado", Value: "enum"},
	}

	deltas := FilterCoveredInferences(inferences, stem)
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

	inferences := []Inference{
		{Type: "enum_values", Field: "tipo", Value: "[task feature]"},
	}

	deltas := FilterCoveredInferences(inferences, stem)
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

	inferences := []Inference{
		{Type: "constant_field", Field: "estado", Value: "Pending"},
	}

	deltas := FilterCoveredInferences(inferences, stem)
	if len(deltas) != 0 {
		t.Errorf("expected constant_field to be covered, got %v", deltas)
	}
}

func TestFilterCoveredInferences_NilStem(t *testing.T) {
	inferences := []Inference{
		{Type: "required_field", Field: "estado"},
	}

	deltas := FilterCoveredInferences(inferences, nil)
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

	inferences := []Inference{
		{Type: "section_patterns", Field: "Contexto", Value: "1.00"},
	}

	deltas := FilterCoveredInferences(inferences, stem)
	if len(deltas) != 1 {
		t.Errorf("expected unknown type to pass through, got %d", len(deltas))
	}
}
