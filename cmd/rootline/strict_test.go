package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupStrictTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create .git marker (WalkUp requires a git boundary).
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// .stem with estado as warn severity, tipo as error severity
	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completado]
    severity: warn
  tipo:
    type: string
    required: true
    severity: error
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	declareTestBoundary(t, dir)
	return dir
}

func TestValidateWarnOnlyNoExitCode(t *testing.T) {
	dir := setupStrictTestDir(t)
	// Missing estado (warn) but has tipo (error) → should be valid (only warnings)
	target := filepath.Join(dir, "warn-only.md")
	mustWriteFile(t, target, []byte("---\ntipo: test\n---\n# Test\n"), 0644)

	out, err := runCmd(t, "validate", target)
	if err != nil {
		t.Fatalf("expected no error for warn-only, got: %v", err)
	}
	// Should show valid=true
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["valid"] != true {
		t.Errorf("expected valid=true with only warnings, got: %v", result["valid"])
	}
}

func TestValidateStrictWarnsExitCode(t *testing.T) {
	dir := setupStrictTestDir(t)
	target := filepath.Join(dir, "warn-strict.md")
	mustWriteFile(t, target, []byte("---\ntipo: test\n---\n# Test\n"), 0644)

	_, err := runCmd(t, "validate", "--strict", target)
	if err == nil {
		t.Fatal("expected error with --strict and warnings")
	}
}

func TestValidateErrorAlwaysExitCode(t *testing.T) {
	dir := setupStrictTestDir(t)
	// Missing tipo (error severity)
	target := filepath.Join(dir, "error.md")
	mustWriteFile(t, target, []byte("---\nestado: Pending\n---\n# Test\n"), 0644)

	_, err := runCmd(t, "validate", target)
	if err == nil {
		t.Fatal("expected error for missing required field with error severity")
	}
}

func TestValidateJSONIncludesSeverity(t *testing.T) {
	dir := setupStrictTestDir(t)
	// Missing both → has errors and warnings
	target := filepath.Join(dir, "both.md")
	mustWriteFile(t, target, []byte("---\n---\n# Test\n"), 0644)

	out, err := runCmd(t, "validate", target)
	_ = err // may have error exit code

	if !strings.Contains(out, `"severity"`) {
		t.Errorf("expected severity field in JSON output, got: %s", out)
	}
}

func TestValidateBatchWarningsCount(t *testing.T) {
	dir := setupStrictTestDir(t)
	f1 := filepath.Join(dir, "a.md")
	f2 := filepath.Join(dir, "b.md")
	mustWriteFile(t, f1, []byte("---\ntipo: test\n---\n# A\n"), 0644)
	mustWriteFile(t, f2, []byte("---\ntipo: prod\n---\n# B\n"), 0644)

	out, _ := runCmd(t, "validate", f1, f2)
	if !strings.Contains(out, `"warnings_count"`) {
		t.Errorf("expected warnings_count in batch output, got: %s", out)
	}
}
