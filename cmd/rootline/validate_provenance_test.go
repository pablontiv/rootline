package main

import (
	"os"
	"path/filepath"
	"reflect"
	"sort"
	"strings"
	"testing"
)

func TestValidateErrorSources_SingleFileCWDIndependent(t *testing.T) {
	root := setupValidateProvenanceProject(t)
	mustChdir(t, filepath.Join(root, "docs", "nested"))

	stdout, err := executeValidate(t, filepath.Join(root, "docs", "parent.md"), "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	assertJSONKeys(t, env, []string{"version", "kind", "results", "structural", "stem_health", "drift_warnings", "notices", "summary"})
	if env["version"] != float64(2) || env["kind"] != "rootline/validate-batch" {
		t.Fatalf("envelope identity = version %v kind %q", env["version"], env["kind"])
	}

	row := firstResult(t, stdout)
	got := validationErrorSourcesByRule(t, row)
	want := map[string][]string{"enum": {".stem"}, "required": {".stem"}}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("sources by rule = %#v, want %#v", got, want)
	}
	if got := validationWarningSourcesByRule(t, row); !reflect.DeepEqual(got, map[string][]string{"enum": {".stem"}}) {
		t.Fatalf("warning sources by rule = %#v", got)
	}
	assertNoAbsoluteOrBackslashInDocumentSources(t, env)
}

func TestValidateErrorSources_AllScanSubdirAndNestedRoots(t *testing.T) {
	root := setupValidateProvenanceProject(t)
	mustChdir(t, filepath.Join(root, "docs"))

	stdout, err := executeValidate(t, "--all", ".", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	assertJSONKeys(t, env, []string{"version", "kind", "results", "structural", "stem_health", "drift_warnings", "notices", "summary"})
	assertNoAbsoluteOrBackslashInDocumentSources(t, env)

	byPath := validationResultsByPath(t, env)
	if got := sortedKeys(byPath); !reflect.DeepEqual(got, []string{"nested/child.md", "parent.md"}) {
		t.Fatalf("result paths = %v", got)
	}
	if got := validationErrorSourcesByRule(t, byPath["parent.md"]); !reflect.DeepEqual(got, map[string][]string{"enum": {".stem"}, "required": {".stem"}}) {
		t.Fatalf("parent sources by rule = %#v", got)
	}
	if got := validationErrorSourcesByRule(t, byPath["nested/child.md"]); !reflect.DeepEqual(got, map[string][]string{"required": {".stem"}}) {
		t.Fatalf("nested sources by rule = %#v", got)
	}
}

func TestValidateErrorSources_TableDoesNotLeakPhysicalSources(t *testing.T) {
	root := setupValidateProvenanceProject(t)
	mustChdir(t, filepath.Join(root, "docs"))

	stdout, err := executeValidate(t, "--all", ".", "-o", "table")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	for _, want := range []string{"File", "Valid", "Errors", "parent.md", "nested/child.md", "required field"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("table output missing %q:\n%s", want, stdout)
		}
	}
	for _, forbidden := range []string{root, `\\`} {
		if strings.Contains(stdout, forbidden) {
			t.Fatalf("table output leaked %q:\n%s", forbidden, stdout)
		}
	}
}

func TestValidateErrorSources_SingleFileParentRelativeRecordSource(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":  "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n",
		"bad.md": "---\ntitle: Bad\n...\nextra: doc\n---\n# Bad\n",
	})
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0o755); err != nil {
		t.Fatal(err)
	}
	mustChdir(t, subdir)

	stdout, err := executeValidate(t, "../bad.md", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	assertNoAbsoluteOrBackslashInDocumentSources(t, env)
	row := firstResult(t, stdout)
	if row["path"] != "../bad.md" {
		t.Fatalf("path = %v, want invocation spelling", row["path"])
	}
	got := validationErrorSourcesByRule(t, row)
	if !reflect.DeepEqual(got, map[string][]string{"multiple_yaml_documents": {"bad.md"}}) {
		t.Fatalf("sources by rule = %#v", got)
	}
	if notices := env["notices"].([]any); len(notices) != 0 {
		t.Fatalf("notices = %#v, want none for in-root parent-relative record", notices)
	}
}

func TestValidateErrorSources_AllRecordOwnedSourcesUseRecordGovernanceRoot(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":                "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n",
		"docs/parent.md":       "---\ntitle: Parent\n...\nextra: doc\n---\n# Parent\n",
		"docs/nested/.stem":    "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n",
		"docs/nested/child.md": "---\ntitle: Child\n...\nextra: doc\n---\n# Child\n",
	})
	mustChdir(t, filepath.Join(root, "docs"))

	stdout, err := executeValidate(t, "--all", ".", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	assertNoAbsoluteOrBackslashInDocumentSources(t, env)
	byPath := validationResultsByPath(t, env)
	if got := validationErrorSourcesByRule(t, byPath["parent.md"]); !reflect.DeepEqual(got, map[string][]string{"multiple_yaml_documents": {"docs/parent.md"}}) {
		t.Fatalf("parent sources by rule = %#v", got)
	}
	if got := validationErrorSourcesByRule(t, byPath["nested/child.md"]); !reflect.DeepEqual(got, map[string][]string{"multiple_yaml_documents": {"child.md"}}) {
		t.Fatalf("nested sources by rule = %#v", got)
	}
}

