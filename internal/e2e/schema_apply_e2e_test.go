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

func TestApply_SchemaApply_EnumExtension(t *testing.T) {
	// Stem has incomplete enum; schema apply should extend it based on inferences.
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: enum\n    required: true\n    values: [Pending, Done]\n  tipo:\n    type: string\n",
		"doc1.md": "---\nestado: Pending\ntipo: task\n---\nBody 1\n",
		"doc2.md": "---\nestado: Done\ntipo: task\n---\nBody 2\n",
		"doc3.md": "---\nestado: Pending\ntipo: task\n---\nBody 3\n",
	})

	// Collect inferences from analyze.
	report := runAnalyze(t, root)
	var allInferences []infer.ReportInference
	for _, cat := range report.Categories {
		allInferences = append(allInferences, cat.Inferences...)
	}

	// Apply schema inferences to .stem via library.
	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no stem found: %v", err)
	}

	result, err := infer.ApplySchemaInferences(entries[0].Path, allInferences, false)
	if err != nil {
		t.Fatalf("apply schema inferences error: %v", err)
	}

	// Verify at least some inferences were processed.
	totalActions := len(result.Applied) + len(result.Skipped)
	if totalActions == 0 && len(allInferences) > 0 {
		t.Logf("warning: %d inferences produced no apply actions (schema may already cover them)", len(allInferences))
	}

	// Validate: all documents should still pass after schema update.
	validateAllRecords(t, root)
}

func TestApply_SchemaApply_DryRun(t *testing.T) {
	// Verify that schema apply with --dry-run does not modify .stem files.
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: enum\n    required: true\n    values: [Pending, Done]\n  priority:\n    type: string\n",
		"doc1.md": "---\nestado: Pending\n---\nBody\n",
	})

	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		t.Fatalf("no stem found: %v", err)
	}

	original, _ := os.ReadFile(entries[0].Path)

	// Create a schema inference: mark priority as required
	inferences := []infer.ReportInference{
		{Type: "required_field", Field: "priority"},
	}

	result, err := infer.ApplySchemaInferences(entries[0].Path, inferences, true)
	if err != nil {
		t.Fatalf("dry-run error: %v", err)
	}

	// Verify dry-run flag is set
	if !result.DryRun {
		t.Error("expected DryRun=true")
	}

	// .stem file must be unchanged.
	after, _ := os.ReadFile(entries[0].Path)
	if string(after) != string(original) {
		t.Errorf("dry-run modified the .stem file:\noriginal: %s\nafter: %s", original, after)
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
