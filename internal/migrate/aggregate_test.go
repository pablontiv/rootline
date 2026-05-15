package migrate

import (
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestGenerateAggregateExpr_SingleValue(t *testing.T) {
	sf := rules.SchemaField{
		Type:   "enum",
		Values: []string{"Done"},
	}
	expr := GenerateAggregateExpr("estado", sf)

	if expr != `"Done"` {
		t.Errorf("expected trivial expression, got: %s", expr)
	}
}

func TestGenerateAggregateExpr_MultiValue(t *testing.T) {
	sf := rules.SchemaField{
		Type:   "enum",
		Values: []string{"Pending", "In Progress", "Blocked", "Completed"},
	}
	expr := GenerateAggregateExpr("estado", sf)

	// Field-agnostic: uses first value as default
	if expr != `"Pending"` {
		t.Errorf("expected first value as default, got: %s", expr)
	}
}

func TestGenerateAggregateExpr_NonEnum(t *testing.T) {
	sf := rules.SchemaField{
		Type: "string",
	}
	expr := GenerateAggregateExpr("name", sf)

	if expr != "" {
		t.Errorf("expected empty for non-enum, got: %s", expr)
	}
}

func TestGenerateAggregates_SkipExisting(t *testing.T) {
	schema := map[string]rules.SchemaField{
		"estado": {Type: "enum", Values: []string{"Pending", "Completed"}},
		"tipo":   {Type: "enum", Values: []string{"A", "B"}},
		"name":   {Type: "string"},
	}
	existing := map[string]any{
		"estado": `"Completed"`,
	}

	result := GenerateAggregates(schema, existing)

	// estado should be skipped (already exists)
	if _, ok := result["estado"]; ok {
		t.Error("expected estado to be skipped")
	}
	// tipo should be generated
	if _, ok := result["tipo"]; !ok {
		t.Error("expected tipo to be generated")
	}
	// name should not appear (non-enum)
	if _, ok := result["name"]; ok {
		t.Error("expected name to be skipped (non-enum)")
	}
}
