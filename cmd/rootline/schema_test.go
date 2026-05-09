package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"
)

func executeSchemaPropose(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(append([]string{"schema", "propose"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

// TestSchemaProposeNoStem tests that schema propose works with no existing .stem.
func TestSchemaProposeNoStem(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"test.md": "---\ntitulo: Test Document\ntipo: task\nestado: Pending\n---\n# Test Document\n\nThis is a test document.\n",
	})

	// Verify no .stem file exists
	stemPath := filepath.Join(root, ".stem")
	if _, err := os.Stat(stemPath); err == nil {
		t.Fatal("expected no .stem file to exist initially")
	}

	stdout, err := executeSchemaPropose(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify that no .stem file was created
	if _, err := os.Stat(stemPath); err == nil {
		t.Fatal("schema propose created a .stem file, but should be read-only")
	}

	var report SchemaProposalsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// Verify report structure
	if report.Version != 1 {
		t.Errorf("expected version 1, got %d", report.Version)
	}
	if report.Kind != "rootline/schema-proposals" {
		t.Errorf("expected kind 'rootline/schema-proposals', got %s", report.Kind)
	}

	// Verify summary  consistency
	if report.Summary.Total != len(report.Proposals) {
		t.Errorf("summary total mismatch: %d != %d", report.Summary.Total, len(report.Proposals))
	}

	// Verify engine_resolved count
	agentRequired := 0
	for _, p := range report.Proposals {
		if p.RequiresAgent {
			agentRequired++
		}
	}
	if agentRequired != report.Summary.RequiresAgent {
		t.Errorf("requires_agent mismatch: %d != %d", agentRequired, report.Summary.RequiresAgent)
	}
	if report.Summary.EngineResolved != (report.Summary.Total - report.Summary.RequiresAgent) {
		t.Errorf("engine_resolved mismatch: %d != %d", report.Summary.EngineResolved, report.Summary.Total-report.Summary.RequiresAgent)
	}
}

// TestSchemaProposeIncremental tests that with --incremental, existing .stem
// results in fewer proposals.
func TestSchemaProposeIncremental(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"test.md": "---\ntitulo: Test Document\ntipo: task\n---\n# Test Document\n\nContent here.\n",
		".stem":   "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  titulo:\n    type: string\n  tipo:\n    type: enum\n    values: [task, epic]\n",
	})

	// Get initial stem content and mtime
	stemPath := filepath.Join(root, ".stem")
	initialContent, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatalf("reading .stem: %v", err)
	}
	initialStat, err := os.Stat(stemPath)
	if err != nil {
		t.Fatalf("stat .stem: %v", err)
	}

	stdout, err := executeSchemaPropose(t, "--incremental", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify .stem file was not modified
	finalContent, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatalf("reading .stem after: %v", err)
	}
	if string(initialContent) != string(finalContent) {
		t.Error("stem file was modified by schema propose")
	}

	finalStat, err := os.Stat(stemPath)
	if err != nil {
		t.Fatalf("stat .stem after: %v", err)
	}
	if !initialStat.ModTime().Equal(finalStat.ModTime()) {
		t.Error("stem file timestamp changed, indicating it was modified")
	}

	var report SchemaProposalsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// In incremental mode with existing .stem, proposals should be filtered
	if report.Incremental != true {
		t.Error("expected incremental flag to be true")
	}

	// With an existing .stem, the bootstrap proposal should be filtered out
	if len(report.Proposals) > 0 {
		for _, p := range report.Proposals {
			if p.Operation == "create_stem" {
				t.Error("in incremental mode with existing .stem, bootstrap create_stem should be filtered")
			}
		}
	}
}

// TestSchemaProposeReadOnly verifies no files are created or modified.
func TestSchemaProposeReadOnly(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"doc1.md": "---\ntitulo: Document One\ntipo: task\nestado: Pending\n---\n# Doc One\n\nContent.\n",
		"doc2.md": "---\ntitulo: Document Two\ntipo: epic\nestado: In Progress\n---\n# Doc Two\n\nMore content.\n",
	})

	// Record initial files and their content
	initialFiles := listFilesWithContent(t, root)

	_, err := executeSchemaPropose(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify no new files were created
	finalFiles := listFilesWithContent(t, root)
	if len(finalFiles) != len(initialFiles) {
		t.Errorf("file count changed: %d -> %d (schema propose created files)", len(initialFiles), len(finalFiles))
	}

	// Verify all files have same content
	for name, initialContent := range initialFiles {
		finalContent, ok := finalFiles[name]
		if !ok {
			t.Errorf("file %s was deleted", name)
			continue
		}
		if initialContent != finalContent {
			t.Errorf("file %s was modified by schema propose", name)
		}
	}
}

// TestSchemaProposeJSONOutput verifies the JSON output format is valid.
func TestSchemaProposeJSONOutput(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"test.md": "---\ntitulo: Test\ntipo: task\n---\n# Test\n\nContent.\n",
	})

	stdout, err := executeSchemaPropose(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report SchemaProposalsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// Verify required fields
	if report.Version == 0 {
		t.Error("version is 0")
	}
	if report.Kind == "" {
		t.Error("kind is empty")
	}
	if report.Path == "" {
		t.Error("path is empty")
	}
	if report.Proposals == nil {
		t.Error("proposals is nil")
	}
}

// TestSchemaProposeEmptyDir tests behavior with empty directory.
func TestSchemaProposeEmptyDir(t *testing.T) {
	root := t.TempDir()

	stdout, err := executeSchemaPropose(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report SchemaProposalsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// Empty directory should have no proposals
	if len(report.Proposals) != 0 {
		t.Errorf("expected 0 proposals for empty dir, got %d", len(report.Proposals))
	}
}

// Helper functions

func listFilesWithContent(t *testing.T, dir string) map[string]string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("listing directory: %v", err)
	}

	files := make(map[string]string)
	for _, entry := range entries {
		if !entry.IsDir() {
			path := filepath.Join(dir, entry.Name())
			content, err := os.ReadFile(path)
			if err != nil {
				t.Fatalf("reading %s: %v", entry.Name(), err)
			}
			files[entry.Name()] = string(content)
		}
	}
	return files
}
