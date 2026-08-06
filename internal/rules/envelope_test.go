package rules

import (
	"encoding/json"
	"testing"
)

func TestStemHealthDiagnostics_MapsStatusToSeverity(t *testing.T) {
	res := &StemHealthResult{Checks: []StemHealthCheck{
		{Name: "yaml-valid", Status: "pass", Path: "ok/.stem"},
		{Name: "yaml-valid", Status: "fail", Path: "bad/.stem", Message: "invalid YAML"},
		{Name: "scope-match", Status: "warn", Path: "c/.stem", Message: "matches no files"},
		{Name: "nested-root-marker", Status: "info", Path: "d/.stem", Message: "nested root"},
		{Name: "stem-files-exist", Status: "warn", Message: "no .stem files found"},
	}}

	got := StemHealthDiagnostics(res)

	if len(got) != 4 {
		t.Fatalf("len = %d, want 4 (pass dropped); got %+v", len(got), got)
	}
	want := []struct {
		check    string
		severity string
		path     string
	}{
		{"yaml-valid", "error", "bad/.stem"},
		{"scope-match", "warn", "c/.stem"},
		{"nested-root-marker", "info", "d/.stem"},
		{"stem-files-exist", "warn", ".stem"},
	}
	for i, w := range want {
		if got[i].Check != w.check {
			t.Errorf("[%d] check = %q, want %q", i, got[i].Check, w.check)
		}
		if got[i].Severity != w.severity {
			t.Errorf("[%d] severity = %q, want %q", i, got[i].Severity, w.severity)
		}
		if got[i].Path != w.path {
			t.Errorf("[%d] path = %q, want %q", i, got[i].Path, w.path)
		}
	}
}

func TestStemHealthDiagnostics_NilResult(t *testing.T) {
	if got := StemHealthDiagnostics(nil); len(got) != 0 {
		t.Fatalf("expected no diagnostics for nil result, got %+v", got)
	}
}

func TestBatchValidationResult_StemHealthNotCountedAsRecord(t *testing.T) {
	results := []*ValidationResult{
		NewValidationResult("a.md", nil),
		NewValidationResult("b.md", nil),
	}
	health := []StemHealthDiagnostic{
		{Path: "sub/.stem", Check: "scope-match", Severity: "warn", Message: "m"},
		{Path: "sub/.stem", Check: "rule-field-exists", Severity: "warn", Message: "m"},
		{Path: "bad/.stem", Check: "yaml-valid", Severity: "error", Message: "m"},
		{Path: "d/.stem", Check: "nested-root-marker", Severity: "info", Message: "m"},
	}

	batch := NewValidationEnvelope(ValidationEnvelopeInput{Results: results, StemHealth: health})

	if batch.Summary.Total != 2 {
		t.Errorf("summary.total = %d, want 2 (records only)", batch.Summary.Total)
	}
	if batch.Summary.Valid != 2 {
		t.Errorf("summary.valid = %d, want 2", batch.Summary.Valid)
	}
	if batch.Summary.WarningsCount != 0 {
		t.Errorf("summary.warnings_count = %d, want 0 (stem health is counted separately)", batch.Summary.WarningsCount)
	}
	if batch.Summary.StemHealthErrorsCount != 1 {
		t.Errorf("stem_health_errors_count = %d, want 1", batch.Summary.StemHealthErrorsCount)
	}
	if batch.Summary.StemHealthWarningsCount != 2 {
		t.Errorf("stem_health_warnings_count = %d, want 2", batch.Summary.StemHealthWarningsCount)
	}
	if batch.Summary.StemHealthInfoCount != 1 {
		t.Errorf("stem_health_info_count = %d, want 1", batch.Summary.StemHealthInfoCount)
	}
	for _, r := range batch.Results {
		if r.Path == "sub/.stem" || r.Path == "bad/.stem" {
			t.Errorf("stem health leaked into results: %s", r.Path)
		}
	}
}

func TestValidationEnvelope_KeysAlwaysPresent(t *testing.T) {
	data, err := json.Marshal(NewValidationEnvelope(ValidationEnvelopeInput{}))
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	var obj map[string]any
	if err := json.Unmarshal(data, &obj); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if obj["version"].(float64) != 2 {
		t.Errorf("version = %v, want 2", obj["version"])
	}
	if obj["kind"] != "rootline/validate-batch" {
		t.Errorf("kind = %v, want rootline/validate-batch", obj["kind"])
	}
	for _, key := range []string{"results", "structural", "stem_health", "drift_warnings", "notices"} {
		val, ok := obj[key]
		if !ok {
			t.Errorf("key %q missing from envelope", key)
			continue
		}
		arr, ok := val.([]any)
		if !ok {
			t.Errorf("key %q = %v, want an array", key, val)
			continue
		}
		if len(arr) != 0 {
			t.Errorf("key %q = %v, want empty array", key, arr)
		}
	}
	summary, ok := obj["summary"].(map[string]any)
	if !ok {
		t.Fatalf("summary missing or not an object: %v", obj["summary"])
	}
	for _, key := range []string{
		"total", "valid", "invalid", "errors_count", "warnings_count",
		"drift_warnings_count", "structural_errors_count", "structural_warnings_count",
		"stem_health_errors_count",
		"stem_health_warnings_count", "stem_health_info_count",
	} {
		if _, ok := summary[key]; !ok {
			t.Errorf("summary key %q missing", key)
		}
	}
}

func TestValidationEnvelope_NoticesCarrySeverityAndCode(t *testing.T) {
	batch := NewValidationEnvelope(ValidationEnvelopeInput{Notices: []Notice{
		{Severity: "error", Code: "scan_failed", Message: "scanning: boom"},
	}})
	if len(batch.Notices) != 1 {
		t.Fatalf("notices = %+v, want 1", batch.Notices)
	}
	n := batch.Notices[0]
	if n.Severity != "error" || n.Code != "scan_failed" {
		t.Errorf("notice = %+v", n)
	}
	if !batch.HasErrorNotice() {
		t.Error("HasErrorNotice() = false, want true")
	}
}
