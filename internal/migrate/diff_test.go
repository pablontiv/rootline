package migrate

import (
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestDiff_NoChanges(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Required: true, Values: []string{"Pending", "Done"}, Severity: "error"},
		},
	}

	result := Diff(".stem", stem, stem)

	if len(result.Changes) != 0 {
		t.Errorf("expected 0 changes, got %d: %v", len(result.Changes), result.Changes)
	}
	if result.TotalCount != 0 {
		t.Errorf("expected total_count=0, got %d", result.TotalCount)
	}
	if result.BreakingCount != 0 {
		t.Errorf("expected breaking_count=0, got %d", result.BreakingCount)
	}
	if result.Version != 1 {
		t.Errorf("expected version=1, got %d", result.Version)
	}
	if result.Kind != "rootline/migrate-diff" {
		t.Errorf("expected kind=rootline/migrate-diff, got %s", result.Kind)
	}
}

func TestDiff_FieldAdded(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Required: true, Values: []string{"Pending"}, Severity: "error"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Required: true, Values: []string{"Pending"}, Severity: "error"},
			"tipo":   {Type: "string", Severity: "error"},
		},
	}

	result := Diff(".stem", before, after)

	if result.TotalCount != 1 {
		t.Fatalf("expected 1 change, got %d", result.TotalCount)
	}
	c := result.Changes[0]
	if c.Kind != FieldAdded {
		t.Errorf("expected field_added, got %s", c.Kind)
	}
	if c.Field != "tipo" {
		t.Errorf("expected field=tipo, got %s", c.Field)
	}
	if c.Breaking {
		t.Errorf("field_added should be non-breaking")
	}
}

func TestDiff_FieldRemoved(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Required: true, Values: []string{"Pending"}, Severity: "error"},
			"tipo":   {Type: "string", Severity: "error"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Required: true, Values: []string{"Pending"}, Severity: "error"},
		},
	}

	result := Diff(".stem", before, after)

	if result.TotalCount != 1 {
		t.Fatalf("expected 1 change, got %d", result.TotalCount)
	}
	c := result.Changes[0]
	if c.Kind != FieldRemoved {
		t.Errorf("expected field_removed, got %s", c.Kind)
	}
	if c.Field != "tipo" {
		t.Errorf("expected field=tipo, got %s", c.Field)
	}
	if !c.Breaking {
		t.Errorf("field_removed should be breaking")
	}
	if result.BreakingCount != 1 {
		t.Errorf("expected breaking_count=1, got %d", result.BreakingCount)
	}
}

func TestDiff_EnumValueAdded(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done"}, Severity: "error"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done", "InProgress"}, Severity: "error"},
		},
	}

	result := Diff(".stem", before, after)

	if result.TotalCount != 1 {
		t.Fatalf("expected 1 change, got %d", result.TotalCount)
	}
	c := result.Changes[0]
	if c.Kind != EnumValueAdded {
		t.Errorf("expected enum_value_added, got %s", c.Kind)
	}
	if c.Breaking {
		t.Errorf("enum_value_added should be non-breaking")
	}
	if c.After != "InProgress" {
		t.Errorf("expected after=InProgress, got %s", c.After)
	}
}

func TestDiff_EnumValueRemoved(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done", "Cancelled"}, Severity: "error"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done"}, Severity: "error"},
		},
	}

	result := Diff(".stem", before, after)

	if result.TotalCount != 1 {
		t.Fatalf("expected 1 change, got %d", result.TotalCount)
	}
	c := result.Changes[0]
	if c.Kind != EnumValueRemoved {
		t.Errorf("expected enum_value_removed, got %s", c.Kind)
	}
	if !c.Breaking {
		t.Errorf("enum_value_removed should be breaking")
	}
	if c.Before != "Cancelled" {
		t.Errorf("expected before=Cancelled, got %s", c.Before)
	}
}

func TestDiff_RequiredFalseToTrue(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Required: false, Severity: "error"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Required: true, Severity: "error"},
		},
	}

	result := Diff(".stem", before, after)

	if result.TotalCount != 1 {
		t.Fatalf("expected 1 change, got %d", result.TotalCount)
	}
	c := result.Changes[0]
	if c.Kind != RequiredAdded {
		t.Errorf("expected required_added, got %s", c.Kind)
	}
	if !c.Breaking {
		t.Errorf("required false->true should be breaking")
	}
}

