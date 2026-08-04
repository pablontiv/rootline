package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/infer"
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
// results in fewer proposals. When all inferences are covered by existing stems,
// no proposals are generated.
func TestSchemaProposeIncremental(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"a.md":  "---\ntitulo: Recipe A\ntipo: task\n---\n# Doc\n",
		"b.md":  "---\ntitulo: Recipe B\ntipo: epic\n---\n# Doc\n",
		"c.md":  "---\ntitulo: Recipe C\ntipo: task\n---\n# Doc\n",
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  titulo:\n    type: enum\n    required: true\n    values: [Recipe A, Recipe B, Recipe C]\n  tipo:\n    type: enum\n    required: true\n    values: [task, epic]\n  doc:\n    type: string\n    required: true\n",
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
// An empty directory (no files, no .stem) should return success with no proposals.
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

// TestSchemaProposeIncrementalPerScope tests that with --incremental, proposals
// for inferences covered by per-scope .stem files are filtered out.
// Uses enum-like data (few repeated values) so GenerateFlatSchema infers enums.
func TestSchemaProposeIncrementalPerScope(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		// concepts scope with .stem defining status as required enum
		"concepts/.stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  status:\n    type: enum\n    required: true\n    values: [done, pending]\n",
		"concepts/a.md":  "---\nstatus: done\n---\n# A\n",
		"concepts/b.md":  "---\nstatus: pending\n---\n# B\n",

		// sources scope with same .stem
		"sources/.stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  status:\n    type: enum\n    required: true\n    values: [done, pending]\n",
		"sources/a.md":  "---\nstatus: pending\n---\n# A\n",
		"sources/b.md":  "---\nstatus: done\n---\n# B\n",
	})

	// Run schema propose --incremental at root
	// Both scopes have identical .stem files with enum status covering [done, pending].
	// GenerateFlatSchema will infer status as enum with the same values.
	// The enum_values inference should be covered by both scopes' .stem files.
	// With per-scope filtering, all inferences should be covered, so no proposal.
	stdout, err := executeSchemaPropose(t, "--incremental", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report SchemaProposalsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}

	// Per-scope filtering: all inferences should be covered
	if len(report.Proposals) > 0 {
		t.Errorf("expected 0 proposals (all inferences covered by per-scope .stems), got %d", len(report.Proposals))
		for _, p := range report.Proposals {
			t.Logf("  proposal: %s at %s", p.ID, p.Target)
		}
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

// TestSchemaApplyAcceptsBothAnalyzeKinds verifies schema apply dispatches an
// analyze report whether it carries the legacy "analyze" kind or the
// namespaced "rootline/analyze" kind (backward-compatible reads).
func TestSchemaApplyAcceptsBothAnalyzeKinds(t *testing.T) {
	for _, kind := range []string{"analyze", "rootline/analyze"} {
		t.Run(kind, func(t *testing.T) {
			root := t.TempDir()
			// Create a minimal .stem file so schema apply can resolve stems
			stemContent := "version: 2\nschema:\n  estado:\n    type: string\n"
			if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(stemContent), 0o644); err != nil {
				t.Fatalf("writing .stem: %v", err)
			}
			report := infer.AnalyzeReport{Version: 1, Kind: kind, Path: root}
			data, _ := json.Marshal(report)
			reportFile := filepath.Join(root, "report.json")
			if err := os.WriteFile(reportFile, data, 0o644); err != nil {
				t.Fatalf("writing report: %v", err)
			}
			if _, err := executeSchemaApply(t, "--report", reportFile); err != nil {
				t.Errorf("expected kind %q to be accepted, got error: %v", kind, err)
			}
		})
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
	patchContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n  author:\n    type: string\n"
	report := SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/schema-proposals",
		Path:    docsDir,
		Root:    root,
		Proposals: []SchemaProposal{
			{
				ID:            "bootstrap-flat",
				Operation:     "create_stem",
				Target:        filepath.Join(docsDir, ".stem"),
				Confidence:    0.85,
				RequiresAgent: false,
				Patch:         patchContent,
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

// TestSchemaApply_AnalyzeReport_ExtendsEnum tests that schema apply consumes an analyze report and extends enum values.
func TestSchemaApply_AnalyzeReport_ExtendsEnum(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stem := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n"
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stem), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: Pending\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: Blocked\n---\n# B\n"), 0644)

	declareTestBoundary(t, dir)

	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Fatalf("analyze: %v\n%s", err, out)
	}
	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, []byte(out), 0644)

	if _, err := runCmd(t, "schema", "apply", "--report", reportFile); err != nil {
		t.Fatalf("schema apply: %v", err)
	}
	got := string(mustReadFile(t, filepath.Join(dir, ".stem")))
	if !strings.Contains(got, "Blocked") {
		t.Fatalf("expected enum extended with Blocked, got:\n%s", got)
	}
}

// TestSchemaApply_AnalyzeReport_DryRunNoWrite tests that --dry-run prevents writes when applying analyze report.
func TestSchemaApply_AnalyzeReport_DryRunNoWrite(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stem := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n"
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stem), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: Pending\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: Blocked\n---\n# B\n"), 0644)

	declareTestBoundary(t, dir)

	// Record initial .stem content
	initialStem := mustReadFile(t, filepath.Join(dir, ".stem"))

	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Fatalf("analyze: %v\n%s", err, out)
	}
	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, []byte(out), 0644)

	if _, err := runCmd(t, "schema", "apply", "--report", reportFile, "--dry-run"); err != nil {
		t.Fatalf("schema apply --dry-run: %v", err)
	}

	// Verify .stem is unchanged
	finalStem := mustReadFile(t, filepath.Join(dir, ".stem"))
	if string(initialStem) != string(finalStem) {
		t.Fatalf("dry-run modified .stem:\nbefore: %s\nafter: %s", initialStem, finalStem)
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

// --- Slice 3d: Mutating Command Error Semantics ---

// TestSchemaProposeWorksWithoutSchema verifies that schema propose succeeds on a tree with no .stem.
// This is a bootstrap test: schema propose must work on ungoverned trees to infer the schema.
func TestSchemaProposeWorksWithoutSchema(t *testing.T) {
	// Create a tree with NO .stem file
	dir := t.TempDir()

	// Create markdown files
	mustWriteFile(t, filepath.Join(dir, "doc1.md"), []byte("---\ntitle: Doc1\nestado: Pending\n---\n# Doc 1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "doc2.md"), []byte("---\ntitle: Doc2\nestado: Completed\n---\n# Doc 2\n"), 0644)

	// Attempt schema propose (should succeed because it's a bootstrap command)
	out, err := runCmd(t, "schema", "propose", dir)

	// This should NOT error
	if err != nil {
		t.Errorf("expected schema propose to succeed on no-schema tree, but got error: %v\noutput: %s", err, out)
	}

	// Should output valid JSON report
	var report map[string]interface{}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Errorf("expected valid JSON report, got error: %v\noutput: %s", err, out)
	}
	if report["kind"] != "rootline/schema-proposals" {
		t.Errorf("expected kind rootline/schema-proposals, got %v", report["kind"])
	}
}

// TestAnalyzeWorksWithoutSchema verifies that analyze succeeds on a tree with no .stem.
// This is a bootstrap test: analyze must work on ungoverned trees to infer the schema.
func TestAnalyzeWorksWithoutSchema(t *testing.T) {
	// Create a tree with NO .stem file
	dir := t.TempDir()

	// Create markdown files
	mustWriteFile(t, filepath.Join(dir, "doc1.md"), []byte("---\ntitle: Doc1\nestado: Pending\n---\n# Doc 1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "doc2.md"), []byte("---\ntitle: Doc2\nestado: Completed\n---\n# Doc 2\n"), 0644)

	// Attempt analyze (should succeed because it's a bootstrap command)
	out, err := runCmd(t, "analyze", dir)

	// This should NOT error
	if err != nil {
		t.Errorf("expected analyze to succeed on no-schema tree, but got error: %v\noutput: %s", err, out)
	}

	// Should output valid JSON report
	var report map[string]interface{}
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Errorf("expected valid JSON report, got error: %v\noutput: %s", err, out)
	}
	if report["kind"] != "rootline/analyze" {
		t.Errorf("expected kind rootline/analyze, got %v", report["kind"])
	}
}

// --- target containment (issue #69) ---

// writeSchemaProposalsReport writes a one-proposal report into root and returns
// its path. scanRoot is what schema apply confines targets to.
func writeSchemaProposalsReport(t *testing.T, root, scanRoot, target string) string {
	t.Helper()

	patchContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	// Root should be the absolute version of the scan root
	absRoot, _ := filepath.Abs(scanRoot)
	report := SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/schema-proposals",
		Path:    scanRoot,
		Root:    absRoot,
		Proposals: []SchemaProposal{
			{
				ID:           "bootstrap-flat",
				Operation:    "create_stem",
				Target:       target,
				Confidence:   0.85,
				Patch:        patchContent,
				PatchPreview: "version: 2\nschema:\n  title:\n    type: string\n",
			},
		},
	}

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	reportFile := filepath.Join(root, "report.json")
	if err := os.WriteFile(reportFile, data, 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}
	return reportFile
}

func decodeSchemaApplyResult(t *testing.T, out string) *SchemaApplyResult {
	t.Helper()

	var result SchemaApplyResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("decoding schema apply result: %v\noutput: %s", err, out)
	}
	return &result
}

func TestSchemaApply_TargetContainment(t *testing.T) {
	// Scenario 6: schema propose emits absolute targets, so an absolute target
	// inside the scan root has to keep working — this is the regression guard on
	// the propose->apply loop, not an edge case.
	t.Run("AbsoluteTargetInsideRootIsApplied", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")
		target := filepath.Join(docsDir, ".stem")

		reportFile := writeSchemaProposalsReport(t, root, docsDir, target)

		out, err := executeSchemaApply(t, "--report", reportFile)
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeSchemaApplyResult(t, out)
		if len(result.Errors) != 0 {
			t.Fatalf("errors = %v, want empty", result.Errors)
		}
		if len(result.Applied) == 0 {
			t.Fatal("applied is empty, want the create_stem recorded")
		}
		if _, err := os.Stat(target); err != nil {
			t.Errorf(".stem was not created: %v", err)
		}
	})

	// Scenario 7: an absolute target above the scan root.
	t.Run("AbsoluteTargetOutsideRootIsRejected", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")
		// One level above the declared scan root.
		target := filepath.Join(root, "escaped.stem")

		reportFile := writeSchemaProposalsReport(t, root, docsDir, target)

		out, err := executeSchemaApply(t, "--report", reportFile)
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeSchemaApplyResult(t, out)
		if len(result.Applied) != 0 {
			t.Errorf("applied = %v, want empty", result.Applied)
		}
		if len(result.Errors) != 1 {
			t.Fatalf("errors = %v, want exactly one containment violation", result.Errors)
		}
		if !strings.Contains(result.Errors[0], "escapes root") {
			t.Errorf("error %q does not give the containment reason", result.Errors[0])
		}
		if _, err := os.Stat(target); err == nil {
			t.Error("a .stem was created outside the scan root")
		}
	})

	// Scenario 5: the target only escapes once cleaned. filepath.Join(root,
	// target) would have folded it back inside and reported it as contained.
	t.Run("TargetEscapingAfterCleanIsRejected", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")
		target := filepath.Join(docsDir, "..", "..", "outside", ".stem")

		reportFile := writeSchemaProposalsReport(t, root, docsDir, target)

		out, err := executeSchemaApply(t, "--report", reportFile)
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeSchemaApplyResult(t, out)
		if len(result.Applied) != 0 {
			t.Errorf("applied = %v, want empty", result.Applied)
		}
		if len(result.Errors) != 1 {
			t.Fatalf("errors = %v, want exactly one containment violation", result.Errors)
		}
		if _, err := os.Stat(filepath.Clean(target)); err == nil {
			t.Error("a .stem was created outside the scan root")
		}
	})

	t.Run("DryRunReportsResolvedTargets", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")
		accepted := filepath.Join(docsDir, ".stem")
		rejected := filepath.Join(root, "escaped.stem")

		report := SchemaProposalsReport{
			Version: 1,
			Kind:    "rootline/schema-proposals",
			Path:    docsDir,
			Proposals: []SchemaProposal{
				{ID: "inside", Operation: "create_stem", Target: accepted},
				{ID: "outside", Operation: "create_stem", Target: rejected},
			},
		}
		data, err := json.Marshal(report)
		if err != nil {
			t.Fatalf("marshal report: %v", err)
		}
		reportFile := filepath.Join(root, "report.json")
		if err := os.WriteFile(reportFile, data, 0o644); err != nil {
			t.Fatalf("writing report: %v", err)
		}

		out, err := executeSchemaApply(t, "--report", reportFile, "--dry-run")
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeSchemaApplyResult(t, out)
		if result.ResolvedTargets == nil {
			t.Fatal("resolved_targets is absent, want it populated in dry-run")
		}
		if got := result.ResolvedTargets.Accepted[accepted]; got != accepted {
			t.Errorf("resolved_targets.accepted[%s] = %q, want %q", accepted, got, accepted)
		}
		if got := result.ResolvedTargets.Rejected[rejected]; got != "escapes root" {
			t.Errorf("resolved_targets.rejected[%s] = %q, want %q", rejected, got, "escapes root")
		}
		if _, err := os.Stat(rejected); err == nil {
			t.Error("dry-run created a .stem outside the scan root")
		}
	})

	t.Run("ResolvedTargetsOmittedOutsideDryRun", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")
		reportFile := writeSchemaProposalsReport(t, root, docsDir, filepath.Join(docsDir, ".stem"))

		out, err := executeSchemaApply(t, "--report", reportFile)
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		if result := decodeSchemaApplyResult(t, out); result.ResolvedTargets != nil {
			t.Errorf("resolved_targets = %+v, want absent outside dry-run", result.ResolvedTargets)
		}
	})
}

