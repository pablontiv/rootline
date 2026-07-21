package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupValidateProject creates a temp directory with .git marker,
// .stem files, and markdown files for validate command testing.
func setupValidateProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	for relPath, content := range files {
		absPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	declareTestBoundary(t, root)
	return root
}

// declareTestBoundary marks the fixture root as a governance boundary.
//
// Every fixture models a real project, and real projects declare where they
// start. Without the marker the walk climbs past the temp directory toward the
// filesystem root, and the boundary preflight fails the command before it runs.
// It appends to an existing root .stem rather than creating one, so the chain
// keeps exactly the layers the test intends. If no .stem exists, it creates one.
func declareTestBoundary(t *testing.T, root string) {
	t.Helper()

	// Mark the outermost .stem the fixture actually created — usually the one
	// at the root, but some fixtures put their only .stem in a subdirectory,
	// and that subtree is then what declares itself.
	stemPath := shallowestStem(root)
	if stemPath == "" {
		// Never manufacture a .stem. Several tests deliberately build a tree
		// with none in order to assert ErrNoSchemaFound, and creating one here
		// would silently convert them into a different scenario.
		return
	}

	content, err := os.ReadFile(stemPath)
	if err != nil {
		return
	}
	if strings.Contains(string(content), "root:") {
		return
	}
	// #nosec G703 -- stemPath is built from the test's own t.TempDir() tree.
	if err := os.WriteFile(stemPath, append([]byte("root: true\n"), content...), 0o644); err != nil {
		t.Fatal(err)
	}
}

// shallowestStem returns the path of the .stem closest to root, or "" when the
// tree has none. Marking a deeper one would truncate the chain and silently
// change what the test is exercising.
func shallowestStem(root string) string {
	if _, err := os.Stat(filepath.Join(root, ".stem")); err == nil {
		return filepath.Join(root, ".stem")
	}

	queue := []string{root}
	for len(queue) > 0 {
		dir := queue[0]
		queue = queue[1:]

		items, err := os.ReadDir(dir)
		if err != nil {
			continue
		}
		var subdirs []string
		for _, item := range items {
			if item.Name() == ".stem" {
				return filepath.Join(dir, ".stem")
			}
			if item.IsDir() && item.Name() != ".git" {
				subdirs = append(subdirs, filepath.Join(dir, item.Name()))
			}
		}
		queue = append(queue, subdirs...)
	}
	return ""
}

// executeValidate runs the validate command with args and returns
// stdout, the error, and whether it failed due to validation errors.
func executeValidate(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Reset global flags to defaults
	resetFlags()

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(append([]string{"validate"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestValidateCmd_SingleFileValid(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"doc.md": "---\ntitle: Hello\n---\n# Hello",
	})

	stdout, err := executeValidate(t, filepath.Join(root, "doc.md"))
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
	if result["kind"] != "rootline/validate" {
		t.Errorf("kind = %v", result["kind"])
	}
	if result["valid"] != true {
		t.Errorf("valid = %v, want true", result["valid"])
	}
}

func TestValidateCmd_SingleFileInvalid(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"doc.md": "---\nstatus: draft\n---\n# No title",
	})

	stdout, err := executeValidate(t, filepath.Join(root, "doc.md"))
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	if result["valid"] != false {
		t.Errorf("valid = %v, want false", result["valid"])
	}
	errors := result["errors"].([]any)
	if len(errors) == 0 {
		t.Error("expected validation errors")
	}
}

func TestValidateCmd_AllMode(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":     "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"valid.md":  "---\ntitle: Good\n---\n",
		"broken.md": "---\nstatus: draft\n---\n",
	})

	// Change to project dir for --all mode
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all")
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	if result["kind"] != "rootline/validate-batch" {
		t.Errorf("kind = %v, want rootline/validate-batch", result["kind"])
	}
	summary := result["summary"].(map[string]any)
	if summary["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", summary["total"])
	}
	if summary["valid"].(float64) != 1 {
		t.Errorf("valid = %v, want 1", summary["valid"])
	}
	if summary["invalid"].(float64) != 1 {
		t.Errorf("invalid = %v, want 1", summary["invalid"])
	}
}

func TestValidateCmd_FieldExtraction(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"doc.md": "---\nstatus: draft\n---\n# No title",
	})

	stdout, err := executeValidate(t, "--field", "errors", filepath.Join(root, "doc.md"))
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v", err)
	}

	// Should be a JSON array of errors
	var errors []any
	if err := json.Unmarshal([]byte(stdout), &errors); err != nil {
		t.Fatalf("invalid JSON array: %v\noutput: %s", err, stdout)
	}
	if len(errors) == 0 {
		t.Error("expected errors in extracted field")
	}
}

func TestValidateCmd_FieldExtractionValid(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"doc.md": "---\ntitle: Hello\n---\n",
	})

	stdout, err := executeValidate(t, "--field", "valid", filepath.Join(root, "doc.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "true\n" {
		t.Errorf("output = %q, want true", stdout)
	}
}

func TestValidateCmd_NoArgs(t *testing.T) {
	_, err := executeValidate(t)
	if err == nil {
		t.Error("expected error for no args")
	}
}

func TestValidateCmd_MultipleFiles(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"a.md":  "---\ntitle: A\n---\n",
		"b.md":  "---\ntitle: B\n---\n",
	})

	stdout, err := executeValidate(t,
		filepath.Join(root, "a.md"),
		filepath.Join(root, "b.md"),
	)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Multiple files → batch result
	if result["kind"] != "rootline/validate-batch" {
		t.Errorf("kind = %v, want rootline/validate-batch", result["kind"])
	}
}

