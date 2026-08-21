package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestValidateAllCorruptStemDiscoveryEmitsOrderedSkippedRecords(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":     "version: 2\nroot: true\nscope:\n  match: [\n",
		"a.md":      "---\ntitle: A\n---\n# A\n",
		"b.md":      "---\ntitle: B\n---\n# B\n",
		"notes.txt": "ignored\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	if strings.TrimSpace(stdout) == "" {
		t.Fatal("validate --all returned nonzero before writing stdout")
	}
	env := decodeEnvelope(t, stdout)
	assertJSONKeys(t, env, []string{"version", "kind", "results", "structural", "stem_health", "drift_warnings", "notices", "summary"})
	assertSkippedResultPaths(t, env, []string{"a.md", "b.md"})
	assertSummaryCounts(t, env, map[string]float64{
		"total":                    2,
		"valid":                    0,
		"invalid":                  2,
		"errors_count":             2,
		"warnings_count":           0,
		"stem_health_errors_count": 1,
	})
	assertNoticeCodes(t, env, []string{"schema_resolution_failed", "schema_resolution_failed"})
	if hasNotice(t, env, "scan_failed") {
		t.Fatalf("notices = %#v, want per-record schema_resolution_failed without scan_failed", env["notices"])
	}
}

func TestValidateAllCorruptStemTableWritesRowsBeforeNonzero(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nroot: true\nscope:\n  match: [\n",
		"a.md":  "---\ntitle: A\n---\n# A\n",
		"b.md":  "---\ntitle: B\n---\n# B\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all", "-o", "table")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	for _, want := range []string{"File", "Valid", "Errors", "a.md", "b.md", "schema_resolution_failed"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("table output missing %q:\n%s", want, stdout)
		}
	}
}

func TestValidateStagedCorruptStemUsesSameSkippedRecordEnvelope(t *testing.T) {
	root := makeStagedRepo(t, map[string]string{
		".stem": "version: 2\nroot: true\nscope:\n  match: [\n",
		"a.md":  "---\ntitle: A\n---\n# A\n",
		"b.md":  "---\ntitle: B\n---\n# B\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--staged", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	assertSkippedResultPaths(t, env, []string{"a.md", "b.md"})
	assertSummaryCounts(t, env, map[string]float64{"total": 2, "valid": 0, "invalid": 2, "errors_count": 2})
	assertNoticeCodes(t, env, []string{"schema_resolution_failed", "schema_resolution_failed"})
}

func TestValidateAllRealDocumentIOFailureEmitsScanFailed(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink creation requires privileges on some Windows hosts")
	}
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n",
	})
	if err := os.Symlink("missing.md", filepath.Join(root, "dangling.md")); err != nil {
		t.Fatalf("creating dangling symlink: %v", err)
	}
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	if got := envelopePaths(t, env); len(got) != 0 {
		t.Fatalf("results = %v, want none for scanner IO failure", got)
	}
	assertNoticeCodes(t, env, []string{"scan_failed"})
	assertSummaryCounts(t, env, map[string]float64{"total": 0, "valid": 0, "invalid": 0})
}

func TestNormalizeValidationResultSourcesRejectsPrivateOutsideRootAdapter(t *testing.T) {
	root := t.TempDir()
	absPath := filepath.Join(root, "doc.md")
	outside := filepath.Join(filepath.Dir(root), "outside.md")
	result := rules.NewValidationResult("doc.md", []rules.ValidationError{{
		Rule:     "private",
		Field:    "x",
		Message:  "outside",
		Source:   outside,
		Severity: rules.SeverityError,
	}})

	_, err := normalizeValidationResultSources(result, "doc.md", absPath, root)
	if err == nil {
		t.Fatal("expected adapter to reject outside-root validation source")
	}
}

func TestCanonicalizeRecordOwnedSourcesRequiresExactRecordSourceOwnership(t *testing.T) {
	root := t.TempDir()
	absPath := filepath.Join(root, "doc.md")
	errs := []rules.ValidationError{{
		Rule:     "private_relative",
		Field:    "x",
		Message:  "relative source merely cleans to record path",
		Source:   "sub/../doc.md",
		Severity: rules.SeverityError,
	}}

	got := canonicalizeRecordOwnedSources(errs, "doc.md", absPath)
	if got[0].Source != "sub/../doc.md" {
		t.Fatalf("source = %q, want ambiguous physical source preserved", got[0].Source)
	}
	if errs[0].Source != "sub/../doc.md" {
		t.Fatalf("input source mutated to %q", errs[0].Source)
	}
}

