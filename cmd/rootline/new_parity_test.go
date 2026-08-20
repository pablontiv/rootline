package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewDryRunRejectsInvalidProspectiveRecordWithoutCreatingTarget(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  status:
    type: enum
    required: true
    values: [Open, Closed]
`)
	target := filepath.Join(dir, "T001-task.md")

	out, err := runCmd(t, "new", target, "--dry-run")
	if err == nil {
		t.Fatalf("expected invalid dry-run to fail, output: %s", out)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("dry-run validation failure must leave target absent, stat err=%v", statErr)
	}
}

func TestNewDryRunPrintsExactValidatedBytes(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  status:
    type: string
    required: true
    default: Open
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    default: seeded notes
`)
	target := filepath.Join(dir, "T001-task.md")

	out, err := runCmd(t, "new", target, "--dry-run")
	if err != nil {
		t.Fatalf("valid dry-run failed: %v\noutput: %s", err, out)
	}
	want := "---\nstatus: Open\n---\n# T001 Task\n\n## Notes\n\nseeded notes\n"
	if out != want {
		t.Fatalf("dry-run bytes mismatch\n--- got ---\n%s\n--- want ---\n%s", out, want)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("dry-run must not create target, stat err=%v", statErr)
	}
}

func TestNewOmitsOptionalEnumWithoutDefault(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  status:
    type: enum
    values: [Open, Closed]
`)
	target := filepath.Join(dir, "T001-task.md")

	out, err := runCmd(t, "new", target, "--dry-run")
	if err != nil {
		t.Fatalf("optional defaultless enum should not make new invalid: %v\noutput: %s", err, out)
	}
	if strings.Contains(out, "status:") {
		t.Fatalf("optional enum without default should be omitted, got:\n%s", out)
	}
}

func TestNewOmitsOptionalDefaultlessSupportedTypes(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  bool_value:
    type: boolean
  enum_value:
    type: enum
    values: [Open, Closed]
  int_value:
    type: integer
  link_value:
    type: link
  list_value:
    type: list
  sequence_value:
    type: sequence
    prefix: T
    digits: 3
  string_value:
    type: string
`)
	target := filepath.Join(dir, "T001-task.md")

	out, err := runCmd(t, "new", target, "--dry-run")
	if err != nil {
		t.Fatalf("optional defaultless fields should not make new invalid: %v\noutput: %s", err, out)
	}
	for _, field := range []string{"bool_value", "enum_value", "int_value", "link_value", "list_value", "sequence_value", "string_value"} {
		if strings.Contains(out, field+":") {
			t.Fatalf("optional defaultless field %q should be omitted, got:\n%s", field, out)
		}
	}
}

func TestValidateProspectiveNewContentIgnoresDirtyValidateStrict(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  reference:
    type: link
    required: true
    default: not-a-wikilink
    severity: warn
`)
	target := filepath.Join(dir, "T001-task.md")
	effective := mustResolveNewEffective(t, dir, target)
	validateStrict = true
	defer func() { validateStrict = false }()

	content := "---\nreference: not-a-wikilink\n---\n# T001 Task\n"
	if err := validateProspectiveNewContent(rootCmd, target, content, effective); err != nil {
		t.Fatalf("new prospective validation must stay non-strict despite dirty global validateStrict: %v", err)
	}
}

func TestNewProspectiveValidationUsesProductionExtractorRegistry(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  status:
    type: string
    required: true
    default: Open
`)
	target := filepath.Join(dir, "T001-task.txt")

	out, err := runCmd(t, "new", target)
	if err == nil {
		t.Fatalf("expected unsupported extension to be rejected, output: %s", out)
	}
	if !strings.Contains(err.Error(), "no extractor") {
		t.Fatalf("expected registry rejection, got: %v", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("unsupported target should remain absent, stat err=%v", statErr)
	}
}

func TestNewProspectiveValidationAcceptsWarningOnlyDiagnostics(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  reference:
    type: link
    required: true
    default: not-a-wikilink
    severity: warn
`)
	target := filepath.Join(dir, "T001-task.md")

	out, err := runCmd(t, "new", target)
	if err != nil {
		t.Fatalf("warning-only prospective validation should not fail new: %v\noutput: %s", err, out)
	}
	if _, err := os.Stat(target); err != nil {
		t.Fatalf("warning-only new should create target: %v", err)
	}
}

func TestNewProspectiveValidationRejectsBrokenGeneratedWikilink(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    default: '[[missing-target]]'
`)
	target := filepath.Join(dir, "T001-task.md")

	out, err := runCmd(t, "new", target)
	if err == nil {
		t.Fatalf("expected broken generated wikilink to fail, output: %s", out)
	}
	if !strings.Contains(err.Error(), "link") {
		t.Fatalf("expected link validation error, got: %v", err)
	}
	if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
		t.Fatalf("broken-link target should remain absent, stat err=%v", statErr)
	}
}

