package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestScaffoldMissingRequiredSections_WithDefault(t *testing.T) {
	stem := `version: 2
schema:
  desbloqueo:
    type: section
    heading: "## Desbloqueo"
    required: true
    default: "<!-- TODO: define unblock criteria -->"
`
	files := map[string]string{
		"task-001.md": "---\nstatus: open\n---\n# Task 001\n\nSome content here.\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{
		RootPath: dir,
		DryRun:   false,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.FilesScaffolded != 1 {
		t.Errorf("expected 1 file scaffolded, got %d", result.FilesScaffolded)
	}
	if result.SectionsAdded != 1 {
		t.Errorf("expected 1 section added, got %d", result.SectionsAdded)
	}

	// Verify section was actually written to the file.
	content, err := os.ReadFile(filepath.Join(dir, "task-001.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)

	if !strings.Contains(text, "## Desbloqueo") {
		t.Error("file is missing '## Desbloqueo' heading")
	}
	if !strings.Contains(text, "<!-- TODO: define unblock criteria -->") {
		t.Error("file is missing default content for Desbloqueo section")
	}
}

func TestScaffoldMissingRequiredSections_PlaceholderWhenNoDefault(t *testing.T) {
	stem := `version: 2
schema:
  notas:
    type: section
    heading: "## Notas"
    required: true
`
	files := map[string]string{
		"note.md": "---\ntitle: My Note\n---\n# Note\n\nBody text.\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{
		RootPath: dir,
		DryRun:   false,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.SectionsAdded != 1 {
		t.Errorf("expected 1 section added, got %d", result.SectionsAdded)
	}

	content, err := os.ReadFile(filepath.Join(dir, "note.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)

	if !strings.Contains(text, "## Notas") {
		t.Error("file is missing '## Notas' heading")
	}
	if !strings.Contains(text, "<!-- TODO -->") {
		t.Error("file is missing '<!-- TODO -->' placeholder")
	}
}

func TestScaffoldSkipsExistingSections(t *testing.T) {
	stem := `version: 2
schema:
  contexto:
    type: section
    heading: "## Contexto"
    required: true
    default: "<!-- TODO: add context -->"
`
	files := map[string]string{
		"task.md": "---\nstatus: open\n---\n# Task\n\n## Contexto\n\nAlready has context here.\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{
		RootPath: dir,
		DryRun:   false,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// No sections should be added — section already exists.
	if result.SectionsAdded != 0 {
		t.Errorf("expected 0 sections added, got %d", result.SectionsAdded)
	}

	// File content should not change.
	content, err := os.ReadFile(filepath.Join(dir, "task.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "Already has context here.") {
		t.Error("file content was unexpectedly modified")
	}
}

func TestScaffoldDryRun(t *testing.T) {
	stem := `version: 2
schema:
  plan:
    type: section
    heading: "## Plan"
    required: true
    default: "<!-- TODO: outline plan -->"
`
	files := map[string]string{
		"work.md": "---\ntype: work\n---\n# Work item\n\nContent.\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{
		RootPath: dir,
		DryRun:   true,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Dry-run should report sections that would be added.
	if result.SectionsAdded != 1 {
		t.Errorf("expected 1 section would be added (dry-run), got %d", result.SectionsAdded)
	}

	// But file should NOT have been modified.
	content, err := os.ReadFile(filepath.Join(dir, "work.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(content), "## Plan") {
		t.Error("dry-run should not have modified the file")
	}
}

func TestScaffoldMultipleSections_OrderedInsertion(t *testing.T) {
	stem := `version: 2
schema:
  contexto:
    type: section
    heading: "## Contexto"
    required: true
    ordered: 1
    default: "<!-- context -->"
  aceptacion:
    type: section
    heading: "## Aceptacion"
    required: true
    ordered: 2
    default: "<!-- acceptance -->"
`
	files := map[string]string{
		"story.md": "---\ntype: story\n---\n# Story\n\nSome body.\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{
		RootPath: dir,
		DryRun:   false,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.SectionsAdded != 2 {
		t.Errorf("expected 2 sections added, got %d", result.SectionsAdded)
	}

	content, err := os.ReadFile(filepath.Join(dir, "story.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)

	// Both sections must be present.
	if !strings.Contains(text, "## Contexto") {
		t.Error("missing ## Contexto")
	}
	if !strings.Contains(text, "## Aceptacion") {
		t.Error("missing ## Aceptacion")
	}

	// Contexto must appear before Aceptacion.
	idxCtx := strings.Index(text, "## Contexto")
	idxAce := strings.Index(text, "## Aceptacion")
	if idxCtx >= idxAce {
		t.Errorf("## Contexto (pos %d) should appear before ## Aceptacion (pos %d)", idxCtx, idxAce)
	}
}

func TestScaffoldNonSectionFieldsIgnored(t *testing.T) {
	stem := `version: 2
schema:
  status:
    type: enum
    required: true
    values: [open, done]
  notes:
    type: section
    heading: "## Notes"
    required: false
    default: "<!-- optional notes -->"
`
	files := map[string]string{
		"item.md": "---\nstatus: open\n---\n# Item\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{
		RootPath: dir,
		DryRun:   false,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Non-required section should not be scaffolded.
	if result.SectionsAdded != 0 {
		t.Errorf("expected 0 sections added, got %d", result.SectionsAdded)
	}
}
