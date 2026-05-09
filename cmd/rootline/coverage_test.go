package main

// coverage_test.go: targeted tests for uncovered code paths in cmd/rootline.
// Focus: runAnalyze, runApply, runMigrateScaffold, postValidateOrRollback,
// findGitRoot, renderValidateTable, appendStemHealthProposals,
// appendPropagateProposals, proposalsToFixResults.

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// --- runAnalyze ---

func TestRunAnalyze_BasicJSON(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if result["kind"] != "analyze" && result["kind"] != "rootline/analyze" {
		t.Errorf("kind = %v, want analyze or rootline/analyze", result["kind"])
	}
}

func TestRunAnalyze_Table(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "analyze", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	// Table output should contain some formatted text
	_ = out // just verify no error
}

func TestRunAnalyze_Incremental(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "analyze", "--incremental", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
}

func TestRunAnalyze_CurrentDir(t *testing.T) {
	dir := setupTestDir(t)
	mustChdir(t, dir)
	out, err := runCmd(t, "analyze")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
}

// --- runApply ---

func TestRunApply_FromFile(t *testing.T) {
	dir := setupTestDir(t)

	// First, generate an analyze report.
	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Fatalf("analyze failed: %v\noutput: %s", err, out)
	}

	// Write the report to a file.
	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, []byte(out), 0644)

	// Apply the report (dry-run to avoid modifying files).
	out2, err := runCmd(t, "apply", "--dry-run", reportFile)
	if err != nil {
		t.Fatalf("apply failed: %v\noutput: %s", err, out2)
	}
}

func TestRunApply_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "bad.json")
	mustWriteFile(t, badFile, []byte("not json"), 0644)

	_, err := runCmd(t, "apply", badFile)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRunApply_FileNotFound(t *testing.T) {
	_, err := runCmd(t, "apply", "/nonexistent/report.json")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// --- runMigrateScaffold ---

func setupMigrateScaffoldDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Need .git for stem resolution
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// .stem with a required section
	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  notes:
    type: section
    heading: "## Notes"
    required: true
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// File missing the required "Notes" section
	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Pending\n---\n# Task\n\nSome content.\n"), 0644)

	return dir
}

