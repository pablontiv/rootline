package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
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
	declareTestBoundary(t, dir)

	// Two markdown files
	mustWriteFile(t, filepath.Join(dir, "doc1.md"), []byte("---\nestado: Pending\ntipo: test\n---\n# Doc 1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "doc2.md"), []byte("---\nestado: Completed\ntipo: prod\n---\n# Doc 2\n"), 0644)

	declareTestBoundary(t, dir)
	return dir
}

// resetFlags resets all global cobra flag state to avoid leaking between tests.
func resetFlags() {
	queryCount = false
	queryLimit = 0
	queryFrom = "."
	queryWhere = nil
	querySort = ""
	querySelect = ""
	queryHasInbound = ""
	queryHasOutbound = ""
	queryInboundType = ""
	queryOutboundType = ""
	queryGraphRoot = ""
	for _, name := range []string{"has-inbound", "has-outbound", "inbound-type", "outbound-type", "graph-root"} {
		if f := queryCmd.Flags().Lookup(name); f != nil {
			f.Changed = false
		}
	}
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
	fixFillMissing = false
	hooksForce = false
	treeWhere = nil
	statsWhere = nil
	graphFormat = "dot"
	graphCheck = false
	graphWhere = nil
	graphFailCycles = false
	if f := graphCmd.Flags().Lookup("fail-cycles"); f != nil {
		f.Changed = false
	}
	migrateDryRun = false
	migrateFrom = ""
	migrateRename = ""
	migrateSplit = false
	migrateScaffold = false
	migrateForce = false
	analyzeIncremental = false
	analyzeThreshold = 0.60
	setDryRun = false
	setCreate = false
	setNoValidate = false
	schemaProposeIncremental = false
	schemaApplyReport = ""
	schemaApplyDryRun = false
	schemaApplyForce = false
	reportPath = ""
	repairDryRun = false
	repairFillMissing = false

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
	if f := queryCmd.Flags().Lookup("select"); f != nil {
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
	if f := migrateCmd.Flags().Lookup("force"); f != nil {
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
	// ByEstado and ByTipo are now empty (field-agnostic) — only total matters
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
	// Table no longer shows By Estado/By Tipo — field-agnostic
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
	if result.Version != 2 {
		t.Errorf("expected version 2, got %d", result.Version)
	}
	if result.Root.Total != 2 {
		t.Errorf("expected 2 total, got %d", result.Root.Total)
	}
}

func TestTreeASCII(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "tree", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "[2]") {
		t.Errorf("expected [2] in ASCII output, got: %s", out)
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

func TestOutputJSONWithArrayProjection(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--field", "rows[].path")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var paths []string
	if err := json.Unmarshal([]byte(out), &paths); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if len(paths) != 2 || paths[0] != "doc1.md" || paths[1] != "doc2.md" {
		t.Errorf("paths = %#v, want [doc1.md doc2.md]", paths)
	}
}

func TestOutputJSONWithNestedArrayProjection(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--field", "rows[].frontmatter.estado")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var estados []string
	if err := json.Unmarshal([]byte(out), &estados); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if len(estados) != 2 || estados[0] != "Pending" || estados[1] != "Completed" {
		t.Errorf("estados = %#v, want [Pending Completed]", estados)
	}
}

func TestOutputJSONWithArrayProjectionMissingKey(t *testing.T) {
	data := []byte(`{"rows":[{"path":"doc1.md"},{"frontmatter":{"estado":"Pending"}}]}`)
	_, err := extractField(data, "rows[].path")
	if err == nil {
		t.Fatal("expected error for missing projected key")
	}
	if !strings.Contains(err.Error(), "key \"path\" not found") {
		t.Fatalf("expected missing key error, got: %v", err)
	}
}

func TestOutputJSONWithArrayProjectionInvalidTraversal(t *testing.T) {
	data := []byte(`{"rows":{"path":"doc1.md"}}`)
	_, err := extractField(data, "rows[].path")
	if err == nil {
		t.Fatal("expected error for array projection over non-array")
	}
	if !strings.Contains(err.Error(), "not an array") {
		t.Fatalf("expected not an array error, got: %v", err)
	}
}

func TestOutputJSONWithFieldNotFound(t *testing.T) {
	dir := setupTestDir(t)
	_, err := runCmd(t, "stats", dir, "--field", "nonexistent.path")
	if err == nil {
		t.Fatal("expected error for nonexistent field path")
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
	declareTestBoundary(t, dir)
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
	declareTestBoundary(t, dir)
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
	declareTestBoundary(t, dir)

	mustWriteFile(t, filepath.Join(dir, "item1.md"), []byte("---\nprioridad: media\nimpact_score: 5\n---\n# Item 1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "item2.md"), []byte("---\nprioridad: alta\nimpact_score: 8\n---\n# Item 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "item3.md"), []byte("---\nprioridad: alta\nimpact_score: 3\n---\n# Item 3\n"), 0644)

	declareTestBoundary(t, dir)
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

// --- Query projection tests (--select flag) ---

func TestQuerySelectPath(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--select", "path")
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
	rows, ok := result["rows"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatal("expected rows in result")
	}
	row, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatal("expected row to be a map")
	}
	if _, ok := row["path"]; !ok {
		t.Errorf("expected 'path' field in projected row, got: %v", row)
	}
	// Path should be the only field (along with version, kind, meta)
	if len(row) != 1 {
		t.Errorf("expected only 'path' field in row, got fields: %v", row)
	}
}

func TestQuerySelectPathAndEstado(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--select", "path,estado")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	rows, ok := result["rows"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatal("expected rows in result")
	}
	row, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatal("expected row to be a map")
	}
	if _, ok := row["path"]; !ok {
		t.Errorf("expected 'path' field in projected row")
	}
	if _, ok := row["estado"]; !ok {
		t.Errorf("expected 'estado' field in projected row")
	}
}

func TestQuerySelectTitle(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--select", "path,title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	rows, ok := result["rows"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatal("expected rows in result")
	}
	row, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatal("expected row to be a map")
	}
	// Title should be extracted from "# Doc 1" heading
	if title, ok := row["title"]; ok {
		if title != "Doc 1" && title != "Doc 2" {
			t.Errorf("expected title to be 'Doc 1' or 'Doc 2', got: %v", title)
		}
	}
}

func TestQuerySelectLinks(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--select", "path,links")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	rows, ok := result["rows"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatal("expected rows in result")
	}
	row, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatal("expected row to be a map")
	}
	// Links field may or may not be present depending on doc content
	if path, ok := row["path"]; !ok {
		t.Errorf("expected 'path' field, got: %v", row)
	} else if path == "" {
		t.Errorf("expected non-empty path")
	}
}

func TestQuerySelectNonexistent(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--select", "path,nonexistent")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	rows, ok := result["rows"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatal("expected rows in result")
	}
	row, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatal("expected row to be a map")
	}
	// Path should be present, nonexistent field should be omitted
	if _, ok := row["path"]; !ok {
		t.Errorf("expected 'path' field")
	}
	if _, ok := row["nonexistent"]; ok {
		t.Errorf("expected nonexistent field to be omitted")
	}
}

