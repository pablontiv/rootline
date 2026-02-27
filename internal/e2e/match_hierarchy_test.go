package e2e

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

// TestMatchHierarchy_PerLevelFieldFiltering tests that v2 match-scoped
// fields are correctly filtered per directory level.
func TestMatchHierarchy_PerLevelFieldFiltering(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Completed]
    required: true
  tipo:
    type: enum
    values: [modulo-sistema, test, documentation]
    match: ["F*", "T*"]
    required:
      match: ["T*"]
`,
		"E01-epic/README.md":                           "---\nestado: Pending\n---\n# Epic",
		"E01-epic/F01-feature/README.md":               "---\nestado: Pending\ntipo: modulo-sistema\n---\n# Feature",
		"E01-epic/F01-feature/S001-story/README.md":    "---\nestado: Pending\n---\n# Story",
		"E01-epic/F01-feature/S001-story/T001-task.md": "---\nestado: Pending\ntipo: modulo-sistema\n---\n# Task",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			t.Fatalf("ResolveForRecord(%s): %v", rec.Path, resolveErr)
		}

		errs := rules.Validate(ctx, rec, effective)
		if len(errs) != 0 {
			t.Errorf("record %s should pass validation, got: %v", rec.Path, errs)
		}
	}
}

// TestMatchHierarchy_MissingRequiredAtTaskLevel tests that a missing
// match-scoped required field is reported at T-level but not F-level.
func TestMatchHierarchy_MissingRequiredAtTaskLevel(t *testing.T) {
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
    values: [modulo-sistema, test]
    match: ["F*", "T*"]
    required:
      match: ["T*"]
`,
		"E01-epic/F01-feature/README.md":               "---\nestado: Pending\n---\n# Feature (no tipo — OK)",
		"E01-epic/F01-feature/S001-story/T001-task.md": "---\nestado: Pending\n---\n# Task (no tipo — ERROR)",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			t.Fatalf("ResolveForRecord(%s): %v", rec.Path, resolveErr)
		}

		errs := rules.Validate(ctx, rec, effective)

		switch {
		case filepath.Base(filepath.Dir(rec.Path)) == "S001-story":
			// T001-task.md is in S001-story dir, has T* in filename
			if len(errs) != 1 {
				t.Errorf("T-level %s: expected 1 error (tipo required), got %d: %v", rec.Path, len(errs), errs)
			} else if errs[0].Field != "tipo" {
				t.Errorf("T-level %s: expected error on 'tipo', got %q", rec.Path, errs[0].Field)
			}
		default:
			// F-level README — tipo is optional
			if len(errs) != 0 {
				t.Errorf("F-level %s: expected 0 errors, got %d: %v", rec.Path, len(errs), errs)
			}
		}
	}
}