func TestValidateErrorSources_ResolveFailureEmitsOrderedEnvelope(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":    "version: 2\nroot: true\nscope:\n  match: [\n",
		"one.md":   "---\ntitle: One\n---\n# One\n",
		"two.md":   "---\ntitle: Two\n---\n# Two\n",
		"skip.txt": "ignored\n",
	})
	mustChdir(t, root)

	stdout, err := executeValidate(t, "one.md", "two.md", "-o", "json")
	if err != ErrValidationFailed {
		t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
	}
	env := decodeEnvelope(t, stdout)
	assertJSONKeys(t, env, []string{"version", "kind", "results", "structural", "stem_health", "drift_warnings", "notices", "summary"})
	results := env["results"].([]any)
	if len(results) != 2 {
		t.Fatalf("results len = %d, want 2", len(results))
	}
	for i, wantPath := range []string{"one.md", "two.md"} {
		row := results[i].(map[string]any)
		assertJSONKeys(t, row, []string{"version", "kind", "path", "valid", "errors", "warnings"})
		if row["path"] != wantPath {
			t.Fatalf("result[%d] path = %v, want %s", i, row["path"], wantPath)
		}
		errs := row["errors"].([]any)
		if len(errs) != 1 {
			t.Fatalf("result[%d] errors = %#v, want one skipped diagnostic", i, errs)
		}
		diag := errs[0].(map[string]any)
		assertJSONKeys(t, diag, []string{"rule", "field", "message", "source", "severity"})
		if diag["rule"] != "skipped" || diag["source"] != "schema" || diag["severity"] != "error" {
			t.Fatalf("result[%d] diagnostic = %#v", i, diag)
		}
	}
	summary := env["summary"].(map[string]any)
	if summary["total"] != float64(2) || summary["valid"] != float64(0) || summary["invalid"] != float64(2) {
		t.Fatalf("summary = %#v, want coherent two-record failure", summary)
	}
	if notices := env["notices"].([]any); len(notices) != 2 || notices[0].(map[string]any)["code"] != "schema_resolution_failed" || notices[1].(map[string]any)["code"] != "schema_resolution_failed" {
		t.Fatalf("notices = %#v, want one schema_resolution_failed per failed record", notices)
	}
}

func setupValidateProvenanceProject(t *testing.T) string {
	t.Helper()
	return setupValidateProject(t, map[string]string{
		".stem":                "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n    required: true\n  estado:\n    type: enum\n    values: [Done]\n  priority:\n    type: enum\n    values: [high]\n    severity: warn\n",
		"docs/parent.md":       "---\nestado: Todo\npriority: low\n---\n# Parent\n",
		"docs/nested/.stem":    "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  status:\n    type: string\n    required: true\n",
		"docs/nested/child.md": "---\nother: value\n---\n# Child\n",
	})
}

func validationResultsByPath(t *testing.T, env map[string]any) map[string]map[string]any {
	t.Helper()
	out := map[string]map[string]any{}
	for _, raw := range env["results"].([]any) {
		row := raw.(map[string]any)
		out[row["path"].(string)] = row
	}
	return out
}

func validationErrorSourcesByRule(t *testing.T, result map[string]any) map[string][]string {
	t.Helper()
	return validationSourcesByRule(t, result, "errors")
}

func validationWarningSourcesByRule(t *testing.T, result map[string]any) map[string][]string {
	t.Helper()
	return validationSourcesByRule(t, result, "warnings")
}

func validationSourcesByRule(t *testing.T, result map[string]any, collection string) map[string][]string {
	t.Helper()
	out := map[string][]string{}
	for _, raw := range result[collection].([]any) {
		err := raw.(map[string]any)
		out[err["rule"].(string)] = append(out[err["rule"].(string)], err["source"].(string))
	}
	for rule := range out {
		sort.Strings(out[rule])
	}
	return out
}

func assertNoAbsoluteOrBackslashInDocumentSources(t *testing.T, env map[string]any) {
	t.Helper()
	for _, rawResult := range env["results"].([]any) {
		row := rawResult.(map[string]any)
		for _, collection := range []string{"errors", "warnings"} {
			for _, rawErr := range row[collection].([]any) {
				source := rawErr.(map[string]any)["source"].(string)
				if filepath.IsAbs(source) || strings.Contains(source, `\\`) {
					t.Fatalf("document %s source leaked physical path: %q", collection, source)
				}
			}
		}
	}
}

func sortedKeys[V any](m map[string]V) []string {
	keys := make([]string, 0, len(m))
	for key := range m {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	return keys
}