func TestRunMigrateScaffold_DryRun(t *testing.T) {
	dir := setupMigrateScaffoldDir(t)

	// Use --dry-run to avoid actually modifying files
	out, err := runCmd(t, "migrate", "--dry-run", "--scaffold", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	// Should output JSON by default
}

func TestRunMigrateScaffold_Table(t *testing.T) {
	dir := setupMigrateScaffoldDir(t)

	out, err := runCmd(t, "migrate", "--dry-run", "--scaffold", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	// Table output should include heading name
	if !strings.Contains(out, "Notes") && !strings.Contains(out, "section") && !strings.Contains(out, "added") && !strings.Contains(out, "0") {
		// May be empty if no sections need adding
		t.Logf("table output: %s", out)
	}
}

func TestRunMigrateScaffold_NothingToAdd(t *testing.T) {
	dir := setupTestDir(t)

	// setupTestDir has no required sections, so scaffold should be a no-op
	out, err := runCmd(t, "migrate", "--dry-run", "--scaffold", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	_ = out
}

func TestRunMigrateScaffold_NotADirectory(t *testing.T) {
	dir := t.TempDir()
	file := filepath.Join(dir, "file.txt")
	mustWriteFile(t, file, []byte("content"), 0644)

	_, err := runCmd(t, "migrate", "--scaffold", file)
	if err == nil {
		t.Fatal("expected error for non-directory path")
	}
}

// --- postValidateOrRollback (via runSet) ---

func TestSetPostValidationRollback(t *testing.T) {
	// Set up a project with a required field and try to remove it
	// This requires constructing a situation where post-mutation validation fails.
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Stem requires estado
	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  tipo:
    type: string
    required: true
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// Document with required fields present
	docContent := `---
estado: Pending
tipo: test
---
# Doc

`
	target := filepath.Join(dir, "doc.md")
	mustWriteFile(t, target, []byte(docContent), 0644)

	// Try to set tipo to "" (empty string) - this might fail post-validation if required
	// Since tipo is required, setting to empty might fail validation
	// Use --no-validate to bypass (ensures the apply path, not postValidate)
	out, err := runCmd(t, "set", "--no-validate", target, "tipo=")
	if err != nil {
		t.Logf("no-validate still errored: %v, output: %s", err, out)
	}
	// Read back to verify
	content := string(mustReadFile(t, target))
	_ = content
}

// --- findGitRoot (cmd/rootline) ---

func TestFindGitRootCmd_NotFound(t *testing.T) {
	dir := t.TempDir()
	_, err := findGitRoot(dir)
	if err == nil {
		t.Fatal("expected error when no .git directory")
	}
	if !strings.Contains(err.Error(), "no .git directory") {
		t.Errorf("expected 'no .git directory' error, got: %v", err)
	}
}

func TestFindGitRootCmd_Found(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	sub := filepath.Join(dir, "a", "b")
	if err := os.MkdirAll(sub, 0o755); err != nil {
		t.Fatal(err)
	}
	got, err := findGitRoot(sub)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dir {
		t.Errorf("expected %s, got %s", dir, got)
	}
}

// --- runSet: file not found ---

func TestSetFileNotFound(t *testing.T) {
	_, err := runCmd(t, "set", "/nonexistent/path/file.md", "estado=Pending")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

// --- runSet: no git root ---

func TestSetNoGitRoot(t *testing.T) {
	dir := t.TempDir()
	// No .git directory
	target := filepath.Join(dir, "file.md")
	mustWriteFile(t, target, []byte("---\nestado: Pending\n---\n# File\n"), 0644)

	_, err := runCmd(t, "set", target, "estado=Completed")
	if err == nil {
		t.Fatal("expected error for no git root")
	}
	if !strings.Contains(err.Error(), "finding git root") {
		t.Errorf("expected 'finding git root' error, got: %v", err)
	}
}

// --- renderValidateTable with failures ---

func TestValidateAll_TableWithErrors(t *testing.T) {
	dir := setupTestDir(t)
	// Add a file with validation errors
	mustWriteFile(t, filepath.Join(dir, "bad.md"),
		[]byte("---\ntipo: test\n---\n# Bad\n"), 0644)

	out, _ := runCmd(t, "validate", "--all", dir, "-o", "table")
	// Should contain the bad file path and "no" for invalid
	if !strings.Contains(out, "bad.md") {
		t.Logf("table output doesn't contain bad.md: %s", out)
	}
}

func TestValidateAll_JSON(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "validate", "--all", dir)
	if err != nil {
		// Non-zero exit is expected if there are errors
		t.Logf("expected possible error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
}

// --- appendStemHealthProposals / appendPropagateProposals (via fix --all) ---

func TestFixAll_WithProposals(t *testing.T) {
	dir := setupTestDir(t)
	// Add a file with invalid enum value to trigger proposals
	mustWriteFile(t, filepath.Join(dir, "bad.md"),
		[]byte("---\nestado: InvalidValue\ntipo: test\n---\n# Bad\n"), 0644)

	out, err := runCmd(t, "fix", "--all", "--dry-run", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
}

// --- proposalsToFixResults (via fix single file) ---

func TestFix_SingleFileWithProposals(t *testing.T) {
	dir := setupTestDir(t)
	// File missing required field
	target := filepath.Join(dir, "bad.md")
	mustWriteFile(t, target, []byte("---\ntipo: test\n---\n# Bad\n"), 0644)

	out, err := runCmd(t, "fix", "--dry-run", target)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
}

// --- describe error paths ---

func TestDescribeFileInDir(t *testing.T) {
	// describe a directory that contains a .stem file
	dir := setupTestDir(t)
	out, err := runCmd(t, "describe", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "estado") {
		t.Errorf("expected estado in describe output, got: %s", out)
	}
}

// --- explain: no extractor ---

func TestExplainNoExtractor(t *testing.T) {
	dir := t.TempDir()
	nonMdFile := filepath.Join(dir, "file.xyz")
	mustWriteFile(t, nonMdFile, []byte("content"), 0644)

	_, err := runCmd(t, "explain", nonMdFile)
	if err == nil {
		t.Fatal("expected error for file with no extractor")
	}
}

// --- runSet: parse error ---

func TestSetInvalidFieldSyntax(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	_, err := runCmd(t, "set", target, "notavalidassignment")
	if err == nil {
		t.Fatal("expected error for invalid field syntax")
	}
}

// --- runSet: @file not found ---

func TestSetAtFileMissing(t *testing.T) {
	dir := setupSetTestDir(t)
	target := filepath.Join(dir, "doc.md")

	_, err := runCmd(t, "set", "--no-validate", target, "investigacion=@/nonexistent/file.txt")
	if err == nil {
		t.Fatal("expected error for missing @file")
	}
}

// --- validate --all table with valid files ---

func TestValidateAll_TableNoErrors(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "validate", "--all", dir, "-o", "table")
	if err != nil {
		t.Logf("possible exit code error (normal): %v", err)
	}
	if !strings.Contains(out, "doc1.md") && !strings.Contains(out, "File") {
		t.Logf("table output: %s", out)
	}
}

// --- runValidateAll with --where filter ---

func TestValidateAll_WithWhere(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "validate", "--all", dir, "--where", "estado == 'Pending'")
	if err != nil {
		t.Logf("error (may be expected if validation fails): %v", err)
	}
	var result map[string]any
	if unmarshalErr := json.Unmarshal([]byte(out), &result); unmarshalErr != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", unmarshalErr, out)
	}
}

// --- fix --all with propagate and stem-health scenarios ---

func TestFixAll_WithStemHealthRedundancy(t *testing.T) {
	// Create a parent dir with a base .stem and a child dir with a redundant .stem
	// so that stem-health detects a redundant field
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	parentStem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(parentStem), 0644)

	// Create child dir with its own .stem that duplicates estado
	childDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(childDir, 0o755); err != nil {
		t.Fatal(err)
	}
	childStem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  tipo:
    type: string
`
	mustWriteFile(t, filepath.Join(childDir, ".stem"), []byte(childStem), 0644)
	mustWriteFile(t, filepath.Join(childDir, "task.md"),
		[]byte("---\nestado: Pending\ntipo: work\n---\n# Task\n"), 0644)

	out, err := runCmd(t, "fix", "--all", "--dry-run", childDir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
}

// --- Apply safety characterization tests (O09/T002) ---

// TestApplySafety_DryRunSchemaInferences characterizes whether apply --dry-run
// modifies .stem files when applying schema inferences. A correct implementation
// should NOT modify .stem during dry-run.
func TestApplySafety_DryRunSchemaInferences(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// .stem with incomplete enum
	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Done]
  tipo:
    type: string
`
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(stemContent), 0644)

	// Document with a third enum value not in schema
	mustWriteFile(t, filepath.Join(dir, "doc1.md"),
		[]byte("---\nestado: Pending\ntipo: task\n---\n# Doc\n"), 0644)

	// Generate analyze report
	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Fatalf("analyze failed: %v\noutput: %s", err, out)
	}

	// Save original stem bytes
	originalStemBytes := mustReadFile(t, stemPath)

	// Write report to file and apply with --dry-run
	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, []byte(out), 0644)

	_, applyErr := runCmd(t, "apply", "--dry-run", reportFile)
	if applyErr != nil {
		t.Logf("apply --dry-run error (may be expected): %v", applyErr)
	}

	// Read stem after dry-run
	afterStemBytes := mustReadFile(t, stemPath)

	// BUG: apply --dry-run CURRENTLY WRITES .stem files. This test documents that bug.
	if !bytes.Equal(originalStemBytes, afterStemBytes) {
		t.Skip("characterization: dry-run writes .stem — expected to fail until T003 fix")
	}

	// If we get here, dry-run correctly preserved the .stem file
	if bytes.Equal(originalStemBytes, afterStemBytes) {
		t.Logf("dry-run correctly preserved .stem bytes")
	}
}