// --- Slice A Tests: Force Flag, Rejected Field, Unrecognized Operations ---

// TestSchemaApplyForceFlag tests that --force flag gates overwrite of existing .stem files.
// RED: test that:
// 1. force flag is recognized
// 2. without force: existing .stem is refused
// 3. with force: existing .stem is overwritten
func TestSchemaApplyForceFlag(t *testing.T) {
	t.Run("ForceFlagIsRecognized", func(t *testing.T) {
		// Use setupValidateProject to create proper project structure
		root := setupValidateProject(t, map[string]string{
			"doc1.md": "---\ntitle: Test\nauthor: Someone\n---\nContent",
		})

		// Create existing .stem
		existingStem := "version: 2\nschema:\n  old_field:\n    type: string\n"
		stemPath := filepath.Join(root, ".stem")
		if err := os.WriteFile(stemPath, []byte(existingStem), 0o644); err != nil {
			t.Fatalf("writing existing .stem: %v", err)
		}

		// Create proposals report targeting the existing .stem
		patchContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  new_field:\n    type: string\n"
		report := SchemaProposalsReport{
			Version: 1,
			Kind:    "rootline/schema-proposals",
			Path:    root,
			Root:    root,
			Proposals: []SchemaProposal{
				{
					ID:            "test-force",
					Operation:     "create_stem",
					Target:        stemPath,
					RequiresAgent: false,
					Patch:         patchContent,
					PatchPreview:  "version: 2\nschema:\n  new_field:\n    type: string\n",
				},
			},
		}
		reportData, _ := json.Marshal(report)
		reportFile := filepath.Join(root, "report.json")
		if err := os.WriteFile(reportFile, reportData, 0o644); err != nil {
			t.Fatalf("writing report: %v", err)
		}

		// Test 1: Without --force, existing .stem should be rejected
		out, err := executeSchemaApply(t, "--report", reportFile)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result := decodeSchemaApplyResult(t, out)
		if len(result.Rejected) == 0 {
			t.Error("expected rejected[]to contain entry for overwrite attempt without --force, got empty")
		}
		if len(result.Applied) > 0 {
			t.Error("expected applied[] to be empty when overwrite is refused")
		}

		// Verify .stem was not modified
		stillExisting := string(mustReadFile(t, stemPath))
		if stillExisting != existingStem {
			t.Error(".stem was modified despite --force not being passed")
		}

		// Test 2: With --force, existing .stem should be overwritten
		out2, err := executeSchemaApply(t, "--report", reportFile, "--force")
		if err != nil {
			t.Fatalf("unexpected error with --force: %v", err)
		}
		result2 := decodeSchemaApplyResult(t, out2)
		if len(result2.Applied) == 0 {
			t.Error("expected applied[] to contain entry when --force is passed")
		}
		if len(result2.Rejected) > 0 {
			t.Errorf("expected no rejections with --force, got: %v", result2.Rejected)
		}
	})

}

