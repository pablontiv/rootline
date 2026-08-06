package main

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupSetTestDir creates a temp directory with .git, .stem (including a section field),
// and a markdown document for set command testing.
func setupSetTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create .git marker (WalkUp requires a git boundary).
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// .stem with frontmatter fields and a section field
	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  tipo:
    type: string
    required: false
  investigacion:
    type: string
    required: false
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	declareTestBoundary(t, dir)

	// Document with frontmatter and a section
	docContent := `---
estado: Pending
tipo: test
---
# Doc

## Investigacion

Initial findings here.
`
	mustWriteFile(t, filepath.Join(dir, "doc.md"), []byte(docContent), 0644)

	declareTestBoundary(t, dir)
	return dir
}

func TestSetFrontmatterField(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	out, err := runCmd(t, "set", target, "estado=Completed")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "set estado") {
		t.Errorf("expected 'set estado' in output, got: %s", out)
	}

	content := string(mustReadFile(t, target))
	if !strings.Contains(content, "estado: Completed") {
		t.Errorf("expected 'estado: Completed' in file, got:\n%s", content)
	}
}

func TestSetDryRun(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	original := string(mustReadFile(t, target))

	out, err := runCmd(t, "set", "--dry-run", target, "estado=Completed")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	if !strings.Contains(out, "would set") {
		t.Errorf("expected 'would set' in dry-run output, got: %s", out)
	}

	// Verify file was NOT modified
	after := string(mustReadFile(t, target))
	if after != original {
		t.Error("dry-run should not modify the file")
	}
}

func TestSetInvalidEnumValue(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	_, err := runCmd(t, "set", target, "estado=InvalidValue")
	if err == nil {
		t.Fatal("expected error for invalid enum value")
	}
	if !strings.Contains(err.Error(), "not in allowed values") {
		t.Errorf("expected 'not in allowed values' error, got: %v", err)
	}
}

func TestSetFromFile(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	// Write a value file
	valueFile := filepath.Join(dir, "value.txt")
	mustWriteFile(t, valueFile, []byte("Content from file"), 0644)

	out, err := runCmd(t, "set", "--no-validate", target, "investigacion=@"+valueFile)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	content := string(mustReadFile(t, target))
	if !strings.Contains(content, "Content from file") {
		t.Errorf("expected file content in document, got:\n%s", content)
	}
}

func TestSetNoArgs(t *testing.T) {
	_, err := runCmd(t, "set")
	if err == nil {
		t.Fatal("expected error for set with no args")
	}
}

func TestSetBadFieldSyntax(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	_, err := runCmd(t, "set", target, "badvalue")
	if err == nil {
		t.Fatal("expected error for bad field syntax")
	}
	if !strings.Contains(err.Error(), "invalid field assignment") {
		t.Errorf("expected 'invalid field assignment' error, got: %v", err)
	}
}

func TestSetAppendNonSection(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	_, err := runCmd(t, "set", target, "estado+=Completed")
	if err == nil {
		t.Fatal("expected error for append on non-section field")
	}
	if !strings.Contains(err.Error(), "no longer supported") {
		t.Errorf("expected 'no longer supported' error, got: %v", err)
	}
}

func TestSetClearField(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	// Clearing tipo (non-required string field) should work
	out, err := runCmd(t, "set", "--no-validate", target, "tipo=")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	content := string(mustReadFile(t, target))
	if !strings.Contains(content, "tipo:") {
		t.Errorf("expected tipo field present (even if empty), got:\n%s", content)
	}
}

// TestSetNonexistentFile verifies that set fails with a clear error when the
// target file does not exist.
func TestSetNonexistentFile(t *testing.T) {
	dir := setupSetTestDir(t)
	missing := filepath.Join(dir, "does-not-exist.md")

	_, err := runCmd(t, "set", missing, "estado=Completed")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' error, got: %v", err)
	}
}

func TestSetRejectsCreateFlag(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	_, err := runCmd(t, "set", "--create", target, "estado=Completed")
	if err == nil {
		t.Fatal("expected --create to be rejected")
	}
	if !strings.Contains(err.Error(), "unknown flag: --create") {
		t.Errorf("expected unknown --create flag error, got: %v", err)
	}
}

