package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestDetectRemoveStemField_FieldOverride(t *testing.T) {
	checks := []rules.StemHealthCheck{
		{Name: "field-override", Status: "warn", Field: "tipo", Path: "sub/.stem"},
	}

	proposals := DetectRemoveStemField(checks)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}
	if proposals[0].Type != RemoveStemField {
		t.Errorf("expected type remove_stem_field, got %s", proposals[0].Type)
	}
	if proposals[0].Field != "tipo" {
		t.Errorf("expected field 'tipo', got %q", proposals[0].Field)
	}
}

func TestDetectRemoveStemField_TypeConsistency(t *testing.T) {
	checks := []rules.StemHealthCheck{
		{Name: "type-consistency", Status: "fail", Field: "estado", Path: "sub/.stem"},
	}

	proposals := DetectRemoveStemField(checks)
	if len(proposals) != 1 {
		t.Fatalf("expected 1 proposal, got %d", len(proposals))
	}
	if proposals[0].Field != "estado" {
		t.Errorf("expected field 'estado', got %q", proposals[0].Field)
	}
}

func TestDetectRemoveStemField_SkipsSequenceField(t *testing.T) {
	checks := []rules.StemHealthCheck{
		{Name: "field-override", Status: "warn", Field: "id", Path: "sub/.stem"},
	}

	proposals := DetectRemoveStemField(checks)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals for id field, got %d", len(proposals))
	}
}

func TestDetectRemoveStemField_SkipsPassChecks(t *testing.T) {
	checks := []rules.StemHealthCheck{
		{Name: "yaml-valid", Status: "pass", Path: "sub/.stem"},
	}

	proposals := DetectRemoveStemField(checks)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals, got %d", len(proposals))
	}
}

func TestDetectRemoveStemField_DeduplicatesSamePathField(t *testing.T) {
	checks := []rules.StemHealthCheck{
		{Name: "type-consistency", Status: "fail", Field: "estado", Path: "sub/.stem"},
		{Name: "field-override", Status: "warn", Field: "estado", Path: "sub/.stem"},
	}

	proposals := DetectRemoveStemField(checks)
	if len(proposals) != 1 {
		t.Errorf("expected 1 deduplicated proposal, got %d", len(proposals))
	}
}

func TestDetectRemoveStemField_SkipsEmptyFieldOrPath(t *testing.T) {
	checks := []rules.StemHealthCheck{
		{Name: "field-override", Status: "warn", Field: "", Path: "sub/.stem"},
		{Name: "field-override", Status: "warn", Field: "tipo", Path: ""},
	}

	proposals := DetectRemoveStemField(checks)
	if len(proposals) != 0 {
		t.Errorf("expected 0 proposals, got %d", len(proposals))
	}
}

func TestDetectRemoveStemField_MultipleFields(t *testing.T) {
	checks := []rules.StemHealthCheck{
		{Name: "field-override", Status: "warn", Field: "tipo", Path: "sub/.stem"},
		{Name: "field-override", Status: "warn", Field: "estado", Path: "sub/.stem"},
		{Name: "field-override", Status: "warn", Field: "id", Path: "sub/.stem"}, // should be skipped
	}

	proposals := DetectRemoveStemField(checks)
	if len(proposals) != 2 {
		t.Errorf("expected 2 proposals (tipo + estado, not id), got %d", len(proposals))
	}
}