// TestSchemaApplyUnrecognizedOperation tests that unknown operations are rejected.
func TestSchemaApplyUnrecognizedOperation(t *testing.T) {
	root := t.TempDir()

	// Create proposals report with unknown operation
	report := SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/schema-proposals",
		Path:    root,
		Proposals: []SchemaProposal{
			{
				ID:            "unknown-op-1",
				Operation:     "update_stem", // This is not a valid proposal-path operation
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

	out, err := executeSchemaApply(t, "--report", reportFile)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	result := decodeSchemaApplyResult(t, out)

	// Verify unknown operation is in rejected[], not errors[] or applied[]
	if len(result.Rejected) == 0 {
		t.Error("expected rejected[] to contain unknown operation, got empty")
	}
	if len(result.Applied) > 0 {
		t.Error("expected applied[] to be empty for unknown operation")
	}

	// Check that the rejection message names the operation
	found := false
	for _, msg := range result.Rejected {
		if strings.Contains(msg, "update_stem") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("rejection message should mention 'update_stem', got: %v", result.Rejected)
	}
}

// TestSchemaApplyDryRunDistinction asserts that a dry run names the action it
// would take. Reporting "create_stem" for a target that already exists is the
// exact blind spot issue #59 describes: the caller approves a create and gets a
// replacement.
func TestSchemaApplyDryRunDistinction(t *testing.T) {
	writeReport := func(t *testing.T, root, name, target string) string {
		t.Helper()
		patchContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
		report := SchemaProposalsReport{
			Version: 1,
			Kind:    "rootline/schema-proposals",
			Path:    filepath.Dir(target),
			Root:    root,
			Proposals: []SchemaProposal{{
				ID:        name,
				Operation: "create_stem",
				Target:    target,
				Patch:     patchContent,
			}},
		}
		data, err := json.Marshal(report)
		if err != nil {
			t.Fatalf("marshaling report: %v", err)
		}
		path := filepath.Join(root, name+".json")
		if err := os.WriteFile(path, data, 0o644); err != nil {
			t.Fatalf("writing report: %v", err)
		}
		return path
	}

	t.Run("absent target reports create", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
		})
		target := filepath.Join(root, "docs", ".stem")
		out, err := executeSchemaApply(t, "--report", writeReport(t, root, "create", target), "--dry-run")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result := decodeSchemaApplyResult(t, out)
		if len(result.Applied) != 1 {
			t.Fatalf("applied = %v, want exactly one entry", result.Applied)
		}
		if !strings.HasPrefix(result.Applied[0], "create_stem: ") {
			t.Errorf("applied[0] = %q, want a create_stem action", result.Applied[0])
		}
		if len(result.Rejected) != 0 {
			t.Errorf("rejected = %v, want empty for a fresh target", result.Rejected)
		}
	})

	t.Run("existing target without force is rejected, not applied", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
			"docs/.stem":   "version: 2\nschema:\n  old:\n    type: string\n",
		})
		target := filepath.Join(root, "docs", ".stem")
		out, err := executeSchemaApply(t, "--report", writeReport(t, root, "reject", target), "--dry-run")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result := decodeSchemaApplyResult(t, out)
		if len(result.Applied) != 0 {
			t.Errorf("applied = %v, want empty when the target exists and --force is absent", result.Applied)
		}
		if len(result.Rejected) != 1 || !strings.Contains(result.Rejected[0], "use --force to overwrite") {
			t.Errorf("rejected = %v, want the already-exists refusal", result.Rejected)
		}
	})

	t.Run("existing target with force reports overwrite, not create", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
			"docs/.stem":   "version: 2\nschema:\n  old:\n    type: string\n",
		})
		target := filepath.Join(root, "docs", ".stem")
		out, err := executeSchemaApply(t, "--report", writeReport(t, root, "force", target), "--dry-run", "--force")
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result := decodeSchemaApplyResult(t, out)
		if len(result.Applied) != 1 {
			t.Fatalf("applied = %v, want exactly one entry", result.Applied)
		}
		if !strings.HasPrefix(result.Applied[0], "overwrite_stem: ") {
			t.Errorf("applied[0] = %q, want an overwrite_stem action so the caller can tell a replacement from a create", result.Applied[0])
		}
	})
}

