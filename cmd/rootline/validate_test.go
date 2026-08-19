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

	env := decodeEnvelope(t, stdout)
	if env["version"].(float64) != 2 {
		t.Errorf("version = %v", env["version"])
	}
	if env["kind"] != "rootline/validate-batch" {
		t.Errorf("kind = %v", env["kind"])
	}
	if result := firstResult(t, stdout); result["valid"] != true {
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

	result := firstResult(t, stdout)
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

	stdout, err := executeValidate(t, "--field", "results[].errors", filepath.Join(root, "doc.md"))
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v", err)
	}

	// Should be one error array per record
	var perRecord [][]any
	if err := json.Unmarshal([]byte(stdout), &perRecord); err != nil {
		t.Fatalf("invalid JSON array: %v\noutput: %s", err, stdout)
	}
	if len(perRecord) != 1 || len(perRecord[0]) == 0 {
		t.Errorf("expected errors in extracted field, got %v", perRecord)
	}
}

func TestValidateCmd_FieldExtractionValid(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"doc.md": "---\ntitle: Hello\n---\n",
	})

	stdout, err := executeValidate(t, "--field", "results[].valid", filepath.Join(root, "doc.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stdout != "[true]\n" {
		t.Errorf("output = %q, want [true]", stdout)
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
	if stdout != "\"rootline/validate-batch\"\n" {
		t.Errorf("output = %q, want \"rootline/validate-batch\"", stdout)
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

	errors := firstResult(t, stdout)["errors"].([]any)
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

func TestValidateAll_StemHealthAcceptsValidNarrowing(t *testing.T) {
	tests := []struct {
		name      string
		rootStem  string
		childStem string
	}{
		{
			name:      "string to enum",
			rootStem:  "version: 2\nschema:\n  estado:\n    type: string\n",
			childStem: "version: 2\nschema:\n  estado:\n    type: enum\n    values: [A, B]\n",
		},
		{
			name:      "enum subset",
			rootStem:  "version: 2\nschema:\n  estado:\n    type: enum\n    values: [A, B, C]\n",
			childStem: "version: 2\nschema:\n  estado:\n    type: enum\n    values: [A, B]\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":      tt.rootStem,
				"sub/.stem":  tt.childStem,
				"sub/doc.md": "---\nestado: A\n---\n",
			})

			mustChdir(t, root)

			stdout, err := executeValidate(t, "--all")
			if err != nil {
				t.Fatalf("unexpected error: %v\noutput: %s", err, stdout)
			}

			env := decodeEnvelope(t, stdout)
			for _, h := range stemHealthChecks(t, env) {
				if h["field"] == "estado" && (h["check"] == "type-consistency" || h["check"] == "field-override" || h["check"] == "monotonic-violations") {
					t.Fatalf("unexpected stem_health compatibility noise: %#v", h)
				}
			}
			if got := env["summary"].(map[string]any)["stem_health_errors_count"].(float64); got != 0 {
				t.Errorf("stem_health_errors_count = %v, want 0", got)
			}
		})
	}
}

func TestValidateAll_StemHealthCompatibilityRejections(t *testing.T) {
	tests := []struct {
		name      string
		rootStem  string
		childStem string
		check     string
		path      string
	}{
		{
			name:      "enum to string",
			rootStem:  "version: 2\nschema:\n  estado:\n    type: enum\n    values: [A, B]\n",
			childStem: "version: 2\nschema:\n  estado:\n    type: string\n",
			check:     "type-consistency",
			path:      "sub/.stem",
		},
		{
			name:      "required loosening",
			rootStem:  "version: 2\nschema:\n  estado:\n    type: string\n    required: true\n",
			childStem: "version: 2\nschema:\n  estado:\n    type: string\n    required: false\n",
			check:     "monotonic-violations",
			path:      "sub/.stem",
		},
		{
			name:      "severity loosening",
			rootStem:  "version: 2\nschema:\n  estado:\n    type: string\n",
			childStem: "version: 2\nschema:\n  estado:\n    type: string\n    severity: warn\n",
			check:     "monotonic-violations",
			path:      "sub/.stem",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":      tt.rootStem,
				"sub/.stem":  tt.childStem,
				"sub/doc.md": "---\nestado: A\n---\n",
			})
			mustChdir(t, root)

			stdout, err := executeValidate(t, "--all")
			if err != ErrValidationFailed {
				t.Fatalf("err = %v, want ErrValidationFailed\noutput: %s", err, stdout)
			}

			env := decodeEnvelope(t, stdout)
			var matches []map[string]any
			for _, h := range stemHealthChecks(t, env) {
				if h["field"] == "estado" {
					matches = append(matches, h)
				}
			}
			if len(matches) != 1 {
				t.Fatalf("stem_health diagnostics for estado = %d, want 1: %#v", len(matches), matches)
			}
			if matches[0]["check"] != tt.check || matches[0]["severity"] != "error" || matches[0]["path"] != tt.path {
				t.Fatalf("stem_health diagnostic = %#v, want check=%q severity=error path=%q", matches[0], tt.check, tt.path)
			}
			if got := env["summary"].(map[string]any)["stem_health_errors_count"].(float64); got != 1 {
				t.Fatalf("stem_health_errors_count = %v, want 1", got)
			}
		})
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

	env := decodeEnvelope(t, stdout)
	foundScopeWarn := false
	for _, h := range stemHealthChecks(t, env) {
		if h["check"] == "scope-match" && h["severity"] == "warn" {
			foundScopeWarn = true
		}
		if h["check"] == "enum-values" {
			t.Fatalf("unexpected enum-values warning in stem_health: %#v", h)
		}
	}
	if !foundScopeWarn {
		t.Errorf("expected scope-match warning in stem_health, got: %s", stdout)
	}
}

