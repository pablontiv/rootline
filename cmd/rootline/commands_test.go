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
	os.WriteFile(filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// Two markdown files
	os.WriteFile(filepath.Join(dir, "doc1.md"), []byte("---\nestado: Pending\ntipo: test\n---\n# Doc 1\n"), 0644)
	os.WriteFile(filepath.Join(dir, "doc2.md"), []byte("---\nestado: Completado\ntipo: prod\n---\n# Doc 2\n"), 0644)

	return dir
}

// resetFlags resets all global cobra flag state to avoid leaking between tests.
func resetFlags() {
	queryCount = false
	queryLimit = 0
	queryFrom = "."
	queryWhere = nil
	validateAll = false
	outputFormat = "json"
	fieldPath = nil
	statsFrom = "."

	// Reset slice flags at the cobra level too (StringSliceVar appends internally)
	if f := queryCmd.Flags().Lookup("where"); f != nil {
		f.Value.Set("")
		f.Changed = false
	}
	if f := rootCmd.PersistentFlags().Lookup("field"); f != nil {
		f.Value.Set("")
		f.Changed = false
	}
	if f := rootCmd.PersistentFlags().Lookup("output"); f != nil {
		f.Value.Set("json")
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
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado eq Pending")
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
	json.Unmarshal([]byte(out), &result)
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
	// Use two separate --where flags to avoid StringSliceVar splitting on comma
	out, err := runCmd(t, "query", "--from", dir, "--where", "tipo in test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md with 'in' operator, got: %s", out)
	}
}

func TestQueryWhereContains(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "tipo contains tes")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md with contains, got: %s", out)
	}
}

func TestQueryWhereNe(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado ne Pending")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc2.md") {
		t.Errorf("expected doc2.md with ne, got: %s", out)
	}
}

func TestQueryWhereExists(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "tipo exists")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected results with exists, got: %s", out)
	}
}

func TestQueryWhereInvalid(t *testing.T) {
	dir := setupTestDir(t)
	_, err := runCmd(t, "query", "--from", dir, "--where", "bad")
	if err == nil {
		t.Fatal("expected error for invalid where expression")
	}
}

func TestQueryWhereUnknownOp(t *testing.T) {
	dir := setupTestDir(t)
	_, err := runCmd(t, "query", "--from", dir, "--where", "estado nope val")
	if err == nil {
		t.Fatal("expected error for unknown operator")
	}
}

func TestQueryMultipleWhere(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado eq Pending", "--where", "tipo eq test")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "doc1.md") {
		t.Errorf("expected doc1.md with multiple where, got: %s", out)
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
	var result treeNode
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result.Total != 2 {
		t.Errorf("expected 2 total, got %d", result.Total)
	}
	if result.Completed != 1 {
		t.Errorf("expected 1 completed, got %d", result.Completed)
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
	os.WriteFile(filepath.Join(dir, "bad.md"), []byte("---\ntipo: test\n---\n# Bad\n"), 0644)
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

// --- parseWhereExpr edge cases ---

func TestParseWhereExprMissingValue(t *testing.T) {
	tests := []struct {
		expr    string
		wantErr bool
	}{
		{"estado eq", true},
		{"estado ne", true},
		{"estado contains", true},
		{"estado in", true},
		{"estado exists", false},
	}
	for _, tt := range tests {
		_, err := parseWhereExpr(tt.expr)
		if tt.wantErr && err == nil {
			t.Errorf("parseWhereExpr(%q): expected error", tt.expr)
		}
		if !tt.wantErr && err != nil {
			t.Errorf("parseWhereExpr(%q): unexpected error: %v", tt.expr, err)
		}
	}
}
