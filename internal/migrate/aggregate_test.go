package migrate

import (
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestGenerateAggregateExpr_EnumEN(t *testing.T) {
	sf := rules.SchemaField{
		Type:   "enum",
		Values: []string{"Pending", "In Progress", "Blocked", "Completed"},
	}
	expr := GenerateAggregateExpr("estado", sf)

	// Terminal: Completed uses all()
	if !strings.Contains(expr, `all(descendants, {.estado == "Completed"}) ? "Completed"`) {
		t.Errorf("expected all() for Completed, got:\n%s", expr)
	}
	// Negative: Blocked uses any()
	if !strings.Contains(expr, `any(descendants, {.estado == "Blocked"}) ? "Blocked"`) {
		t.Errorf("expected any() for Blocked, got:\n%s", expr)
	}
	// Active: In Progress uses any()
	if !strings.Contains(expr, `any(descendants, {.estado == "In Progress"}) ? "In Progress"`) {
		t.Errorf("expected any() for In Progress, got:\n%s", expr)
	}
	// Default: Pending (first neutral value)
	if !strings.HasSuffix(expr, `"Pending"`) {
		t.Errorf("expected Pending as default, got:\n%s", expr)
	}
}

func TestGenerateAggregateExpr_EnumES(t *testing.T) {
	sf := rules.SchemaField{
		Type:   "enum",
		Values: []string{"Pending", "En Progreso", "Bloqueada", "Diferida", "Completado", "Obsoleto"},
	}
	expr := GenerateAggregateExpr("estado", sf)

	// Terminal: Completado and Obsoleto use all()
	if !strings.Contains(expr, `all(descendants, {.estado == "Completado"})`) {
		t.Errorf("expected all() for Completado, got:\n%s", expr)
	}
	if !strings.Contains(expr, `all(descendants, {.estado == "Obsoleto"})`) {
		t.Errorf("expected all() for Obsoleto, got:\n%s", expr)
	}
	// Negative: Bloqueada and Diferida use any()
	if !strings.Contains(expr, `any(descendants, {.estado == "Bloqueada"})`) {
		t.Errorf("expected any() for Bloqueada, got:\n%s", expr)
	}
	if !strings.Contains(expr, `any(descendants, {.estado == "Diferida"})`) {
		t.Errorf("expected any() for Diferida, got:\n%s", expr)
	}
	// Active: En Progreso
	if !strings.Contains(expr, `any(descendants, {.estado == "En Progreso"})`) {
		t.Errorf("expected any() for En Progreso, got:\n%s", expr)
	}
}

func TestGenerateAggregateExpr_NoKeywords(t *testing.T) {
	sf := rules.SchemaField{
		Type:   "enum",
		Values: []string{"Alpha", "Beta", "Gamma"},
	}
	expr := GenerateAggregateExpr("status", sf)

	// Fallback: last value (Gamma) is terminal with all()
	if !strings.Contains(expr, `all(descendants, {.status == "Gamma"}) ? "Gamma"`) {
		t.Errorf("expected all() for Gamma (fallback terminal), got:\n%s", expr)
	}
	// Default should be Alpha (first value)
	if !strings.HasSuffix(expr, `"Alpha"`) {
		t.Errorf("expected Alpha as default, got:\n%s", expr)
	}
}

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
		"estado": `all(descendants, {.estado == "Completed"}) ? "Completed" : "Pending"`,
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
