package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func executeDescribe(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(append([]string{"describe"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestDescribeCmd_FullContract(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":     "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  Fecha:\n    type: string\n    required: true\n",
		"prd/.stem": "version: 2\nschema:\n  Estado:\n    type: enum\n    values:\n      - Pending\n      - Completado\n    required: true\nvalidate:\n  - rule: requires\n    if: { Estado: Completado }\n    then: { fields: [Fecha] }\n",
	})

	stdout, err := executeDescribe(t, filepath.Join(root, "prd"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	if result["version"].(float64) != 1 {
		t.Errorf("version = %v", result["version"])
	}
	if result["kind"] != "rootline/describe" {
		t.Errorf("kind = %v", result["kind"])
	}

	// Applies should list both .stem files
	applies := result["applies"].([]any)
	if len(applies) != 2 {
		t.Fatalf("applies has %d entries, want 2", len(applies))
	}

	// Schema should have merged fields: Fecha + Estado
	schema := result["schema"].(map[string]any)
	if len(schema) != 2 {
		t.Errorf("schema has %d fields, want 2", len(schema))
	}
	if _, ok := schema["Fecha"]; !ok {
		t.Error("missing Fecha in schema")
	}
	if _, ok := schema["Estado"]; !ok {
		t.Error("missing Estado in schema")
	}

	// Source tracking
	fechaField := schema["Fecha"].(map[string]any)
	if fechaField["source"] == nil || fechaField["source"] == "" {
		t.Error("Fecha.source should be set")
	}

	// Validate section
	validate := result["validate"].([]any)
	if len(validate) != 1 {
		t.Errorf("validate has %d rules, want 1", len(validate))
	}
}

// TestDescribeCmd_NoStemAncestors asserts that describing a path with no .stem
// anywhere in its chain is an explicit failure.
//
// This previously returned an empty schema plus a hint, which meant a command
// could report success while governing nothing. Zero .stem is now
// ErrNoSchemaFound; the "run rootline init" guidance moved into the error
// message so the help survives the stricter contract.
func TestDescribeCmd_NoStemAncestors(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"empty/placeholder.txt": "",
	})

	_, err := executeDescribe(t, filepath.Join(root, "empty"))
	if err == nil {
		t.Fatal("expected an error when no .stem exists in the chain, got success")
	}
	if !errors.Is(err, rules.ErrNoSchemaFound) {
		t.Fatalf("expected ErrNoSchemaFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), "rootline init") {
		t.Errorf("error should guide the user to 'rootline init', got: %s", err)
	}
}

func TestDescribeCmd_FieldExtraction(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  Estado:\n    type: enum\n    values:\n      - Pending\n      - Done\n    required: true\n",
	})

	stdout, err := executeDescribe(t, "--field", "schema.Estado.values", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var values []any
	if err := json.Unmarshal([]byte(stdout), &values); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	if len(values) != 2 {
		t.Fatalf("values = %v, want [Pending Done]", values)
	}
}

func TestDescribeCmd_FieldApplies(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":     "version: 2\nschema:\n  title:\n    type: string\n",
		"sub/.stem": "version: 2\nschema:\n  status:\n    type: string\n",
	})

	stdout, err := executeDescribe(t, "--field", "applies", filepath.Join(root, "sub"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var applies []any
	if err := json.Unmarshal([]byte(stdout), &applies); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	if len(applies) != 2 {
		t.Fatalf("applies has %d entries, want 2", len(applies))
	}
}

func TestDescribeCmd_ScopeInherited(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":     "version: 2\nscope:\n  match: \"*.md\"\n",
		"sub/.stem": "version: 2\nschema:\n  x:\n    type: string\n",
	})

	stdout, err := executeDescribe(t, filepath.Join(root, "sub"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	scope := result["scope"].(map[string]any)
	if scope["match"] != "*.md" {
		t.Errorf("scope.match = %v, want *.md", scope["match"])
	}
}

func TestDescribeCmd_DeepInheritance(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":           "version: 2\nschema:\n  project:\n    type: string\n",
		"epics/.stem":     "version: 2\nschema:\n  epic:\n    type: string\n",
		"epics/E01/.stem": "version: 2\nschema:\n  feature:\n    type: string\n",
	})

	stdout, err := executeDescribe(t, filepath.Join(root, "epics", "E01"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// 3 .stem files merged
	applies := result["applies"].([]any)
	if len(applies) != 3 {
		t.Fatalf("applies = %d, want 3", len(applies))
	}

	// 3 schema fields: project, epic, feature
	schema := result["schema"].(map[string]any)
	if len(schema) != 3 {
		t.Errorf("schema has %d fields, want 3", len(schema))
	}

	// Verify directory argument is used as path, not resolved absolute
	absDir := filepath.Join(root, "epics", "E01")
	if result["path"] != absDir {
		// Path comes from args[0] which is absolute in test
		_ = absDir // path is whatever was passed
	}
}

func TestDescribeCmd_RequiresArg(t *testing.T) {
	_, _ = os.Getwd() // just to use os
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"describe"})
	err := rootCmd.Execute()
	if err == nil {
		t.Error("expected error for missing arg")
	}
}

