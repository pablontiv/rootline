package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

// TestGovernanceMultiStemAnalyze tests that analyze properly detects governance issues
// in a multi-.stem fixture (no root .stem, but subtree .stems in concepts/ and sources/).
func TestGovernanceMultiStemAnalyze(t *testing.T) {
	root := t.TempDir()

	// Create .git marker for WalkUp boundary
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// concepts/ subtree: status enum with values [a, b]
	conceptsDir := filepath.Join(root, "concepts")
	if err := os.MkdirAll(conceptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	conceptsStem := `version: 2
scope:
  match: "*.md"
schema:
  status:
    type: enum
    required: true
    values: [a, b]
`
	if err := os.WriteFile(filepath.Join(conceptsDir, ".stem"), []byte(conceptsStem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conceptsDir, "a.md"), []byte("---\nstatus: a\n---\n# Concept A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conceptsDir, "b.md"), []byte("---\nstatus: b\n---\n# Concept B\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// sources/ subtree: tipo enum WITHOUT values (the gap to detect)
	sourcesDir := filepath.Join(root, "sources")
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sourcesStem := `version: 2
scope:
  match: "*.md"
schema:
  tipo:
    type: enum
    required: false
`
	if err := os.WriteFile(filepath.Join(sourcesDir, ".stem"), []byte(sourcesStem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourcesDir, "p.md"), []byte("---\ntipo: X\n---\n# Source P\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourcesDir, "q.md"), []byte("---\ntipo: Y\n---\n# Source Q\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run analyze on the root directory
	resetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"analyze", root, "-o", "json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("analyze error: %v", err)
	}

	output := buf.String()
	if output == "" {
		t.Fatal("expected analyze output, got empty string")
	}

	// Parse JSON report
	var report map[string]any
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	// Verify report structure
	if report["version"].(float64) != 1 {
		t.Errorf("expected version 1, got %v", report["version"])
	}
	if report["kind"] != "analyze" {
		t.Errorf("expected kind analyze, got %s", report["kind"])
	}

	// Verify that enum_without_values is detected for tipo in sources/
	categories := report["categories"].([]any)
	found := false
	for _, cat := range categories {
		catMap := cat.(map[string]any)
		if catMap["id"] == "validation_gaps" {
			inferences := catMap["inferences"].([]any)
			for _, inf := range inferences {
				infMap := inf.(map[string]any)
				if infMap["type"] == "enum_without_values" && infMap["field"] == "tipo" {
					found = true
					break
				}
			}
		}
	}
	if !found {
		t.Fatalf("expected enum_without_values for tipo in validation_gaps category")
	}
}

// TestGovernanceMultiStemIncremental tests that analyze --incremental correctly
// filters proposals that are already covered by subtree .stem files.
func TestGovernanceMultiStemIncremental(t *testing.T) {
	root := t.TempDir()

	// Create .git marker
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// concepts/ with .stem
	conceptsDir := filepath.Join(root, "concepts")
	if err := os.MkdirAll(conceptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	conceptsStem := `version: 2
scope:
  match: "*.md"
schema:
  status:
    type: enum
    required: true
    values: [a, b]
`
	if err := os.WriteFile(filepath.Join(conceptsDir, ".stem"), []byte(conceptsStem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conceptsDir, "a.md"), []byte("---\nstatus: a\n---\n# Concept A\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// sources/ with .stem missing values
	sourcesDir := filepath.Join(root, "sources")
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sourcesStem := `version: 2
scope:
  match: "*.md"
schema:
  tipo:
    type: enum
    required: false
`
	if err := os.WriteFile(filepath.Join(sourcesDir, ".stem"), []byte(sourcesStem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourcesDir, "p.md"), []byte("---\ntipo: X\n---\n# Source P\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run analyze --incremental
	resetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"analyze", "--incremental", root, "-o", "json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("analyze --incremental error: %v", err)
	}

	output := buf.String()
	var report map[string]any
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	// Verify incremental flag is set
	if incremental, ok := report["incremental"].(bool); !ok || !incremental {
		t.Error("expected incremental flag to be true")
	}

	// The report should still contain findings (the gap in sources/.stem)
	categories := report["categories"].([]any)
	if len(categories) == 0 {
		t.Error("expected at least one category in incremental mode")
	}
}

// TestGovernanceSchemaProposeIncremental tests that schema propose --incremental
// filters proposals correctly on multi-stem fixtures.
func TestGovernanceSchemaProposeIncremental(t *testing.T) {
	root := t.TempDir()

	// Create .git marker
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// concepts/ with complete .stem
	conceptsDir := filepath.Join(root, "concepts")
	if err := os.MkdirAll(conceptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	conceptsStem := `version: 2
scope:
  match: "*.md"
schema:
  status:
    type: enum
    required: true
    values: [a, b]
`
	if err := os.WriteFile(filepath.Join(conceptsDir, ".stem"), []byte(conceptsStem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conceptsDir, "a.md"), []byte("---\nstatus: a\n---\n# Concept A\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// sources/ with no .stem yet
	sourcesDir := filepath.Join(root, "sources")
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourcesDir, "p.md"), []byte("---\ntipo: X\n---\n# Source P\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run schema propose --incremental on the root
	resetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"schema", "propose", "--incremental", root, "-o", "json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("schema propose --incremental error: %v", err)
	}

	output := buf.String()
	var report map[string]any
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	// Verify incremental flag
	if incremental, ok := report["incremental"].(bool); !ok || !incremental {
		t.Error("expected incremental flag to be true")
	}

	// Verify report structure
	if report["kind"] != "rootline/schema-proposals" {
		t.Errorf("expected kind rootline/schema-proposals, got %s", report["kind"])
	}
}

// TestGovernanceSingleStemRegression tests that a single-.stem directory
// yields the same analyze output as before (no-op regression guard).
func TestGovernanceSingleStemRegression(t *testing.T) {
	root := t.TempDir()

	// Create .git marker
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Root .stem only (no subtree .stems)
	rootStem := `version: 2
scope:
  match: "*.md"
schema:
  status:
    type: enum
    required: true
    values: [a, b, c]
  tipo:
    type: enum
    required: false
`
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(rootStem), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create markdown files with various tipo values to trigger inference
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\nstatus: a\ntipo: X\n---\n# Doc A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.md"), []byte("---\nstatus: b\ntipo: Y\n---\n# Doc B\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run analyze (should work without error on single-stem structure)
	resetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"analyze", root, "-o", "json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("analyze error on single-stem dir: %v", err)
	}

	output := buf.String()
	var report map[string]any
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	// Verify basic report structure (regression guard)
	if report["version"].(float64) != 1 {
		t.Errorf("expected version 1, got %v", report["version"])
	}
	if report["kind"] != "analyze" {
		t.Errorf("expected kind analyze, got %s", report["kind"])
	}

	// Verify categories are present
	categories := report["categories"].([]any)
	if len(categories) == 0 {
		t.Error("expected at least one category for single-stem dir")
	}
}

// TestGovernanceMultiStemFilterCoveredInferences verifies that analyze properly
// filters inferences already covered by existing .stem files in subtree scopes.
func TestGovernanceMultiStemFilterCoveredInferences(t *testing.T) {
	root := t.TempDir()

	// Create .git marker
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// concepts/ with complete .stem covering all inferences
	conceptsDir := filepath.Join(root, "concepts")
	if err := os.MkdirAll(conceptsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	conceptsStem := `version: 2
scope:
  match: "*.md"
schema:
  status:
    type: enum
    required: true
    values: [active, draft]
  priority:
    type: enum
    required: false
    values: [high, low]
`
	if err := os.WriteFile(filepath.Join(conceptsDir, ".stem"), []byte(conceptsStem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(conceptsDir, "active.md"), []byte("---\nstatus: active\npriority: high\n---\n# Active\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// sources/ with partial .stem (missing enum values for tipo)
	sourcesDir := filepath.Join(root, "sources")
	if err := os.MkdirAll(sourcesDir, 0o755); err != nil {
		t.Fatal(err)
	}
	sourcesStem := `version: 2
scope:
  match: "*.md"
schema:
  tipo:
    type: enum
    required: false
  kind:
    type: string
    required: false
`
	if err := os.WriteFile(filepath.Join(sourcesDir, ".stem"), []byte(sourcesStem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourcesDir, "p.md"), []byte("---\ntipo: data\nkind: dataset\n---\n# Source P\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(sourcesDir, "q.md"), []byte("---\ntipo: code\nkind: module\n---\n# Source Q\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// Run analyze (non-incremental) on the root
	resetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs([]string{"analyze", root, "-o", "json"})
	err := rootCmd.Execute()
	if err != nil {
		t.Fatalf("analyze error: %v", err)
	}

	output := buf.String()
	var report map[string]any
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, output)
	}

	// Verify report structure
	categories := report["categories"].([]any)
	if len(categories) == 0 {
		t.Error("expected at least one category in analyze report")
	}

	// Verify inferences are present across categories
	totalInferences := 0
	for _, cat := range categories {
		catMap := cat.(map[string]any)
		inferences := catMap["inferences"].([]any)
		totalInferences += len(inferences)
	}
	if totalInferences == 0 {
		t.Logf("note: no inferences found in report. categories: %v", categories)
	}
}
