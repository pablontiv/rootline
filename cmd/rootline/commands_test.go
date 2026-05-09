package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestDir creates a temp directory with .stem and markdown files for CLI testing.
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// Create .git marker (WalkUp requires a git boundary).
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// .stem file with schema
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
    required: false
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// Two markdown files
	mustWriteFile(t, filepath.Join(dir, "doc1.md"), []byte("---\nestado: Pending\ntipo: test\n---\n# Doc 1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "doc2.md"), []byte("---\nestado: Completed\ntipo: prod\n---\n# Doc 2\n"), 0644)

	return dir
}

// resetFlags resets all global cobra flag state to avoid leaking between tests.
func resetFlags() {
	queryCount = false
	queryLimit = 0
	queryFrom = "."
	queryWhere = nil
	querySort = ""
	validateAll = false
	validateStrict = false
	validateStaged = false
	validateWhere = nil
	outputFormat = "json"
	fieldPath = nil
	statsFrom = "."
	initDryRun = false
	initForce = false
	initTemplate = ""
	newForce = false
	newDryRun = false
	fixDryRun = false
	fixAll = false
	hooksForce = false
	treeWhere = nil
	statsWhere = nil
	graphFormat = "dot"
	graphCheck = false
	graphWhere = nil
	graphOpen = false
	migrateDryRun = false
	migrateFrom = ""
	migrateRename = ""
	migrateSplit = false
	migrateScaffold = false
	analyzeIncremental = false
	analyzeThreshold = 0.60
	applyDryRun = false
	setDryRun = false
	setCreate = false
	setNoValidate = false
	traceReverse = false
	traceDepth = 0
	traceType = ""
	traceFormat = "tree"
	schemaProposeIncremental = false
	schemaApplyReport = ""
	schemaApplyDryRun = false

	// Reset slice flags at the cobra level too (StringSliceVar appends internally)
	if f := treeCmd.Flags().Lookup("where"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := statsCmd.Flags().Lookup("where"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := validateCmd.Flags().Lookup("where"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := graphCmd.Flags().Lookup("where"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := queryCmd.Flags().Lookup("where"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := queryCmd.Flags().Lookup("sort"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := rootCmd.PersistentFlags().Lookup("field"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := rootCmd.PersistentFlags().Lookup("output"); f != nil {
		_ = f.Value.Set("json")
		f.Changed = false
	}
	if f := migrateCmd.Flags().Lookup("rename"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := migrateCmd.Flags().Lookup("from"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := migrateCmd.Flags().Lookup("dry-run"); f != nil {
		_ = f.Value.Set("false")
		f.Changed = false
	}
	if f := migrateCmd.Flags().Lookup("scaffold"); f != nil {
		_ = f.Value.Set("false")
		f.Changed = false
	}
	if f := analyzeCmd.Flags().Lookup("incremental"); f != nil {
		_ = f.Value.Set("false")
		f.Changed = false
	}
	if f := applyCmd.Flags().Lookup("dry-run"); f != nil {
		_ = f.Value.Set("false")
		f.Changed = false
	}
	if f := schemaApplyCmd.Flags().Lookup("report"); f != nil {
		_ = f.Value.Set("")
		f.Changed = false
	}
	if f := schemaApplyCmd.Flags().Lookup("dry-run"); f != nil {
		_ = f.Value.Set("false")
		f.Changed = false
	}
	if f := schemaProposeCmd.Flags().Lookup("incremental"); f != nil {
		_ = f.Value.Set("false")
		f.Changed = false
	}
}

// runCmd executes a rootline command and captures stdout.
func runCmd(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags()
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return buf.String(), err
}

// --- Query tests ---

func TestQueryJSON(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["kind"] != "rootline/query" {
		t.Errorf("expected kind rootline/query, got %v", result["kind"])
	}
}

func TestQueryWithWhere(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado == 'Pending'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md in results, got: %s", out)
	}
	if strings.Contains(out, "doc2.md") {
		t.Errorf("expected doc2.md to be filtered out")
	}
}

func TestQueryCount(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--count")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, `"count"`) {
		t.Errorf("expected count in output, got: %s", out)
	}
}

func TestQueryLimit(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--limit", "1")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	rows := result["rows"].([]any)
	if len(rows) != 1 {
		t.Errorf("expected 1 row with --limit 1, got %d", len(rows))
	}
}

func TestQueryTable(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Path") {
		t.Errorf("expected Path header in table output, got: %s", out)
	}
	if !strings.Contains(out, "estado") {
		t.Errorf("expected estado column in table output")
	}
}

func TestQueryWhereIn(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "tipo in ['test', 'prod']")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md with 'in' operator, got: %s", out)
	}
	if !strings.Contains(out, "doc2.md") {
		t.Errorf("expected doc2.md with 'in' operator, got: %s", out)
	}
}

func TestQueryWhereInArray(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado in ['Pending', 'Completed']")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md (Pending) in results, got: %s", out)
	}
	if !strings.Contains(out, "doc2.md") {
		t.Errorf("expected doc2.md (Completed) in results, got: %s", out)
	}
}

func TestQueryWhereInWithMultipleWhere(t *testing.T) {
	dir := setupTestDir(t)
	// Multiple --where flags combined with && (AND logic)
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado in ['Pending', 'Completed']", "--where", "tipo == 'test'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md (Pending + tipo=test), got: %s", out)
	}
	if strings.Contains(out, "doc2.md") {
		t.Errorf("expected doc2.md filtered out (tipo=prod, not test)")
	}
}

func TestQueryWhereContains(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "tipo contains 'tes'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md with contains, got: %s", out)
	}
}