// writeProposalsReport builds a single-proposal report on disk. Shared by the
// patch-fidelity and root-resolution tests, which differ only in the fields
// they exercise.
func writeProposalsReport(t *testing.T, scanPath, scanRoot, target, patch string) string {
	t.Helper()
	report := SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/schema-proposals",
		Path:    scanPath,
		Root:    scanRoot,
		Proposals: []SchemaProposal{{
			ID:        "p1",
			Operation: "create_stem",
			Target:    target,
			Patch:     patch,
		}},
	}
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshaling report: %v", err)
	}
	path := filepath.Join(t.TempDir(), "proposals.json")
	if err := os.WriteFile(path, data, 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}
	return path
}

// TestSchemaProposePopulatesFullPatch asserts that propose records the whole
// inferred schema, not just the 200-char display preview. Without the full
// patch there is nothing for apply to write except a re-derivation.
func TestSchemaProposePopulatesFullPatch(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"task1.md": "---\nstatus: done\ntype: feature\n---\n# Task 1\n",
		"task2.md": "---\nstatus: pending\ntype: bug\n---\n# Task 2\n",
	})

	stdout, err := executeSchemaPropose(t, root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var report SchemaProposalsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, stdout)
	}
	if len(report.Proposals) == 0 {
		t.Fatal("expected at least one proposal")
	}

	p := report.Proposals[0]
	if p.Patch == "" {
		t.Fatal("patch is empty; propose must record the full YAML it inferred")
	}
	for _, want := range []string{"version: 2", "schema:", "status", "type"} {
		if !strings.Contains(p.Patch, want) {
			t.Errorf("patch is missing %q; got:\n%s", want, p.Patch)
		}
	}
	if p.PatchPreview == "" {
		t.Error("patch_preview must stay populated for display")
	}
	if len(p.PatchPreview) > 203 {
		t.Errorf("patch_preview is %d chars; it must stay truncated for display", len(p.PatchPreview))
	}
}

