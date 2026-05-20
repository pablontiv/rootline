package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestGenerateAggregateExpr_NonEnum(t *testing.T) {
	got := generateAggregateExpr("name", rules.SchemaField{Type: "string"})
	if got != "" {
		t.Errorf("expected empty for non-enum, got %q", got)
	}
}

func TestGenerateAggregateExpr_EmptyValues(t *testing.T) {
	got := generateAggregateExpr("estado", rules.SchemaField{Type: "enum", Values: nil})
	if got != "" {
		t.Errorf("expected empty for enum without values, got %q", got)
	}
}

func TestGenerateAggregateExpr_SingleValue(t *testing.T) {
	got := generateAggregateExpr("estado", rules.SchemaField{Type: "enum", Values: []string{"Done"}})
	if got != `"Done"` {
		t.Errorf("expected quoted single value, got %q", got)
	}
}

func TestGenerateAggregateExpr_MultiValue(t *testing.T) {
	got := generateAggregateExpr("estado", rules.SchemaField{Type: "enum", Values: []string{"Pending", "Done"}})
	if got != `"Pending"` {
		t.Errorf("expected first value, got %q", got)
	}
}

func TestGenerateAggregateExpressions(t *testing.T) {
	schema := map[string]rules.SchemaField{
		"estado": {Type: "enum", Values: []string{"Pending", "Done"}},
		"name":   {Type: "string"},
		"empty":  {Type: "enum"},
	}
	got := generateAggregateExpressions(schema)
	if len(got) != 1 {
		t.Errorf("expected 1 aggregate expr, got %d: %v", len(got), got)
	}
	if got["estado"] != `"Pending"` {
		t.Errorf("expected estado=\"Pending\", got %q", got["estado"])
	}
}