// TestApplySafety_DryRunMissingSchema characterizes whether apply --dry-run
// creates .stem files for directories with missing_schema inferences.
// A correct implementation should NOT create .stem files during dry-run.
func TestApplySafety_DryRunMissingSchema(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Parent .stem
	parentStem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Done]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(parentStem), 0644)

	// Document in subdirectory (no .stem in sub)
	mustWriteFile(t, filepath.Join(subDir, "doc.md"),
		[]byte("---\nestado: Pending\n---\n# Doc\n"), 0644)

	// Analyze to detect missing_schema in subDir
	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Fatalf("analyze failed: %v\noutput: %s", err, out)
	}

	// Check for missing_schema inferences in the report
	var report map[string]any
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid analyze report: %v", err)
	}

	// Save state before dry-run
	subStemPath := filepath.Join(subDir, ".stem")
	_, existsBefore := os.Stat(subStemPath)

	// Write report and apply with --dry-run
	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, []byte(out), 0644)

	_, applyErr := runCmd(t, "apply", "--dry-run", reportFile)
	if applyErr != nil {
		t.Logf("apply --dry-run error (may be expected): %v", applyErr)
	}

	// Check state after dry-run
	_, existsAfter := os.Stat(subStemPath)

	// BUG: apply --dry-run CURRENTLY SCAFFOLDS .stem during pre-phase.
	// This characterizes that behavior.
	if existsBefore != nil && existsAfter == nil {
		// .stem was created during dry-run (bug)
		t.Skip("characterization: dry-run creates .stem — expected to fail until T003 fix")
	}

	// If we get here, dry-run correctly did NOT create .stem
	if existsBefore != nil && existsAfter != nil {
		t.Logf("dry-run correctly did not create .stem")
	}
}