// TestSchemaApplyWritesProposedPatchVerbatim is the propose->apply contract:
// the bytes a reviewer approved are the bytes that land on disk. Before this
// change apply ignored the proposal body and re-derived an untyped schema, so
// approving a proposal approved something the tool would never produce.
func TestSchemaApplyWritesProposedPatchVerbatim(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"task1.md": "---\nstatus: done\n---\n# Task 1\n",
		"task2.md": "---\nstatus: pending\n---\n# Task 2\n",
	})

	stdout, err := executeSchemaPropose(t, root)
	if err != nil {
		t.Fatalf("propose: %v", err)
	}
	var report SchemaProposalsReport
	if err := json.Unmarshal([]byte(stdout), &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if len(report.Proposals) == 0 {
		t.Fatal("expected at least one proposal")
	}
	proposed := report.Proposals[0].Patch

	reportPath := filepath.Join(t.TempDir(), "proposals.json")
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshaling report: %v", err)
	}
	if err := os.WriteFile(reportPath, data, 0o644); err != nil {
		t.Fatalf("writing report: %v", err)
	}

	if _, err := executeSchemaApply(t, "--report", reportPath); err != nil {
		t.Fatalf("apply: %v", err)
	}

	written := string(mustReadFile(t, report.Proposals[0].Target))
	if written != proposed {
		t.Errorf("written .stem does not match the proposed patch.\nproposed:\n%s\nwritten:\n%s", proposed, written)
	}
	// The regression this guards: the old path emitted untyped shells.
	if strings.Contains(written, "type: string") && !strings.Contains(proposed, "type: string") {
		t.Error("written schema was re-derived rather than taken from the proposal")
	}
}