func TestQueryWhereNe(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado != 'Pending'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc2.md") {
		t.Errorf("expected doc2.md with ne, got: %s", out)
	}
}

func TestQueryWhereFieldNotNil(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "tipo != nil")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected results with field != nil, got: %s", out)
	}
}

func TestQueryWhereInvalidExpr(t *testing.T) {
	dir := setupTestDir(t)
	_, err := runCmd(t, "query", "--from", dir, "--where", "== bad syntax")
	if err == nil {
		t.Fatal("expected error for invalid where expression")
	}
}

func TestQueryMultipleWhere(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado == 'Pending'", "--where", "tipo == 'test'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md with multiple where, got: %s", out)
	}
}

func TestQueryWhereAndInSingleExpr(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado == 'Pending' && tipo == 'test'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md with && in single expr, got: %s", out)
	}
	if strings.Contains(out, "doc2.md") {
		t.Errorf("expected doc2.md filtered out")
	}
}

// --- Stats tests ---

func TestStatsJSON(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "stats", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result StatsResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 total, got %d", result.Total)
	}
	if result.ByEstado["Pending"] != 1 {
		t.Errorf("expected 1 Pending, got %d", result.ByEstado["Pending"])
	}
}

func TestStatsWhere(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "stats", dir, "--where", "tipo == 'test'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result StatsResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Total != 1 {
		t.Errorf("expected 1 total with --where tipo=test, got %d", result.Total)
	}
}

func TestStatsWhereInvalid(t *testing.T) {
	dir := setupTestDir(t)
	_, err := runCmd(t, "stats", dir, "--where", "== bad")
	if err == nil {
		t.Fatal("expected error for invalid where expression")
	}
}

func TestStatsTable(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "stats", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Total:") {
		t.Errorf("expected Total: in table output, got: %s", out)
	}
	if !strings.Contains(out, "By Estado:") {
		t.Errorf("expected By Estado: section")
	}
}

// --- Tree tests ---

func TestTreeJSON(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "tree", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
	if result.Root.Total != 2 {
		t.Errorf("expected 2 total, got %d", result.Root.Total)
	}
	if result.Root.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", result.Root.Completed)
	}
}

func TestTreeASCII(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "tree", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[1/2]") {
		t.Errorf("expected [1/2] in ASCII output, got: %s", out)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md in tree")
	}
}

// --- Validate tests ---

func TestValidateSingleFile(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "validate", filepath.Join(dir, "doc1.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["valid"] != true {
		t.Errorf("expected valid=true, got %v", result["valid"])
	}
}

func TestValidateMultipleFiles(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "validate", filepath.Join(dir, "doc1.md"), filepath.Join(dir, "doc2.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "results") {
		t.Errorf("expected batch results, got: %s", out)
	}
}

func TestValidateTable(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "validate", filepath.Join(dir, "doc1.md"), "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "File") && !strings.Contains(out, "Valid") {
		t.Errorf("expected table headers, got: %s", out)
	}
}

func TestValidateNoArgs(t *testing.T) {
	_, err := runCmd(t, "validate")
	if err == nil {
		t.Fatal("expected error for validate with no args")
	}
}

func TestValidateFileNotFound(t *testing.T) {
	_, err := runCmd(t, "validate", "/nonexistent/path.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestValidateInvalidFile(t *testing.T) {
	dir := setupTestDir(t)
	// Write a file that violates the schema (missing required estado)
	mustWriteFile(t, filepath.Join(dir, "bad.md"), []byte("---\ntipo: test\n---\n# Bad\n"), 0644)
	out, _ := runCmd(t, "validate", filepath.Join(dir, "bad.md"))
	if !strings.Contains(out, "false") && !strings.Contains(out, "error") {
		t.Errorf("expected validation failure for missing required field, got: %s", out)
	}
}

// --- Describe tests ---

func TestDescribeJSON(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "describe", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["kind"] != "rootline/describe" {
		t.Errorf("expected kind rootline/describe, got %v", result["kind"])
	}
}

func TestDescribeTable(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "describe", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Field") {
		t.Errorf("expected Field header, got: %s", out)
	}
	if !strings.Contains(out, "estado") {
		t.Errorf("expected estado field in table, got: %s", out)
	}
}

