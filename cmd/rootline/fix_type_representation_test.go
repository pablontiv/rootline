package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupTypeRepresentationFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	stem := `version: 2
root: true
scope:
  match: "*.md"
schema:
  date:
    type: string
  boolean:
    type: string
  integer:
    type: string
  object:
    type: string
`
	doc := `---
date: 2026-06-22T00:00:00Z
boolean: TRUE
integer: 042
object:
  nested: value
---
# Probe
`
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(doc), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestFixAllDryRunReportsSafeTypeRepairsAndUnsupportedFindings(t *testing.T) {
	dir := setupTypeRepresentationFixture(t)
	before, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	out, err := runCmd(t, "fix", "--all", dir, "--dry-run")
	if err != nil {
		t.Fatalf("dry-run: %v\n%s", err, out)
	}
	var report struct {
		Proposals    []map[string]any `json:"proposals"`
		TypeFindings []map[string]any `json:"type_findings"`
	}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("JSON: %v\n%s", err, out)
	}
	if len(report.Proposals) != 3 || len(report.TypeFindings) != 1 {
		t.Fatalf("proposals=%d findings=%d output=%s", len(report.Proposals), len(report.TypeFindings), out)
	}
	if !strings.Contains(out, `"from_representation":"timestamp"`) ||
		!strings.Contains(out, `"from_representation":"boolean"`) ||
		!strings.Contains(out, `"from_representation":"integer"`) {
		t.Fatalf("missing typed proposal evidence: %s", out)
	}
	after, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(after) != string(before) {
		t.Fatal("dry-run modified the document")
	}
}

func TestFixAllAppliesExactTypeRepairsAndRemainsIdempotent(t *testing.T) {
	dir := setupTypeRepresentationFixture(t)
	firstOut, err := runCmd(t, "fix", "--all", dir)
	if err != nil {
		t.Fatalf("first fix: %v\n%s", err, firstOut)
	}
	firstBytes, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, quoted := range []string{`date: "2026-06-22T00:00:00Z"`, `boolean: "TRUE"`, `integer: "042"`} {
		if !strings.Contains(string(firstBytes), quoted) {
			t.Errorf("missing %s:\n%s", quoted, firstBytes)
		}
	}
	if !strings.Contains(firstOut, "type_findings") || !strings.Contains(firstOut, "mapping") {
		t.Fatalf("applied run hid unsupported finding: %s", firstOut)
	}

	secondOut, err := runCmd(t, "fix", "--all", dir)
	if err != nil {
		t.Fatalf("second fix: %v\n%s", err, secondOut)
	}
	secondBytes, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(secondBytes) != string(firstBytes) {
		t.Fatal("second fix was not idempotent")
	}
	if strings.Contains(secondOut, "from_representation") {
		t.Fatalf("second run proposed already-applied scalar repairs: %s", secondOut)
	}
	if !strings.Contains(secondOut, "type_findings") {
		t.Fatalf("unsupported mapping finding disappeared: %s", secondOut)
	}
}

func TestFixAllTypeFindingsTable(t *testing.T) {
	dir := setupTypeRepresentationFixture(t)
	for _, tc := range []struct {
		name string
		args []string
	}{
		{
			name: "dry-run",
			args: []string{"fix", "--all", dir, "--dry-run", "--output", "table"},
		},
		{
			name: "apply",
			args: []string{"fix", "--all", dir, "--output", "table"},
		},
	} {
		out, err := runCmd(t, tc.args...)
		if err != nil {
			t.Fatalf("table %s: %v\n%s", tc.name, err, out)
		}
		for _, want := range []string{
			"Type findings: 1 (reported, not repaired)",
			"a.md", "object", "mapping",
		} {
			if !strings.Contains(out, want) {
				t.Errorf("table %s output missing %q:\n%s", tc.name, want, out)
			}
		}
	}
}

func TestFixTypeFindingsDoNotReplaceValidateExitContract(t *testing.T) {
	dir := t.TempDir()
	stem := "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  object:\n    type: string\n"
	doc := "---\nobject:\n  nested: value\n---\n# Probe\n"
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(doc), 0o644); err != nil {
		t.Fatal(err)
	}
	for _, args := range [][]string{
		{"fix", "--all", dir, "--dry-run"},
		{"fix", "--all", dir},
	} {
		out, err := runCmd(t, args...)
		if err != nil {
			t.Fatalf("%v returned failure: %v\n%s", args, err, out)
		}
		if !strings.Contains(out, "type_findings") {
			t.Fatalf("%v hid finding: %s", args, out)
		}
	}
	out, err := runCmd(t, "validate", "--all", dir)
	if err == nil {
		t.Fatalf("validate unexpectedly accepted mapping-to-string mismatch: %s", out)
	}
}