// TestSchemaApplyEmptyPatchRejected covers legacy reports that predate the
// patch field. Refusing is the point: the old fallback silently wrote untyped
// shells over whatever was there.
func TestSchemaApplyEmptyPatchRejected(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"test.md": "---\ntitle: Test\n---\nContent",
	})
	absRoot, err := filepath.Abs(root)
	if err != nil {
		t.Fatalf("resolving root: %v", err)
	}
	stemPath := filepath.Join(absRoot, ".stem")

	reportPath := writeProposalsReport(t, absRoot, absRoot, stemPath, "")

	out, err := executeSchemaApply(t, "--report", reportPath)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	result := decodeSchemaApplyResult(t, out)

	found := false
	for _, e := range result.Errors {
		if strings.Contains(e, "patch content required") && strings.Contains(e, "schema propose") {
			found = true
		}
	}
	if !found {
		t.Errorf("errors = %v, want one telling the caller to re-run schema propose", result.Errors)
	}
	if len(result.Applied) != 0 {
		t.Errorf("applied = %v, want empty for a proposal with no patch", result.Applied)
	}
	if _, statErr := os.Stat(stemPath); statErr == nil {
		t.Error("a .stem was written despite the proposal carrying no patch")
	}
}

// TestSchemaApplyRootResolution covers the scan root the post-apply validation
// runs against. report.Root is absolute, so a report stays applicable from any
// working directory; reports without it keep the old CWD-relative behaviour.
func TestSchemaApplyRootResolution(t *testing.T) {
	const patch = "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"

	t.Run("propose records an absolute root", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"test.md": "---\ntitle: Test\n---\nContent",
		})
		stdout, err := executeSchemaPropose(t, root)
		if err != nil {
			t.Fatalf("propose: %v", err)
		}
		var report SchemaProposalsReport
		if err := json.Unmarshal([]byte(stdout), &report); err != nil {
			t.Fatalf("invalid JSON: %v", err)
		}
		if !filepath.IsAbs(report.Root) {
			t.Errorf("root = %q, want an absolute path so the report survives a change of directory", report.Root)
		}
	})

	t.Run("root wins over a path that would resolve elsewhere", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/test.md": "---\ntitle: Test\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")
		// Path is a bare relative name that would resolve against the test
		// process CWD; Root is the real location.
		reportPath := writeProposalsReport(t, "docs", docsDir, filepath.Join(docsDir, ".stem"), patch)

		out, err := executeSchemaApply(t, "--report", reportPath)
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		result := decodeSchemaApplyResult(t, out)
		if len(result.Applied) != 1 {
			t.Fatalf("applied = %v (errors %v), want the proposal applied via report.Root", result.Applied, result.Errors)
		}
		if _, statErr := os.Stat(filepath.Join(docsDir, ".stem")); statErr != nil {
			t.Errorf("no .stem at the root-resolved location: %v", statErr)
		}
	})

	t.Run("legacy report without root keeps path-based resolution", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"test.md": "---\ntitle: Test\n---\nContent",
		})
		absRoot, err := filepath.Abs(root)
		if err != nil {
			t.Fatalf("resolving root: %v", err)
		}
		reportPath := writeProposalsReport(t, absRoot, "", filepath.Join(absRoot, ".stem"), patch)

		out, err := executeSchemaApply(t, "--report", reportPath)
		if err != nil {
			t.Fatalf("apply: %v", err)
		}
		result := decodeSchemaApplyResult(t, out)
		if len(result.Applied) != 1 {
			t.Errorf("applied = %v (errors %v), want legacy path resolution to still work", result.Applied, result.Errors)
		}
	})
}

