package migrate

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func acceptScaffoldValidation(ctx context.Context, in ScaffoldValidationInput) (*rules.ValidationResult, error) {
	return rules.NewValidationResult(in.Path, nil), nil
}

func TestScaffoldMissingRequiredSections_WithDefault(t *testing.T) {
	stem := `version: 2
schema:
  desbloqueo:
    type: string
    source: 'body.section["## Desbloqueo"]'
    required: true
    default: "<!-- TODO: define unblock criteria -->"
`
	files := map[string]string{
		"task-001.md": "---\nstatus: open\n---\n# Task 001\n\nSome content here.\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{
		RootPath:  dir,
		DryRun:    false,
		Validator: acceptScaffoldValidation,
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
    type: string
    source: 'body.section["## Notas"]'
    required: true
`
	files := map[string]string{
		"note.md": "---\ntitle: My Note\n---\n# Note\n\nBody text.\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{
		RootPath:  dir,
		DryRun:    false,
		Validator: acceptScaffoldValidation,
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
    type: string
    source: 'body.section["## Contexto"]'
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
    type: string
    source: 'body.section["## Plan"]'
    required: true
    default: "<!-- TODO: outline plan -->"
`
	files := map[string]string{
		"work.md": "---\ntype: work\n---\n# Work item\n\nContent.\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{
		RootPath:  dir,
		DryRun:    true,
		Validator: acceptScaffoldValidation,
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
    type: string
    source: 'body.section["## Contexto"]'
    required: true
    default: "<!-- context -->"
  aceptacion:
    type: string
    source: 'body.section["## Aceptacion"]'
    required: true
    default: "<!-- acceptance -->"
`
	files := map[string]string{
		"story.md": "---\ntype: story\n---\n# Story\n\nSome body.\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{
		RootPath:  dir,
		DryRun:    false,
		Validator: acceptScaffoldValidation,
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

	// Sections are materialized in exact lexical heading order.
	idxCtx := strings.Index(text, "## Contexto")
	idxAce := strings.Index(text, "## Aceptacion")
	if idxAce >= idxCtx {
		t.Errorf("## Aceptacion (pos %d) should appear before ## Contexto (pos %d)", idxAce, idxCtx)
	}
}

func TestScaffoldRequiredSectionSourceLegacySectionFieldsAreRejected(t *testing.T) {
	stem := `version: 2
schema:
  notas:
    type: section
    heading: "## Notas"
    required: true
`
	files := map[string]string{
		"item.md": "---\ntitle: Item\n---\n# Item\n",
	}
	dir := setupTestDir(t, stem, files)

	op := &ScaffoldOperation{RootPath: dir}

	_, err := op.Execute()
	if err == nil {
		t.Fatal("expected legacy section declaration error")
	}
	if !strings.Contains(err.Error(), `field "notas"`) {
		t.Fatalf("error = %v, want field declaration context", err)
	}
	content, readErr := os.ReadFile(filepath.Join(dir, "item.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(content), "## Notas") {
		t.Fatalf("legacy declaration must not be written, got:\n%s", content)
	}
}

func TestScaffoldRequiredSectionSourceValidateAndAppendInLexicalOrder(t *testing.T) {
	stem := `version: 2
schema:
  zeta:
    type: string
    required: true
    source: 'body.section["## Zeta"]'
    default: zed
  alpha:
    type: string
    required: true
    source: 'body.section["## Alpha"]'
`
	files := map[string]string{
		"story.md": "---\ntitle: Story\n---\n# Story\n",
	}
	dir := setupTestDir(t, stem, files)
	var validated ScaffoldValidationInput

	op := &ScaffoldOperation{
		RootPath: dir,
		Validator: func(ctx context.Context, in ScaffoldValidationInput) (*rules.ValidationResult, error) {
			validated = in
			return rules.NewValidationResult(in.Path, nil), nil
		},
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}
	if result.FilesScaffolded != 1 || result.SectionsAdded != 2 || len(result.Details) != 2 {
		t.Fatalf("result = %+v, want one file and two validated sections", result)
	}
	if result.Details[0].Heading != "## Alpha" || result.Details[1].Heading != "## Zeta" {
		t.Fatalf("details not lexical: %+v", result.Details)
	}
	if validated.Path != "story.md" {
		t.Fatalf("validator Path = %q, want logical scanner path", validated.Path)
	}
	if !filepath.IsAbs(validated.AbsPath) || validated.AbsPath != filepath.Join(dir, "story.md") {
		t.Fatalf("validator AbsPath = %q, want exact absolute record path", validated.AbsPath)
	}
	if !strings.Contains(string(validated.Content), "## Alpha\n\n<!-- TODO -->\n\n## Zeta\n\nzed\n") {
		t.Fatalf("validator saw unexpected prospective content:\n%s", validated.Content)
	}
	content, readErr := os.ReadFile(filepath.Join(dir, "story.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != string(validated.Content) {
		t.Fatalf("written bytes differ from validator bytes:\nvalidated:\n%s\nwritten:\n%s", validated.Content, content)
	}
	idxAlpha := strings.Index(string(content), "## Alpha")
	idxZeta := strings.Index(string(content), "## Zeta")
	if idxAlpha < 0 || idxZeta < 0 || idxAlpha > idxZeta {
		t.Fatalf("written sections not lexical:\n%s", content)
	}
}

func TestScaffoldRequiredSectionSourceNilValidatorFailsClosedOnlyWhenCandidateExists(t *testing.T) {
	stem := `version: 2
schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
`
	dir := setupTestDir(t, stem, map[string]string{
		"missing.md": "---\ntitle: Missing\n---\n# Missing\n",
	})

	_, err := (&ScaffoldOperation{RootPath: dir}).Execute()
	if err == nil {
		t.Fatal("expected nil validator to fail closed for a mutation candidate")
	}
	content, readErr := os.ReadFile(filepath.Join(dir, "missing.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(content), "## Notes") {
		t.Fatalf("nil validator wrote candidate:\n%s", content)
	}

	noCandidateDir := setupTestDir(t, stem, map[string]string{
		"present.md": "---\ntitle: Present\n---\n# Present\n\n## Notes\n\nAlready present.\n",
	})
	result, err := (&ScaffoldOperation{RootPath: noCandidateDir}).Execute()
	if err != nil {
		t.Fatalf("nil validator with no candidates should remain a zero-result success: %v", err)
	}
	if result.SectionsAdded != 0 || result.FilesScaffolded != 0 {
		t.Fatalf("result = %+v, want no candidates", result)
	}
}

func TestScaffoldRequiredSectionSourceRejectsMalformedZeroCandidateDryOrWrite(t *testing.T) {
	stem := `version: 2
schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
`
	original := "---\ntitle: [broken\n---\n# Present\n\n## Notes\n\nAlready present.\n"
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry", false: "write"}[dryRun], func(t *testing.T) {
			dir := setupTestDir(t, stem, map[string]string{"present.md": original})
			_, err := (&ScaffoldOperation{RootPath: dir, DryRun: dryRun}).Execute()
			if err == nil {
				t.Fatal("expected malformed frontmatter to fail despite zero candidates")
			}
			if !strings.Contains(err.Error(), "present.md") || !strings.Contains(err.Error(), "malformed YAML") {
				t.Fatalf("error = %v, want contextual extraction failure", err)
			}
			content, readErr := os.ReadFile(filepath.Join(dir, "present.md"))
			if readErr != nil {
				t.Fatal(readErr)
			}
			if string(content) != original {
				t.Fatalf("malformed record changed in dryRun=%v:\n%s", dryRun, content)
			}
		})
	}
}

func TestScaffoldRequiredSectionSourceUsesLogicalPathForRelativeExcludes(t *testing.T) {
	stem := `version: 2
schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    excludes:
      match: "docs/README.md"
`
	dir := setupTestDir(t, stem, map[string]string{
		"docs/README.md": "---\ntitle: Docs\n---\n# Docs\n",
	})
	result, err := (&ScaffoldOperation{RootPath: dir}).Execute()
	if err != nil {
		t.Fatalf("logical relative exclude should yield clean zero result: %v", err)
	}
	if result.SectionsAdded != 0 || result.FilesScaffolded != 0 {
		t.Fatalf("result = %+v, want relative exclude to suppress scaffold", result)
	}
	content, readErr := os.ReadFile(filepath.Join(dir, "docs", "README.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(content), "## Notes") {
		t.Fatalf("excluded relative path was scaffolded:\n%s", content)
	}
}

func TestScaffoldRequiredSectionSourceValidationFailureDoesNotWriteOrCountCandidate(t *testing.T) {
	stem := `version: 2
schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
`
	original := "---\ntitle: Broken\n---\n# Broken\n"
	dir := setupTestDir(t, stem, map[string]string{"broken.md": original})
	target := filepath.Join(dir, "broken.md")
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}

	op := &ScaffoldOperation{
		RootPath: dir,
		Validator: func(ctx context.Context, in ScaffoldValidationInput) (*rules.ValidationResult, error) {
			return rules.NewValidationResult(in.Path, []rules.ValidationError{{Rule: "test", Severity: "error", Message: "boom"}}), nil
		},
	}
	_, err := op.Execute()
	if err == nil {
		t.Fatal("expected validation failure")
	}
	content, readErr := os.ReadFile(target)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if string(content) != original {
		t.Fatalf("failed candidate changed bytes:\n%s", content)
	}
	info, statErr := os.Stat(target)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600 after validation failure", got)
	}
}

func TestScaffoldRequiredSectionSourceKeepsEarlierWriteWhenLaterFileFails(t *testing.T) {
	stem := `version: 2
schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
`
	dir := setupTestDir(t, stem, map[string]string{
		"a.md": "---\ntitle: A\n---\n# A\n",
		"b.md": "---\ntitle: B\n---\n# B\n",
	})
	op := &ScaffoldOperation{
		RootPath: dir,
		Validator: func(ctx context.Context, in ScaffoldValidationInput) (*rules.ValidationResult, error) {
			if in.Path == "b.md" {
				return rules.NewValidationResult(in.Path, []rules.ValidationError{{Rule: "test", Severity: "error", Message: "later failure"}}), nil
			}
			return rules.NewValidationResult(in.Path, nil), nil
		},
	}
	_, err := op.Execute()
	if err == nil {
		t.Fatal("expected later file validation failure")
	}
	aContent, readErr := os.ReadFile(filepath.Join(dir, "a.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(aContent), "## Notes") {
		t.Fatalf("first successful write was rolled back or skipped:\n%s", aContent)
	}
	bContent, readErr := os.ReadFile(filepath.Join(dir, "b.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(bContent), "## Notes") {
		t.Fatalf("later failed file was written:\n%s", bContent)
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
    type: string
    source: 'body.section["## Notes"]'
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

func TestScaffoldInvalidLaterSchemaKeepsEarlierWriteButDryRunWritesNone(t *testing.T) {
	stem := `version: 2
root: true
schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
  id:
    type: sequence
    match:
      "BAD*": {prefix: BAD, digits: 2.0}
`
	files := map[string]string{
		"A.md":      "---\ntitle: A\n---\n# A\n",
		"BAD001.md": "---\ntitle: Bad\n---\n# Bad\n",
	}

	writeDir := setupTestDir(t, stem, files)
	_, writeErr := (&ScaffoldOperation{RootPath: writeDir, Validator: acceptScaffoldValidation}).Execute()
	if writeErr == nil || !strings.Contains(writeErr.Error(), "BAD001.md") || !strings.Contains(writeErr.Error(), "digits") {
		t.Fatalf("write error = %v, want BAD001 digits cause", writeErr)
	}
	writtenA, err := os.ReadFile(filepath.Join(writeDir, "A.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(writtenA), "## Notes") {
		t.Fatalf("normal mode did not preserve earlier validated write:\n%s", writtenA)
	}

	dryDir := setupTestDir(t, stem, files)
	_, dryErr := (&ScaffoldOperation{RootPath: dryDir, DryRun: true, Validator: acceptScaffoldValidation}).Execute()
	if dryErr == nil || !strings.Contains(dryErr.Error(), "BAD001.md") || !strings.Contains(dryErr.Error(), "digits") {
		t.Fatalf("dry-run error = %v, want same BAD001 digits cause", dryErr)
	}
	if dryErr.Error() != writeErr.Error() {
		t.Fatalf("dry-run error = %q, want same causal error as write mode %q", dryErr, writeErr)
	}
	dryA, err := os.ReadFile(filepath.Join(dryDir, "A.md"))
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(dryA), "## Notes") {
		t.Fatalf("dry-run wrote earlier candidate despite later failure:\n%s", dryA)
	}
}
