package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestDetectExtendEnum(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completado"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Obsoleto" not in allowed values: [Pending, Completado]`}},
		"b.md": {{Rule: "enum", Field: "estado", Message: `value "Obsoleto" not in allowed values: [Pending, Completado]`}},
		"c.md": {{Rule: "enum", Field: "estado", Message: `value "Obsoleto" not in allowed values: [Pending, Completado]`}},
	}

	proposals := detectExtendEnum(stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Type != ExtendEnum {
		t.Errorf("type = %q, want extend_enum", proposals[0].Type)
	}
	if proposals[0].Value != "Obsoleto" {
		t.Errorf("value = %q, want Obsoleto", proposals[0].Value)
	}
	if len(proposals[0].Paths) != 3 {
		t.Errorf("paths = %d, want 3", len(proposals[0].Paths))
	}
}

func TestDetectCorrectValue(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completado", "In Progress"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Completo" not in allowed values: [Pending, Completado, In Progress]`}},
	}

	proposals := detectCorrectValue(stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Type != CorrectValue {
		t.Errorf("type = %q, want correct_value", proposals[0].Type)
	}
	if proposals[0].From != "Completo" {
		t.Errorf("from = %q, want Completo", proposals[0].From)
	}
	if proposals[0].To != "Completado" {
		t.Errorf("to = %q, want Completado", proposals[0].To)
	}
}

func TestDetectAddField(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completado"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "required", Field: "estado", Message: `required field "estado" is missing`}},
	}

	proposals := detectAddField(stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Type != AddField {
		t.Errorf("type = %q, want add_field", proposals[0].Type)
	}
	if proposals[0].Value != "Pending" {
		t.Errorf("value = %q, want Pending (first enum value)", proposals[0].Value)
	}
}

func TestAnalyze_NoErrors(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending"}, Required: true},
		},
	}

	report := Analyze([]*extract.Record{}, stem, map[string][]rules.ValidationError{})
	if len(report.Proposals) != 0 {
		t.Errorf("got %d proposals, want 0", len(report.Proposals))
	}
	if report.Summary.Total != 0 {
		t.Errorf("summary.total = %d, want 0", report.Summary.Total)
	}
}

func TestAnalyze_Mixed(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completado"}, Required: true},
			"tipo":   {Type: "enum", Values: []string{"feature", "bug"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {
			{Rule: "enum", Field: "estado", Message: `value "Completo" not in allowed values: [Pending, Completado]`},
		},
		"b.md": {
			{Rule: "required", Field: "tipo", Message: `required field "tipo" is missing`},
		},
	}

	report := Analyze([]*extract.Record{}, stem, errs)
	if report.Summary.Total != 2 {
		t.Errorf("summary.total = %d, want 2", report.Summary.Total)
	}
	if report.Summary.CorrectValue != 1 {
		t.Errorf("summary.correct_value = %d, want 1", report.Summary.CorrectValue)
	}
	if report.Summary.AddField != 1 {
		t.Errorf("summary.add_field = %d, want 1", report.Summary.AddField)
	}
}
