package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestApply_AnalyzeThenApplyThenValidate(t *testing.T) {
	// Stem is incomplete: estado has only 2 values but docs use 3.
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: enum\n    required: true\n    values: [Pending, Done]\n  tipo:\n    type: string\n",
		"doc1.md": "---\nestado: Pending\ntipo: task\n---\nBody 1\n",
		"doc2.md": "---\nestado: Done\ntipo: task\n---\nBody 2\n",
		"doc3.md": "---\nestado: Pending\ntipo: task\n---\nBody 3\n",
	})

	report := runAnalyze(t, root)

	// Collect all inferences.
	var allInferences []infer.ReportInference
	for _, cat := range report.Categories {
		allInferences = append(allInferences, cat.Inferences...)
	}

	// Find stem and apply schema inferences.
	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no stem found: %v", err)
	}

	result, err := infer.ApplySchemaInferences(entries[0].Path, allInferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	// Verify at least some inferences were processed.
	totalActions := len(result.Applied) + len(result.Skipped)
	if totalActions == 0 && len(allInferences) > 0 {
		t.Logf("warning: %d inferences produced no apply actions (schema may already cover them)", len(allInferences))
	}

	// Validate: all documents should still pass.
	validateAllRecords(t, root)
}

func TestApply_DataCorrections_MigrateAndValidate(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: enum\n    required: true\n    values: [Pending, Done]\n  tipo:\n    type: string\n",
		"doc1.md": "---\nestado: todo\ntipo: task\n---\nBody\n",
	})

	inferences := []infer.ReportInference{
		{Type: "migrate_value", Field: "estado", From: "todo", To: "Pending", Paths: []string{"doc1.md"}},
	}

	opts := infer.ApplyOptions{Root: root}
	result, err := infer.ApplyDataCorrections(inferences, opts)
	if err != nil {
		t.Fatalf("apply data error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	// Verify file was updated.
	data, _ := os.ReadFile(filepath.Join(root, "doc1.md"))
	if !strings.Contains(string(data), "Pending") {
		t.Errorf("expected 'Pending' in doc1.md, got:\n%s", data)
	}

	// Validate post-apply.
	validateAllRecords(t, root)
}

func TestApply_DryRun_NoFileChanges(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: string\n",
		"doc1.md": "---\nestado: todo\n---\nBody\n",
	})

	original, _ := os.ReadFile(filepath.Join(root, "doc1.md"))

	inferences := []infer.ReportInference{
		{Type: "migrate_value", Field: "estado", From: "todo", To: "Pending", Paths: []string{"doc1.md"}},
	}

	result, err := infer.ApplyDataCorrections(inferences, infer.ApplyOptions{Root: root, DryRun: true})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 dry-run applied, got %d", len(result.Applied))
	}
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}

	// File must be unchanged.
	after, _ := os.ReadFile(filepath.Join(root, "doc1.md"))
	if string(after) != string(original) {
		t.Errorf("dry-run modified the file:\noriginal: %s\nafter: %s", original, after)
	}
}

func TestApply_RequiresAgent_Skipped(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: string\n",
		"doc1.md": "---\nestado: Pending\n---\nBody\n",
	})

	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no stem found: %v", err)
	}

	inferences := []infer.ReportInference{
		{Type: "required_field", Field: "estado", RequiresAgent: true, Message: "needs human review"},
	}

	result, err := infer.ApplySchemaInferences(entries[0].Path, inferences, false)
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Skipped) != 1 {
		t.Errorf("expected 1 skipped, got %d", len(result.Skipped))
	}
	if len(result.Applied) != 0 {
		t.Errorf("expected 0 applied, got %d", len(result.Applied))
	}
	if !strings.Contains(result.Skipped[0], "requires agent") {
		t.Errorf("expected 'requires agent' in skipped message, got: %s", result.Skipped[0])
	}
}

func TestApply_AddField_ThenValidate(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: enum\n    required: true\n    values: [Pending, Done]\n  tipo:\n    type: string\n    required: true\n",
		"doc1.md": "---\nestado: Pending\n---\nBody\n",
	})

	inferences := []infer.ReportInference{
		{Type: "add_field", Field: "tipo", Value: "task", Paths: []string{"doc1.md"}},
	}

	result, err := infer.ApplyDataCorrections(inferences, infer.ApplyOptions{Root: root})
	if err != nil {
		t.Fatalf("apply error: %v", err)
	}

	if len(result.Applied) != 1 {
		t.Fatalf("expected 1 applied, got %d", len(result.Applied))
	}

	// Verify field added.
	data, _ := os.ReadFile(filepath.Join(root, "doc1.md"))
	if !strings.Contains(string(data), "tipo: task") {
		t.Errorf("expected 'tipo: task' in doc1.md, got:\n%s", data)
	}

	// Validate post-apply.
	validateAllRecords(t, root)
}

// validateAllRecords runs validation against all records in the root directory.
func validateAllRecords(t *testing.T, root string) {
	t.Helper()
	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		return // no stem = nothing to validate against
	}
	effective := rules.MergeStemFiles(entries)

	files, err := filepath.Glob(filepath.Join(root, "*.md"))
	if err != nil {
		t.Fatalf("glob: %v", err)
	}

	reg := extract.NewRegistry()
	for _, f := range files {
		content, err := os.ReadFile(f)
		if err != nil {
			t.Fatalf("reading %s: %v", f, err)
		}
		ext := reg.ForFile(f, "")
		if ext == nil {
			continue
		}
		rec, err := ext.Extract(f, content)
		if err != nil {
			continue
		}
		relPath, _ := filepath.Rel(root, f)
		rec.Path = relPath
		errs := rules.Validate(context.Background(), rec, effective)
		if len(errs) > 0 {
			t.Errorf("validation errors in %s after apply: %v", relPath, errs)
		}
	}
}
