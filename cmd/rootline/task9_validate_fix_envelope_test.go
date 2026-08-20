package main

import (
	"bytes"
	"encoding/json"
	"reflect"
	"strings"
	"testing"
)

func runTask9CmdSplit(t *testing.T, args ...string) (stdout string, stderr string, err error) {
	t.Helper()
	resetFlags()
	out := new(bytes.Buffer)
	errOut := new(bytes.Buffer)
	rootCmd.SetOut(out)
	rootCmd.SetErr(errOut)
	rootCmd.SetArgs(args)
	err = rootCmd.Execute()
	return out.String(), errOut.String(), err
}

func TestTask9ValidateAllEnrichmentFailureEmitsValidateEnvelope(t *testing.T) {
	dir := setupTask9SourceProject(t, true)

	stdout, stderr, err := runTask9CmdSplit(t, "validate", "--all", dir, "-o", "json")
	if err == nil {
		t.Fatalf("validate --all succeeded unexpectedly; stdout=%s stderr=%s", stdout, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("validate --all wrote stderr %q, want JSON envelope on stdout only", stderr)
	}

	var envelope map[string]any
	if err := json.Unmarshal([]byte(stdout), &envelope); err != nil {
		t.Fatalf("validate --all stdout is not JSON envelope: %v\nstdout=%q", err, stdout)
	}
	assertJSONKeys(t, envelope, []string{"version", "kind", "results", "structural", "stem_health", "drift_warnings", "notices", "summary"})
	if envelope["version"] != float64(2) || envelope["kind"] != "rootline/validate-batch" {
		t.Fatalf("validate envelope identity = version %v kind %q", envelope["version"], envelope["kind"])
	}

	results := envelope["results"].([]any)
	if len(results) != 3 {
		t.Fatalf("validate results len = %d, want scanned document count 3", len(results))
	}
	wantResultPaths := []string{"body.md", "empty.md", "override.md"}
	for i, raw := range results {
		row := raw.(map[string]any)
		assertJSONKeys(t, row, []string{"version", "kind", "path", "valid", "errors", "warnings"})
		if row["version"] != float64(1) || row["kind"] != "rootline/validate" || row["path"] != wantResultPaths[i] || row["valid"] != false {
			t.Fatalf("validate incomplete row = %#v, want invalid rootline/validate result for %s", row, wantResultPaths[i])
		}
		if len(row["errors"].([]any)) != 1 || len(row["warnings"].([]any)) != 0 {
			t.Fatalf("validate incomplete row diagnostics = %#v/%#v, want one error and empty warnings", row["errors"], row["warnings"])
		}
		errObj := row["errors"].([]any)[0].(map[string]any)
		assertJSONKeys(t, errObj, []string{"rule", "field", "message", "source", "severity"})
		if errObj["rule"] != "skipped" || errObj["field"] != "" || errObj["source"] != "schema" || errObj["severity"] != "error" {
			t.Fatalf("validate incomplete diagnostic = %#v, want generic skipped fail-closed error", errObj)
		}
		msg := errObj["message"].(string)
		if msg != "validation incomplete because schema resolution failed; see notices" {
			t.Fatalf("validate incomplete diagnostic message = %q, want generic unvalidated status", msg)
		}
		for _, foreign := range []string{"body.md", "empty.md", "override.md", "notes", "ambiguous body section source"} {
			if strings.Contains(msg, foreign) {
				t.Fatalf("validate incomplete diagnostic message %q leaked causal/path detail %q", msg, foreign)
			}
		}
	}

	for _, key := range []string{"structural", "stem_health", "drift_warnings"} {
		if got := len(envelope[key].([]any)); got != 0 {
			t.Fatalf("validate %s len = %d, want empty collection", key, got)
		}
	}
	notices := envelope["notices"].([]any)
	if len(notices) != 1 {
		t.Fatalf("validate notices = %#v, want one run-level error", notices)
	}
	notice := notices[0].(map[string]any)
	if notice["severity"] != "error" || notice["code"] != "schema_resolution_failed" {
		t.Fatalf("validate notice = %#v, want stable schema_resolution_failed error", notice)
	}
	if !strings.Contains(notice["message"].(string), "ambiguous body section source") {
		t.Fatalf("validate notice message = %q, want ambiguity", notice["message"])
	}

	summary := envelope["summary"].(map[string]any)
	wantSummary := map[string]any{
		"total": float64(3), "valid": float64(0), "invalid": float64(3),
		"errors_count": float64(3), "warnings_count": float64(0),
		"drift_warnings_count": float64(0), "structural_errors_count": float64(0), "structural_warnings_count": float64(0),
		"stem_health_errors_count": float64(0), "stem_health_warnings_count": float64(0), "stem_health_info_count": float64(0),
	}
	if !reflect.DeepEqual(summary, wantSummary) {
		t.Fatalf("validate summary = %#v, want %#v", summary, wantSummary)
	}
}

func TestTask9FixAllSuccessPreservesBaseJSONShape(t *testing.T) {
	dir := setupTask9SourceProject(t, false)

	stdout, stderr, err := runTask9CmdSplit(t, "fix", "--all", dir, "-o", "json")
	if err != nil {
		t.Fatalf("fix --all success failed: %v\nstdout=%s stderr=%s", err, stdout, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("fix --all success wrote stderr %q", stderr)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("fix --all success stdout is not JSON: %v\nstdout=%q", err, stdout)
	}
	assertJSONKeys(t, result, []string{"version", "kind", "results", "summary"})
	if result["version"] != float64(1) || result["kind"] != "rootline/fix-batch" {
		t.Fatalf("fix success identity = version %v kind %q", result["version"], result["kind"])
	}
	if _, exists := result["complete"]; exists {
		t.Fatalf("fix success JSON added complete key: %#v", result)
	}
}

func TestTask9FixAllEnrichmentFailureEmitsIncompleteResultEnvelope(t *testing.T) {
	dir := setupTask9SourceProject(t, true)

	stdout, stderr, err := runTask9CmdSplit(t, "fix", "--all", dir, "-o", "json")
	if err == nil {
		t.Fatalf("fix --all succeeded unexpectedly; stdout=%s stderr=%s", stdout, stderr)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("fix --all wrote stderr %q, want result envelope on stdout only", stderr)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(stdout), &result); err != nil {
		t.Fatalf("fix --all stdout is not JSON result envelope: %v\nstdout=%q", err, stdout)
	}
	assertJSONKeys(t, result, []string{"version", "kind", "complete", "results", "summary", "errors"})
	if result["version"] != float64(1) || result["kind"] != "rootline/fix-batch" {
		t.Fatalf("fix envelope identity = version %v kind %q", result["version"], result["kind"])
	}
	if result["complete"] != false {
		t.Fatalf("fix failure complete = %#v, want false", result["complete"])
	}
	if len(result["results"].([]any)) != 0 {
		t.Fatalf("fix failure results = %#v, want empty", result["results"])
	}
	errs := result["errors"].([]any)
	if len(errs) != 1 || !strings.Contains(errs[0].(string), "ambiguous body section source") {
		t.Fatalf("fix errors = %#v, want enrichment ambiguity", errs)
	}
}

func TestTask9FixAllEnrichmentFailureTableRendersErrorsAndReturnsNonzero(t *testing.T) {
	dir := setupTask9SourceProject(t, true)

	stdout, stderr, err := runTask9CmdSplit(t, "fix", "--all", dir, "-o", "table")
	if err == nil {
		t.Fatalf("fix --all table succeeded unexpectedly; stdout=%s", stdout)
	}
	if strings.TrimSpace(stderr) != "" {
		t.Fatalf("fix --all table wrote stderr %q", stderr)
	}
	for _, want := range []string{"File", "Fixed", "Changes", "Errors", "ambiguous body section source"} {
		if !strings.Contains(stdout, want) {
			t.Fatalf("fix failure table missing %q:\n%s", want, stdout)
		}
	}
}

func assertJSONKeys(t *testing.T, got map[string]any, want []string) {
	t.Helper()
	gotKeys := make([]string, 0, len(got))
	for key := range got {
		gotKeys = append(gotKeys, key)
	}
	if !reflect.DeepEqual(stringSet(gotKeys), stringSet(want)) {
		t.Fatalf("JSON keys = %v, want exactly %v in object %#v", gotKeys, want, got)
	}
}

func stringSet(values []string) map[string]bool {
	set := make(map[string]bool, len(values))
	for _, value := range values {
		set[value] = true
	}
	return set
}
