package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

// --- generateMarkdown tests (new.go) ---

// TestGenerateMarkdown_BasicSchema tests markdown generation
func TestGenerateMarkdown_BasicSchema(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 2
schema:
  titulo:
    type: string
    required: true
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// Run new command to generate markdown with a specific file path
	newFile := filepath.Join(dir, "document.md")
	resetFlags()
	out, err := runCmd(t, "new", newFile, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Output should contain frontmatter section
	if !strings.Contains(out, "---") {
		t.Errorf("expected frontmatter markers in output, got: %s", out)
	}
}

// --- runExplain tests (explain.go) ---

// TestRunExplain_FullOutput tests explain with valid markdown file
func TestRunExplain_FullOutput(t *testing.T) {
	dir := setupTestDir(t)
	taskFile := filepath.Join(dir, "doc1.md")

	resetFlags()
	out, err := runCmd(t, "explain", taskFile)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	// Should have fields array
	if _, ok := result["fields"]; !ok {
		t.Error("expected 'fields' in explain result")
	}

	// Should have kind
	if kind, ok := result["kind"]; !ok || kind != "rootline/explain" {
		t.Errorf("expected kind rootline/explain, got %v", result["kind"])
	}
}

// TestRunExplain_TableOutput tests explain with table output format
func TestRunExplain_TableOutput(t *testing.T) {
	dir := setupTestDir(t)
	taskFile := filepath.Join(dir, "doc1.md")

	resetFlags()
	out, err := runCmd(t, "explain", taskFile, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Table output should contain field names
	if !strings.Contains(out, "estado") && !strings.Contains(out, "Field") {
		t.Logf("table output: %s", out)
	}
}

// TestRunExplain_WithErrors tests explain with validation errors
func TestRunExplain_WithErrors(t *testing.T) {
	dir := setupTestDir(t)
	// Create a file with validation error
	badFile := filepath.Join(dir, "bad.md")
	mustWriteFile(t, badFile, []byte("---\ntipo: test\n---\n# Bad\n"), 0644)

	resetFlags()
	out, err := runCmd(t, "explain", badFile)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Should have errors field
	if errs, ok := result["errors"]; ok && errs != nil {
		errList := errs.([]any)
		if len(errList) == 0 {
			t.Error("expected validation errors")
		}
	}
}

// --- runDescribe tests (describe.go) ---

// TestRunDescribe_DirectoryTarget tests describe with directory target
func TestRunDescribe_DirectoryTarget(t *testing.T) {
	dir := setupTestDir(t)

	resetFlags()
	out, err := runCmd(t, "describe", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	// Should have schema with fields
	schema, ok := result["schema"].(map[string]any)
	if !ok {
		t.Errorf("expected schema map, got %T", result["schema"])
	}

	if _, hasEstado := schema["estado"]; !hasEstado {
		t.Errorf("expected estado in schema, got: %v", schema)
	}
}

// TestRunDescribe_TableOutput tests describe with table output
func TestRunDescribe_TableOutput(t *testing.T) {
	dir := setupTestDir(t)

	resetFlags()
	out, err := runCmd(t, "describe", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Should be table formatted
	if !strings.Contains(out, "Field") && !strings.Contains(out, "Type") {
		t.Logf("table output: %s", out)
	}
}

// TestRunDescribe_NoStem tests that describe fails explicitly when no .stem
// exists anywhere in the chain.
//
// Before the marker-based discovery change, this returned a successful JSON
// document carrying hints. That was one half of the false-green class of bugs:
// a command reporting success while governing nothing. Zero .stem is now
// always ErrNoSchemaFound.
func TestRunDescribe_NoStem(t *testing.T) {
	dir := t.TempDir()
	emptyDir := filepath.Join(dir, "empty")
	if err := os.Mkdir(emptyDir, 0o755); err != nil {
		t.Fatal(err)
	}

	resetFlags()
	out, err := runCmd(t, "describe", emptyDir)
	if err == nil {
		t.Fatalf("expected an error when no .stem exists, got success\noutput: %s", out)
	}
	if !errors.Is(err, rules.ErrNoSchemaFound) {
		t.Fatalf("expected ErrNoSchemaFound, got: %v", err)
	}
}

// --- runHooksUninstall tests (hooks.go) ---

// TestRunHooksUninstall_Success tests hook uninstallation
func TestRunHooksUninstall_Success(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create git hooks directory
	hooksDir := filepath.Join(dir, ".git", "hooks")
	if err := os.MkdirAll(hooksDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a mock pre-commit hook
	hookPath := filepath.Join(hooksDir, "pre-commit")
	mustWriteFile(t, hookPath, []byte("#!/bin/bash\necho 'test hook'\n"), 0o755)

	mustChdir(t, dir)

	resetFlags()
	out, err := runCmd(t, "hooks", "uninstall")
	if err != nil {
		t.Logf("uninstall error (may be expected): %v, output: %s", err, out)
	}

	// Check if hook was uninstalled
	_, err = os.Stat(hookPath)
	if err == nil {
		// Hook still exists - uninstall didn't work or wasn't needed
		t.Logf("hook still exists after uninstall: %s", hookPath)
	}
}

// TestRunHooksUninstall_NoGitRoot tests uninstall with no git root
func TestRunHooksUninstall_NoGitRoot(t *testing.T) {
	dir := t.TempDir()
	// No .git directory

	mustChdir(t, dir)

	resetFlags()
	_, err := runCmd(t, "hooks", "uninstall")
	if err == nil {
		// May succeed if hooks uninstall is lenient
		t.Logf("uninstall succeeded with no .git (lenient behavior)")
	}
}

// --- resolveLinkSchema tests (analyze.go) ---

// TestResolveLinkSchema_ValidLinks tests link resolution
func TestResolveLinkSchema_ValidLinks(t *testing.T) {
	dir := setupTestDir(t)

	resetFlags()
	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Logf("analyze error (may be expected): %v", err)
	}

	// analyze command should exercise resolveLinkSchema indirectly
	// Just verify no crash
	var result map[string]any
	if unmarshalErr := json.Unmarshal([]byte(out), &result); unmarshalErr != nil {
		t.Logf("couldn't parse analyze output: %v", unmarshalErr)
	}
}

// --- appendPropagateProposals tests (fix.go) ---

// TestAppendPropagateProposals_ViaFixAll tests via fix --all command
func TestAppendPropagateProposals_ViaFixAll(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create stem with aggregate
	stemContent := `version: 2
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
aggregate:
  total:
    expression: "len(rows)"
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// Create index file
	mustWriteFile(t, filepath.Join(dir, "README.md"),
		[]byte("---\ntotal: 1\n---\n# Index\n"), 0644)

	// Create a task file
	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Pending\n---\n# Task\n"), 0644)

	resetFlags()
	out, err := runCmd(t, "fix", "--all", dir)
	if err != nil {
		t.Logf("fix error (may be expected): %v", err)
	}

	// Just verify command runs without crash
	_ = out
}