// TestValidateAll_NoSchemaAnywhere tests that validate --all exits non-zero
// when no .stem files exist in the tree.
func TestValidateAll_NoSchemaAnywhere(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// Create a markdown file but NO .stem files anywhere
	mdFile := filepath.Join(root, "doc.md")
	if err := os.WriteFile(mdFile, []byte("---\ntitle: Test\n---\n# Test"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Note: deliberately NOT calling declareTestBoundary so there is no .stem

	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all")
	// Should exit with error since no schema is found
	if err == nil {
		t.Fatalf("expected error when no .stem files exist, got success\noutput: %s", stdout)
	}
}

// TestValidateAll_HardErrorStructuralValidation tests that validate --all
// propagates hard errors from structural validation (line ~199 in validate.go).
func TestValidateAll_HardErrorStructuralValidation(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a valid .stem at root with structural rules
	if err := os.WriteFile(
		filepath.Join(root, ".stem"),
		[]byte("version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\nstructural:\n  require_index: true\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory
	subDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a markdown file in subdirectory so it gets validated
	// This will add sub to visitedDirs
	if err := os.WriteFile(
		filepath.Join(subDir, "doc.md"),
		[]byte("---\ntitle: Test\n---\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Write an unparseable STEM file in subdirectory
	// This will cause WalkUp to fail during structural validation phase
	if err := os.WriteFile(
		filepath.Join(subDir, ".stem"),
		[]byte("version: 2\ninvalid: [unclosed\nschema:\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all")
	// Should fail due to unparseable .stem during structural validation
	if err == nil {
		t.Fatalf("expected error when .stem is unparseable during structural validation, got success\noutput: %s", stdout)
	}
}

// TestValidateAll_HardErrorDriftDetection tests that validate --all
// propagates hard errors from drift detection (line ~225 in validate.go).
func TestValidateAll_HardErrorDriftDetection(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Create a valid .stem at root
	if err := os.WriteFile(
		filepath.Join(root, ".stem"),
		[]byte("version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Create a subdirectory with markdown files for drift detection
	subDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create an index file (README.md) - this triggers drift detection for this parent
	if err := os.WriteFile(
		filepath.Join(subDir, "README.md"),
		[]byte("---\ntitle: Index\n---\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Create a child file - this makes parentChildren[subDir] non-empty
	if err := os.WriteFile(
		filepath.Join(subDir, "child.md"),
		[]byte("---\ntitle: Child\n---\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	// Create an unparseable .stem in the subdirectory
	// This will cause WalkUp to fail during drift detection phase
	if err := os.WriteFile(
		filepath.Join(subDir, ".stem"),
		[]byte("version: 2\ninvalid: [unclosed\nschema:\n"),
		0o644,
	); err != nil {
		t.Fatal(err)
	}

	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all")
	// Should fail due to unparseable .stem encountered during drift detection
	if err == nil {
		t.Fatalf("expected error during drift detection when .stem is unparseable, got success\noutput: %s", stdout)
	}
}

// TestValidate_DirectoryArgumentError names the mistake instead of its
// consequence.
//
// `validate docs` is a plausible thing to type — `--all` is the flag that is
// easy to forget — and the answer was "no extractor for docs", which describes
// an extractor-registry miss and sends the reader looking for a missing file
// type. The directory is the actual condition, and the flag that handles it is
// one word away.
func TestValidate_DirectoryArgumentError(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":        "version: 2\nscope:\n  match: \"*.md\"\n",
		"docs/a.md":    "---\ntitulo: x\n---\n# x\n",
		"docs/b.md":    "---\ntitulo: y\n---\n# y\n",
		"CHANGELOG.md": "---\ntitulo: z\n---\n# z\n",
	})
	target := filepath.Join(root, "docs")

	_, err := runCmd(t, "validate", target)
	if err == nil {
		t.Fatal("validate must reject a directory argument")
	}
	want := target + " is a directory; use 'rootline validate --all " + target + "'"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