func TestValidateCmd_FileNotFound(t *testing.T) {
	_, err := executeValidate(t, "/nonexistent/path/doc.md")
	if err == nil {
		t.Error("expected error for nonexistent file")
	}
}

func TestValidateCmd_NoExtractor(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"data.json": `{"key": "value"}`,
	})

	_, err := executeValidate(t, filepath.Join(root, "data.json"))
	if err == nil {
		t.Error("expected error for unsupported file type")
	}
}

func TestValidateCmd_FieldExtractionNested(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"doc.md": "---\ntitle: Hello\n---\n",
	})

	stdout, err := executeValidate(t, "--field", "kind", filepath.Join(root, "doc.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stdout != "\"rootline/validate\"\n" {
		t.Errorf("output = %q, want \"rootline/validate\"", stdout)
	}
}

func TestValidateCmd_FieldExtractionNotFound(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"doc.md": "---\ntitle: Hello\n---\n",
	})

	_, err := executeValidate(t, "--field", "nonexistent.deep.path", filepath.Join(root, "doc.md"))
	if err == nil {
		t.Error("expected error for nonexistent field path")
	}
}

func TestSplitDotPath(t *testing.T) {
	tests := []struct {
		input    string
		expected []string
	}{
		{"errors", []string{"errors"}},
		{"summary.total", []string{"summary", "total"}},
		{"a.b.c", []string{"a", "b", "c"}},
		{"", nil},
	}
	for _, tt := range tests {
		got := splitDotPath(tt.input)
		if len(got) != len(tt.expected) {
			t.Errorf("splitDotPath(%q) = %v, want %v", tt.input, got, tt.expected)
			continue
		}
		for i := range got {
			if got[i] != tt.expected[i] {
				t.Errorf("splitDotPath(%q)[%d] = %q, want %q", tt.input, i, got[i], tt.expected[i])
			}
		}
	}
}

func TestValidateCmd_AllWhere(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":     "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n  tipo:\n    type: string\n",
		"good.md":   "---\nestado: Pending\ntipo: test\n---\n",
		"bad.md":    "---\nestado: Completed\n---\n",
		"broken.md": "---\ntipo: test\n---\n",
	})

	mustChdir(t, root)

	// Filter to only tipo=test records (good.md and broken.md)
	stdout, err := executeValidate(t, "--all", "--where", "tipo == 'test'")
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed (broken.md missing estado), got: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	summary := result["summary"].(map[string]any)
	// Should validate 2 files (good.md + broken.md), not 3
	if summary["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2 (filtered)", summary["total"])
	}
}

func TestValidateCmd_AllWhereInvalid(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n",
		"doc.md": "---\ntitle: Test\n---\n",
	})

	mustChdir(t, root)
	_, err := executeValidate(t, "--all", "--where", "== bad syntax")
	if err == nil {
		t.Fatal("expected error for invalid where expression")
	}
}

func TestValidateCmd_EnumError(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values:\n      - Pending\n      - Completado\n    required: true\n",
		"doc.md": "---\nestado: Invalid\n---\n",
	})

	stdout, err := executeValidate(t, filepath.Join(root, "doc.md"))
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	errors := result["errors"].([]any)
	foundEnum := false
	for _, e := range errors {
		errMap := e.(map[string]any)
		if errMap["rule"] == "enum" && errMap["field"] == "estado" {
			foundEnum = true
		}
	}
	if !foundEnum {
		t.Error("expected enum validation error for estado")
	}
}

func TestValidateAll_StemHealthChecks(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":      "version: 2\nschema:\n  estado:\n    type: string\n",
		"sub/.stem":  "version: 2\nschema:\n  estado:\n    type: enum\n    values: [A, B]\n",
		"sub/doc.md": "---\nestado: A\n---\n",
	})

	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all")
	// type-consistency failure should cause validation failed
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v\noutput: %s", err, stdout)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// Check that stem health errors appear in results
	results := result["results"].([]any)
	foundStemHealth := false
	for _, r := range results {
		rm := r.(map[string]any)
		errs, ok := rm["errors"].([]any)
		if !ok {
			continue
		}
		for _, e := range errs {
			em := e.(map[string]any)
			if em["source"] == "stem-health" && em["rule"] == "type-consistency" {
				foundStemHealth = true
			}
		}
	}
	if !foundStemHealth {
		t.Errorf("expected stem-health type-consistency error in results, got: %s", stdout)
	}
}

func TestValidateAll_StemHealthWarnings(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.xyz\"\nschema:\n  estado:\n    type: enum\n    values: [OnlyOne]\n",
		"doc.md": "---\nestado: OnlyOne\n---\n",
	})

	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, stdout)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// Check that warnings appear
	results := result["results"].([]any)
	foundScopeWarn := false
	foundEnumWarn := false
	for _, r := range results {
		rm := r.(map[string]any)
		warns, ok := rm["warnings"].([]any)
		if !ok {
			continue
		}
		for _, w := range warns {
			wm := w.(map[string]any)
			if wm["rule"] == "scope-match" {
				foundScopeWarn = true
			}
			if wm["rule"] == "enum-values" {
				foundEnumWarn = true
			}
		}
	}
	if !foundScopeWarn {
		t.Errorf("expected scope-match warning in results, got: %s", stdout)
	}
	if !foundEnumWarn {
		t.Errorf("expected enum-values warning in results, got: %s", stdout)
	}
}