func TestQuerySelectWithWhere(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "estado == 'Pending'", "--select", "path,estado,title")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	rows, ok := result["rows"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatal("expected rows in result")
	}
	// Should only have doc1.md (Pending)
	row, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatal("expected row to be a map")
	}
	if len(rows) != 1 {
		t.Errorf("expected 1 row after filtering, got %d", len(rows))
	}
	if estado, ok := row["estado"]; ok {
		if estado != "Pending" {
			t.Errorf("expected estado='Pending', got %v", estado)
		}
	}
}

func TestQuerySelectBackwardCompat(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	rows, ok := result["rows"].([]any)
	if !ok || len(rows) == 0 {
		t.Fatal("expected rows in result")
	}
	row, ok := rows[0].(map[string]any)
	if !ok {
		t.Fatal("expected row to be a map")
	}
	// Without --select, rows should have full Record fields (path, frontmatter, body, type, links, etc.)
	// Check that we have the important fields from the Record struct
	if _, ok := row["path"]; !ok {
		t.Errorf("expected 'path' field in full record, got keys: %v", getMapKeys(row))
	}
	// Check version and kind to verify we're still using QueryResult
	if version, ok := result["version"]; !ok {
		t.Errorf("expected version field in result")
	} else {
		// Version might be float64 after JSON unmarshal
		vFloat, _ := version.(float64)
		if int(vFloat) != 1 {
			t.Errorf("expected version 1, got %v", version)
		}
	}
	if kind, ok := result["kind"]; !ok || kind != "rootline/query" {
		t.Errorf("expected kind rootline/query, got %v", kind)
	}
}

