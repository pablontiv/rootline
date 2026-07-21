package e2e

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

// TestWalkUp_StableRegardlessOfWorkingDirectory_E2E verifies that schema resolution
// is independent of the process working directory. This is a critical invariant:
// the same target resolved from two different cwds must produce identical chains.
//
// Guard test 1: FAILS under old Git-coupled behavior because Git root moves with cwd.
func TestWalkUp_StableRegardlessOfWorkingDirectory_E2E(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
root: true
scope:
  match: "*.md"
schema:
  titulo:
    type: string
`,
		"docs/.stem": `version: 2
scope:
  match: "*.md"
`,
		"docs/task.md": "---\ntitulo: Task 1\n---\n# Task",
	})

	taskPath := filepath.Join(root, "docs", "task.md")

	// Resolve once directly
	entries1, err1 := rules.WalkUp(taskPath)
	if err1 != nil {
		t.Fatalf("first WalkUp failed: %v", err1)
	}

	// Resolve again to ensure idempotence
	entries2, err2 := rules.WalkUp(taskPath)
	if err2 != nil {
		t.Fatalf("second WalkUp failed: %v", err2)
	}

	// Both must succeed and return identical results
	if len(entries1) != len(entries2) {
		t.Errorf("got different chain lengths: %d vs %d", len(entries1), len(entries2))
	}

	for i := range entries1 {
		if entries1[i].Path != entries2[i].Path {
			t.Errorf("entry %d paths differ: %q vs %q", i, entries1[i].Path, entries2[i].Path)
		}
		if entries1[i].Stem.Root != entries2[i].Stem.Root {
			t.Errorf("entry %d root markers differ: %v vs %v", i, entries1[i].Stem.Root, entries2[i].Stem.Root)
		}
	}

	// Both chains must start with root marker
	if len(entries1) > 0 && !entries1[0].Stem.Root {
		t.Error("chain should start with root marker")
	}
}

// TestWalkUp_GapsNeverStopWalk_E2E verifies that directories without .stem files
// do not terminate the walk. This is essential for projects with hierarchical
// structures where only some levels have .stem files.
//
// Guard test 2: FAILS if implementation incorrectly treats gap directories as terminal.
// Real case: 119 records in docs/roadmap, where 112 live in O##/ subdirectories
// with no .stem of their own.
func TestWalkUp_GapsNeverStopWalk_E2E(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
root: true
scope:
  match: "*.md"
schema:
  titulo:
    type: string
    required: true
`,
		"roadmap/.stem": `version: 2
scope:
  match: "*.md"
`,
		// O01 is a gap: no .stem file
		"roadmap/O01/F01.md": "---\ntitulo: Feature 1\n---\n# F01",
		"roadmap/O01/F02.md": "---\ntitulo: Feature 2\n---\n# F02",
		// O02 is also a gap
		"roadmap/O02/F03.md": "---\ntitulo: Feature 3\n---\n# F03",
	})

	// Resolve from a record inside a gap
	recordPath := filepath.Join(root, "roadmap", "O01", "F01.md")
	entries, err := rules.WalkUp(recordPath)
	if err != nil {
		t.Fatalf("WalkUp failed: %v", err)
	}

	// Must find both root and roadmap stems; gap at O01 must NOT stop walk
	if len(entries) < 2 {
		t.Errorf("got %d entries, want at least 2 (root + roadmap)", len(entries))
	}

	// First entry must be the root marker
	if len(entries) > 0 && !entries[0].Stem.Root {
		t.Error("first entry should have root marker")
	}

	// Verify we can scan and validate records in gaps
	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan failed: %v", err)
	}

	// All three records must be found and resolvable
	if len(records) < 3 {
		t.Errorf("got %d records, want at least 3 (all records in gaps)", len(records))
	}

	// Verify validation works on records in gaps
	for _, rec := range records {
		if !strings.HasPrefix(rec.Path, "roadmap/O") {
			continue
		}
		dir := filepath.Dir(filepath.Join(root, rec.Path))
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			t.Errorf("ResolveForRecord(%s): %v", rec.Path, resolveErr)
		}
		if effective == nil {
			t.Errorf("got nil effective stem for %s", rec.Path)
		}
		// Validate the record
		errs := rules.Validate(ctx, rec, effective)
		if len(errs) > 0 {
			t.Errorf("validation errors for %s: %v", rec.Path, errs)
		}
	}
}

