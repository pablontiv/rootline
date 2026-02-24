package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupMigrateDir creates a temp dir with .git, .stem, and markdown files for migrate CLI tests.
func setupMigrateDir(t *testing.T, stemContent string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	// Create .git marker so WalkUp has a boundary.
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	if stemContent != "" {
		mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		mustWriteFile(t, path, []byte(content), 0644)
	}

	return dir
}

// --- migrate --rename tests ---

func TestMigrateRenameJSON(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
    required: true
  estado:
    type: enum
    values: [Pending, Done]
`
	files := map[string]string{
		"doc1.md": "---\ntitulo: Hello\nestado: Pending\n---\n# Content\n",
		"doc2.md": "---\ntitulo: World\nestado: Done\n---\n# More\n",
	}
	dir := setupMigrateDir(t, stem, files)

	out, err := runCmd(t, "migrate", dir, "--rename", "titulo=title")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if result["kind"] != "rootline/migrate-rename" {
		t.Errorf("kind = %v, want rootline/migrate-rename", result["kind"])
	}
	summary := result["summary"].(map[string]any)
	if summary["files_updated"].(float64) != 2 {
		t.Errorf("expected 2 files updated, got %v", summary["files_updated"])
	}
}

func TestMigrateRenameTable(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
    required: true
`
	files := map[string]string{
		"doc1.md": "---\ntitulo: Hello\n---\n# Content\n",
	}
	dir := setupMigrateDir(t, stem, files)

	out, err := runCmd(t, "migrate", dir, "--rename", "titulo=title", "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "updated") {
		t.Errorf("expected 'updated' in table output, got: %s", out)
	}
	if !strings.Contains(out, "titulo") || !strings.Contains(out, "title") {
		t.Errorf("expected field names in output, got: %s", out)
	}
}

func TestMigrateRenameTableNoMatch(t *testing.T) {
	stem := `version: 1
schema:
  estado:
    type: string
`
	files := map[string]string{
		"doc1.md": "---\nestado: Pending\n---\n# Content\n",
	}
	dir := setupMigrateDir(t, stem, files)

	out, err := runCmd(t, "migrate", dir, "--rename", "nonexistent=something", "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No files affected") {
		t.Errorf("expected 'No files affected' message, got: %s", out)
	}
}

func TestMigrateRenameDryRunTable(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
`
	files := map[string]string{
		"doc1.md": "---\ntitulo: Hello\n---\n# Content\n",
	}
	dir := setupMigrateDir(t, stem, files)

	out, err := runCmd(t, "migrate", dir, "--rename", "titulo=title", "--dry-run", "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "would") {
		t.Errorf("expected 'would' prefix for dry-run, got: %s", out)
	}

	// Verify file was NOT modified.
	content, _ := os.ReadFile(filepath.Join(dir, "doc1.md"))
	if !strings.Contains(string(content), "titulo:") {
		t.Error("dry-run should not modify files")
	}
}

func TestMigrateRenameInvalidFormat(t *testing.T) {
	dir := setupMigrateDir(t, "", nil)

	_, err := runCmd(t, "migrate", dir, "--rename", "badformat")
	if err == nil {
		t.Fatal("expected error for invalid rename format")
	}
}

func TestMigrateRenameWithStemUpdate(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
  estado:
    type: string
`
	files := map[string]string{
		"doc1.md": "---\ntitulo: Test\nestado: Pending\n---\n",
	}
	dir := setupMigrateDir(t, stem, files)

	out, err := runCmd(t, "migrate", dir, "--rename", "titulo=title", "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, ".stem") {
		t.Errorf("expected .stem in output, got: %s", out)
	}
}

// --- migrate diff tests ---

func TestMigrateDiffNoStemFiles(t *testing.T) {
	dir := t.TempDir()

	_, err := runCmd(t, "migrate", dir)
	if err == nil {
		t.Fatal("expected error for no .stem files")
	}
}

func TestMigrateDiffSingleStemWithFrom(t *testing.T) {
	stem := `version: 1
schema:
  estado:
    type: enum
    values: [Pending, Done]
    required: true
`
	dir := setupMigrateDir(t, stem, nil)

	// Create a "previous" version of the .stem.
	prevStem := `version: 1
schema:
  estado:
    type: enum
    values: [Pending, Done, Cancelled]
    required: true
`
	prevPath := filepath.Join(dir, "prev.stem")
	mustWriteFile(t, prevPath, []byte(prevStem), 0644)

	out, err := runCmd(t, "migrate", filepath.Join(dir, ".stem"), "--from", prevPath)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if result["kind"] != "rootline/migrate-diff" {
		t.Errorf("kind = %v, want rootline/migrate-diff", result["kind"])
	}
}