func TestResolveValidationRecordContextMatchesRulesResolveForRecord(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nroot: true\nscope:\n  match: \"*.txt\"\nschema:\n  parent_only:\n    type: string\n    required: true\n",
		"docs/.stem": `version: 2
root: true
scope:
  match: "*.md"
schema:
  local:
    type: string
    required: true
  task_required:
    type: string
    required:
      match: ["T*"]
  task_or_feature:
    type: enum
    values: [feature, task]
    match: ["F*", "T*"]
`,
		"docs/T001-task.md": "---\nlocal: yes\n---\n# Task\n",
		"docs/README.md":    "---\nlocal: yes\n---\n# Readme\n",
	})
	for _, tt := range []struct {
		recordPath string
		wantFields []string
	}{
		{recordPath: "docs/T001-task.md", wantFields: []string{"local", "task_or_feature", "task_required"}},
		{recordPath: "docs/README.md", wantFields: []string{"local", "task_required"}},
	} {
		t.Run(tt.recordPath, func(t *testing.T) {
			absPath := filepath.Join(root, tt.recordPath)
			ctx, err := resolveValidationRecordContext(absPath, tt.recordPath)
			if err != nil {
				t.Fatal(err)
			}
			want, err := rules.ResolveForRecord(filepath.Dir(absPath), tt.recordPath)
			if err != nil {
				t.Fatal(err)
			}
			if !reflect.DeepEqual(ctx.effective, want) {
				t.Fatalf("context effective = %#v, want exact ResolveForRecord %#v", ctx.effective, want)
			}
			if got := sortedKeys(ctx.effective.Schema); !reflect.DeepEqual(got, tt.wantFields) {
				t.Fatalf("effective schema fields = %v, want %v", got, tt.wantFields)
			}
			if ctx.governanceRoot != filepath.Join(root, "docs") {
				t.Fatalf("governanceRoot = %q, want nested root", ctx.governanceRoot)
			}
		})
	}
}

func assertSkippedResultPaths(t *testing.T, env map[string]any, want []string) {
	t.Helper()
	results := env["results"].([]any)
	if len(results) != len(want) {
		t.Fatalf("results len = %d, want %d: %#v", len(results), len(want), results)
	}
	for i, wantPath := range want {
		row := results[i].(map[string]any)
		assertJSONKeys(t, row, []string{"version", "kind", "path", "valid", "errors", "warnings"})
		if row["path"] != wantPath || row["valid"] != false {
			t.Fatalf("result[%d] = %#v, want invalid %s", i, row, wantPath)
		}
		errs := row["errors"].([]any)
		if len(errs) != 1 {
			t.Fatalf("result[%d] errors = %#v, want one skipped diagnostic", i, errs)
		}
		diag := errs[0].(map[string]any)
		assertJSONKeys(t, diag, []string{"rule", "field", "message", "source", "severity"})
		if diag["rule"] != "skipped" || diag["field"] != "" || diag["source"] != "schema" || diag["severity"] != rules.SeverityError {
			t.Fatalf("result[%d] diagnostic = %#v", i, diag)
		}
		if warnings := row["warnings"].([]any); len(warnings) != 0 {
			t.Fatalf("result[%d] warnings = %#v, want none", i, warnings)
		}
	}
}

func assertSummaryCounts(t *testing.T, env map[string]any, want map[string]float64) {
	t.Helper()
	summary := env["summary"].(map[string]any)
	for key, wantValue := range want {
		if got := summary[key].(float64); got != wantValue {
			t.Fatalf("summary[%s] = %v, want %v; full summary=%#v", key, got, wantValue, summary)
		}
	}
}

func assertNoticeCodes(t *testing.T, env map[string]any, want []string) {
	t.Helper()
	notices := env["notices"].([]any)
	if len(notices) != len(want) {
		t.Fatalf("notices len = %d, want %d: %#v", len(notices), len(want), notices)
	}
	for i, wantCode := range want {
		notice := notices[i].(map[string]any)
		assertJSONKeys(t, notice, []string{"severity", "code", "message"})
		if notice["severity"] != rules.SeverityError || notice["code"] != wantCode {
			t.Fatalf("notice[%d] = %#v, want error %s", i, notice, wantCode)
		}
	}
}