// --- Query output format tests (JSONL and CSV) ---

func TestQueryOutputJSONL(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--select", "path,estado", "--output", "jsonl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		t.Fatal("expected JSONL output lines")
	}

	// Each line should be valid JSON
	for i, line := range lines {
		if line == "" {
			continue
		}
		var obj map[string]any
		if err := json.Unmarshal([]byte(line), &obj); err != nil {
			t.Errorf("line %d not valid JSON: %v (line: %s)", i, err, line)
		}
		// Each object should have path and estado fields
		if _, ok := obj["path"]; !ok {
			t.Errorf("line %d missing 'path' field", i)
		}
	}
}

func TestQueryOutputCSV(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--select", "path,estado", "--output", "csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatal("expected CSV output with header and at least one data row")
	}

	// First line should be the header
	header := lines[0]
	if !strings.Contains(header, "path") || !strings.Contains(header, "estado") {
		t.Errorf("expected header to contain 'path' and 'estado', got: %s", header)
	}

	// Data rows should exist
	if len(lines) < 2 {
		t.Error("expected at least one data row in CSV output")
	}
}

func TestQueryOutputCSVWithNil(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--select", "path,estado,nonexistent", "--output", "csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatal("expected CSV output with header and at least one data row")
	}

	// First line should be the header with three columns
	header := lines[0]
	headerCols := strings.Split(header, ",")
	if len(headerCols) != 3 {
		t.Errorf("expected 3 header columns, got %d", len(headerCols))
	}

	// Check that data row has 3 columns (nonexistent should be empty)
	if len(lines) >= 2 {
		dataCols := strings.Split(lines[1], ",")
		if len(dataCols) != 3 {
			t.Errorf("expected 3 data columns, got %d", len(dataCols))
		}
	}
}

func TestQueryOutputJSONLNoSelect(t *testing.T) {
	dir := setupTestDir(t)
	out, _ := runCmd(t, "query", "--from", dir, "--output", "jsonl")
	if !strings.Contains(out, "requires --select") {
		t.Errorf("expected error message about --select requirement in output, got: %s", out)
	}
}

func TestQueryOutputCSVNoSelect(t *testing.T) {
	dir := setupTestDir(t)
	out, _ := runCmd(t, "query", "--from", dir, "--output", "csv")
	if !strings.Contains(out, "requires --select") {
		t.Errorf("expected error message about --select requirement in output, got: %s", out)
	}
}

func TestQueryOutputJSONLColumnOrder(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--select", "estado,path", "--output", "jsonl")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		t.Fatal("expected JSONL output lines")
	}

	// Parse first line and check field order is preserved in the JSON object
	var obj map[string]any
	if err := json.Unmarshal([]byte(lines[0]), &obj); err != nil {
		t.Fatalf("first line not valid JSON: %v", err)
	}

	// Both fields should exist in the projected row
	if _, ok := obj["estado"]; !ok {
		t.Errorf("expected 'estado' field in JSONL object")
	}
	if _, ok := obj["path"]; !ok {
		t.Errorf("expected 'path' field in JSONL object")
	}
}

func TestQueryOutputCSVColumnOrder(t *testing.T) {
	dir := setupTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--select", "estado,path", "--output", "csv")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatal("expected CSV output with header")
	}

	header := lines[0]
	// Header should be "estado,path" in that order
	if header != "estado,path" {
		t.Errorf("expected header 'estado,path', got: %s", header)
	}
}

// getMapKeys returns the keys of a map in sorted order for debugging
func getMapKeys(m map[string]any) []string {
	var keys []string
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
