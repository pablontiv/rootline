package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestDoctorValidStems(t *testing.T) {
	dir := t.TempDir()
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
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	mustWriteFile(t, filepath.Join(dir, "test.md"), []byte("---\nestado: draft\n---\n# Test\n"), 0644)

	out, err := runCmd(t, "doctor", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "✗") {
		t.Errorf("expected no failures, got: %s", out)
	}
}

func TestDoctorInvalidYAML(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("{{invalid yaml"), 0644)

	out, _ := runCmd(t, "doctor", dir, "-o", "table")
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
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	out, _ := runCmd(t, "doctor", dir, "-o", "table")
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
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	out, err := runCmd(t, "doctor", dir, "-o", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result DoctorResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
	if result.Kind != "rootline/doctor" {
		t.Errorf("expected kind rootline/doctor, got %s", result.Kind)
	}
}

func TestDoctorEnumValues(t *testing.T) {
	dir := t.TempDir()
	stemContent := `version: 1
schema:
  estado:
    type: enum
    values: [OnlyOne]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	out, _ := runCmd(t, "doctor", dir, "-o", "table")
	if !strings.Contains(out, "enum-values") {
		t.Errorf("expected enum-values warning, got: %s", out)
	}
}

func TestDoctorTypeConsistency(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	parentStem := `version: 1
schema:
  estado:
    type: string
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(parentStem), 0644)

	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	childStem := `version: 1
schema:
  estado:
    type: enum
    values: [A, B]
`
	mustWriteFile(t, filepath.Join(sub, ".stem"), []byte(childStem), 0644)

	out, _ := runCmd(t, "doctor", dir, "-o", "table")
	if !strings.Contains(out, "type-consistency") {
		t.Errorf("expected type-consistency failure, got: %s", out)
	}
}

func TestDoctorRuleFieldExists(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	stemContent := `version: 1
schema:
  estado:
    type: string
validate:
  - rule: non_empty
    field: nonexistent_field
    severity: error
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	out, _ := runCmd(t, "doctor", dir, "-o", "table")
	if !strings.Contains(out, "rule-field-exists") {
		t.Errorf("expected rule-field-exists warning, got: %s", out)
	}
}

func TestDoctorFieldOverride(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	parentStem := `version: 1
schema:
  estado:
    type: string
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(parentStem), 0644)

	sub := filepath.Join(dir, "sub")
	if err := os.MkdirAll(sub, 0755); err != nil {
		t.Fatal(err)
	}
	childStem := `version: 1
schema:
  estado:
    type: string
    required: true
`
	mustWriteFile(t, filepath.Join(sub, ".stem"), []byte(childStem), 0644)

	out, _ := runCmd(t, "doctor", dir, "-o", "table")
	if !strings.Contains(out, "field-override") {
		t.Errorf("expected field-override warning, got: %s", out)
	}
}

func TestDoctorTableWithFailures(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("{{invalid yaml"), 0644)

	out, _ := runCmd(t, "doctor", dir, "-o", "table")
	if !strings.Contains(out, "failed") {
		t.Errorf("expected 'failed' in summary, got: %s", out)
	}
}

func TestDoctorNoStems(t *testing.T) {
	dir := t.TempDir()

	out, _ := runCmd(t, "doctor", dir, "-o", "table")
	if !strings.Contains(out, "no .stem") {
		t.Errorf("expected no .stem warning, got: %s", out)
	}
}
