package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorValidStems(t *testing.T) {
	dir := t.TempDir()
	// Create a valid .stem
	stemContent := `version: 1
scope:
  match: "*.md"
schema:
  estado:
    type: string
    required: true
  tipo:
    type: enum
    values: [a, b]
`
	os.WriteFile(filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	os.WriteFile(filepath.Join(dir, "test.md"), []byte("---\nestado: draft\n---\n# Test\n"), 0644)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"doctor", dir})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out := buf.String()
	if strings.Contains(out, "✗") {
		t.Errorf("expected no failures, got: %s", out)
	}
}

func TestDoctorInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	os.WriteFile(filepath.Join(dir, ".stem"), []byte("{{invalid yaml"), 0644)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"doctor", dir})
	rootCmd.Execute() // may return error
	out := buf.String()
	if !strings.Contains(out, "yaml-valid") {
		t.Errorf("expected yaml-valid check failure, got: %s", out)
	}
}

func TestDoctorOrphanScope(t *testing.T) {
	dir := t.TempDir()
	stemContent := `version: 1
scope:
  match: "*.xyz"
`
	os.WriteFile(filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"doctor", dir})
	rootCmd.Execute()
	out := buf.String()
	if !strings.Contains(out, "scope-match") {
		t.Errorf("expected scope-match warning, got: %s", out)
	}
}

func TestDoctorJSONOutput(t *testing.T) {
	dir := t.TempDir()
	stemContent := `version: 1
schema:
  estado:
    type: string
`
	os.WriteFile(filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"doctor", dir, "-o", "json"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result DoctorResult
	if err := json.Unmarshal(buf.Bytes(), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, buf.String())
	}
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
	if result.Kind != "rootline/doctor" {
		t.Errorf("expected kind rootline/doctor, got %s", result.Kind)
	}
}

func TestDoctorNoStems(t *testing.T) {
	dir := t.TempDir()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetArgs([]string{"doctor", dir})
	rootCmd.Execute()
	out := buf.String()
	if !strings.Contains(out, "no .stem") {
		t.Errorf("expected no .stem warning, got: %s", out)
	}
}