func TestMigrateDiffTableOutput(t *testing.T) {
	stem := `version: 1
schema:
  estado:
    type: enum
    values: [Pending, Done]
    required: true
`
	dir := setupMigrateDir(t, stem, nil)

	prevStem := `version: 1
schema:
  estado:
    type: enum
    values: [Pending, Done, Cancelled]
    required: true
`
	prevPath := filepath.Join(dir, "prev.stem")
	mustWriteFile(t, prevPath, []byte(prevStem), 0644)

	out, err := runCmd(t, "migrate", filepath.Join(dir, ".stem"), "--from", prevPath, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Schema changes") {
		t.Errorf("expected 'Schema changes' in table output, got: %s", out)
	}
	if !strings.Contains(out, "YES") {
		t.Errorf("expected 'YES' for breaking changes, got: %s", out)
	}
}

func TestMigrateDiffNoChanges(t *testing.T) {
	stem := `version: 1
schema:
  estado:
    type: string
`
	dir := setupMigrateDir(t, stem, nil)

	// Same content for "from" file.
	prevPath := filepath.Join(dir, "prev.stem")
	mustWriteFile(t, prevPath, []byte(stem), 0644)

	out, err := runCmd(t, "migrate", filepath.Join(dir, ".stem"), "--from", prevPath, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "no changes") {
		t.Errorf("expected 'no changes' message, got: %s", out)
	}
}

func TestMigrateDiffFieldExtraction(t *testing.T) {
	stem := `version: 1
schema:
  estado:
    type: enum
    values: [Pending, Done]
`
	dir := setupMigrateDir(t, stem, nil)

	prevStem := `version: 1
schema:
  estado:
    type: enum
    values: [Pending, Done, Cancelled]
`
	prevPath := filepath.Join(dir, "prev.stem")
	mustWriteFile(t, prevPath, []byte(prevStem), 0644)

	out, err := runCmd(t, "migrate", filepath.Join(dir, ".stem"), "--from", prevPath, "--field", "total_count")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(strings.TrimSpace(out), "1") {
		t.Errorf("expected total_count=1, got: %s", out)
	}
}

func TestMigrateDiffFromOnlyWithSingleStem(t *testing.T) {
	// --from can only be used with a single .stem file target.
	stem := `version: 1
schema:
  estado:
    type: string
`
	dir := setupMigrateDir(t, stem, nil)

	// Create another .stem in a subdir.
	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(subDir, ".stem"), []byte(stem), 0644)

	prevPath := filepath.Join(dir, "prev.stem")
	mustWriteFile(t, prevPath, []byte(stem), 0644)

	_, err := runCmd(t, "migrate", dir, "--from", prevPath)
	if err == nil {
		t.Fatal("expected error for --from with multiple stems")
	}
}

// --- migrate batch diff tests ---

func TestMigrateBatchDiffJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	stem1 := `version: 1
schema:
  estado:
    type: string
`
	stem2 := `version: 1
schema:
  tipo:
    type: string
`

	// Create two subdirectories, each with its own .stem.
	sub1 := filepath.Join(dir, "sub1")
	sub2 := filepath.Join(dir, "sub2")
	if err := os.Mkdir(sub1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(sub2, 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(sub1, ".stem"), []byte(stem1), 0644)
	mustWriteFile(t, filepath.Join(sub2, ".stem"), []byte(stem2), 0644)

	// Without --from, it uses git which may fail — that's OK,
	// the code handles the error path and produces a diff with nil "before".
	out, err := runCmd(t, "migrate", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var batch MigrateBatchResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	if batch.Version != 1 {
		t.Errorf("version = %d, want 1", batch.Version)
	}
	if batch.Kind != "rootline/migrate-batch" {
		t.Errorf("kind = %s, want rootline/migrate-batch", batch.Kind)
	}
	if batch.Summary.StemsChecked != 2 {
		t.Errorf("stems_checked = %d, want 2", batch.Summary.StemsChecked)
	}
}

func TestMigrateBatchDiffTable(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	stem1 := `version: 1
schema:
  estado:
    type: string
`
	sub1 := filepath.Join(dir, "sub1")
	sub2 := filepath.Join(dir, "sub2")
	if err := os.Mkdir(sub1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(sub2, 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(sub1, ".stem"), []byte(stem1), 0644)
	mustWriteFile(t, filepath.Join(sub2, ".stem"), []byte(stem1), 0644)

	out, err := runCmd(t, "migrate", dir, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	// Should show either changes or "no changes" or summary.
	if !strings.Contains(out, "Summary") && !strings.Contains(out, "no changes") {
		t.Errorf("expected Summary or 'no changes' in batch table, got: %s", out)
	}
}

func TestMigrateBatchDiffFieldExtraction(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	stem := `version: 1
schema:
  estado:
    type: string
`
	sub1 := filepath.Join(dir, "sub1")
	sub2 := filepath.Join(dir, "sub2")
	if err := os.Mkdir(sub1, 0755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(sub2, 0755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(sub1, ".stem"), []byte(stem), 0644)
	mustWriteFile(t, filepath.Join(sub2, ".stem"), []byte(stem), 0644)

	out, err := runCmd(t, "migrate", dir, "--field", "summary.stems_checked")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(strings.TrimSpace(out), "2") {
		t.Errorf("expected stems_checked=2, got: %s", out)
	}
}

// --- migrate --split tests ---

func setupSplitDir(t *testing.T) string {
	t.Helper()
	stem := `version: 1
scope:
  match: "*.md"
schema:
  id:
    type: sequence
    prefix: E
    digits: 2
  estado:
    type: enum
    values: [Pending, Completed, In Progress]
    required: true
  tipo:
    type: enum
    values: [feature, historia]
    severity: warn
structural:
  subdirs:
    require_index: README.md
    min_children: 2
    severity: warn
links:
  allowed: [blocks, reference]
  blocks:
    target: "^T\\d{3}-"
    field: blocked_by
derive:
  estado: "hold != nil ? \"On Hold\" : estado"
aggregate:
  estado: "all(descendants, {.estado == \"Completed\"}) ? \"Completed\" : \"Pending\""
`
	files := map[string]string{
		"E01-infra/README.md":             "---\nestado: Pending\n---\n# E01\n",
		"E02-platform/README.md":          "---\nestado: Completed\n---\n# E02\n",
		"E01-infra/F01-net/README.md":     "---\nestado: Pending\ntipo: feature\n---\n# F01\n",
		"E01-infra/F02-store/README.md":   "---\nestado: Completed\ntipo: historia\n---\n# F02\n",
		"E02-platform/F01-auth/README.md": "---\nestado: Pending\ntipo: feature\n---\n# F01\n",
		"E02-platform/F02-api/README.md":  "---\nestado: Completed\ntipo: feature\n---\n# F02\n",
	}
	return setupMigrateDir(t, stem, files)
}

func TestMigrateSplit(t *testing.T) {
	dir := setupSplitDir(t)

	out, err := runCmd(t, "migrate", dir, "--split")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Split .stem into") {
		t.Errorf("expected split message, got: %s", out)
	}

	// Root .stem should still exist with common fields.
	rootContent, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	rootStr := string(rootContent)
	if !strings.Contains(rootStr, "prefix: E") {
		t.Errorf("expected prefix: E in root .stem, got: %s", rootStr)
	}
	if !strings.Contains(rootStr, "estado") {
		t.Errorf("expected estado in root .stem, got: %s", rootStr)
	}

	// Child .stem should exist.
	childContent, err := os.ReadFile(filepath.Join(dir, "E01-infra", ".stem"))
	if err != nil {
		t.Fatalf("expected child .stem: %v", err)
	}
	childStr := string(childContent)
	if !strings.Contains(childStr, "prefix: F") {
		t.Errorf("expected prefix: F in child .stem, got: %s", childStr)
	}
}

func TestMigrateSplitDryRun(t *testing.T) {
	dir := setupSplitDir(t)

	out, err := runCmd(t, "migrate", dir, "--split", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "# --- ") {
		t.Errorf("expected file separators, got: %s", out)
	}
	if !strings.Contains(out, "prefix: E") {
		t.Errorf("expected prefix: E in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "prefix: F") {
		t.Errorf("expected prefix: F in dry-run output, got: %s", out)
	}

	// No child .stem files should be written.
	if _, err := os.Stat(filepath.Join(dir, "E01-infra", ".stem")); err == nil {
		t.Error("expected no child .stem in dry-run mode")
	}
}

func TestMigrateSplitNoHierarchy(t *testing.T) {
	stem := `version: 1
schema:
  estado:
    type: string
`
	files := map[string]string{
		"a.md": "---\nestado: draft\n---\n# A\n",
		"b.md": "---\nestado: done\n---\n# B\n",
	}
	dir := setupMigrateDir(t, stem, files)

	_, err := runCmd(t, "migrate", dir, "--split")
	if err == nil {
		t.Fatal("expected error for non-hierarchical directory")
	}
	if !strings.Contains(err.Error(), "no hierarchy") {
		t.Errorf("expected 'no hierarchy' error, got: %v", err)
	}
}

func TestMigrateSplitPreservesCustom(t *testing.T) {
	dir := setupSplitDir(t)

	out, err := runCmd(t, "migrate", dir, "--split")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Verify derive/aggregate/links/structural preserved in root .stem.
	rootContent, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	rootStr := string(rootContent)

	for _, section := range []string{"derive:", "aggregate:", "links:", "structural:"} {
		if !strings.Contains(rootStr, section) {
			t.Errorf("expected %q preserved in root .stem, got: %s", section, rootStr)
		}
	}
}