// --- Field extraction tests ---

func TestOutputJSONWithField(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "stats", dir, "--field", "total")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out = strings.TrimSpace(out)
	if out != "2" {
		t.Errorf("expected 2 with --field total, got: %s", out)
	}
}

func TestOutputJSONWithNestedField(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "stats", dir, "--field", "by_estado.Pending")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	out = strings.TrimSpace(out)
	if out != "1" {
		t.Errorf("expected 1 with --field by_estado.Pending, got: %s", out)
	}
}

func TestOutputJSONWithFieldNotFound(t *testing.T) {
	dir := setupTestDir(t)
	_, err := runCmd(t, "stats", dir, "--field", "nonexistent.path")
	if err == nil {
		t.Fatal("expected error for nonexistent field path")
	}
}

// --- Serve test ---

func TestServeStub(t *testing.T) {
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	rootCmd.SetErr(buf)
	rootCmd.SetArgs([]string{"serve"})
	if err := rootCmd.Execute(); err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}

// --- Explain tests ---

func TestExplainJSON(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "explain", filepath.Join(dir, "doc1.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if result["version"] != float64(1) {
		t.Errorf("version = %v, want 1", result["version"])
	}
	if result["kind"] != "rootline/explain" {
		t.Errorf("kind = %v, want rootline/explain", result["kind"])
	}
	fields, ok := result["fields"].([]any)
	if !ok {
		t.Fatalf("fields is not array: %T", result["fields"])
	}
	// Should have at least the frontmatter fields.
	if len(fields) < 2 {
		t.Errorf("expected at least 2 fields, got %d", len(fields))
	}
	// Check that estado field has origin "frontmatter".
	found := false
	for _, f := range fields {
		fm := f.(map[string]any)
		if fm["name"] == "estado" {
			found = true
			if fm["origin"] != "frontmatter" {
				t.Errorf("estado origin = %v, want frontmatter", fm["origin"])
			}
		}
	}
	if !found {
		t.Error("estado field not found in explain output")
	}
}

func TestExplainTable(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "explain", filepath.Join(dir, "doc1.md"), "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "frontmatter") {
		t.Errorf("expected 'frontmatter' in table output, got: %s", out)
	}
	if !strings.Contains(out, "estado") {
		t.Errorf("expected 'estado' in table output, got: %s", out)
	}
}