// TestPostApplyValidationErrorPropagation tests that validation errors surface in results.
func TestPostApplyValidationErrorPropagation(t *testing.T) {
	// A scan root that does not exist must surface as an error. This is issue
	// #59 sub-defect 4b: applying a report from another directory resolved the
	// scan root to a path that was never there, and the swallowed scan error
	// was reported as total_files: 0 — indistinguishable from a clean run.
	t.Run("unscannable root surfaces an error instead of an all-zero summary", func(t *testing.T) {
		missing := filepath.Join(t.TempDir(), "does-not-exist")
		reportPath := writeProposalsReport(t, missing, missing,
			filepath.Join(missing, ".stem"), "version: 2\nschema:\n  title:\n    type: string\n")

		out, err := executeSchemaApply(t, "--report", reportPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result := decodeSchemaApplyResult(t, out)

		foundScanError := false
		for _, e := range result.Errors {
			if strings.Contains(e, "post-apply validation scan") {
				foundScanError = true
			}
		}
		if !foundScanError {
			t.Errorf("errors = %v, want one naming the failed post-apply validation scan", result.Errors)
		}
		if result.ValidationSummary != nil {
			t.Errorf("validation_summary = %+v, want it omitted when the scan failed", result.ValidationSummary)
		}
	})

	// The complement: a scan that succeeds must report the real numbers,
	// including failures caused by the schema that was just written.
	t.Run("successful scan reports real counts for the freshly written schema", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/a.md": "---\nestado: Pending\n---\nContent",
			"docs/b.md": "---\nestado: Bogus\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")
		target := filepath.Join(docsDir, ".stem")
		patch := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    required: true\n    values: [Pending, Done]\n"
		reportPath := writeProposalsReport(t, docsDir, docsDir, target, patch)

		out, err := executeSchemaApply(t, "--report", reportPath)
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		result := decodeSchemaApplyResult(t, out)

		if result.ValidationSummary == nil {
			t.Fatalf("validation_summary is nil; a successful scan must report one. errors=%v", result.Errors)
		}
		if result.ValidationSummary.TotalFiles != 2 {
			t.Errorf("total_files = %d, want 2", result.ValidationSummary.TotalFiles)
		}
		// b.md violates the enum the patch just introduced, so a green summary
		// here would mean validation ran against something that cannot fail.
		if result.ValidationSummary.InvalidFiles == 0 || result.ValidationSummary.TotalErrors == 0 {
			t.Errorf("summary = %+v, want the enum violation in b.md counted", result.ValidationSummary)
		}
	})
}