func TestDiff_RequiredTrueToFalse(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Required: true, Severity: "error"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Required: false, Severity: "error"},
		},
	}

	result := Diff(".stem", before, after)

	if result.TotalCount != 1 {
		t.Fatalf("expected 1 change, got %d", result.TotalCount)
	}
	c := result.Changes[0]
	if c.Kind != RequiredRemoved {
		t.Errorf("expected required_removed, got %s", c.Kind)
	}
	if c.Breaking {
		t.Errorf("required true->false should be non-breaking")
	}
}

func TestDiff_TypeChanged(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "string", Severity: "error"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"A"}, Severity: "error"},
		},
	}

	result := Diff(".stem", before, after)

	found := false
	for _, c := range result.Changes {
		if c.Kind == TypeChanged {
			found = true
			if !c.Breaking {
				t.Errorf("type_changed should be breaking")
			}
			if c.Before != "string" || c.After != "enum" {
				t.Errorf("expected string->enum, got %s->%s", c.Before, c.After)
			}
		}
	}
	if !found {
		t.Errorf("expected type_changed change")
	}
}

func TestDiff_NilBefore(t *testing.T) {
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending"}, Severity: "error"},
			"tipo":   {Type: "string", Severity: "error"},
		},
	}

	result := Diff(".stem", nil, after)

	if result.TotalCount != 2 {
		t.Errorf("expected 2 changes (2 fields added), got %d", result.TotalCount)
	}
	if result.BreakingCount != 0 {
		t.Errorf("new fields should be non-breaking, got breaking_count=%d", result.BreakingCount)
	}
}

func TestDiff_NilAfter(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending"}, Severity: "error"},
		},
	}

	result := Diff(".stem", before, nil)

	if result.TotalCount != 1 {
		t.Errorf("expected 1 change (1 field removed), got %d", result.TotalCount)
	}
	if result.BreakingCount != 1 {
		t.Errorf("removed fields should be breaking, got breaking_count=%d", result.BreakingCount)
	}
}

func TestDiff_MultipleChanges(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado":        {Type: "enum", Values: []string{"Pending", "Done", "Cancelled"}, Required: true, Severity: "error"},
			"old_field":     {Type: "string", Severity: "error"},
			"ejecutable_en": {Type: "string", Severity: "error"},
		},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado":        {Type: "enum", Values: []string{"Pending", "Done", "InProgress"}, Required: true, Severity: "error"},
			"new_field":     {Type: "string", Severity: "error"},
			"ejecutable_en": {Type: "string", Required: true, Severity: "error"},
		},
	}

	result := Diff(".stem", before, after)

	// Cancelled removed (breaking), InProgress added (non-breaking),
	// old_field removed (breaking), new_field added (non-breaking),
	// ejecutable_en required false->true (breaking)
	if result.TotalCount != 5 {
		t.Errorf("expected 5 changes, got %d", result.TotalCount)
		for _, c := range result.Changes {
			t.Logf("  %s %s breaking=%v", c.Kind, c.Field, c.Breaking)
		}
	}
	if result.BreakingCount != 3 {
		t.Errorf("expected 3 breaking changes, got %d", result.BreakingCount)
	}
}

func TestDiff_ValidationRules(t *testing.T) {
	before := &rules.StemFile{
		Validate: []rules.ValidationRule{
			{Rule: "requires", Field: "tipo", Severity: "error"},
		},
	}
	after := &rules.StemFile{
		Validate: []rules.ValidationRule{
			{Rule: "requires", Field: "tipo", Severity: "error"},
			{Rule: "requires", Field: "estado", Severity: "error"},
		},
	}

	result := Diff(".stem", before, after)

	if result.TotalCount != 1 {
		t.Fatalf("expected 1 change, got %d", result.TotalCount)
	}
	c := result.Changes[0]
	if c.Kind != RuleAdded {
		t.Errorf("expected rule_added, got %s", c.Kind)
	}
}

func TestDiff_DeterministicOrder(t *testing.T) {
	before := &rules.StemFile{
		Schema: map[string]rules.SchemaField{},
	}
	after := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"z_field": {Type: "string", Severity: "error"},
			"a_field": {Type: "string", Severity: "error"},
			"m_field": {Type: "string", Severity: "error"},
		},
	}

	result := Diff(".stem", before, after)

	if len(result.Changes) != 3 {
		t.Fatalf("expected 3 changes, got %d", len(result.Changes))
	}
	if result.Changes[0].Field != "a_field" {
		t.Errorf("expected first change on a_field, got %s", result.Changes[0].Field)
	}
	if result.Changes[1].Field != "m_field" {
		t.Errorf("expected second change on m_field, got %s", result.Changes[1].Field)
	}
	if result.Changes[2].Field != "z_field" {
		t.Errorf("expected third change on z_field, got %s", result.Changes[2].Field)
	}
}
