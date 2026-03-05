package infer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestApplySchemaInferences_ExtendEnum(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: enum\n    values: [a, b]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "enum_values", Field: "tipo", Value: "[a b c]"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	// Verify the stem was updated.
	data, _ := os.ReadFile(stemPath)
	stem, _ := rules.ParseStem(stemPath, data)

	sf := stem.Schema["tipo"]
	found := false
	for _, v := range sf.Values {
		if v == "c" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected 'c' in enum values, got %v", sf.Values)
	}
}

func TestApplySchemaInferences_RequiresAgentSkipped(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "required_field", Field: "estado", RequiresAgent: true, Message: "needs agent"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied for agent-required, got %d", len(result.Applied))
	}
	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(result.Skipped))
	}
}

func TestApplySchemaInferences_AddRequired(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "required_field", Field: "estado"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(stemPath)
	stem, _ := rules.ParseStem(stemPath, data)
	if !stem.Schema["estado"].Required {
		t.Error("expected estado to be required")
	}
}

func TestApplySchemaInferences_NoModifications(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	original := []byte("version: 2\nschema:\n  estado:\n    type: string\n")
	if err := os.WriteFile(stemPath, original, 0o644); err != nil {
		t.Fatal(err)
	}

	// Empty inferences list → no modifications.
	result, err := ApplySchemaInferences(stemPath, nil)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied, got %d", len(result.Applied))
	}
}
