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

func TestParseBlockingInfo_Simple(t *testing.T) {
	base, targets, notes := ParseBlockingInfo("Pending (blocked by T001)")
	if base != "Pending" {
		t.Errorf("base = %q, want Pending", base)
	}
	if len(targets) != 1 || targets[0] != "T001" {
		t.Errorf("targets = %v, want [T001]", targets)
	}
	if len(notes) != 0 {
		t.Errorf("notes = %v, want []", notes)
	}
}

func TestParseBlockingInfo_PathTarget(t *testing.T) {
	base, targets, _ := ParseBlockingInfo("Pending (blocked by E04/F01)")
	if base != "Pending" {
		t.Errorf("base = %q, want Pending", base)
	}
	if len(targets) != 1 || targets[0] != "E04/F01" {
		t.Errorf("targets = %v, want [E04/F01]", targets)
	}
}

func TestParseBlockingInfo_Compound(t *testing.T) {
	base, targets, notes := ParseBlockingInfo("Pending (blocked by E04 + E03/F05 + human)")
	if base != "Pending" {
		t.Errorf("base = %q, want Pending", base)
	}
	if len(targets) != 2 || targets[0] != "E04" || targets[1] != "E03/F05" {
		t.Errorf("targets = %v, want [E04, E03/F05]", targets)
	}
	if len(notes) != 1 || notes[0] != "human" {
		t.Errorf("notes = %v, want [human]", notes)
	}
}

func TestParseBlockingInfo_NoParens(t *testing.T) {
	base, targets, notes := ParseBlockingInfo("Pending")
	if base != "Pending" {
		t.Errorf("base = %q, want Pending", base)
	}
	if targets != nil || notes != nil {
		t.Errorf("expected nil targets and notes for simple value")
	}
}

func TestDetectMigrateValue(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Bloqueada", "Completado"}, Required: true},
		},
	}

	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "enum", Field: "estado", Message: `value "Pending (blocked by T001)" not in allowed values: [Pending, Bloqueada, Completado]`}},
	}

	proposals := detectMigrateValue(stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	p := proposals[0]
	if p.Type != MigrateValue {
		t.Errorf("type = %q, want migrate_value", p.Type)
	}
	if p.From != "Pending (blocked by T001)" {
		t.Errorf("from = %q, want full original value", p.From)
	}
	if p.To != "Pending" {
		t.Errorf("to = %q, want Pending", p.To)
	}
	if len(p.WikiLinks) != 1 || p.WikiLinks[0] != "[[blocks:T001]]" {
		t.Errorf("wiki_links = %v, want [[blocks:T001]]", p.WikiLinks)
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