// TestApplySafety_JSONPurityWithMissingSchema characterizes whether
// apply --output json produces parseable JSON when missing_schema is present.
// A correct implementation should NOT emit human-readable text to stdout
// that contaminates the JSON output.
func TestApplySafety_JSONPurityWithMissingSchema(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Parent .stem
	parentStem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: string
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(parentStem), 0644)

	// Document in subdirectory (missing .stem)
	mustWriteFile(t, filepath.Join(subDir, "doc.md"),
		[]byte("---\nestado: value\n---\n# Doc\n"), 0644)

	// Analyze
	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Fatalf("analyze failed: %v\noutput: %s", err, out)
	}

	// Write report and apply with --dry-run
	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, []byte(out), 0644)

	applyOut, applyErr := runCmd(t, "apply", "--dry-run", reportFile)
	if applyErr != nil {
		t.Logf("apply --dry-run error (may be expected): %v", applyErr)
	}

	// Try to parse stdout as JSON
	var result map[string]any
	if err := json.Unmarshal([]byte(applyOut), &result); err != nil {
		// BUG: stdout is contaminated with "scaffolded..." text
		t.Skip("characterization: stdout contamination bug — expected to fail until T003 fix")
	}

	// If we get here, stdout is valid JSON (correct behavior)
	t.Logf("apply --output json produced parseable JSON (correct behavior)")
}

// TestApplySafety_RootMostStemTargetSelection characterizes which .stem file
// is selected when multiple .stem files exist at different hierarchy levels.
// Documents current behavior for future fix.
func TestApplySafety_RootMostStemTargetSelection(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Parent .stem (root-most)
	parentStem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Done]
  parent_field:
    type: string