func TestNewProspectiveValidationAcceptsAbsentSelfLink(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    default: '[[T001-task]]'
`)
	target := filepath.Join(dir, "T001-task.md")

	if out, err := runCmd(t, "new", target); err != nil {
		t.Fatalf("self-link to absent prospective target should validate: %v\noutput: %s", err, out)
	}
	if out, err := runCmd(t, "validate", target); err != nil {
		t.Fatalf("written self-link should validate after creation: %v\noutput: %s", err, out)
	}
}

func TestNewProspectiveValidationAcceptsAbsentSelfAnchor(t *testing.T) {
	dir := newSectionSourceProject(t, `links:
  checks:
    anchors: true
schema:
  anchor:
    type: string
    required: true
    source: 'body.section["## Generated Anchor"]'
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    default: '[[T001-task#generated-anchor]]'
`)
	target := filepath.Join(dir, "T001-task.md")

	if out, err := runCmd(t, "new", target); err != nil {
		t.Fatalf("self-anchor to absent prospective target should validate: %v\noutput: %s", err, out)
	}
	if out, err := runCmd(t, "validate", target); err != nil {
		t.Fatalf("written self-anchor should validate after creation: %v\noutput: %s", err, out)
	}
}

func TestNewForceValidatesAnchorsAgainstProspectiveBytes(t *testing.T) {
	dir := newSectionSourceProject(t, `links:
  checks:
    anchors: true
schema:
  anchor:
    type: string
    required: true
    source: 'body.section["## New Anchor"]'
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    default: '[[existing#new-anchor]]'
`)
	target := filepath.Join(dir, "existing.md")
	old := "---\n---\n# Existing\n\n## Old Anchor\n"
	mustWriteFile(t, target, []byte(old), 0o644)

	if out, err := runCmd(t, "new", target, "--force"); err != nil {
		t.Fatalf("force should accept anchor introduced only by prospective bytes: %v\noutput: %s", err, out)
	}
	if got := string(mustReadFile(t, target)); !strings.Contains(got, "## New Anchor") || strings.Contains(got, "## Old Anchor") {
		t.Fatalf("forced target should contain prospective bytes only, got:\n%s", got)
	}
}

func TestNewForceRejectsAnchorRemovedFromProspectiveBytes(t *testing.T) {
	dir := newSectionSourceProject(t, `links:
  checks:
    anchors: true
schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    default: '[[existing#old-anchor]]'
  replacement:
    type: string
    required: true
    source: 'body.section["## Replacement"]'
`)
	target := filepath.Join(dir, "existing.md")
	old := "---\n---\n# Existing\n\n## Old Anchor\n"
	mustWriteFile(t, target, []byte(old), 0o644)

	out, err := runCmd(t, "new", target, "--force")
	if err == nil {
		t.Fatalf("force should reject anchor absent from prospective bytes, output: %s", out)
	}
	if got := string(mustReadFile(t, target)); got != old {
		t.Fatalf("failed force must leave existing target unchanged\ngot: %q\nwant:%q", got, old)
	}
}

func TestNewGeneratedH1SatisfiesRequiredBodyH1Source(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  title:
    type: string
    required: true
    source: body.h1
`)
	target := filepath.Join(dir, "T001-task.md")

	out, err := runCmd(t, "new", target)
	if err != nil {
		t.Fatalf("new with required body.h1 failed: %v\noutput: %s", err, out)
	}
	got := string(mustReadFile(t, target))
	want := "---\n---\n# T001 Task\n"
	if got != want {
		t.Fatalf("body.h1 bytes mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
	if strings.Contains(got, "title:") {
		t.Fatalf("body.h1 source-backed field must not be frontmatter-shadowed:\n%s", got)
	}
}

func TestNewMaterializesExactRequiredH1Section(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  overview:
    type: string
    required: true
    source: 'body.section["# Required Overview"]'
`)
	target := filepath.Join(dir, "T001-task.md")

	out, err := runCmd(t, "new", target)
	if err != nil {
		t.Fatalf("new with materialized H1 section failed: %v\noutput: %s", err, out)
	}
	want := "---\n---\n# T001 Task\n\n# Required Overview\n\n<!-- TODO -->\n"
	if got := string(mustReadFile(t, target)); got != want {
		t.Fatalf("materialized H1 bytes mismatch\n--- got ---\n%s\n--- want ---\n%s", got, want)
	}
}

func TestNewWritesFileThatValidateAccepts(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  status:
    type: string
    required: true
    default: Open
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
`)
	target := filepath.Join(dir, "T001-task.md")

	if out, err := runCmd(t, "new", target); err != nil {
		t.Fatalf("new failed: %v\noutput: %s", err, out)
	}
	if out, err := runCmd(t, "validate", target); err != nil {
		t.Fatalf("validate after new failed: %v\noutput: %s", err, out)
	}
}