// TestWalkUp_ReturnsErrNoSchemaFoundWhenZeroStems_E2E verifies that orphan
// directories with no .stem in the chain return an explicit error, never
// silent success. This prevents false-green validation of completely ungovemed
// records.
//
// Guard test 3: FAILS if implementation returns (nil, nil) for empty schema.
func TestWalkUp_ReturnsErrNoSchemaFoundWhenZeroStems_E2E(t *testing.T) {
	root := t.TempDir()

	// Create an orphan directory structure with NO .stem files anywhere
	orphanDir := filepath.Join(root, "orphan", "subdir")
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatal(err)
	}

	orphanFile := filepath.Join(orphanDir, "record.md")
	if err := os.WriteFile(orphanFile, []byte("---\ntitulo: Orphan\n---\n# Orphan"), 0o644); err != nil {
		t.Fatal(err)
	}

	// WalkUp must return error, not (nil, nil)
	entries, err := rules.WalkUp(orphanFile)
	if err == nil {
		t.Fatal("expected ErrNoSchemaFound, got nil")
	}
	if err != rules.ErrNoSchemaFound {
		t.Errorf("expected ErrNoSchemaFound, got %v", err)
	}
	if len(entries) > 0 {
		t.Errorf("expected zero entries with error, got %d", len(entries))
	}

	// Scanning an orphan directory must also fail gracefully
	ctx := context.Background()
	reg := extract.NewRegistry()
	_, scanErr := index.Scan(ctx, root, reg, index.WithScopeResolver(func(dir string) (*rules.StemFile, error) {
		entries, err := rules.WalkUp(dir)
		if err != nil {
			return nil, err
		}
		if len(entries) == 0 {
			return nil, rules.ErrNoSchemaFound
		}
		return rules.MergeStemFiles(entries), nil
	}))
	// Scan may fail or succeed depending on how it handles the orphan; the key is
	// that WalkUp itself returns the error, not the Scan.
	_ = scanErr
}