// TestMatchHierarchy_SequenceIDPerPattern tests that v2 map-form match
// on id field resolves correct prefix/digits per directory level.
func TestMatchHierarchy_SequenceIDPerPattern(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
scope:
  match: "*.md"
schema:
  id:
    type: sequence
    match:
      "E*": {prefix: E, digits: 2}
      "F*": {prefix: F, digits: 2}
      "S*": {prefix: S, digits: 3}
      "T*": {prefix: T, digits: 3}
`,
		"E01-epic/README.md":                           "---\nid: E01\n---\n",
		"E01-epic/F01-feature/README.md":               "---\nid: F01\n---\n",
		"E01-epic/F01-feature/S001-story/README.md":    "---\nid: S001\n---\n",
		"E01-epic/F01-feature/S001-story/T001-task.md": "---\nid: T001\n---\n",
	})

	tests := []struct {
		dir        string
		recordPath string
		wantPrefix string
		wantDigits int
	}{
		{"E01-epic", "E01-epic/README.md", "E", 2},
		{"E01-epic/F01-feature", "E01-epic/F01-feature/README.md", "F", 2},
		{"E01-epic/F01-feature/S001-story", "E01-epic/F01-feature/S001-story/README.md", "S", 3},
		{"E01-epic/F01-feature/S001-story", "E01-epic/F01-feature/S001-story/T001-task.md", "T", 3},
	}

	for _, tt := range tests {
		t.Run(tt.recordPath, func(t *testing.T) {
			dir := filepath.Join(root, tt.dir)
			effective, err := rules.ResolveForRecord(dir, tt.recordPath)
			if err != nil {
				t.Fatalf("ResolveForRecord: %v", err)
			}

			id, ok := effective.Schema["id"]
			if !ok {
				t.Fatal("expected id in schema")
			}
			if id.Prefix != tt.wantPrefix {
				t.Errorf("prefix = %q, want %q", id.Prefix, tt.wantPrefix)
			}
			if id.Digits != tt.wantDigits {
				t.Errorf("digits = %d, want %d", id.Digits, tt.wantDigits)
			}
		})
	}
}

// TestMatchHierarchy_DescribeOutput tests that describe produces correct
// schema for different levels with v2 stems.
func TestMatchHierarchy_DescribeOutput(t *testing.T) {
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
    values: [modulo-sistema, test]
    match: ["F*", "T*"]
`,
		"E01-epic/README.md":             "---\nestado: Pending\n---\n",
		"E01-epic/F01-feature/README.md": "---\nestado: Pending\ntipo: modulo-sistema\n---\n",
	})

	// E-level: should have estado but not tipo
	epicDir := filepath.Join(root, "E01-epic")
	effective, err := rules.ResolveForRecord(epicDir, "E01-epic/README.md")
	if err != nil {
		t.Fatalf("ResolveForRecord: %v", err)
	}
	if _, ok := effective.Schema["tipo"]; ok {
		t.Error("E-level describe should not include tipo (match restricts to F* and T*)")
	}
	if _, ok := effective.Schema["estado"]; !ok {
		t.Error("E-level describe should include estado (global field)")
	}

	// F-level: should have both estado and tipo
	featureDir := filepath.Join(root, "E01-epic", "F01-feature")
	effective, err = rules.ResolveForRecord(featureDir, "E01-epic/F01-feature/README.md")
	if err != nil {
		t.Fatalf("ResolveForRecord: %v", err)
	}
	if _, ok := effective.Schema["tipo"]; !ok {
		t.Error("F-level describe should include tipo")
	}
	if _, ok := effective.Schema["estado"]; !ok {
		t.Error("F-level describe should include estado")
	}
}

// TestMatchHierarchy_ValidateConditionalMatch tests that validate rules
// with match conditions work through the full pipeline.
func TestMatchHierarchy_ValidateConditionalMatch(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
scope:
  match: "*.md"
schema:
  tipo:
    type: enum
    values: [modulo-sistema, test]
    match: ["T*"]
validate:
  - rule: requires
    if: {match: "T*", tipo: modulo-sistema}
    then: {fields: [ejecutable_en]}
`,
		"E01-epic/F01-feature/S001-story/T001-task.md": "---\ntipo: modulo-sistema\n---\n# Missing ejecutable_en",
		"E01-epic/F01-feature/README.md":               "---\ntipo: modulo-sistema\n---\n# Feature OK",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			t.Fatalf("ResolveForRecord(%s): %v", rec.Path, resolveErr)
		}

		errs := rules.Validate(ctx, rec, effective)

		if filepath.Base(rec.Path) == "T001-task.md" {
			// Should have error: ejecutable_en required when tipo=modulo-sistema at T-level
			if len(errs) != 1 {
				t.Errorf("T001: expected 1 error, got %d: %v", len(errs), errs)
			}
		} else {
			// Feature-level: match "T*" doesn't fire, no error
			if len(errs) != 0 {
				t.Errorf("Feature: expected 0 errors, got %d: %v", len(errs), errs)
			}
		}
	}
}