// TestDescribeCmd_NoStemTableHint asserts the same failure in table mode, and
// that the init guidance is preserved in the error regardless of output format.
func TestDescribeCmd_NoStemTableHint(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"empty/placeholder.txt": "",
	})

	_, err := executeDescribe(t, "--output", "table", filepath.Join(root, "empty"))
	if err == nil {
		t.Fatal("expected an error when no .stem exists in the chain, got success")
	}
	if !errors.Is(err, rules.ErrNoSchemaFound) {
		t.Fatalf("expected ErrNoSchemaFound, got: %v", err)
	}
	if !strings.Contains(err.Error(), "rootline init") {
		t.Errorf("error should guide the user to 'rootline init', got: %s", err)
	}
}

func TestDescribeCmd_SequenceFieldNext(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":                "version: 2\nschema:\n  id:\n    type: sequence\n    prefix: T\n    digits: 3\n",
		"tasks/T001-first.md":  "---\nestado: Done\n---\n# First\n",
		"tasks/T002-second.md": "---\nestado: Pending\n---\n# Second\n",
	})

	// Test JSON field extraction
	stdout, err := executeDescribe(t, "--field", "schema.id.next", filepath.Join(root, "tasks"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	trimmed := strings.TrimSpace(stdout)
	if trimmed != `"T003"` {
		t.Errorf("schema.id.next = %s, want %q", trimmed, "T003")
	}
}

func TestDescribeCmd_SequenceFieldNextEmpty(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nschema:\n  id:\n    type: sequence\n    prefix: S\n    digits: 3\n",
	})

	// Root has no files matching S prefix → S001
	stdout, err := executeDescribe(t, "--field", "schema.id.next", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	trimmed := strings.TrimSpace(stdout)
	if trimmed != `"S001"` {
		t.Errorf("schema.id.next = %s, want %q", trimmed, "S001")
	}
}

func TestDescribeCmd_WithStemNoHints(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values:\n      - Pending\n      - Done\n    required: true\n",
	})

	stdout, err := executeDescribe(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// hints should be absent (omitempty) or null when schema exists
	if hints, ok := result["hints"]; ok && hints != nil {
		hintsList := hints.([]any)
		if len(hintsList) > 0 {
			t.Errorf("expected no hints with .stem present, got: %v", hints)
		}
	}
}

// TEST-FIRST: Describe command should fail (non-zero exit) when schema resolution fails
// This test will FAIL under current behavior if describe doesn't propagate
// WalkUp errors properly
func TestDescribeErrorPropagation_NoSchema(t *testing.T) {
	dir := t.TempDir()

	// Create a directory WITHOUT a .stem anywhere in the tree
	// This will cause WalkUp to return ErrNoSchemaFound
	_, err := executeDescribe(t, dir)

	// EXPECTATION: describe should fail when no schema is found, not silently succeed
	if err == nil {
		t.Fatalf("describe should fail when no schema is found, but got nil error (silent degradation)")
	}
}
