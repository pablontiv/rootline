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

// Schema Apply Tests

func executeSchemaApply(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(append([]string{"schema", "apply"}, args...))
	err := rootCmd.Execute()
	return buf.String(), err
}

// TestSchemaApplyInvalidKind tests that wrong report kind is rejected.
func TestSchemaApplyInvalidKind(t *testing.T) {
	root := t.TempDir()

	// Create a report with wrong kind
	report := SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/invalid-kind",
		Path:    root,
		Proposals: []SchemaProposal{
			{
				ID:            "test",
				Operation:     "create_stem",
				Target:        filepath.Join(root, ".stem"),
				RequiresAgent: false,
			},
		},
	}

	reportData, _ := json.Marshal(report)
	reportFile := filepath.Join(root, "report.json")
	if err := os.WriteFile(reportFile, reportData, 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	_, err := executeSchemaApply(t, "--report", reportFile)
	if err == nil {
		t.Error("expected error for wrong report kind, got nil")
	}
}

// TestSchemaApplyInvalidVersion tests that wrong report version is rejected.
func TestSchemaApplyInvalidVersion(t *testing.T) {
	root := t.TempDir()

	// Create a report with wrong version
	report := SchemaProposalsReport{
		Version: 2,
		Kind:    "rootline/schema-proposals",
		Path:    root,
		Proposals: []SchemaProposal{
			{
				ID:            "test",
				Operation:     "create_stem",
				Target:        filepath.Join(root, ".stem"),
				RequiresAgent: false,
			},
		},
	}

	reportData, _ := json.Marshal(report)
	reportFile := filepath.Join(root, "report.json")
	if err := os.WriteFile(reportFile, reportData, 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	_, err := executeSchemaApply(t, "--report", reportFile)
	if err == nil {
		t.Error("expected error for wrong version, got nil")
	}
}

// TestSchemaApplyDryRun tests that --dry-run does not write files.
func TestSchemaApplyDryRun(t *testing.T) {
	root := t.TempDir()

	// Create a test directory with a markdown file
	testDir := filepath.Join(root, "testdir")
	if err := os.Mkdir(testDir, 0o755); err != nil {
		t.Fatalf("creating test directory: %v", err)
	}

	testFile := filepath.Join(testDir, "test.md")
	if err := os.WriteFile(testFile, []byte("---\ntitle: Test\n---\nContent"), 0o644); err != nil {
		t.Fatalf("writing test file: %v", err)
	}

	// Create a schema proposals report with create_stem
	report := SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/schema-proposals",
		Path:    testDir,
		Proposals: []SchemaProposal{
			{
				ID:            "bootstrap-flat",
				Operation:     "create_stem",
				Target:        filepath.Join(testDir, ".stem"),
				Confidence:    0.85,
				RequiresAgent: false,
				PatchPreview:  "version: 2\nschema:\n  title:\n    type: string\n",
			},
		},
	}

	reportData, _ := json.Marshal(report)
	reportFile := filepath.Join(root, "report.json")
	if err := os.WriteFile(reportFile, reportData, 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	// Record files before apply
	filesBefore := listFilesWithContent(t, testDir)

	_, err := executeSchemaApply(t, "--report", reportFile, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Record files after dry-run
	filesAfter := listFilesWithContent(t, testDir)

	// Verify no files were created
	if len(filesAfter) != len(filesBefore) {
		t.Errorf("dry-run created files: before=%d, after=%d", len(filesBefore), len(filesAfter))
	}

	// Verify existing files unchanged
	for name, beforeContent := range filesBefore {
		afterContent, ok := filesAfter[name]
		if !ok {
			t.Errorf("file %s was deleted by dry-run", name)
			continue
		}
		if beforeContent != afterContent {
			t.Errorf("file %s was modified by dry-run", name)
		}
	}
}

// TestSchemaApplySkipsRequiresAgent tests that proposals with requires_agent are skipped.
func TestSchemaApplySkipsRequiresAgent(t *testing.T) {
	root := t.TempDir()

	// Create a schema proposals report with requires_agent=true
	report := SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/schema-proposals",
		Path:    root,
		Proposals: []SchemaProposal{
			{
				ID:            "test-agent",
				Operation:     "create_stem",
				Target:        filepath.Join(root, ".stem"),
				RequiresAgent: true,
				PatchPreview:  "version: 2\nschema:\n  test:\n    type: string\n",
			},
		},
	}

	reportData, _ := json.Marshal(report)
	reportFile := filepath.Join(root, "report.json")
	if err := os.WriteFile(reportFile, reportData, 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	stdout, err := executeSchemaApply(t, "--report", reportFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SchemaApplyResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// Verify proposal was skipped
	if len(result.Skipped) == 0 {
		t.Error("expected skipped proposal, got none")
	}

	// Verify stem file was not created
	stemPath := filepath.Join(root, ".stem")
	if _, err := os.Stat(stemPath); err == nil {
		t.Error("stem file should not be created for requires_agent proposal")
	}
}

// TestSchemaApplyCreateStem tests that valid create_stem proposals create .stem files.
func TestSchemaApplyCreateStem(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"docs/doc1.md": "---\ntitle: Test\nauthor: Someone\n---\nContent",
	})

	docsDir := filepath.Join(root, "docs")

	// Create a schema proposals report with create_stem
	report := SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/schema-proposals",
		Path:    docsDir,
		Proposals: []SchemaProposal{
			{
				ID:            "bootstrap-flat",
				Operation:     "create_stem",
				Target:        filepath.Join(docsDir, ".stem"),
				Confidence:    0.85,
				RequiresAgent: false,
				PatchPreview:  "version: 2\nschema:\n  title:\n    type: string\n  author:\n    type: string\n",
			},
		},
	}

	reportData, _ := json.Marshal(report)
	reportFile := filepath.Join(root, "report.json")
	if err := os.WriteFile(reportFile, reportData, 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	stdout, err := executeSchemaApply(t, "--report", reportFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SchemaApplyResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// Verify report structure
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
	if result.Kind != "rootline/schema-apply" {
		t.Errorf("expected kind 'rootline/schema-apply', got %s", result.Kind)
	}

	// Verify proposal was applied
	if len(result.Applied) == 0 {
		t.Error("expected applied proposal, got none")
	}

	// Verify stem file was created
	stemPath := filepath.Join(docsDir, ".stem")
	if _, err := os.Stat(stemPath); err != nil {
		t.Errorf("stem file not created: %v", err)
	}
}

// TestSchemaApplyResultStructure tests that the result JSON has correct structure.
func TestSchemaApplyResultStructure(t *testing.T) {
	root := t.TempDir()

	// Create empty report
	report := SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/schema-proposals",
		Path:    root,
	}

	reportData, _ := json.Marshal(report)
	reportFile := filepath.Join(root, "report.json")
	if err := os.WriteFile(reportFile, reportData, 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	stdout, err := executeSchemaApply(t, "--report", reportFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result SchemaApplyResult
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// Verify required fields are present
	if result.Version == 0 {
		t.Error("version is 0")
	}
	if result.Kind == "" {
		t.Error("kind is empty")
	}
	if result.Applied == nil {
		t.Error("applied is nil")
	}
	if result.Skipped == nil {
		t.Error("skipped is nil")
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
