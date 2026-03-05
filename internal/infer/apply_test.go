package infer

import (
	"os"
	"path/filepath"
	"strings"
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

	result, err := ApplySchemaInferences(stemPath, inferences)
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

	result, err := ApplySchemaInferences(stemPath, inferences)
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
