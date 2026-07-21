package main

import (
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
// target file does not exist, both with and without --create.
// --create creates missing sections, not missing files.
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

func TestSetCreateOnNonexistentFile(t *testing.T) {
	dir := setupSetTestDir(t)
	missing := filepath.Join(dir, "does-not-exist.md")

	// --create does NOT create files; the same file-not-found error must occur.
	_, err := runCmd(t, "set", "--create", missing, "estado=Completed")
	if err == nil {
		t.Fatal("expected error for nonexistent file even with --create")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' error, got: %v", err)
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