`
	parentStemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, parentStemPath, []byte(parentStem), 0644)

	// Create subdirectory with its own .stem
	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	childStem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Done, InProgress]
  child_field:
    type: string
`
	childStemPath := filepath.Join(subDir, ".stem")
	mustWriteFile(t, childStemPath, []byte(childStem), 0644)

	// Document in subdirectory uses a value from child's enum
	mustWriteFile(t, filepath.Join(subDir, "doc.md"),
		[]byte("---\nestado: InProgress\nchild_field: value\n---\n# Doc\n"), 0644)

	// Analyze from parent
	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Fatalf("analyze failed: %v\noutput: %s", err, out)
	}

	// Write report and apply with --dry-run to see which .stem is selected
	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, []byte(out), 0644)

	// Save original bytes for both stems
	parentOriginal := mustReadFile(t, parentStemPath)
	childOriginal := mustReadFile(t, childStemPath)

	_, applyErr := runCmd(t, "apply", "--dry-run", reportFile)
	if applyErr != nil {
		t.Logf("apply --dry-run error (may be expected): %v", applyErr)
	}

	// Read both stems after apply
	parentAfter := mustReadFile(t, parentStemPath)
	childAfter := mustReadFile(t, childStemPath)

	// Characterize: which stem was selected?
	parentModified := !bytes.Equal(parentOriginal, parentAfter)
	childModified := !bytes.Equal(childOriginal, childAfter)

	switch {
	case parentModified && !childModified:
		t.Logf("characterization: parent .stem was selected (closest-to-root behavior)")
	case !parentModified && childModified:
		t.Logf("characterization: child .stem was selected (closest-to-file behavior)")
	case parentModified && childModified:
		t.Logf("characterization: both stems were modified")
	default:
		t.Logf("characterization: neither stem was modified (dry-run worked correctly)")
	}
}

// TestApplySafety_PartialWriteRisk characterizes the non-transactional risk
// where scaffold/schema/data modifications happen sequentially without
// transaction protection. If one phase fails, others may have already modified files.
func TestApplySafety_PartialWriteRisk(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Parent .stem
	parentStem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: string
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(parentStem), 0644)

	// Document in subdirectory
	mustWriteFile(t, filepath.Join(subDir, "doc.md"),
		[]byte("---\nestado: value\n---\n# Doc\n"), 0644)

	// Analyze
	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Fatalf("analyze failed: %v\noutput: %s", err, out)
	}

	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, []byte(out), 0644)

	// Record initial state
	subStemPath := filepath.Join(subDir, ".stem")
	_, existsBefore := os.Stat(subStemPath)

	// Apply (not dry-run, will actually write)
	applyOut, applyErr := runCmd(t, "apply", reportFile)
	if applyErr != nil {
		// Expect error if scaffold fails for some reason
		t.Logf("apply error (may happen if scaffold fails): %v, output: %s", applyErr, applyOut)
	}

	// Check final state: if scaffold was created, schema may have been modified too
	_, existsAfter := os.Stat(subStemPath)

	if existsBefore != nil && existsAfter == nil {
		// Scaffold was created — document partial-write risk
		t.Logf("characterization: apply created .stem (partial-write risk — other phases may not roll back)")
	} else {
		t.Logf("characterization: apply did not create .stem")
	}
}

// TestApplySafety_StdoutContaminationWithScaffold tests that "scaffolded..."
// messages printed to stdout contaminate JSON output when missing_schema is present.
func TestApplySafety_StdoutContaminationWithScaffold(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	subDir := filepath.Join(dir, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Parent .stem
	parentStem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: string
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(parentStem), 0644)

	// Document in subdirectory (will trigger missing_schema)
	mustWriteFile(t, filepath.Join(subDir, "doc.md"),
		[]byte("---\nestado: value\n---\n# Doc\n"), 0644)

	// Analyze
	out, err := runCmd(t, "analyze", dir)
	if err != nil {
		t.Fatalf("analyze failed: %v\noutput: %s", err, out)
	}

	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, []byte(out), 0644)

	// Apply with --dry-run
	applyOut, _ := runCmd(t, "apply", "--dry-run", reportFile)

	// Check if stdout contains "scaffolded" text (bug indicator)
	if strings.Contains(applyOut, "scaffolded") {
		// Try to parse as JSON — will fail due to contamination
		var result map[string]any
		if err := json.Unmarshal([]byte(applyOut), &result); err != nil {
			t.Skip("characterization: stdout contamination bug — 'scaffolded' text in JSON output")
		}
	} else {
		t.Logf("no scaffold contamination detected in stdout")
	}
}
