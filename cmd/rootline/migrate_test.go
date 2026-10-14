package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func runCmdWithSeparateOutput(t *testing.T, args ...string) (string, string, error) {
	t.Helper()
	resetFlags()
	stdout := new(bytes.Buffer)
	stderr := new(bytes.Buffer)
	rootCmd.SetOut(stdout)
	rootCmd.SetErr(stderr)
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return stdout.String(), stderr.String(), err
}

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
		declareTestBoundary(t, dir)
	}

	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		mustWriteFile(t, path, []byte(content), 0644)
	}

	declareTestBoundary(t, dir)
	return dir
}

// --- migrate --rename tests ---

func TestMigrateRenameJSON(t *testing.T) {
	stem := `version: 2
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
	stem := `version: 2
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
	stem := `version: 2
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
	stem := `version: 2
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
	stem := `version: 2
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

func TestMigrateRenameCollisionNoLog(t *testing.T) {
	const stem = "version: 2\nschema:\n  titulo:\n    type: string\n"
	tests := []struct {
		name          string
		stem          string
		files         map[string]string
		collidingPath string
	}{
		{
			name: "later document",
			stem: stem,
			files: map[string]string{
				"a.md": "---\ntitulo: Rename me\n---\n# First\n",
				"z.md": "---\ntitulo: Keep me\ntitle: Existing\n---\n# Last\n",
			},
			collidingPath: "z.md",
		},
		{
			name: "stem",
			stem: stem + "  title:\n    type: string\n",
			files: map[string]string{
				"a.md": "---\ntitulo: Rename me\n---\n# First\n",
			},
			collidingPath: ".stem",
		},
	}
	for _, tt := range tests {
		for _, existingLog := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/existing-log=%v", tt.name, existingLog), func(t *testing.T) {
				dir := setupMigrateDir(t, tt.stem, tt.files)
				before := make(map[string]string)
				for path := range tt.files {
					content, err := os.ReadFile(filepath.Join(dir, path))
					if err != nil {
						t.Fatal(err)
					}
					before[path] = string(content)
				}
				stemContent, err := os.ReadFile(filepath.Join(dir, ".stem"))
				if err != nil {
					t.Fatal(err)
				}
				before[".stem"] = string(stemContent)
				logPath := filepath.Join(dir, ".rootline-migrations")
				const logContent = "{\"operation\":\"rename\",\"old_field\":\"legacy\",\"new_field\":\"titulo\"}\n"
				if existingLog {
					mustWriteFile(t, logPath, []byte(logContent), 0644)
				}
				var dryRunError string
				for _, dryRun := range []bool{true, false} {
					args := []string{"migrate", dir, "--rename", "titulo=title"}
					if dryRun {
						args = append(args, "--dry-run")
					}
					out, _, err := runCmdWithSeparateOutput(t, args...)
					if err == nil {
						t.Fatalf("dry-run=%v: expected collision refusal", dryRun)
					}
					if !strings.Contains(err.Error(), tt.collidingPath) || !strings.Contains(err.Error(), `"title"`) {
						t.Errorf("dry-run=%v: missing colliding path or destination key: %v", dryRun, err)
					}
					if dryRun {
						dryRunError = err.Error()
					} else if err.Error() != dryRunError {
						t.Errorf("real refusal %q differs from dry-run %q", err.Error(), dryRunError)
					}
					if out != "" {
						t.Errorf("dry-run=%v: refused rename emitted stdout: %s", dryRun, out)
					}
					for path, want := range before {
						content, readErr := os.ReadFile(filepath.Join(dir, path))
						if readErr != nil {
							t.Fatal(readErr)
						}
						if string(content) != want {
							t.Errorf("dry-run=%v: %s changed on refusal", dryRun, path)
						}
					}
					content, readErr := os.ReadFile(logPath)
					if existingLog {
						if readErr != nil {
							t.Fatal(readErr)
						}
						if string(content) != logContent {
							t.Errorf("dry-run=%v: migration log changed on refusal", dryRun)
						}
					} else if !os.IsNotExist(readErr) {
						t.Errorf("dry-run=%v: expected no migration log, got error %v", dryRun, readErr)
					}
				}
			})
		}
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
	stem := `version: 2
schema:
  estado:
    type: enum
    values: [Pending, Done]
    required: true
`
	dir := setupMigrateDir(t, stem, nil)

	// Create a "previous" version of the .stem.
	prevStem := `version: 2
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

func TestMigrateDiffAffectedFilesScanFailure(t *testing.T) {
	const current = "version: 2\nroot: true\nschema: {}\n"
	dir := setupMigrateDir(t, current, map[string]string{
		"doc.md": "---\nremoved: Present\n---\n",
	})
	beforePath := filepath.Join(dir, "before.stem")
	mustWriteFile(t, beforePath, []byte("version: 2\nschema:\n  removed:\n    type: string\n"), 0644)
	docPath := filepath.Join(dir, "doc.md")
	if err := os.Chmod(docPath, 0); err != nil {
		t.Skipf("permission restrictions unavailable: %v", err)
	}
	defer func() {
		if err := os.Chmod(docPath, 0600); err != nil {
			t.Error(err)
		}
	}()
	if _, err := os.ReadFile(docPath); !errors.Is(err, os.ErrPermission) {
		t.Skip("platform or privileges do not enforce unreadable file permissions")
	}
	for _, format := range []string{"json", "table"} {
		for _, dryRun := range []bool{false, true} {
			args := []string{"migrate", dir, "--from", beforePath, "-o", format}
			if dryRun {
				args = append(args, "--dry-run")
			}
			stdout, _, err := runCmdWithSeparateOutput(t, args...)
			if err == nil || !strings.Contains(err.Error(), "scanning records for "+filepath.Join(dir, ".stem")) {
				t.Fatalf("expected record scan error, got %v, stdout %q", err, stdout)
			}
			if stdout != "" {
				t.Errorf("scan failure emitted fabricated counts or partial stdout: %s", stdout)
			}
		}
	}
}

func TestMigrateDiffBatchBaselineFailurePrecedesScan(t *testing.T) {
	dir := t.TempDir()
	runFixtureGit(t, dir, "init", "--quiet")
	rootStem := filepath.Join(dir, ".stem")
	subDir := filepath.Join(dir, "sub")
	if err := os.Mkdir(subDir, 0755); err != nil {
		t.Fatal(err)
	}
	subStem := filepath.Join(subDir, ".stem")
	mustWriteFile(t, rootStem, []byte("version: 2\nroot: true\n"), 0644)
	mustWriteFile(t, subStem, []byte("version: 1\n"), 0644)
	runFixtureGit(t, dir, "add", ".stem", "sub/.stem")
	runFixtureGit(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--quiet", "-m", "baseline")
	mustWriteFile(t, rootStem, []byte("version: 2\nroot: true\nschema:\n  added:\n    type: string\n"), 0644)
	mustWriteFile(t, subStem, []byte("version: 2\n"), 0644)
	docPath := filepath.Join(dir, "doc.md")
	mustWriteFile(t, docPath, []byte("---\ninvalid: [\n---\n"), 0644)
	for _, format := range []string{"json", "table"} {
		assertMigrateBaselineRejected(t, []string{"migrate", dir, "-o", format},
			"parsing git HEAD:sub/.stem baseline", rootStem, subStem, docPath)
	}
}

func TestMigrateDiffAffectedFiles(t *testing.T) {
	const beforeStem = `version: 2
root: true
scope:
  match: "T*.md"
schema:
  estado:
    type: enum
    values: [Pending, Done, Cancelled]
  owner:
    type: string
  tipo:
    type: string
  unused:
    type: string
`
	const afterStem = `version: 2
root: true
scope:
  match: "T*.md"
schema:
  a_note:
    type: string
  estado:
    type: enum
    values: [Pending, Done]
  owner:
    type: string
    required: true
  tipo:
    type: integer
`
	dir := setupMigrateDir(t, afterStem, map[string]string{
		"T001.md":             "---\nestado: Cancelled\ntipo: task\n---\n",
		"T002.md":             "---\nestado: Done\ntipo: task\nowner: Ana\n---\n",
		"sub/.stem":           "version: 2\n",
		"sub/T003.md":         "---\nestado: Cancelled\nowner: Ana\n---\n",
		"T004-ignored.md":     "---\nestado: Cancelled\ntipo: task\n---\n",
		".stemignore":         "T004-ignored.md\n",
		"outside.md":          "---\nestado: Cancelled\ntipo: task\n---\n",
		"independent/.stem":   "version: 2\nroot: true\n",
		"independent/T005.md": "---\nestado: Cancelled\ntipo: task\n---\n",
		"narrow/.stem":        "version: 2\nscope:\n  match: \"README.md\"\n",
		"narrow/T006.md":      "---\nestado: Cancelled\ntipo: task\n---\n",
		"narrow/README.md":    "---\nestado: Done\nowner: Ana\n---\n",
	})
	beforePath := filepath.Join(dir, "before.stem")
	mustWriteFile(t, beforePath, []byte(beforeStem), 0644)
	stemPath := filepath.Join(dir, ".stem")
	wantChanges := []struct {
		kind     string
		field    string
		affected int
	}{
		{"enum_value_removed", "estado", 2},
		{"required_added", "owner", 1},
		{"type_changed", "tipo", 2},
		{"field_removed", "unused", 0},
		{"field_added", "a_note", 0},
	}
	for _, dryRun := range []bool{false, true} {
		t.Run(fmt.Sprintf("dry-run=%v", dryRun), func(t *testing.T) {
			args := []string{"migrate", stemPath, "--from", beforePath, "-o", "json"}
			if dryRun {
				args = append(args, "--dry-run")
			}
			var first string
			for i := range 3 {
				stdout, stderr, err := runCmdWithSeparateOutput(t, args...)
				if err != nil {
					t.Fatalf("migrate failed: %v, stderr %q", err, stderr)
				}
				if i == 0 {
					first = stdout
				} else if stdout != first {
					t.Fatalf("JSON order changed on run %d:\n%s\nwant:\n%s", i, stdout, first)
				}
				var result struct {
					Version       int    `json:"version"`
					Kind          string `json:"kind"`
					StemPath      string `json:"stem_path"`
					BreakingCount int    `json:"breaking_count"`
					TotalCount    int    `json:"total_count"`
					Changes       []struct {
						Kind          string `json:"kind"`
						Field         string `json:"field"`
						AffectedFiles int    `json:"affected_files"`
					} `json:"changes"`
				}
				if err := json.Unmarshal([]byte(stdout), &result); err != nil {
					t.Fatal(err)
				}
				if result.Version != 1 || result.Kind != "rootline/migrate-diff" || result.StemPath != stemPath ||
					result.BreakingCount != 4 || result.TotalCount != len(wantChanges) || len(result.Changes) != len(wantChanges) {
					t.Fatalf("unexpected diff envelope: %s", stdout)
				}
				for j, want := range wantChanges {
					got := result.Changes[j]
					if got.Kind != want.kind || got.Field != want.field || got.AffectedFiles != want.affected {
						t.Errorf("change %d = %+v, want %s/%s affected_files=%d", j, got, want.kind, want.field, want.affected)
					}
				}
			}
		})
	}

	var firstTable string
	for i := range 3 {
		out, err := runCmd(t, "migrate", stemPath, "--from", beforePath, "-o", "table")
		if err != nil {
			t.Fatalf("table failed: %v\n%s", err, out)
		}
		if i == 0 {
			firstTable = out
		} else if out != firstTable {
			t.Fatalf("table changed on run %d:\n%s\nwant:\n%s", i, out, firstTable)
		}
		var lines []string
		for _, line := range strings.Split(out, "\n") {
			if fields := strings.Fields(line); len(fields) > 0 {
				lines = append(lines, strings.Join(fields, " "))
			}
		}
		want := strings.Join([]string{
			"Schema changes in " + stemPath + ":",
			"Kind Field Breaking Message",
			"- - - -",
			"enum_value_removed estado YES enum value Cancelled removed from estado",
			"required_added owner YES field owner is now required",
			"type_changed tipo YES field tipo type changed from string to integer",
			"field_removed unused YES field unused removed",
			"field_added a_note no field a_note added (type: string)",
			"Total: 5 changes (4 breaking)",
		}, "\n")
		if got := strings.Join(lines, "\n"); got != want {
			t.Fatalf("table text or ordering changed:\n%s\nwant:\n%s", got, want)
		}
	}
}

func TestMigrateDiffAffectedFilesBatch(t *testing.T) {
	dir := t.TempDir()
	runFixtureGit(t, dir, "init", "--quiet")
	before := map[string]string{
		".stem":   "version: 2\nroot: true\nschema:\n  parent:\n    type: string\n",
		"a/.stem": "version: 2\nschema:\n  local:\n    type: string\n",
		"z/.stem": "version: 2\nroot: true\nschema:\n  local:\n    type: string\n",
	}
	for name, content := range before {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		mustWriteFile(t, path, []byte(content), 0644)
	}
	runFixtureGit(t, dir, "add", ".stem", "a/.stem", "z/.stem")
	runFixtureGit(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--quiet", "-m", "baseline")
	for name, content := range map[string]string{
		".stem":   "version: 2\nroot: true\nschema:\n  added:\n    type: string\n",
		"a/.stem": "version: 2\nschema:\n  added:\n    type: string\n",
		"z/.stem": "version: 2\nroot: true\nschema:\n  added:\n    type: string\n",
		"doc.md":  "---\nparent: Root\nlocal: Outside child scope\n---\n",
		"a/1.md":  "---\nparent: Inherited\nlocal: First\n---\n",
		"a/2.md":  "---\nlocal: Second\n---\n",
		"z/1.md":  "---\nparent: Independent\nlocal: Separate\n---\n",
	} {
		mustWriteFile(t, filepath.Join(dir, name), []byte(content), 0644)
	}
	var first string
	for i := range 3 {
		out, err := runCmd(t, "migrate", dir)
		if err != nil {
			t.Fatalf("batch failed: %v\n%s", err, out)
		}
		if i == 0 {
			first = out
		} else if out != first {
			t.Fatalf("batch order changed on run %d:\n%s\nwant:\n%s", i, out, first)
		}
		var batch MigrateBatchResult
		if err := json.Unmarshal([]byte(out), &batch); err != nil {
			t.Fatal(err)
		}
		if batch.Version != 1 || batch.Kind != "rootline/migrate-batch" || len(batch.Results) != 3 ||
			batch.Summary != (MigrateBatchSummary{StemsChecked: 3, TotalChanges: 6, BreakingCount: 3}) {
			t.Fatalf("unexpected batch envelope: %s", out)
		}
		for j, want := range []struct {
			path     string
			field    string
			affected int
		}{
			{".stem", "parent", 2},
			{"a/.stem", "local", 2},
			{"z/.stem", "local", 1},
		} {
			result := batch.Results[j]
			if result.StemPath != filepath.Join(dir, want.path) || len(result.Changes) != 2 {
				t.Fatalf("unexpected stem ordering or changes: %+v", result)
			}
			removed, added := result.Changes[0], result.Changes[1]
			if string(removed.Kind) != "field_removed" || removed.Field != want.field || removed.AffectedFiles != want.affected ||
				string(added.Kind) != "field_added" || added.AffectedFiles != 0 {
				t.Errorf("%s: unexpected affected counts/order: %+v", want.path, result.Changes)
			}
		}
	}
}

func TestMigrateDiffAlternateSchemaFilename(t *testing.T) {
	dir := t.TempDir()
	current := filepath.Join(dir, "current.yaml")
	before := filepath.Join(dir, "before.yaml")
	mustWriteFile(t, current, []byte("version: 2\nschema: {}\n"), 0644)
	mustWriteFile(t, before, []byte("version: 2\nschema:\n  removed:\n    type: string\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "doc.md"), []byte("---\nremoved: Present\n---\n"), 0644)
	out, err := runCmd(t, "migrate", current, "--from", before)
	if err != nil {
		t.Fatalf("alternate schema filename was rejected: %v\n%s", err, out)
	}
	var result struct {
		Changes []struct {
			Kind          string `json:"kind"`
			AffectedFiles int    `json:"affected_files"`
		} `json:"changes"`
	}
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if len(result.Changes) != 1 || result.Changes[0].Kind != "field_removed" || result.Changes[0].AffectedFiles != 0 {
		t.Fatalf("non-governing schema input changed behavior: %s", out)
	}
}

func TestMigrateDiffRejectsExplicitBaseline(t *testing.T) {
	const current = "version: 2\nschema:\n  title:\n    type: string\n"
	for _, tc := range []struct {
		name    string
		content string
	}{
		{"missing", ""},
		{"unreadable-directory", ""},
		{"permission-denied", "version: 2\n"},
		{"malformed", "version: 2\nschema: [\n"},
		{"unsupported-version", "version: 1\nschema: {}\n"},
		{"refused-null-field", "version: 2\nschema:\n  title: null\n"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			stemPath := filepath.Join(dir, ".stem")
			docPath := filepath.Join(dir, "doc.md")
			fromPath := filepath.Join(dir, "previous.stem")
			mustWriteFile(t, stemPath, []byte(current), 0644)
			mustWriteFile(t, docPath, []byte("---\ntitle: Original\n---\n# Body\n"), 0644)
			if tc.name == "unreadable-directory" {
				if err := os.Mkdir(fromPath, 0755); err != nil {
					t.Fatal(err)
				}
			} else if tc.content != "" {
				mustWriteFile(t, fromPath, []byte(tc.content), 0644)
			}
			if tc.name == "permission-denied" {
				if err := os.Chmod(fromPath, 0); err != nil {
					t.Skipf("permission restrictions unavailable: %v", err)
				}
				t.Cleanup(func() {
					if err := os.Chmod(fromPath, 0600); err != nil {
						t.Error(err)
					}
				})
				if _, err := os.ReadFile(fromPath); !errors.Is(err, os.ErrPermission) {
					t.Skip("platform or privileges do not enforce unreadable file permissions")
				}
			}
			for _, target := range []string{stemPath, dir} {
				for _, format := range []string{"json", "table"} {
					for _, dryRun := range []bool{false, true} {
						args := []string{"migrate", target, "--from", fromPath, "-o", format}
						if dryRun {
							args = append(args, "--dry-run")
						}
						assertMigrateBaselineRejected(t, args, "--from baseline "+fromPath, stemPath, docPath)
					}
				}
			}
			if tc.content != "" && tc.name != "permission-denied" {
				content, err := os.ReadFile(fromPath)
				if err != nil || string(content) != tc.content {
					t.Fatalf("baseline changed: content %q, error %v", content, err)
				}
			}
		})
	}
}

func assertMigrateBaselineRejected(t *testing.T, args []string, context string, paths ...string) {
	t.Helper()
	before := make(map[string][]byte, len(paths))
	for _, path := range paths {
		content, err := os.ReadFile(path)
		if err != nil {
			t.Fatal(err)
		}
		before[path] = content
	}
	stdout, stderr, err := runCmdWithSeparateOutput(t, args...)
	if err == nil {
		t.Fatalf("%v: expected baseline error, stdout %q", args, stdout)
	}
	if stdout != "" {
		t.Errorf("%v: fabricated or partial stdout %q", args, stdout)
	}
	if !strings.Contains(err.Error(), context) || !strings.Contains(stderr, context) {
		t.Errorf("%v: missing baseline context %q: error %v, stderr %q", args, context, err, stderr)
	}
	for path, content := range before {
		after, readErr := os.ReadFile(path)
		if readErr != nil || !bytes.Equal(after, content) {
			t.Errorf("%v: file %s changed after refusal: %v", args, path, readErr)
		}
	}
}

func TestMigrateDiffRejectsGitBaseline(t *testing.T) {
	const current = "version: 2\nschema:\n  title:\n    type: string\n"
	for _, baseline := range []string{"version: 2\nschema: [\n", "version: 1\nschema: {}\n"} {
		t.Run(baseline, func(t *testing.T) {
			dir := t.TempDir()
			runFixtureGit(t, dir, "init", "--quiet")
			rootStem := filepath.Join(dir, ".stem")
			subDir := filepath.Join(dir, "sub")
			if err := os.Mkdir(subDir, 0755); err != nil {
				t.Fatal(err)
			}
			stemPath := filepath.Join(subDir, ".stem")
			docPath := filepath.Join(subDir, "doc.md")
			mustWriteFile(t, rootStem, []byte("version: 2\n"), 0644)
			mustWriteFile(t, stemPath, []byte(baseline), 0644)
			runFixtureGit(t, dir, "add", ".stem", "sub/.stem")
			runFixtureGit(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--quiet", "-m", "baseline")
			mustWriteFile(t, stemPath, []byte(current), 0644)
			mustWriteFile(t, docPath, []byte("---\ntitle: Original\n---\n# Body\n"), 0644)
			for _, target := range []string{stemPath, dir} {
				for _, format := range []string{"json", "table"} {
					for _, dryRun := range []bool{false, true} {
						args := []string{"migrate", target, "-o", format}
						if dryRun {
							args = append(args, "--dry-run")
						}
						assertMigrateBaselineRejected(t, args, "parsing git HEAD:sub/.stem baseline", rootStem, stemPath, docPath)
					}
				}
			}
		})
	}
}

func TestMigrateDiffImplicitBaselineAbsent(t *testing.T) {
	const current = "version: 2\nschema:\n  title:\n    type: string\n"
	for _, scenario := range []string{"no-repository", "unborn-head", "missing-path"} {
		t.Run(scenario, func(t *testing.T) {
			dir := t.TempDir()
			if scenario != "no-repository" {
				runFixtureGit(t, dir, "init", "--quiet")
			}
			if scenario == "missing-path" {
				mustWriteFile(t, filepath.Join(dir, "tracked.txt"), []byte("baseline\n"), 0644)
				runFixtureGit(t, dir, "add", "tracked.txt")
				runFixtureGit(t, dir, "-c", "user.name=Test", "-c", "user.email=test@example.com", "commit", "--quiet", "-m", "baseline")
			}
			stemPath := filepath.Join(dir, ".stem")
			mustWriteFile(t, stemPath, []byte(current), 0644)
			stdout, stderr, err := runCmdWithSeparateOutput(t, "migrate", stemPath)
			if err != nil {
				t.Fatalf("absent implicit baseline failed: %v, stderr %q", err, stderr)
			}
			var result struct {
				Kind       string `json:"kind"`
				TotalCount int    `json:"total_count"`
				Changes    []struct {
					Kind  string `json:"kind"`
					Field string `json:"field"`
				} `json:"changes"`
			}
			if err := json.Unmarshal([]byte(stdout), &result); err != nil {
				t.Fatal(err)
			}
			if result.Kind != "rootline/migrate-diff" || result.TotalCount != 1 || len(result.Changes) != 1 ||
				result.Changes[0].Kind != "field_added" || result.Changes[0].Field != "title" {
				t.Fatalf("absent-baseline behavior changed: %s", stdout)
			}
			after, readErr := os.ReadFile(stemPath)
			if readErr != nil || string(after) != current {
				t.Fatalf("diff changed current stem: %q, error %v", after, readErr)
			}
		})
	}
}

func TestMigrateDiffGitFailureIsNotAbsence(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte("version: 2\nschema:\n  title:\n    type: string\n"), 0644)
	assertMigrateBaselineRejected(t, []string{"migrate", stemPath}, "git show HEAD:.stem", stemPath)
}

func TestMigrateDiffTableOutput(t *testing.T) {
	stem := `version: 2
schema:
  estado:
    type: enum
    values: [Pending, Done]
    required: true
`
	dir := setupMigrateDir(t, stem, nil)

	prevStem := `version: 2
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

func TestMigrateDiffAttributeSurfaces(t *testing.T) {
	const beforeStem = `version: 2
schema:
  id:
    type: sequence
    prefix: T
    digits: 3
    match: [T*]
    required: {match: [T*]}
    excludes: {match: README.md}
validate:
  - rule: requires
    if: {status: draft}
    then: {fields: [owner]}
    severity: error
`
	const afterStem = `version: 2
schema:
  id:
    type: sequence
    prefix: F
    digits: 2
    match: [F*]
    required: {match: [F*]}
    excludes: {match: index.md}
validate:
  - rule: requires
    if: {status: published}
    then: {fields: [reviewer]}
    severity: error
`
	dir := setupMigrateDir(t, afterStem, nil)
	beforePath := filepath.Join(dir, "before.stem")
	mustWriteFile(t, beforePath, []byte(beforeStem), 0644)
	stemPath := filepath.Join(dir, ".stem")
	snapshots := []struct {
		kind   string
		before string
		after  string
	}{
		{"prefix_changed", `{"prefix":"T"}`, `{"prefix":"F"}`},
		{"digits_changed", `{"digits":3}`, `{"digits":2}`},
		{"match_changed", `{"match":{"patterns":["T*"]}}`, `{"match":{"patterns":["F*"]}}`},
		{"required_match_changed", `{"required_match":{"patterns":["T*"]}}`, `{"required_match":{"patterns":["F*"]}}`},
		{"excludes_changed", `{"excludes":{"match":"README.md"}}`, `{"excludes":{"match":"index.md"}}`},
		{"rule_removed", `{"field":"","if":{"status":"draft"},"rule":"requires","severity":"error","then":{"fields":["owner"]}}`, ""},
		{"rule_added", "", `{"field":"","if":{"status":"published"},"rule":"requires","severity":"error","then":{"fields":["reviewer"]}}`},
	}
	for _, format := range []string{"json", "table"} {
		t.Run(format, func(t *testing.T) {
			var first string
			for i := range 5 {
				out, err := runCmd(t, "migrate", stemPath, "--from", beforePath, "-o", format)
				if err != nil {
					t.Fatalf("unexpected error: %v\noutput: %s", err, out)
				}
				if i == 0 {
					first = out
				} else if out != first {
					t.Fatalf("run %d changed output:\n%s\nwant:\n%s", i, out, first)
				}
				if format == "table" {
					for _, snapshot := range snapshots {
						for _, value := range []string{snapshot.kind, snapshot.before, snapshot.after} {
							if !strings.Contains(out, value) {
								t.Errorf("table omits %q:\n%s", value, out)
							}
						}
					}
					continue
				}
				var envelope map[string]json.RawMessage
				if err := json.Unmarshal([]byte(out), &envelope); err != nil {
					t.Fatal(err)
				}
				if len(envelope) != 6 || string(envelope["version"]) != "1" || string(envelope["kind"]) != `"rootline/migrate-diff"` {
					t.Fatalf("diff envelope changed: %s", out)
				}
				if string(envelope["total_count"]) != "7" || string(envelope["breaking_count"]) != "6" {
					t.Fatalf("attribute changes were lost: %s", out)
				}
				var changes []struct {
					Kind   string `json:"kind"`
					Before string `json:"before"`
					After  string `json:"after"`
				}
				if err := json.Unmarshal(envelope["changes"], &changes); err != nil {
					t.Fatal(err)
				}
				if len(changes) != len(snapshots) {
					t.Fatalf("changes = %d, want %d", len(changes), len(snapshots))
				}
				for _, snapshot := range snapshots {
					found := false
					for _, change := range changes {
						if change.Kind == snapshot.kind {
							found = true
							if change.Before != snapshot.before || change.After != snapshot.after {
								t.Errorf("%s snapshots = %q → %q, want %q → %q", change.Kind, change.Before, change.After, snapshot.before, snapshot.after)
							}
						}
					}
					if !found {
						t.Errorf("missing %s", snapshot.kind)
					}
				}
			}
		})
	}
}

func TestMigrateDiffNoChanges(t *testing.T) {
	stem := `version: 2
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
	stem := `version: 2
schema:
  estado:
    type: enum
    values: [Pending, Done]
`
	dir := setupMigrateDir(t, stem, nil)

	prevStem := `version: 2
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

func TestMigrateDiffRepeatableFieldExtraction(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte("version: 2\nroot: true\nschema: {}\n"), 0o644)

	for _, tc := range []struct {
		name   string
		fields []string
		want   string
	}{
		{"flag order", []string{"kind", "version"}, `["rootline/migrate-diff",1]`},
		{"reverse order", []string{"version", "kind"}, `[1,"rootline/migrate-diff"]`},
		{"repeated path", []string{"kind", "kind"}, `["rootline/migrate-diff","rootline/migrate-diff"]`},
		{"empty path ignored", []string{"", "kind"}, `"rootline/migrate-diff"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"migrate", stemPath, "--from", stemPath}
			for _, field := range tc.fields {
				args = append(args, "--field", field)
			}

			out, err := runCmd(t, args...)
			if err != nil {
				t.Fatalf("unexpected error: %v\noutput: %s", err, out)
			}
			if got := strings.TrimSpace(out); got != tc.want {
				t.Errorf("output = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestMigrateDiffRepeatableFieldFailureIsAtomic(t *testing.T) {
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte("version: 2\nroot: true\nschema: {}\n"), 0o644)

	for _, fields := range [][]string{
		{"kind", "nonexistent"},
		{"nonexistent", "kind"},
	} {
		args := []string{"migrate", stemPath, "--from", stemPath}
		for _, field := range fields {
			args = append(args, "--field", field)
		}

		stdout, stderr, err := runCmdWithSeparateOutput(t, args...)
		if err == nil {
			t.Fatalf("fields %v: expected an error, got stdout %q", fields, stdout)
		}
		if stdout != "" {
			t.Errorf("fields %v: stdout = %q, want no partial output", fields, stdout)
		}
		if !strings.Contains(err.Error(), "nonexistent") || !strings.Contains(stderr, "nonexistent") {
			t.Errorf("fields %v: error = %v, stderr = %q, want missing path diagnostic", fields, err, stderr)
		}
	}
}

func TestMigrateDiffFromOnlyWithSingleStem(t *testing.T) {
	// --from can only be used with a single .stem file target.
	stem := `version: 2
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
	runFixtureGit(t, dir, "init", "--quiet")

	stem1 := `version: 2
schema:
  estado:
    type: string
`
	stem2 := `version: 2
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

	// An unborn Git HEAD has no baseline, so current fields are additions.
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
	runFixtureGit(t, dir, "init", "--quiet")

	stem1 := `version: 2
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
	runFixtureGit(t, dir, "init", "--quiet")

	stem := `version: 2
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

func TestMigrateBatchDiffRepeatableFieldExtraction(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nschema: {}\n"), 0o644)
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(child, ".stem"), []byte("version: 2\nschema: {}\n"), 0o644)

	for _, tc := range []struct {
		name   string
		fields []string
		want   string
	}{
		{"flag order", []string{"kind", "version"}, `["rootline/migrate-batch",1]`},
		{"reverse order", []string{"version", "kind"}, `[1,"rootline/migrate-batch"]`},
		{"repeated path", []string{"kind", "kind"}, `["rootline/migrate-batch","rootline/migrate-batch"]`},
		{"empty path ignored", []string{"", "kind"}, `"rootline/migrate-batch"`},
	} {
		t.Run(tc.name, func(t *testing.T) {
			args := []string{"migrate", dir}
			for _, field := range tc.fields {
				args = append(args, "--field", field)
			}

			out, err := runCmd(t, args...)
			if err != nil {
				t.Fatalf("unexpected error: %v\noutput: %s", err, out)
			}
			if got := strings.TrimSpace(out); got != tc.want {
				t.Errorf("output = %s, want %s", got, tc.want)
			}
		})
	}
}

func TestMigrateBatchDiffRepeatableFieldFailureIsAtomic(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nschema: {}\n"), 0o644)
	child := filepath.Join(dir, "child")
	if err := os.Mkdir(child, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(child, ".stem"), []byte("version: 2\nschema: {}\n"), 0o644)

	for _, fields := range [][]string{
		{"kind", "nonexistent"},
		{"nonexistent", "kind"},
	} {
		args := []string{"migrate", dir}
		for _, field := range fields {
			args = append(args, "--field", field)
		}

		stdout, stderr, err := runCmdWithSeparateOutput(t, args...)
		if err == nil {
			t.Fatalf("fields %v: expected an error, got stdout %q", fields, stdout)
		}
		if stdout != "" {
			t.Errorf("fields %v: stdout = %q, want no partial output", fields, stdout)
		}
		if !strings.Contains(err.Error(), "nonexistent") || !strings.Contains(stderr, "nonexistent") {
			t.Errorf("fields %v: error = %v, stderr = %q, want missing path diagnostic", fields, err, stderr)
		}
	}
}

// --- migrate --split tests ---

func setupSplitDir(t *testing.T) string {
	t.Helper()
	stem := `version: 2
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
	stem := `version: 2
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

func TestMigrateSplitPreservesValidateRules(t *testing.T) {
	stem := `version: 2
scope:
  match: "*.md"
schema:
  id:
    type: sequence
    prefix: E
    digits: 2
  estado:
    type: enum
    values: [Pending, Completed]
    required: true
  tipo:
    type: enum
    values: [feature, historia]
  ejecutable_en:
    type: string
validate:
  - field: estado
    rule: non_empty
  - rule: requires
    if:
      tipo: ci-cd
    then:
      fields: [ejecutable_en]
`
	files := map[string]string{
		"E01-infra/README.md":           "---\nestado: Pending\n---\n# E01\n",
		"E02-platform/README.md":        "---\nestado: Completed\n---\n# E02\n",
		"E01-infra/F01-net/README.md":   "---\nestado: Pending\ntipo: feature\n---\n# F01\n",
		"E01-infra/F02-store/README.md": "---\nestado: Completed\ntipo: historia\n---\n# F02\n",
	}
	dir := setupMigrateDir(t, stem, files)

	out, err := runCmd(t, "migrate", dir, "--split")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	rootContent, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	rootStr := string(rootContent)

	if !strings.Contains(rootStr, "validate:") {
		t.Fatalf("expected validate section in root .stem, got: %s", rootStr)
	}
	if !strings.Contains(rootStr, "if:") {
		t.Errorf("expected 'if:' in validate rules, got: %s", rootStr)
	}
	if !strings.Contains(rootStr, "then:") {
		t.Errorf("expected 'then:' in validate rules, got: %s", rootStr)
	}
	if !strings.Contains(rootStr, "ejecutable_en") {
		t.Errorf("expected 'ejecutable_en' in validate then clause, got: %s", rootStr)
	}
}

func TestMigrateSplit_GeneratesAggregate(t *testing.T) {
	// .stem with enum estado but no aggregate section.
	stem := `version: 2
scope:
  match: "*.md"
schema:
  id:
    type: sequence
    prefix: E
    digits: 2
  estado:
    type: enum
    values: [Pending, In Progress, Blocked, Completed]
    required: true
`
	files := map[string]string{
		"E01-infra/README.md":           "---\nestado: Pending\n---\n# E01\n",
		"E02-platform/README.md":        "---\nestado: Completed\n---\n# E02\n",
		"E01-infra/F01-net/README.md":   "---\nestado: Pending\n---\n# F01\n",
		"E01-infra/F02-store/README.md": "---\nestado: In Progress\n---\n# F02\n",
	}
	dir := setupMigrateDir(t, stem, files)

	out, err := runCmd(t, "migrate", dir, "--split")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Should contain note about auto-generated aggregate.
	if !strings.Contains(out, "Note: auto-generated aggregate for 'estado'") {
		t.Errorf("expected aggregate note in output, got: %s", out)
	}

	rootContent, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	rootStr := string(rootContent)

	if !strings.Contains(rootStr, "aggregate:") {
		t.Errorf("expected aggregate section in root .stem, got:\n%s", rootStr)
	}
	// Field-agnostic: simple positional fallback (first value)
	if !strings.Contains(rootStr, `"Pending"`) {
		t.Errorf("expected first value as default, got:\n%s", rootStr)
	}
}

func TestMigrateSplit_PreservesExistingAggregate(t *testing.T) {
	// .stem with existing aggregate — should not be replaced.
	dir := setupSplitDir(t) // has aggregate: estado already

	out, err := runCmd(t, "migrate", dir, "--split")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Should NOT contain note about auto-generated aggregate (it already exists).
	if strings.Contains(out, "Note: auto-generated aggregate for 'estado'") {
		t.Errorf("expected no aggregate note when existing aggregate present, got: %s", out)
	}

	rootContent, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	rootStr := string(rootContent)

	// Should preserve the original aggregate expression.
	if !strings.Contains(rootStr, "aggregate:") {
		t.Errorf("expected aggregate section preserved, got:\n%s", rootStr)
	}
	if !strings.Contains(rootStr, `estado:`) || !strings.Contains(rootStr, `Completed`) {
		t.Errorf("expected original aggregate expression preserved, got:\n%s", rootStr)
	}
}

// sentinelStem is written to a child target before invoking --split so tests can
// prove byte-for-byte that no partial write happened.
const sentinelStem = "version: 2\nschema:\n  test_marker:\n    type: string\n"

func writeSentinelChildStem(t *testing.T, dir string) string {
	t.Helper()
	path := filepath.Join(dir, "E01-infra", ".stem")
	mustWriteFile(t, path, []byte(sentinelStem), 0644)
	return path
}

func TestRunMigrateSplit_PreFlightGuardCollisionsRefused(t *testing.T) {
	dir := setupSplitDir(t)
	sentinel := writeSentinelChildStem(t, dir)

	out, err := runCmd(t, "migrate", dir, "--split")
	if err == nil {
		t.Fatalf("expected collision error, got success\noutput: %s", out)
	}
	if !strings.Contains(err.Error(), "already exist") ||
		!strings.Contains(err.Error(), "use --force to overwrite") {
		t.Errorf("expected collision message, got: %v", err)
	}
	if !strings.Contains(err.Error(), filepath.Join("E01-infra", ".stem")) {
		t.Errorf("expected colliding child path in message, got: %v", err)
	}

	// Zero partial writes: the sentinel must be byte-identical.
	got, readErr := os.ReadFile(sentinel)
	if readErr != nil {
		t.Fatalf("reading sentinel: %v", readErr)
	}
	if string(got) != sentinelStem {
		t.Errorf("sentinel child .stem was modified:\n%s", got)
	}

	// No other child .stem may have been created either.
	if _, statErr := os.Stat(filepath.Join(dir, "E02-platform", ".stem")); statErr == nil {
		t.Error("expected no child .stem written when the guard refuses")
	}
}

func TestRunMigrateSplit_PreFlightGuardForceBypass(t *testing.T) {
	dir := setupSplitDir(t)
	sentinel := writeSentinelChildStem(t, dir)

	out, err := runCmd(t, "migrate", dir, "--split", "--force")
	if err != nil {
		t.Fatalf("unexpected error with --force: %v\noutput: %s", err, out)
	}

	got, readErr := os.ReadFile(sentinel)
	if readErr != nil {
		t.Fatalf("reading child .stem: %v", readErr)
	}
	if string(got) == sentinelStem {
		t.Error("expected child .stem to be overwritten with --force")
	}
	if !strings.Contains(string(got), "prefix: F") {
		t.Errorf("expected regenerated child .stem, got:\n%s", got)
	}
}

func TestRunMigrateSplit_IntegralFloatSequenceConfigRejectsBeforeDryRunOrWrite(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(fmt.Sprintf("dry_run=%t", dryRun), func(t *testing.T) {
			dir := setupSplitDir(t)
			rootPath := filepath.Join(dir, ".stem")
			beforeRoot := mustReadFile(t, rootPath)
			rootInfo, err := os.Stat(rootPath)
			if err != nil {
				t.Fatal(err)
			}

			invalid := strings.Replace(string(beforeRoot), `  estado:
    type: enum
    values: [Pending, Completed, In Progress]
    required: true
`, `  estado:
    type: sequence
    prefix: E
    digits: 2
    match:
      "E*": {prefix: E, digits: 2.0}
`, 1)
			if invalid == string(beforeRoot) {
				t.Fatal("fixture did not replace the applicable root field")
			}
			mustWriteFile(t, rootPath, []byte(invalid), rootInfo.Mode())
			beforeRoot = []byte(invalid)
			sentinel := writeSentinelChildStem(t, dir)
			sentinelBefore := mustReadFile(t, sentinel)
			sentinelInfo, err := os.Stat(sentinel)
			if err != nil {
				t.Fatal(err)
			}

			args := []string{"migrate", dir, "--split", "--force"}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := runCmd(t, args...)
			if err == nil || !strings.Contains(err.Error(), "serializing split stems") || !strings.Contains(err.Error(), "digits") {
				t.Fatalf("expected integral-float serialization failure, got out=%q err=%v", out, err)
			}
			assertUnchanged := func(path string, want []byte, wantMode os.FileMode) {
				t.Helper()
				got, readErr := os.ReadFile(path)
				if readErr != nil || string(got) != string(want) {
					t.Fatalf("%s changed: bytes=%q readErr=%v", path, got, readErr)
				}
				info, statErr := os.Stat(path)
				if statErr != nil || info.Mode() != wantMode {
					t.Fatalf("%s mode changed: info=%v statErr=%v want=%v", path, info, statErr, wantMode)
				}
			}
			assertUnchanged(rootPath, beforeRoot, rootInfo.Mode())
			assertUnchanged(sentinel, sentinelBefore, sentinelInfo.Mode())
			if _, statErr := os.Stat(filepath.Join(dir, "E02-platform", ".stem")); !os.IsNotExist(statErr) {
				t.Fatalf("absent split target was created: %v", statErr)
			}
		})
	}
}

func TestRunMigrateSplit_SerializationErrorLeavesNoPartialOutput(t *testing.T) {
	stem := `version: 2
schema:
  code:
    type: sequence
    match:
      E*: bad-config
`
	files := map[string]string{
		"E01-infra/README.md":             "---\nid: E01\ncode: E01\n---\n# E01\n",
		"E02-platform/README.md":          "---\nid: E02\ncode: E02\n---\n# E02\n",
		"E01-infra/F01-net/README.md":     "---\nid: F01\ncode: F01\n---\n# F01\n",
		"E01-infra/F02-store/README.md":   "---\nid: F02\ncode: F02\n---\n# F02\n",
		"E02-platform/F01-auth/README.md": "---\nid: F01\ncode: F01\n---\n# F01\n",
		"E02-platform/F02-api/README.md":  "---\nid: F02\ncode: F02\n---\n# F02\n",
	}
	dir := setupMigrateDir(t, stem, files)
	before, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatalf("reading input stem: %v", err)
	}

	out, err := runCmd(t, "migrate", dir, "--split")
	if err == nil || !strings.Contains(err.Error(), "serializing split stems") || !strings.Contains(err.Error(), "sequence field") {
		t.Fatalf("expected serialization error, got out=%q err=%v", out, err)
	}
	after, readErr := os.ReadFile(filepath.Join(dir, ".stem"))
	if readErr != nil {
		t.Fatalf("reading root after error: %v", readErr)
	}
	if string(after) != string(before) {
		t.Fatalf("root .stem changed despite serialization error:\nbefore:\n%s\nafter:\n%s", before, after)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "E01-infra", ".stem")); !os.IsNotExist(statErr) {
		t.Fatalf("expected no child .stem after serialization error, stat err=%v", statErr)
	}
}

func TestRunMigrateSplit_DryRunAnnotatesCollisions(t *testing.T) {
	dir := setupSplitDir(t)
	sentinel := writeSentinelChildStem(t, dir)

	out, err := runCmd(t, "migrate", dir, "--split", "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	header := fmt.Sprintf("# --- %s ---\n# (existing file will be overwritten)\n", filepath.Join("E01-infra", ".stem"))
	if !strings.Contains(out, header) {
		t.Errorf("expected collision annotation directly after the header, got:\n%s", out)
	}
	// The root .stem is the input being rewritten; it is never a collision.
	if strings.Contains(out, "# --- .stem ---\n# (existing file will be overwritten)") {
		t.Errorf("root .stem must not be annotated as a collision, got:\n%s", out)
	}
	// A non-colliding child gets no annotation.
	if strings.Contains(out, fmt.Sprintf("# --- %s ---\n# (existing file will be overwritten)", filepath.Join("E02-platform", ".stem"))) {
		t.Errorf("non-colliding child must not be annotated, got:\n%s", out)
	}

	// Dry-run writes nothing.
	got, readErr := os.ReadFile(sentinel)
	if readErr != nil {
		t.Fatalf("reading sentinel: %v", readErr)
	}
	if string(got) != sentinelStem {
		t.Errorf("dry-run modified the sentinel:\n%s", got)
	}
	if _, statErr := os.Stat(filepath.Join(dir, "E02-platform", ".stem")); statErr == nil {
		t.Error("expected no child .stem written in dry-run mode")
	}
}
