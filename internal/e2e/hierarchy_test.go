package e2e

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

// TestHierarchyLevels tests the full pipeline with a .stem that uses levels:
// scan → extract → ResolveForRecord → validate with per-level schema.
func TestHierarchyLevels(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 1
scope:
  match: "*.md"
schema:
  titulo:
    type: string
    required: true
  estado:
    type: string
levels:
  epic:
    match: "E*"
    children: [feature]
    schema:
      objetivo:
        type: string
  feature:
    match: "F*"
    children: [story]
  story:
    match: "S*"
    children: [task]
  task:
    match: "T*"
    schema:
      ejecutable_en:
        type: string
        required: true
`,
		// Epic level
		"E01-infra/README.md": "---\ntitulo: Infrastructure\nestado: Specified\nobjetivo: Build foundation\n---\n",

		// Feature level (child of epic)
		"E01-infra/F01-core/README.md": "---\ntitulo: Core Engine\nestado: Specified\n---\n",

		// Story level (child of feature)
		"E01-infra/F01-core/S001-parsing/README.md": "---\ntitulo: Parsing\nestado: Specified\n---\n",

		// Task level (child of story) — has ejecutable_en required by levels
		"E01-infra/F01-core/S001-parsing/T001-parser.md": "---\ntitulo: Build parser\nestado: Specified\nejecutable_en: 1 sesion\n---\n",

		// Task without ejecutable_en — should fail validation
		"E01-infra/F01-core/S001-parsing/T002-lexer.md": "---\ntitulo: Build lexer\nestado: Specified\n---\n",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// Validate each record using ResolveForRecord
	type valResult struct {
		path string
		errs []rules.ValidationError
	}
	var results []valResult

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			t.Fatalf("ResolveForRecord(%s): %v", rec.Path, resolveErr)
		}
		errs := rules.Validate(ctx, rec, effective)
		results = append(results, valResult{path: rec.Path, errs: errs})
	}

	// T001 (has ejecutable_en) should pass
	for _, r := range results {
		if r.path == "E01-infra/F01-core/S001-parsing/T001-parser.md" {
			if len(r.errs) != 0 {
				t.Errorf("T001 should pass validation, got errors: %v", r.errs)
			}
		}
	}

	// T002 (missing ejecutable_en) should fail with required field error
	found := false
	for _, r := range results {
		if r.path == "E01-infra/F01-core/S001-parsing/T002-lexer.md" {
			found = true
			hasRequiredErr := false
			for _, e := range r.errs {
				if e.Field == "ejecutable_en" && e.Rule == "required" {
					hasRequiredErr = true
				}
			}
			if !hasRequiredErr {
				t.Errorf("T002 should fail with required ejecutable_en, got: %v", r.errs)
			}
		}
	}
	if !found {
		t.Error("T002-lexer.md not found in results")
	}

	// Feature level should NOT require ejecutable_en
	for _, r := range results {
		if r.path == "E01-infra/F01-core/README.md" {
			for _, e := range r.errs {
				if e.Field == "ejecutable_en" {
					t.Errorf("Feature should not require ejecutable_en, got: %v", e)
				}
			}
		}
	}

	// Epic level defines objetivo as a typed field (not required), so no error
	for _, r := range results {
		if r.path == "E01-infra/README.md" {
			if len(r.errs) != 0 {
				t.Errorf("Epic should pass validation, got errors: %v", r.errs)
			}
		}
	}
}

// TestHierarchyLevelsNesting tests that nesting violations are detected
// when a record's path violates the parent-child hierarchy defined in levels.
func TestHierarchyLevelsNesting(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 1
scope:
  match: "*.md"
schema:
  titulo:
    type: string
levels:
  epic:
    match: "E*"
    children: [feature]
  feature:
    match: "F*"
    children: [story]
  story:
    match: "S*"
    children: [task]
  task:
    match: "T*"
`,
		// Valid: task under story
		"E01-x/F01-y/S001-z/T001-ok.md": "---\ntitulo: Valid task\n---\n",

		// Invalid: task directly under epic (skipping feature and story)
		"E01-x/T001-bad.md": "---\ntitulo: Bad task\n---\n",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	// Check nesting for the invalid task (task under epic)
	for _, rec := range records {
		if rec.Path != "E01-x/T001-bad.md" {
			continue
		}
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			t.Fatalf("ResolveForRecord: %v", resolveErr)
		}

		nestingErrs := rules.CheckNesting(effective.Levels, rec.Path)
		if len(nestingErrs) == 0 {
			t.Error("expected nesting violation for task under epic, got none")
		}
		hasNestingErr := false
		for _, e := range nestingErrs {
			if e.Rule == "nesting" {
				hasNestingErr = true
			}
		}
		if !hasNestingErr {
			t.Errorf("expected nesting rule error, got: %v", nestingErrs)
		}
	}

	// Check valid task has no nesting errors
	for _, rec := range records {
		if rec.Path != "E01-x/F01-y/S001-z/T001-ok.md" {
			continue
		}
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			t.Fatalf("ResolveForRecord: %v", resolveErr)
		}

		nestingErrs := rules.CheckNesting(effective.Levels, rec.Path)
		if len(nestingErrs) != 0 {
			t.Errorf("valid task should have no nesting errors, got: %v", nestingErrs)
		}
	}
}

// TestHierarchyLevelsBackwardCompat tests that a .stem without levels
// behaves exactly the same as before — ResolveForRecord just returns
// the merged stem without any levels expansion.
func TestHierarchyLevelsBackwardCompat(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 1
scope:
  match: "*.md"
schema:
  titulo:
    type: string
    required: true
  estado:
    type: string
    enum: [Pending, Active, Done]
`,
		"docs/readme.md": "---\ntitulo: My Doc\nestado: Active\n---\n",
		"docs/notes.md":  "---\ntitulo: Notes\nestado: Pending\n---\n",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	// Both should validate cleanly
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			t.Fatalf("ResolveForRecord(%s): %v", rec.Path, resolveErr)
		}

		if effective.Levels != nil {
			t.Errorf("expected no levels for backward compat stem, got: %v", effective.Levels)
		}

		errs := rules.Validate(ctx, rec, effective)
		if len(errs) != 0 {
			t.Errorf("record %s should pass validation, got: %v", rec.Path, errs)
		}
	}
}
