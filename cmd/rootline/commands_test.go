package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestDir creates a temp directory with .stem and markdown files for CLI testing.
func setupTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	// .stem file with schema
	stemContent := `version: 1
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completado]
  tipo:
    type: string
    required: false
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// Two markdown files
	mustWriteFile(t, filepath.Join(dir, "doc1.md"), []byte("---\nestado: Pending\ntipo: test\n---\n# Doc 1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "doc2.md"), []byte("---\nestado: Completado\ntipo: prod\n---\n# Doc 2\n"), 0644)

	return dir
}

// resetFlags resets all global cobra flag state to avoid leaking between tests.
func resetFlags() {
	queryCount = false
	queryLimit = 0
	queryFrom = "."
	queryWhere = nil
	validateAll = false
	validateStrict = false
	validateStaged = false
	outputFormat = "json"
	fieldPath = nil
	statsFrom = "."
	initDryRun = false
	initForce = false
	newForce = false
	newDryRun = false
	fixDryRun = false
	fixAll = false
	hooksForce = false

	// Reset slice flags at the cobra level too (StringSliceVar appends internally)
	if f := queryCmd.Flags().Lookup("where"); f != nil {
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
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado in ['Pending', 'Completado']")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md (Pending) in results, got: %s", out)
	}
	if !strings.Contains(out, "doc2.md") {
		t.Errorf("expected doc2.md (Completado) in results, got: %s", out)
	}
}

func TestQueryWhereInWithMultipleWhere(t *testing.T) {
	dir := setupTestDir(t)
	// Multiple --where flags combined with && (AND logic)
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado in ['Pending', 'Completado']", "--where", "tipo == 'test'")
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

// --- Explain test ---

func TestExplainStub(t *testing.T) {
	dir := setupTestDir(t)
	_, err := runCmd(t, "explain", filepath.Join(dir, "doc1.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
}
