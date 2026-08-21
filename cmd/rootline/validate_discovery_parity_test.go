package main

import (
	"errors"
	"reflect"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestValidateAllScopeResolverKeepsMissingLocalAndCorruptDiscoverable(t *testing.T) {
	resolver := resilientValidateAllScopeResolver()

	missingRoot := t.TempDir()
	stem, err := resolver(missingRoot)
	if stem != nil || !errors.Is(err, rules.ErrNoSchemaFound) {
		t.Fatalf("missing schema outcome = stem %#v err %v, want nil + canonical ErrNoSchemaFound", stem, err)
	}

	corruptRoot := setupValidateProject(t, map[string]string{
		".stem": "version: 2\nroot: true\nscope:\n  match: [\n",
	})
	stem, err = resolver(corruptRoot)
	if err != nil || stem == nil {
		t.Fatalf("corrupt schema outcome = stem %#v err %v, want discoverable empty stem without resolver error", stem, err)
	}
}

func TestValidateAllMixedUngovernedSiblingKeepsNestedGovernedPopulation(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"orphan.md":      "---\ntitle: Orphan\n---\n# Orphan\n",
		"notes.txt":      "ordinary sibling\n",
		"nested/.stem":   "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n",
		"nested/good.md": "---\ntitle: Good\n---\n# Good\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all", ".", "-o", "json")
	if err != nil {
		t.Fatalf("validate --all mixed governance err = %v\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	if got := envelopePaths(t, env); !reflect.DeepEqual(got, []string{"nested/good.md"}) {
		t.Fatalf("result paths = %v, want only nested governed record; env=%#v", got, env)
	}
	row := env["results"].([]any)[0].(map[string]any)
	if row["valid"] != true || len(row["errors"].([]any)) != 0 || len(row["warnings"].([]any)) != 0 {
		t.Fatalf("nested governed row = %#v, want valid with no diagnostics", row)
	}
	assertSummaryCounts(t, env, map[string]float64{"total": 1, "valid": 1, "invalid": 0, "errors_count": 0})
	if hasNotice(t, env, "scan_failed") {
		t.Fatalf("notices = %#v, want no scan_failed for mixed local ungoverned sibling", env["notices"])
	}
}

func TestValidateAllPureUngovernedStillScanFailed(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		"orphan.md": "---\ntitle: Orphan\n---\n# Orphan\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all", ".", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	if got := envelopePaths(t, env); len(got) != 0 {
		t.Fatalf("results = %v, want none for pure ungoverned scan failure", got)
	}
	assertNoticeCodes(t, env, []string{"scan_failed"})
	assertSummaryCounts(t, env, map[string]float64{"total": 0, "valid": 0, "invalid": 0})
}

func TestValidateAllCorruptStemignoreExcludesIgnoredCandidates(t *testing.T) {
	root := setupValidateProject(t, corruptStemignoreFixture())
	mustChdir(t, root)

	stdout, err := executeValidate(t, "--all", ".", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	assertSkippedResultPaths(t, env, []string{"keep.md"})
	assertNoticeCodes(t, env, []string{"schema_resolution_failed"})
}

func TestValidateExplicitMultiAndStagedCorruptStemignoreKeepIgnoredScopeSkipped(t *testing.T) {
	t.Run("explicit", func(t *testing.T) {
		root := setupValidateProject(t, corruptStemignoreFixture())
		mustChdir(t, root)

		stdout, err := executeValidate(t, "ignored.md", "-o", "json")
		if err != nil {
			t.Fatalf("ignored explicit err = %v, want warning-only skipped result\nstdout=%s", err, stdout)
		}
		env := decodeEnvelope(t, stdout)
		assertIgnoredScopeSkippedResult(t, env, []string{"ignored.md"})
		if notices := env["notices"].([]any); len(notices) != 0 {
			t.Fatalf("notices = %#v, want none for ignored explicit file", notices)
		}
	})

	t.Run("multi", func(t *testing.T) {
		root := setupValidateProject(t, corruptStemignoreFixture())
		mustChdir(t, root)

		stdout, err := executeValidate(t, "keep.md", "ignored.md", "-o", "json")
		if err != ErrValidationFailed {
			t.Fatalf("multi err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
		}
		env := decodeEnvelope(t, stdout)
		if got := envelopePaths(t, env); !reflect.DeepEqual(got, []string{"keep.md", "ignored.md"}) {
			t.Fatalf("multi paths = %v", got)
		}
		assertSkippedResultPaths(t, map[string]any{"results": []any{env["results"].([]any)[0]}}, []string{"keep.md"})
		assertIgnoredScopeSkippedResult(t, map[string]any{"results": []any{env["results"].([]any)[1]}}, []string{"ignored.md"})
		assertNoticeCodes(t, env, []string{"schema_resolution_failed"})
	})

	t.Run("staged", func(t *testing.T) {
		root := makeStagedRepo(t, corruptStemignoreFixture())
		mustChdir(t, root)

		stdout, err := executeValidate(t, "--staged", "-o", "json")
		if err != ErrValidationFailed {
			t.Fatalf("staged err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
		}
		env := decodeEnvelope(t, stdout)
		if got := envelopePaths(t, env); !reflect.DeepEqual(got, []string{"ignored.md", "keep.md"}) {
			t.Fatalf("staged paths = %v", got)
		}
		assertIgnoredScopeSkippedResult(t, map[string]any{"results": []any{env["results"].([]any)[0]}}, []string{"ignored.md"})
		assertSkippedResultPaths(t, map[string]any{"results": []any{env["results"].([]any)[1]}}, []string{"keep.md"})
		assertNoticeCodes(t, env, []string{"schema_resolution_failed"})
	})
}

func corruptStemignoreFixture() map[string]string {
	return map[string]string{
		".stem":       "version: 2\nroot: true\nscope:\n  match: [\n",
		".stemignore": "ignored.md\n",
		"keep.md":     "---\ntitle: Keep\n---\n# Keep\n",
		"ignored.md":  "---\ntitle: Ignored\n---\n# Ignored\n",
	}
}

func assertIgnoredScopeSkippedResult(t *testing.T, env map[string]any, want []string) {
	t.Helper()
	results := env["results"].([]any)
	if len(results) != len(want) {
		t.Fatalf("results len = %d, want %d: %#v", len(results), len(want), results)
	}
	for i, wantPath := range want {
		row := results[i].(map[string]any)
		assertJSONKeys(t, row, []string{"version", "kind", "path", "valid", "errors", "warnings"})
		if row["path"] != wantPath || row["valid"] != true {
			t.Fatalf("ignored result[%d] = %#v, want valid skipped warning result for %s", i, row, wantPath)
		}
		if errs := row["errors"].([]any); len(errs) != 0 {
			t.Fatalf("ignored result[%d] errors = %#v, want none", i, errs)
		}
		warnings := row["warnings"].([]any)
		if len(warnings) != 1 {
			t.Fatalf("ignored result[%d] warnings = %#v, want one", i, warnings)
		}
		diag := warnings[0].(map[string]any)
		assertJSONKeys(t, diag, []string{"rule", "field", "message", "source", "severity"})
		if diag["rule"] != "skipped" || diag["source"] != "scope" || diag["severity"] != rules.SeverityWarn {
			t.Fatalf("ignored diagnostic = %#v, want scope skipped warning", diag)
		}
	}
}
