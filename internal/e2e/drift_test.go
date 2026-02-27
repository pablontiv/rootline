package e2e

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

// TestDrift_UnanimousChildrenMismatch tests that drift is detected when
// all children unanimously differ from the parent.
func TestDrift_UnanimousChildrenMismatch(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, "In Progress", Completed]
    required: true
`,
		"project/README.md": "---\nestado: In Progress\n---\n# Project",
		"project/task1.md":  "---\nestado: Completed\n---\n# Task 1",
		"project/task2.md":  "---\nestado: Completed\n---\n# Task 2",
	})

	parent, children := loadParentChildren(t, root, "project")
	effective := resolveEffective(t, filepath.Join(root, "project"))

	warnings := rules.DetectDrift(parent, children, effective.Schema)
	if len(warnings) != 1 {
		t.Fatalf("got %d drift warnings, want 1", len(warnings))
	}
	if warnings[0].Field != "estado" {
		t.Errorf("field = %q, want estado", warnings[0].Field)
	}
	if warnings[0].ParentValue != "In Progress" {
		t.Errorf("parent_value = %v, want In Progress", warnings[0].ParentValue)
	}
	if warnings[0].ChildrenValue != "Completed" {
		t.Errorf("children_value = %v, want Completed", warnings[0].ChildrenValue)
	}
}

// TestDrift_MixedChildrenNoDrift tests that no drift warning is produced
// when children have mixed values (no consensus).
func TestDrift_MixedChildrenNoDrift(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, "In Progress", Completed]
    required: true
`,
		"project/README.md": "---\nestado: In Progress\n---\n# Project",
		"project/task1.md":  "---\nestado: Completed\n---\n# Task 1",
		"project/task2.md":  "---\nestado: Pending\n---\n# Task 2",
	})

	parent, children := loadParentChildren(t, root, "project")
	effective := resolveEffective(t, filepath.Join(root, "project"))

	warnings := rules.DetectDrift(parent, children, effective.Schema)
	if len(warnings) != 0 {
		t.Errorf("got %d drift warnings, want 0 (mixed children)", len(warnings))
	}
}

// TestDrift_MatchScopedFieldExcluded tests that match-scoped fields
// (like tipo with match: ["F*", "T*"]) do not produce drift warnings.
func TestDrift_MatchScopedFieldExcluded(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Completed]
  tipo:
    type: enum
    values: [feature, task]
    match: ["F*", "T*"]
`,
		"E01-epic/README.md":             "---\nestado: Pending\ntipo: feature\n---\n# Epic",
		"E01-epic/F01-feature/README.md": "---\nestado: Pending\ntipo: task\n---\n# Feature",
	})

	parent, children := loadParentChildren(t, root, "E01-epic")
	effective := resolveEffective(t, filepath.Join(root, "E01-epic"))

	warnings := rules.DetectDrift(parent, children, effective.Schema)
	for _, w := range warnings {
		if w.Field == "tipo" {
			t.Error("tipo should not produce drift warning (match-scoped field)")
		}
	}
}

// TestDrift_BatchResultIncludesDrift tests that the batch validation result
// includes drift warnings in the JSON output.
func TestDrift_BatchResultIncludesDrift(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Completed]
    required: true
`,
		"project/README.md": "---\nestado: Pending\n---\n# Project",
		"project/task1.md":  "---\nestado: Completed\n---\n# Task 1",
		"project/task2.md":  "---\nestado: Completed\n---\n# Task 2",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// Build validation results
	var results []*rules.ValidationResult
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			t.Fatalf("ResolveForRecord(%s): %v", rec.Path, resolveErr)
		}
		errs := rules.Validate(ctx, rec, effective)
		results = append(results, rules.NewValidationResult(rec.Path, errs))
	}

	// Detect drift
	parent, children := loadParentChildren(t, root, "project")
	effective := resolveEffective(t, filepath.Join(root, "project"))
	driftWarnings := rules.DetectDrift(parent, children, effective.Schema)

	batch := rules.NewBatchValidationResultWithDrift(results, driftWarnings)

	if len(batch.DriftWarnings) != 1 {
		t.Fatalf("batch drift_warnings len = %d, want 1", len(batch.DriftWarnings))
	}
	if batch.Summary.DriftWarningsCount != 1 {
		t.Errorf("summary drift_warnings_count = %d, want 1", batch.Summary.DriftWarningsCount)
	}

	// Verify JSON output
	jsonBytes, err := batch.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	jsonStr := string(jsonBytes)
	if !containsSubstring(jsonStr, "drift_warnings") {
		t.Error("JSON output should contain drift_warnings key")
	}
	if !containsSubstring(jsonStr, "estado") {
		t.Error("JSON output should contain drifted field name")
	}
}

// TestDrift_NoDriftWhenValuesMatch tests that no warnings are produced
// when parent and children agree.
func TestDrift_NoDriftWhenValuesMatch(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Completed]
    required: true
`,
		"project/README.md": "---\nestado: Completed\n---\n# Project",
		"project/task1.md":  "---\nestado: Completed\n---\n# Done 1",
		"project/task2.md":  "---\nestado: Completed\n---\n# Done 2",
	})

	parent, children := loadParentChildren(t, root, "project")
	effective := resolveEffective(t, filepath.Join(root, "project"))

	warnings := rules.DetectDrift(parent, children, effective.Schema)
	if len(warnings) != 0 {
		t.Errorf("got %d drift warnings, want 0 (values match)", len(warnings))
	}
}

// loadParentChildren scans a subdirectory and returns the index file as parent
// and the rest as children.
func loadParentChildren(t *testing.T, root, subdir string) (extract.Record, []extract.Record) {
	t.Helper()
	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	var parent *extract.Record
	var children []extract.Record
	for _, rec := range records {
		recDir := filepath.Dir(rec.Path)
		if recDir != subdir {
			continue
		}
		if filepath.Base(rec.Path) == "README.md" {
			r := *rec
			parent = &r
		} else {
			children = append(children, *rec)
		}
	}
	if parent == nil {
		t.Fatalf("no parent found in %s", subdir)
	}
	return *parent, children
}

// resolveEffective resolves the effective stem for a directory.
func resolveEffective(t *testing.T, dir string) *rules.StemFile {
	t.Helper()
	entries, err := rules.WalkUp(dir)
	if err != nil {
		t.Fatalf("WalkUp(%s): %v", dir, err)
	}
	return rules.MergeStemFiles(entries)
}

func containsSubstring(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && findSubstring(s, sub))
}

func findSubstring(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