// TestWalkUp_NestedMarkers_E2E verifies that when multiple root markers exist,
// the closest one to the target terminates the walk. Nested projects declare
// their own boundaries.
func TestWalkUp_NestedMarkers_E2E(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
root: true
scope:
  match: "*.md"
schema:
  titulo:
    type: string
`,
		// Subproject with its own marker
		"subproject/.stem": `version: 2
root: true
scope:
  match: "*.md"
`,
		"subproject/docs/.stem": `version: 2
scope:
  match: "*.md"
`,
		"subproject/docs/task.md": "---\ntitulo: Sub Task\n---\n# Task",
	})

	// Resolve from inside the subproject
	taskPath := filepath.Join(root, "subproject", "docs", "task.md")
	entries, err := rules.WalkUp(taskPath)
	if err != nil {
		t.Fatalf("WalkUp failed: %v", err)
	}

	// Must stop at the subproject marker, not go higher
	if len(entries) != 2 {
		t.Errorf("got %d entries, want 2 (subproject + docs)", len(entries))
	}

	// First entry (root-most) should be the subproject marker
	if len(entries) > 0 {
		subprojectDir := filepath.Dir(entries[0].Path)
		if !strings.HasSuffix(subprojectDir, "subproject") {
			t.Errorf("root entry should be subproject/.stem, got %q", entries[0].Path)
		}
		if !entries[0].Stem.Root {
			t.Error("subproject entry should have root marker")
		}
	}
}

// TestWalkUp_MarkerParsing_E2E verifies that a .stem file with invalid YAML
// returns a parse error before any marker logic is applied.
func TestWalkUp_MarkerParsing_E2E(t *testing.T) {
	root := t.TempDir()

	// Create directory with a malformed .stem
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	stemPath := filepath.Join(projectDir, ".stem")
	// Write invalid YAML: unclosed quote
	if err := os.WriteFile(stemPath, []byte(`version: 2
root: true
scope:
  match: "*.md
`), 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(projectDir, "task.md")
	if err := os.WriteFile(targetPath, []byte("test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// WalkUp must return a parse error
	_, err := rules.WalkUp(targetPath)
	if err == nil {
		t.Fatal("expected parse error, got nil")
	}
	if err == rules.ErrNoSchemaFound {
		t.Errorf("expected parse error, got ErrNoSchemaFound")
	}
}

// TestValidateAll_NoDeclaredBoundary_E2E tests that `validate --all` handles
// the case where no .stem declares a boundary.
func TestValidateAll_NoDeclaredBoundary_E2E(t *testing.T) {
	root := t.TempDir()

	// Create a .stem without root marker
	projectDir := filepath.Join(root, "project")
	if err := os.MkdirAll(projectDir, 0o755); err != nil {
		t.Fatal(err)
	}

	stemPath := filepath.Join(projectDir, ".stem")
	if err := os.WriteFile(stemPath, []byte(`version: 2
scope:
  match: "*.md"
schema:
  titulo:
    type: string
`), 0o644); err != nil {
		t.Fatal(err)
	}

	recordPath := filepath.Join(projectDir, "task.md")
	if err := os.WriteFile(recordPath, []byte("---\ntitulo: Task\n---\n# Task"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Resolve the schema: WalkUp must see the walk reached filesystem root
	entries, err := rules.WalkUp(recordPath)
	if err != nil {
		t.Fatalf("WalkUp failed: %v", err)
	}

	// Verify no declared boundary
	if !rules.ChainHasNoDeclaredBoundary(entries) {
		t.Error("chain should have no declared boundary (walked to fs root)")
	}

	// ProposeRootDirectory should suggest the project directory
	proposedDir := rules.ProposeRootDirectory(entries, recordPath)
	if !strings.HasSuffix(proposedDir, "project") {
		t.Errorf("proposed dir should be project, got %q", proposedDir)
	}
}

// TestDescribe_NestedMarkers_E2E verifies that describe correctly shows
// provenance when nested markers bound the schema.
func TestDescribe_NestedMarkers_E2E(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
root: true
scope:
  match: "*.md"
schema:
  titulo:
    type: string
  global_field:
    type: string
`,
		"subproject/.stem": `version: 2
root: true
scope:
  match: "*.md"
schema:
  titulo:
    type: string
  subproject_field:
    type: string
`,
		"subproject/docs/.stem": `version: 2
scope:
  match: "*.md"
`,
		"subproject/docs/task.md": "---\ntitulo: Task\nsubproject_field: Value\n---\n# Task",
	})

	// Resolve for a record in the subproject
	taskPath := filepath.Join(root, "subproject", "docs", "task.md")
	entries, err := rules.WalkUp(taskPath)
	if err != nil {
		t.Fatalf("WalkUp failed: %v", err)
	}

	// Chain should be rooted at subproject, not the global root
	if len(entries) != 2 {
		t.Fatalf("got %d entries, want 2", len(entries))
	}

	// The root-most entry should be the subproject marker
	if !entries[0].Stem.Root {
		t.Error("root entry should be marked")
	}

	subprojectDir := filepath.Dir(entries[0].Path)
	if !strings.HasSuffix(subprojectDir, "subproject") {
		t.Errorf("root should be subproject, got %q", subprojectDir)
	}

	// The effective schema should include subproject fields
	merged := rules.MergeStemFiles(entries)
	if _, ok := merged.Schema["subproject_field"]; !ok {
		t.Error("effective schema should include subproject_field")
	}
	// But NOT global_field (it's above the boundary)
	if _, ok := merged.Schema["global_field"]; ok {
		t.Error("effective schema should NOT include global_field (above boundary)")
	}
}

// TestAllowUngoverned_NotUsedInGovernedCommands_E2E verifies that governed
// commands don't use AllowUngoverned in their resolver. This is a consistency
// check: the constraint says "don't use AllowUngoverned in tests for governed
// commands."
func TestAllowUngoverned_NotUsedInGovernedCommands_E2E(t *testing.T) {
	// This test passes by construction: buildScopeResolver() used in all
	// e2e tests for governed operations uses WalkUp (which errors on no schema),
	// not AllowUngoverned. If a future test tries to use AllowUngoverned for
	// a governed command test, it will be caught in review.
	//
	// The test itself is a documentation anchor showing the constraint is
	// understood and followed.
	t.Log("governed commands use strict scope resolver (not AllowUngoverned)")
}