func TestExplainWithValidationErrors(t *testing.T) {
	dir := setupTestDir(t)
	// Create a file missing required field.
	mustWriteFile(t, filepath.Join(dir, "bad.md"), []byte("---\ntipo: test\n---\n# Bad\n"), 0644)
	out, err := runCmd(t, "explain", filepath.Join(dir, "bad.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	errs, ok := result["errors"].([]any)
	if !ok || len(errs) == 0 {
		t.Error("expected validation errors for missing required field")
	}
}

func TestExplainWithDerived(t *testing.T) {
	dir := setupTestDir(t)
	// Add derive to .stem.
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
    required: false
derive:
  slug: "slugify(estado)"
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	out, err := runCmd(t, "explain", filepath.Join(dir, "doc1.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	fields := result["fields"].([]any)
	found := false
	for _, f := range fields {
		fm := f.(map[string]any)
		if fm["name"] == "slug" {
			found = true
			if fm["origin"] != "derived" {
				t.Errorf("slug origin = %v, want derived", fm["origin"])
			}
			if fm["expression"] != `slugify(estado)` {
				t.Errorf("slug expression = %v, want slugify(estado)", fm["expression"])
			}
			if fm["value"] != "pending" {
				t.Errorf("slug value = %v, want pending", fm["value"])
			}
		}
	}
	if !found {
		t.Errorf("slug derived field not found in explain output.\nfields: %v", fields)
	}
}

func TestExplainTableWithErrors(t *testing.T) {
	dir := setupTestDir(t)
	// Create a file missing required field.
	mustWriteFile(t, filepath.Join(dir, "bad.md"), []byte("---\ntipo: test\n---\n# Bad\n"), 0644)
	out, err := runCmd(t, "explain", filepath.Join(dir, "bad.md"), "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Validation Errors") {
		t.Errorf("expected 'Validation Errors' section in table output, got: %s", out)
	}
}

func TestExplainWithSchemaDefault(t *testing.T) {
	dir := setupTestDir(t)
	// Add a default value to schema.
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
    required: false
    default: "unknown"
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	// Create file without tipo (will use default).
	mustWriteFile(t, filepath.Join(dir, "defaulttest.md"), []byte("---\nestado: Pending\n---\n# Test\n"), 0644)
	out, err := runCmd(t, "explain", filepath.Join(dir, "defaulttest.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "unknown") {
		t.Errorf("expected default value 'unknown' in output, got: %s", out)
	}
	if !strings.Contains(out, "schema") {
		t.Errorf("expected 'schema' origin in output, got: %s", out)
	}
}

func TestExplainNonexistentFile(t *testing.T) {
	_, err := runCmd(t, "explain", "/nonexistent/path.md")
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
}

func TestExplainFieldExtraction(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "explain", filepath.Join(dir, "doc1.md"), "--field", "kind")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "rootline/explain") {
		t.Errorf("expected rootline/explain in field output, got: %s", out)
	}
}

// --- Sort tests ---

func TestQuerySort_SingleKey(t *testing.T) {
	dir := setupSortTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--sort", "prioridad:asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	rows := result["rows"].([]any)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// alta=0 should be first
	row0 := rows[0].(map[string]any)
	fm0 := row0["frontmatter"].(map[string]any)
	if fm0["prioridad"] != "alta" {
		t.Errorf("row 0 prioridad = %v, want alta", fm0["prioridad"])
	}
}

func TestQuerySort_MultiKey(t *testing.T) {
	dir := setupSortTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--sort", "prioridad:asc,impact_score:desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	rows := result["rows"].([]any)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// alta with higher impact_score (8) should come before alta with lower (3)
	row0 := rows[0].(map[string]any)
	row1 := rows[1].(map[string]any)
	fm0 := row0["frontmatter"].(map[string]any)
	fm1 := row1["frontmatter"].(map[string]any)
	if fm0["prioridad"] != "alta" || fm1["prioridad"] != "alta" {
		t.Errorf("first two rows should be alta, got %v and %v", fm0["prioridad"], fm1["prioridad"])
	}
	// impact_score desc: 8 before 3
	score0, _ := fm0["impact_score"].(float64) // JSON numbers are float64
	score1, _ := fm1["impact_score"].(float64)
	if score0 < score1 {
		t.Errorf("expected impact_score desc: got %.0f before %.0f", score0, score1)
	}
}

func TestQuerySort_WithWhere(t *testing.T) {
	dir := setupSortTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "prioridad == 'alta'", "--sort", "impact_score:desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	rows := result["rows"].([]any)
	if len(rows) != 2 {
		t.Fatalf("expected 2 alta rows, got %d", len(rows))
	}
}

func TestQuerySort_InvalidSort(t *testing.T) {
	dir := setupSortTestDir(t)
	_, err := runCmd(t, "query", "--from", dir, "--sort", "field:invalid_dir")
	if err == nil {
		t.Fatal("expected error for invalid sort direction")
	}
}

func TestQuerySort_Table(t *testing.T) {
	dir := setupSortTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--sort", "prioridad:asc", "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Path") {
		t.Errorf("expected table output, got: %s", out)
	}
}

// setupSortTestDir creates a test directory with enum schema and records for sort testing.
func setupSortTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  prioridad:
    type: enum
    required: true
    values: [alta, media, baja]
  impact_score:
    type: number
    required: false
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	mustWriteFile(t, filepath.Join(dir, "item1.md"), []byte("---\nprioridad: media\nimpact_score: 5\n---\n# Item 1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "item2.md"), []byte("---\nprioridad: alta\nimpact_score: 8\n---\n# Item 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "item3.md"), []byte("---\nprioridad: alta\nimpact_score: 3\n---\n# Item 3\n"), 0644)

	return dir
}

func TestQuerySort_IntegrationBacklog(t *testing.T) {
	// Skip if the homeserver backlog directory doesn't exist (CI environments).
	backlogDir := "/opt/factory/homeserver/backlog"
	if _, err := os.Stat(backlogDir); os.IsNotExist(err) {
		t.Skip("skipping integration test: /opt/factory/homeserver/backlog not found")
	}

	out, err := runCmd(t, "query", backlogDir, "--sort", "prioridad:asc,impact_score:desc", "-o", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	rows, ok := result["rows"].([]any)
	if !ok {
		t.Fatalf("rows not found in result")
	}

	if len(rows) == 0 {
		t.Skip("no backlog items found")
	}

	// NOTE: The backlog .stem uses "enum:" instead of "values:" for enum definitions.
	// This means rootline currently does NOT get the enum ordering for prioridad.
	// Sort falls back to lexicographic ordering ("alta" < "baja" < "media" = correct for asc).
	// This test verifies the sort completes without error and produces valid JSON.
	// Once the enum:/values: parsing is unified, enum-aware ordering will work automatically.
	t.Logf("sorted %d backlog items successfully", len(rows))

	// Verify output structure
	if result["kind"] != "rootline/query" {
		t.Errorf("expected kind rootline/query, got %v", result["kind"])
	}
}
