package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
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
	return root
}

// executeValidate runs the validate command with args and returns
// stdout, the error, and whether it failed due to validation errors.
func executeValidate(t *testing.T, args ...string) (string, error) {
	t.Helper()

	// Reset global flags to defaults
	fieldPath = nil
	validateAll = false

	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(append([]string{"validate"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

func TestValidateCmd_SingleFileValid(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
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
		".stem":  "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
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
		".stem":     "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
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
		".stem":  "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
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
		".stem":  "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
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
		".stem": "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
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
		".stem":  "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
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
		".stem":  "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
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

func TestValidateCmd_EnumError(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values:\n      - Pending\n      - Completado\n    required: true\n",
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