func TestSetHelpDoesNotAdvertiseUnsupportedSyntax(t *testing.T) {
	buf := new(bytes.Buffer)
	setCmd.SetOut(buf)
	t.Cleanup(func() { setCmd.SetOut(nil) })
	if err := setCmd.Help(); err != nil {
		t.Fatalf("unexpected help error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "--create") {
		t.Errorf("set help advertises removed --create flag: %s", out)
	}
	if strings.Contains(out, "+=") {
		t.Errorf("set help advertises unsupported append syntax: %s", out)
	}
}

func TestSetMultipleFields(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	out, err := runCmd(t, "set", target, "estado=Completed", "tipo=production")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	content := string(mustReadFile(t, target))
	if !strings.Contains(content, "estado: Completed") {
		t.Errorf("expected 'estado: Completed', got:\n%s", content)
	}
	if !strings.Contains(content, "tipo: production") {
		t.Errorf("expected 'tipo: production', got:\n%s", content)
	}
}

// --- Slice 3d: Mutating Command Error Semantics ---

// TestSetRefusesNoSchema verifies that set fails when no schema is present.
// This is a slice 3d test: currently set might succeed silently on ungoverned trees.
// After slice 3d, it should fail.
func TestSetRefusesNoSchema(t *testing.T) {
	// Create a tree with NO .stem file
	dir := t.TempDir()

	// Create a markdown file
	target := filepath.Join(dir, "doc.md")
	mustWriteFile(t, target, []byte("---\ntitle: Test\n---\n# Test\n"), 0644)

	// Attempt set (should fail because no schema exists)
	out, err := runCmd(t, "set", target, "field=value")

	// After slice 3d: this should error
	if err == nil {
		t.Errorf("expected error when set has no schema, but succeeded with output: %s", out)
	}
}

// TestSetRefusesBadSchema verifies that set fails when the schema is unparseable.
func TestSetRefusesBadSchema(t *testing.T) {
	// Create a tree with an unparseable .stem
	dir := t.TempDir()

	// Create unparseable .stem
	badStem := "version: 2\nthis: [is not valid YAML"
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(badStem), 0644)

	// Create a markdown file
	target := filepath.Join(dir, "doc.md")
	mustWriteFile(t, target, []byte("---\ntitle: Test\n---\n# Test\n"), 0644)

	// Attempt set (should fail because schema parse error)
	out, err := runCmd(t, "set", target, "field=value")

	// After slice 3d: this should error
	if err == nil {
		t.Errorf("expected error when set has bad schema, but succeeded with output: %s", out)
	}
}

// --- Slice 4: Set.go Root Computation ---

// TestSetProposalPaths_MatchRecordPaths verifies that proposal paths and record paths
// use the same namespace (invocation root), not the marker root. This is a critical
// requirement for fix.go to find and apply proposals.
func TestSetProposalPaths_MatchRecordPaths(t *testing.T) {
	dir := setupSetTestDir(t)

	// Run set from the directory root, targeting a file in docs/
	target := filepath.Join(dir, "doc.md")
	out, err := runCmd(t, "set", target, "estado=Completed")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Verify the file was actually modified (proposal was applied)
	content := string(mustReadFile(t, target))
	if !strings.Contains(content, "estado: Completed") {
		t.Errorf("expected 'estado: Completed' in file after set, got:\n%s", content)
	}

	// The test passes if set succeeded and modified the file, which means
	// the proposal paths matched the record paths during fix.ApplyProposals.
	// If paths diverged, the lookup in fix.go:29-31 would silently skip the record
	// and the file would remain unchanged.
}

// TestSetWithMarkerBoundary verifies that set respects the marker-based schema
// boundary, not a Git boundary. This test creates a marker at a specific location
// and verifies set works correctly with it.
func TestSetWithMarkerBoundary(t *testing.T) {
	// Create a simple project with a marker at the root
	dir := t.TempDir()

	// Create the root marker with `root: true`
	stemContent := `version: 2
root: true
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// Create a document
	docContent := `---
estado: Pending
---
# Test Doc
`
	mustWriteFile(t, filepath.Join(dir, "doc.md"), []byte(docContent), 0644)

	// Run set
	target := filepath.Join(dir, "doc.md")
	out, err := runCmd(t, "set", target, "estado=Completed")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Verify the change was applied
	content := string(mustReadFile(t, target))
	if !strings.Contains(content, "estado: Completed") {
		t.Errorf("expected 'estado: Completed' after set, got:\n%s", content)
	}
}

// TestSetOutsideGit verifies that set works on a file in a project with no .git directory.
// This is a critical requirement: set must not depend on .git to compute paths.
func TestSetOutsideGit(t *testing.T) {
	// Create a project with NO .git directory at all
	dir := t.TempDir()

	// Create a marker (required by new implementation)
	stemContent := `version: 2
root: true
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// Create a markdown file
	docContent := `---
estado: Pending
---
# Test Doc
`
	target := filepath.Join(dir, "doc.md")
	mustWriteFile(t, target, []byte(docContent), 0644)

	// Attempt set (should work even without .git)
	out, err := runCmd(t, "set", target, "estado=Completed")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Verify the change was applied
	content := string(mustReadFile(t, target))
	if !strings.Contains(content, "estado: Completed") {
		t.Errorf("expected 'estado: Completed' after set, got:\n%s", content)
	}
}
