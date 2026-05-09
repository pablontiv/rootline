package infer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
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

	result, err := ApplySchemaInferences(stemPath, inferences, false)
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

	result, err := ApplySchemaInferences(stemPath, inferences, false)
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

	result, err := ApplySchemaInferences(stemPath, inferences, false)
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
	result, err := ApplySchemaInferences(stemPath, nil, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied, got %d", len(result.Applied))
	}
}

func TestApplyDataCorrections_MigrateValue(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	if err := os.WriteFile(docPath, []byte("---\nestado: todo\ntipo: task\n---\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "migrate_value", Field: "estado", From: "todo", To: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(docPath)
	if !strings.Contains(string(data), "Pending") {
		t.Errorf("expected 'Pending' in file, got:\n%s", data)
	}
	if strings.Contains(string(data), "todo") {
		t.Errorf("expected 'todo' to be replaced, got:\n%s", data)
	}
}

func TestApplyDataCorrections_CorrectValue(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	if err := os.WriteFile(docPath, []byte("---\nestado: Pendng\n---\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "correct_value", Field: "estado", From: "Pendng", To: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(docPath)
	if !strings.Contains(string(data), "Pending") {
		t.Errorf("expected 'Pending' in file, got:\n%s", data)
	}
}

func TestApplyDataCorrections_AddField(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	if err := os.WriteFile(docPath, []byte("---\ntipo: task\n---\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "add_field", Field: "estado", Value: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(docPath)
	if !strings.Contains(string(data), "estado: Pending") {
		t.Errorf("expected 'estado: Pending' in file, got:\n%s", data)
	}
}

func TestApplyDataCorrections_AddField_SkipsExisting(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	if err := os.WriteFile(docPath, []byte("---\nestado: Done\ntipo: task\n---\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "add_field", Field: "estado", Value: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied (field exists), got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(docPath)
	if !strings.Contains(string(data), "Done") {
		t.Errorf("expected original 'Done' preserved, got:\n%s", data)
	}
}

func TestApplyDataCorrections_DryRun(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	original := "---\nestado: todo\n---\nBody\n"
	if err := os.WriteFile(docPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "migrate_value", Field: "estado", From: "todo", To: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir, DryRun: true})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied in dry-run, got %d", len(result.Applied))
	}
	if !result.DryRun {
		t.Error("expected DryRun flag to be true")
	}

	// File should NOT be modified.
	data, _ := os.ReadFile(docPath)
	if string(data) != original {
		t.Errorf("dry-run should not modify file, got:\n%s", data)
	}
}

func TestApplyDataCorrections_RequiresAgentSkipped(t *testing.T) {
	inferences := []ReportInference{
		{Type: "migrate_value", Field: "estado", RequiresAgent: true, Message: "needs agent"},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied, got %d", len(result.Applied))
	}
	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(result.Skipped))
	}
}

func TestApplySchemaInferences_AddDefault(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "constant_field", Field: "estado", Value: "Pending"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(data), "default: Pending") {
		t.Errorf("expected 'default: Pending' in stem, got:\n%s", data)
	}
}

func TestApplySchemaInferences_SetType(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  count:\n    required: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "field_type", Field: "count", Value: "integer"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(data), "type: integer") {
		t.Errorf("expected 'type: integer' in stem, got:\n%s", data)
	}
}

func TestRewriteFrontmatter_NoPrior(t *testing.T) {
	content := "Just body text"
	fm := map[string]any{"estado": "Pending"}
	result := rewriteFrontmatter(content, fm)
	if !strings.HasPrefix(result, "---\n") {
		t.Errorf("expected frontmatter prefix, got:\n%s", result)
	}
	if !strings.Contains(result, "estado: Pending") {
		t.Errorf("expected field in output, got:\n%s", result)
	}
	if !strings.Contains(result, "Just body text") {
		t.Errorf("expected body preserved, got:\n%s", result)
	}
}

func TestApplyDataCorrections_MigrateValueMismatch(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	if err := os.WriteFile(docPath, []byte("---\nestado: Done\n---\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "migrate_value", Field: "estado", From: "todo", To: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied (value mismatch), got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_EnumNoValuesKey(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	// Field "tipo" exists but has no "values" key in YAML.
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: enum\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "enum_values", Field: "tipo", Value: "[a b]"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	// The YAML node has "tipo" with "type: enum" but no "values" key,
	// so applyEnumExtensionNode can't find the sequence to extend.
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied (no values key in YAML), got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_UntypedField(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  mystery:\n    required: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ApplySchemaInferences(stemPath, []ReportInference{
		{Type: "untyped_field", Field: "mystery", Value: "string"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) == 0 {
		t.Fatal("expected applied changes")
	}

	data, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(data), "type: string") {
		t.Errorf("expected type: string in stem, got:\n%s", data)
	}
}

func TestApplySchemaInferences_SequenceIncomplete(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  id:\n    type: sequence\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ApplySchemaInferences(stemPath, []ReportInference{
		{Type: "sequence_incomplete", Field: "id", Value: "T:3"},
	}, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(result.Applied) == 0 {
		t.Fatal("expected applied changes")
	}

	data, _ := os.ReadFile(stemPath)
	content := string(data)
	if !strings.Contains(content, "prefix:") || !strings.Contains(content, "digits:") {
		t.Errorf("expected prefix and digits in stem, got:\n%s", content)
	}
}

func TestApplySchemaInferences_EnumNoField(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// enum_values for a field not in the schema → skipped.
	inferences := []ReportInference{
		{Type: "enum_values", Field: "nonexistent", Value: "[a b]"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied for nonexistent field, got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_EnumEmptyValue(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: enum\n    values: [a]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Empty value list → no changes.
	inferences := []ReportInference{
		{Type: "enum_values", Field: "tipo", Value: "[]"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied for empty value list, got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_EnumAllExisting(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: enum\n    values: [a, b]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// All values already exist → no changes.
	inferences := []ReportInference{
		{Type: "enum_values", Field: "tipo", Value: "[a b]"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied when all values exist, got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_RequiredAlreadySet(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n    required: true\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Field already required → no change.
	inferences := []ReportInference{
		{Type: "required_field", Field: "estado"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied (already required), got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_DefaultAlreadySet(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n    default: Done\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Field already has default → no change.
	inferences := []ReportInference{
		{Type: "constant_field", Field: "estado", Value: "Pending"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied (default exists), got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_TypeAlreadySet(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  count:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Field already has type → no change.
	inferences := []ReportInference{
		{Type: "field_type", Field: "count", Value: "integer"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied (type already set), got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_RequiredFieldNotInSchema(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Field not in schema → cannot find node → no change.
	inferences := []ReportInference{
		{Type: "required_field", Field: "nonexistent"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied for missing field, got %d", len(result.Applied))
	}
}

func TestApplyDataCorrections_AddFieldWithToFallback(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	if err := os.WriteFile(docPath, []byte("---\ntipo: task\n---\nBody\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// add_field with empty Value, uses To as fallback.
	inferences := []ReportInference{
		{Type: "add_field", Field: "estado", Value: "", To: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(docPath)
	if !strings.Contains(string(data), "estado: Pending") {
		t.Errorf("expected 'estado: Pending' in file, got:\n%s", data)
	}
}

func TestApplyDataCorrections_AddFieldNoPaths(t *testing.T) {
	inferences := []ReportInference{
		{Type: "add_field", Field: "estado", Value: "Pending"},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied with no paths, got %d", len(result.Applied))
	}
}

func TestApplyDataCorrections_AddFieldDryRun(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	original := "---\ntipo: task\n---\nBody\n"
	if err := os.WriteFile(docPath, []byte(original), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "add_field", Field: "estado", Value: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir, DryRun: true})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied in dry-run, got %d", len(result.Applied))
	}

	// File should NOT be modified.
	data, _ := os.ReadFile(docPath)
	if string(data) != original {
		t.Errorf("dry-run should not modify file, got:\n%s", data)
	}
}

func TestApplyDataCorrections_AddFieldNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	if err := os.WriteFile(docPath, []byte("Just body text without frontmatter\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "add_field", Field: "estado", Value: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(docPath)
	if !strings.Contains(string(data), "estado: Pending") {
		t.Errorf("expected 'estado: Pending' in file, got:\n%s", data)
	}
}

func TestApplyDataCorrections_MigrateValueNoPaths(t *testing.T) {
	inferences := []ReportInference{
		{Type: "migrate_value", Field: "estado", From: "todo", To: "Pending"},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied with no paths, got %d", len(result.Applied))
	}
}

func TestApplyDataCorrections_CorrectValueNoPaths(t *testing.T) {
	inferences := []ReportInference{
		{Type: "correct_value", Field: "estado", From: "Pendng", To: "Pending"},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: t.TempDir()})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied with no paths, got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_UpdateExistingProperty(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	// Field has required: false initially, we update to true.
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  estado:\n    type: string\n    required: false\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "required_field", Field: "estado"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(data), "required: \"true\"") && !strings.Contains(string(data), "required: 'true'") && !strings.Contains(string(data), "required: true") {
		t.Errorf("expected 'required: true' in stem, got:\n%s", data)
	}
}

func TestFindSchemaFieldNode_NoSchemaKey(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("version: 2\nother: value\n"), &doc); err != nil {
		t.Fatal(err)
	}
	node := findSchemaFieldNode(&doc, "estado")
	if node != nil {
		t.Error("expected nil for document without schema key")
	}
}

func TestFindSchemaFieldNode_SchemaNotMapping(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("version: 2\nschema: not_a_mapping\n"), &doc); err != nil {
		t.Fatal(err)
	}
	node := findSchemaFieldNode(&doc, "estado")
	if node != nil {
		t.Error("expected nil for schema that is not a mapping")
	}
}

func TestFindSchemaFieldNode_FieldNotInSchema(t *testing.T) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("version: 2\nschema:\n  tipo:\n    type: string\n"), &doc); err != nil {
		t.Fatal(err)
	}
	node := findSchemaFieldNode(&doc, "nonexistent")
	if node != nil {
		t.Error("expected nil for field not in schema")
	}
}

func TestSetFieldProperty_NonMappingNode(t *testing.T) {
	node := &yaml.Node{Kind: yaml.ScalarNode, Value: "test"}
	// Should not panic, just return.
	setFieldProperty(node, "key", "value")
	if node.Value != "test" {
		t.Error("non-mapping node should not be modified")
	}
}

func TestSetFieldProperty_UpdateExisting(t *testing.T) {
	node := &yaml.Node{
		Kind: yaml.MappingNode,
		Content: []*yaml.Node{
			{Kind: yaml.ScalarNode, Value: "type"},
			{Kind: yaml.ScalarNode, Value: "string"},
		},
	}
	setFieldProperty(node, "type", "integer")
	if node.Content[1].Value != "integer" {
		t.Errorf("expected updated value 'integer', got %q", node.Content[1].Value)
	}
}

func TestApplySchemaInferences_DefaultNotInSchema(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// constant_field for field NOT in schema → should not find node → no change.
	inferences := []ReportInference{
		{Type: "constant_field", Field: "nonexistent", Value: "Pending"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied, got %d", len(result.Applied))
	}
}

func TestApplySchemaInferences_FieldTypeNotInSchema(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nschema:\n  tipo:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// field_type for field NOT in schema → should return false.
	inferences := []ReportInference{
		{Type: "field_type", Field: "nonexistent", Value: "integer"},
	}

	result, err := ApplySchemaInferences(stemPath, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied, got %d", len(result.Applied))
	}
}

func TestRewriteFrontmatter_NoClosingDelimiter(t *testing.T) {
	// Frontmatter with opening --- but no closing --- → original returned unchanged.
	content := "---\nestado: Pending\nno closing delimiter here"
	fm := map[string]any{"estado": "Done"}
	result := rewriteFrontmatter(content, fm)
	if result != content {
		t.Errorf("expected original returned for malformed frontmatter, got:\n%s", result)
	}
}

func TestFindSchemaFieldNode_RootNotMapping(t *testing.T) {
	// A YAML document with a scalar value (not a mapping).
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte("just a string"), &doc); err != nil {
		t.Fatal(err)
	}
	node := findSchemaFieldNode(&doc, "anything")
	if node != nil {
		t.Error("expected nil for non-mapping root")
	}
}

func TestApplyDataCorrections_CorrectValueFileReadError(t *testing.T) {
	inferences := []ReportInference{
		{Type: "correct_value", Field: "estado", From: "Pendng", To: "Pending", Paths: []string{"nonexistent.md"}},
	}

	_, err := ApplyDataCorrections(inferences, ApplyOptions{Root: t.TempDir()})
	if err == nil {
		t.Error("expected error for nonexistent file in correct_value")
	}
}

func TestParseFrontmatter_InvalidYAML(t *testing.T) {
	// Frontmatter with invalid YAML content.
	fm := parseFrontmatter("---\n: invalid: yaml: [unclosed\n---\nBody")
	if fm != nil {
		t.Error("expected nil for invalid YAML frontmatter")
	}
}

func TestParseValueList(t *testing.T) {
	tests := []struct {
		input string
		want  int
	}{
		{"[a b c]", 3},
		{"a b c", 3},
		{"[]", 0},
		{"", 0},
		{"[single]", 1},
	}
	for _, tt := range tests {
		got := parseValueList(tt.input)
		if len(got) != tt.want {
			t.Errorf("parseValueList(%q) = %v (len %d), want len %d", tt.input, got, len(got), tt.want)
		}
	}
}

func TestApplySchemaInferences_ReadError(t *testing.T) {
	_, err := ApplySchemaInferences("/nonexistent/.stem", nil, false)
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestApplySchemaInferences_InvalidYAML(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, []byte(":\n  bad:\n    yaml: [unclosed"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, err := ApplySchemaInferences(stemPath, nil, false)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

func TestApplyDataCorrections_CorrectValueFileWrite(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	if err := os.WriteFile(docPath, []byte("---\nestado: Pendng\ntipo: task\n---\nBody text\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	inferences := []ReportInference{
		{Type: "correct_value", Field: "estado", From: "Pendng", To: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	data, _ := os.ReadFile(docPath)
	if strings.Contains(string(data), "Pendng") {
		t.Errorf("expected 'Pendng' to be corrected, got:\n%s", data)
	}
}

func TestApplyDataCorrections_DocCorrectionNoFrontmatter(t *testing.T) {
	dir := t.TempDir()
	docPath := filepath.Join(dir, "doc1.md")
	if err := os.WriteFile(docPath, []byte("Just body text\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// migrate_value on file without frontmatter → parseFrontmatter returns nil → skip.
	inferences := []ReportInference{
		{Type: "migrate_value", Field: "estado", From: "todo", To: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := ApplyDataCorrections(inferences, ApplyOptions{Root: dir})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied (no frontmatter), got %d", len(result.Applied))
	}
}

func TestApplyDataCorrections_DocCorrectionFileReadError(t *testing.T) {
	inferences := []ReportInference{
		{Type: "migrate_value", Field: "estado", From: "todo", To: "Pending", Paths: []string{"nonexistent.md"}},
	}

	_, err := ApplyDataCorrections(inferences, ApplyOptions{Root: t.TempDir()})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestApplyDataCorrections_AddFieldFileReadError(t *testing.T) {
	inferences := []ReportInference{
		{Type: "add_field", Field: "estado", Value: "Pending", Paths: []string{"nonexistent.md"}},
	}

	_, err := ApplyDataCorrections(inferences, ApplyOptions{Root: t.TempDir()})
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestParseFrontmatter_TrailingNoNewline(t *testing.T) {
	// Test frontmatter that ends without trailing newline after ---
	fm := parseFrontmatter("---\nestado: Done\n---")
	if fm == nil {
		t.Error("expected non-nil frontmatter for trailing --- without newline")
	}
}

func TestParseFrontmatter(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    bool
	}{
		{"valid", "---\nestado: Pending\n---\nBody", true},
		{"no frontmatter", "Just text", false},
		{"empty frontmatter", "---\n---\nBody", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fm := parseFrontmatter(tt.content)
			if (fm != nil) != tt.want {
				t.Errorf("parseFrontmatter(%q) returned %v, want non-nil=%v", tt.content, fm, tt.want)
			}
		})
	}
}

func TestScaffoldSchema(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, "doc1.md"), []byte("---\ntitle: Hello\nestado: draft\n---\n# Doc\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "doc2.md"), []byte("---\ntitle: World\n---\n# Doc 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	err := ScaffoldSchema(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	stemPath := filepath.Join(dir, ".stem")
	data, readErr := os.ReadFile(stemPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	content := string(data)
	if !strings.Contains(content, "version: 2") {
		t.Error("expected version: 2")
	}
	if !strings.Contains(content, "estado:") {
		t.Error("expected estado field")
	}
	if !strings.Contains(content, "title:") {
		t.Error("expected title field")
	}
}

func TestScaffoldSchema_NoMarkdown(t *testing.T) {
	dir := t.TempDir()
	err := ScaffoldSchema(dir, false)
	if err == nil {
		t.Error("expected error when no markdown files found")
	}
}
