package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/migrate"
)

func newMigrateScaffoldSectionSourceProject(t *testing.T, schema, body string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	stem := "version: 2\nscope:\n  match: \"*.md\"\n" + schema
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stem), 0o644)
	if body == "" {
		body = "---\ntitle: Task\n---\n# Task\n"
	}
	mustWriteFile(t, filepath.Join(dir, "T001-task.md"), []byte(body), 0o640)
	declareTestBoundary(t, dir)
	return dir
}

func runMigrateScaffoldJSON(t *testing.T, dir string, args ...string) (migrate.ScaffoldResult, string, error) {
	t.Helper()
	cmdArgs := append([]string{"migrate", "--scaffold"}, args...)
	cmdArgs = append(cmdArgs, dir)
	out, err := runCmd(t, cmdArgs...)
	if err != nil {
		return migrate.ScaffoldResult{}, out, err
	}
	var result migrate.ScaffoldResult
	if unmarshalErr := json.Unmarshal([]byte(out), &result); unmarshalErr != nil {
		t.Fatalf("unmarshaling scaffold JSON %q: %v", out, unmarshalErr)
	}
	return result, out, nil
}

func TestMigrateScaffoldRequiredSectionSourceWarningsDoNotBlockWrite(t *testing.T) {
	dir := newMigrateScaffoldSectionSourceProject(t, `schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
  advisory:
    type: string
    required: true
    severity: warn
`, "")

	result, out, err := runMigrateScaffoldJSON(t, dir)
	if err != nil {
		t.Fatalf("warning-only prospective validation should not fail scaffold: %v\noutput: %s", err, out)
	}
	if result.SectionsAdded != 1 || len(result.Details) != 1 {
		t.Fatalf("result = %+v, want one validated scaffold", result)
	}
	content, readErr := os.ReadFile(filepath.Join(dir, "T001-task.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(content), "## Notes") {
		t.Fatalf("warning-only scaffold did not write section:\n%s", content)
	}
}

func TestMigrateScaffoldRequiredSectionSourceRejectsBrokenProspectiveLinkWithoutWrite(t *testing.T) {
	dir := newMigrateScaffoldSectionSourceProject(t, `schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    default: '[[missing-target]]'
`, "")

	out, err := runCmd(t, "migrate", "--scaffold", dir)
	if err == nil {
		t.Fatalf("expected broken prospective link to fail, output: %s", out)
	}
	if !strings.Contains(err.Error(), "link") {
		t.Fatalf("error = %v, want link validation failure", err)
	}
	content, readErr := os.ReadFile(filepath.Join(dir, "T001-task.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(content), "## Notes") {
		t.Fatalf("broken candidate was written:\n%s", content)
	}
}

func TestMigrateScaffoldRequiredSectionSourceDryRunParityAndNoWrite(t *testing.T) {
	schema := `schema:
  zeta:
    type: string
    required: true
    source: 'body.section["## Zeta"]'
  alpha:
    type: string
    required: true
    source: 'body.section["## Alpha"]'
`
	dryDir := newMigrateScaffoldSectionSourceProject(t, schema, "")
	writeDir := newMigrateScaffoldSectionSourceProject(t, schema, "")

	dryResult, out, err := runMigrateScaffoldJSON(t, dryDir, "--dry-run")
	if err != nil {
		t.Fatalf("dry-run scaffold failed: %v\noutput: %s", err, out)
	}
	writeResult, out, err := runMigrateScaffoldJSON(t, writeDir)
	if err != nil {
		t.Fatalf("write scaffold failed: %v\noutput: %s", err, out)
	}
	if dryResult.SectionsAdded != writeResult.SectionsAdded || len(dryResult.Details) != len(writeResult.Details) {
		t.Fatalf("dry result %+v does not match write result %+v", dryResult, writeResult)
	}
	if dryResult.Details[0].Heading != "## Alpha" || dryResult.Details[1].Heading != "## Zeta" {
		t.Fatalf("dry-run details not lexical: %+v", dryResult.Details)
	}
	dryContent, readErr := os.ReadFile(filepath.Join(dryDir, "T001-task.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(dryContent), "## Alpha") || strings.Contains(string(dryContent), "## Zeta") {
		t.Fatalf("dry-run wrote candidate:\n%s", dryContent)
	}
}

func TestMigrateScaffoldRequiredSectionSourcePreservesModeAndValidatesAfterWrite(t *testing.T) {
	dir := newMigrateScaffoldSectionSourceProject(t, `schema:
  anchor:
    type: string
    required: true
    source: 'body.section["## Generated Anchor"]'
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    default: '[[T001-task#generated-anchor]]'
`, "")
	target := filepath.Join(dir, "T001-task.md")
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}

	result, out, err := runMigrateScaffoldJSON(t, dir)
	if err != nil {
		t.Fatalf("self-anchor should validate against prospective scaffold bytes: %v\noutput: %s", err, out)
	}
	if result.SectionsAdded != 2 {
		t.Fatalf("result = %+v, want two scaffolded sections", result)
	}
	info, statErr := os.Stat(target)
	if statErr != nil {
		t.Fatal(statErr)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
	if out, err := runCmd(t, "validate", target); err != nil {
		t.Fatalf("scaffolded record should validate after write: %v\noutput: %s", err, out)
	}
}

func TestMigrateScaffoldPassesLogicalPathToProspectiveValidation(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  metadata:
    type: string
    required: true
    excludes:
      match: "docs/T001-task.md"
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
`), 0o644)
	if err := os.Mkdir(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "docs", "T001-task.md"), []byte("---\ntitle: Task\n---\n# Task\n"), 0o640)
	declareTestBoundary(t, dir)

	result, out, err := runMigrateScaffoldJSON(t, dir)
	if err != nil {
		t.Fatalf("logical excludes should apply during prospective validation: %v\noutput: %s", err, out)
	}
	if result.SectionsAdded != 1 || result.Details[0].File != "docs/T001-task.md" {
		t.Fatalf("result = %+v, want one scaffolded logical docs path", result)
	}
	content, readErr := os.ReadFile(filepath.Join(dir, "docs", "T001-task.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(content), "## Notes") {
		t.Fatalf("logical-path validated scaffold was not written:\n%s", content)
	}
}

func TestMigrateScaffoldRejectsProspectiveStructureErrorWithoutWrite(t *testing.T) {
	dir := newMigrateScaffoldSectionSourceProject(t, `schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
`, "---\ntitle: Task\n...\nextra: true\n---\n# Task\n")

	out, err := runCmd(t, "migrate", "--scaffold", dir)
	if err == nil {
		t.Fatalf("expected prospective structure rejection, output: %s", out)
	}
	if !strings.Contains(err.Error(), "frontmatter contains") {
		t.Fatalf("error = %v, want ValidateStructure rejection", err)
	}
	content, readErr := os.ReadFile(filepath.Join(dir, "T001-task.md"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(content), "## Notes") {
		t.Fatalf("structurally invalid candidate was written:\n%s", content)
	}
}

func TestMigrateScaffoldResolutionFailureErrorsInsteadOfZeroResult(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "ungoverned.md"), []byte("# Ungoverned\n"), 0o644)
	declareTestBoundary(t, dir)
	out, err := runCmd(t, "migrate", "--scaffold", dir)
	if err == nil {
		t.Fatalf("expected resolution failure instead of zero-result success, output: %s", out)
	}
}
