package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestValidateSingleFileNoBoundaryStemEmitsEnvelope verifies that validate
// emits a proper JSON envelope when a single file target has no .stem anywhere.
// Issue #149: validate <file> should emit envelope, not raw error.
func TestValidateSingleFileNoBoundaryStemEmitsEnvelope(t *testing.T) {
	tmpdir := t.TempDir()
	mdFile := filepath.Join(tmpdir, "test.md")
	if err := os.WriteFile(mdFile, []byte("---\n---\n# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Save current dir and change to tmpdir with no .stem
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldCwd)
	}()

	if err := os.Chdir(tmpdir); err != nil {
		t.Fatal(err)
	}

	// Run validate <file> -o json
	var stdout bytes.Buffer
	cmd := rootCmd
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"validate", "test.md", "-o", "json"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("validate should fail with no .stem")
	}

	// Check that stdout has valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout should be valid JSON, got: %s, error: %v", stdout.String(), err)
	}

	// Check envelope structure
	if version, ok := result["version"]; !ok || version != float64(2) {
		t.Errorf("envelope should have version 2, got: %v", version)
	}
	if kind, ok := result["kind"]; !ok || kind != "rootline/validate-batch" {
		t.Errorf("envelope should have kind rootline/validate-batch, got: %v", kind)
	}

	// Check that notices contain scan_failed or schema_resolution_failed
	notices, ok := result["notices"].([]interface{})
	if !ok {
		t.Errorf("envelope should have notices array, got: %v", result["notices"])
	}
	if len(notices) == 0 {
		t.Error("envelope should have at least one notice for the missing schema")
	}
}

// TestValidateAllNoBoundaryStemEmitsEnvelope verifies that validate --all
// emits a proper JSON envelope when there's no declared boundary.
// Issue #149: validate --all should emit envelope, not raw error.
func TestValidateAllNoBoundaryStemEmitsEnvelope(t *testing.T) {
	tmpdir := t.TempDir()

	// Create a .stem without root: true
	stemPath := filepath.Join(tmpdir, ".stem")
	if err := os.WriteFile(stemPath, []byte("version: 2\nscope:\n  match: \"*.md\"\nschema:\n  titulo:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create a test markdown file
	mdFile := filepath.Join(tmpdir, "test.md")
	if err := os.WriteFile(mdFile, []byte("---\ntitulo: Test\n---\n# Test\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Save current dir and change to tmpdir
	oldCwd, err := os.Getwd()
	if err != nil {
		t.Fatal(err)
	}
	defer func() {
		_ = os.Chdir(oldCwd)
	}()

	if err := os.Chdir(tmpdir); err != nil {
		t.Fatal(err)
	}

	// Run validate --all . -o json
	var stdout bytes.Buffer
	cmd := rootCmd
	cmd.SetOut(&stdout)
	cmd.SetArgs([]string{"validate", "--all", ".", "-o", "json"})

	err = cmd.Execute()
	if err == nil {
		t.Fatal("validate should fail with no declared boundary")
	}

	// Check that stdout has valid JSON
	var result map[string]interface{}
	if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
		t.Fatalf("stdout should be valid JSON, got: %s, error: %v", stdout.String(), err)
	}

	// Check envelope structure
	if version, ok := result["version"]; !ok || version != float64(2) {
		t.Errorf("envelope should have version 2, got: %v", version)
	}
	if kind, ok := result["kind"]; !ok || kind != "rootline/validate-batch" {
		t.Errorf("envelope should have kind rootline/validate-batch, got: %v", kind)
	}

	// Check that notices contain schema_resolution_failed or similar error
	notices, ok := result["notices"].([]interface{})
	if !ok {
		t.Errorf("envelope should have notices array, got: %v", result["notices"])
	}
	if len(notices) == 0 {
		t.Error("envelope should have at least one notice for the boundary issue")
	}
}
