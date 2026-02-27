package rules

import (
	"encoding/json"
	"testing"
)

func TestValidationResult_ValidDocument(t *testing.T) {
	result := NewValidationResult("docs/readme.md", nil)

	if !result.Valid {
		t.Error("expected Valid=true for no errors")
	}
	if result.Version != 1 {
		t.Errorf("version = %d, want 1", result.Version)
	}
	if result.Kind != "rootline/validate" {
		t.Errorf("kind = %q, want rootline/validate", result.Kind)
	}
	if len(result.Errors) != 0 {
		t.Errorf("errors = %v, want empty", result.Errors)
	}
}

func TestValidationResult_InvalidDocument(t *testing.T) {
	errs := []ValidationError{
		{Rule: "required", Field: "title", Message: "required field \"title\" is missing", Source: "root/.stem"},
		{Rule: "enum", Field: "estado", Message: "value Invalid not in values", Source: "tasks/.stem"},
	}
	result := NewValidationResult("docs/task.md", errs)

	if result.Valid {
		t.Error("expected Valid=false with errors")
	}
	if len(result.Errors) != 2 {
		t.Errorf("got %d errors, want 2", len(result.Errors))
	}
	if result.Path != "docs/task.md" {
		t.Errorf("path = %q", result.Path)
	}
}

func TestValidationResult_ToJSON(t *testing.T) {
	result := NewValidationResult("test.md", []ValidationError{
		{Rule: "required", Field: "title", Message: "missing", Source: ".stem"},
	})

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	// Parse back to verify structure
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if parsed["version"].(float64) != 1 {
		t.Errorf("version = %v", parsed["version"])
	}
	if parsed["kind"] != "rootline/validate" {
		t.Errorf("kind = %v", parsed["kind"])
	}
	if parsed["valid"] != false {
		t.Errorf("valid = %v, want false", parsed["valid"])
	}
	if parsed["path"] != "test.md" {
		t.Errorf("path = %v", parsed["path"])
	}
	errors := parsed["errors"].([]any)
	if len(errors) != 1 {
		t.Errorf("errors length = %d", len(errors))
	}
}

func TestValidationResult_ToJSON_ValidEmpty(t *testing.T) {
	result := NewValidationResult("valid.md", nil)

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if parsed["valid"] != true {
		t.Errorf("valid = %v, want true", parsed["valid"])
	}
	// Errors should be empty array, not null
	errors := parsed["errors"].([]any)
	if len(errors) != 0 {
		t.Errorf("errors = %v, want empty array", errors)
	}
}

func TestValidationResult_JSONStable(t *testing.T) {
	result := NewValidationResult("doc.md", []ValidationError{
		{Rule: "enum", Field: "estado", Message: "bad value", Source: "x/.stem"},
	})

	data1, _ := result.ToJSON()
	data2, _ := result.ToJSON()

	if string(data1) != string(data2) {
		t.Error("JSON output not stable across calls")
	}
}

func TestBatchValidationResult_Mixed(t *testing.T) {
	results := []*ValidationResult{
		NewValidationResult("valid.md", nil),
		NewValidationResult("invalid.md", []ValidationError{
			{Rule: "required", Field: "title", Message: "missing", Source: ".stem"},
		}),
		NewValidationResult("also-valid.md", nil),
	}

	batch := NewBatchValidationResult(results)

	if batch.Version != 1 {
		t.Errorf("version = %d", batch.Version)
	}
	if batch.Kind != "rootline/validate-batch" {
		t.Errorf("kind = %q", batch.Kind)
	}
	if batch.Summary.Total != 3 {
		t.Errorf("total = %d", batch.Summary.Total)
	}
	if batch.Summary.Valid != 2 {
		t.Errorf("valid = %d", batch.Summary.Valid)
	}
	if batch.Summary.Invalid != 1 {
		t.Errorf("invalid = %d", batch.Summary.Invalid)
	}
}

func TestBatchValidationResult_AllValid(t *testing.T) {
	results := []*ValidationResult{
		NewValidationResult("a.md", nil),
		NewValidationResult("b.md", nil),
	}

	batch := NewBatchValidationResult(results)

	if batch.Summary.Invalid != 0 {
		t.Errorf("invalid = %d, want 0", batch.Summary.Invalid)
	}
	if batch.Summary.Valid != 2 {
		t.Errorf("valid = %d, want 2", batch.Summary.Valid)
	}
}

func TestBatchValidationResult_AllInvalid(t *testing.T) {
	results := []*ValidationResult{
		NewValidationResult("a.md", []ValidationError{{Rule: "required", Field: "x", Message: "m", Source: "s"}}),
		NewValidationResult("b.md", []ValidationError{{Rule: "enum", Field: "y", Message: "m", Source: "s"}}),
	}

	batch := NewBatchValidationResult(results)

	if batch.Summary.Valid != 0 {
		t.Errorf("valid = %d, want 0", batch.Summary.Valid)
	}
	if batch.Summary.Invalid != 2 {
		t.Errorf("invalid = %d, want 2", batch.Summary.Invalid)
	}
}

func TestBatchValidationResult_Empty(t *testing.T) {
	batch := NewBatchValidationResult(nil)

	if batch.Summary.Total != 0 {
		t.Errorf("total = %d", batch.Summary.Total)
	}
}

func TestBatchValidationResult_WithDrift(t *testing.T) {
	results := []*ValidationResult{
		NewValidationResult("project/README.md", nil),
		NewValidationResult("project/task1.md", nil),
	}
	driftWarnings := []DriftWarning{
		{
			Field:         "estado",
			ParentValue:   "In Progress",
			ChildrenValue: "Completed",
			ParentPath:    "project/README.md",
			ChildPaths:    []string{"project/task1.md"},
		},
	}

	batch := NewBatchValidationResultWithDrift(results, driftWarnings)

	if len(batch.DriftWarnings) != 1 {
		t.Errorf("drift_warnings len = %d, want 1", len(batch.DriftWarnings))
	}
	if batch.Summary.DriftWarningsCount != 1 {
		t.Errorf("drift_warnings_count = %d, want 1", batch.Summary.DriftWarningsCount)
	}
	// Drift warnings don't affect validity
	if batch.Summary.Invalid != 0 {
		t.Errorf("invalid = %d, want 0 (drift is warning only)", batch.Summary.Invalid)
	}
}

func TestBatchValidationResult_DriftInJSON(t *testing.T) {
	results := []*ValidationResult{
		NewValidationResult("a.md", nil),
	}
	driftWarnings := []DriftWarning{
		{Field: "estado", ParentValue: "In Progress", ChildrenValue: "Completed", ParentPath: "a.md"},
	}
	batch := NewBatchValidationResultWithDrift(results, driftWarnings)

	data, err := batch.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	dw, ok := parsed["drift_warnings"].([]any)
	if !ok {
		t.Fatal("drift_warnings not in JSON output")
	}
	if len(dw) != 1 {
		t.Errorf("drift_warnings len = %d, want 1", len(dw))
	}

	summary := parsed["summary"].(map[string]any)
	if summary["drift_warnings_count"].(float64) != 1 {
		t.Errorf("drift_warnings_count = %v, want 1", summary["drift_warnings_count"])
	}
}

func TestBatchValidationResult_NoDriftIsEmptyArray(t *testing.T) {
	batch := NewBatchValidationResult(nil)

	data, err := batch.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	dw, ok := parsed["drift_warnings"].([]any)
	if !ok {
		t.Fatal("drift_warnings should be empty array, not null")
	}
	if len(dw) != 0 {
		t.Errorf("drift_warnings len = %d, want 0", len(dw))
	}
}

func TestBatchValidationResult_ToJSON(t *testing.T) {
	results := []*ValidationResult{
		NewValidationResult("a.md", nil),
		NewValidationResult("b.md", []ValidationError{
			{Rule: "required", Field: "x", Message: "m", Source: "s"},
		}),
	}
	batch := NewBatchValidationResult(results)

	data, err := batch.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if parsed["version"].(float64) != 1 {
		t.Errorf("version = %v", parsed["version"])
	}
	if parsed["kind"] != "rootline/validate-batch" {
		t.Errorf("kind = %v", parsed["kind"])
	}

	resultsArr := parsed["results"].([]any)
	if len(resultsArr) != 2 {
		t.Errorf("results length = %d", len(resultsArr))
	}

	summary := parsed["summary"].(map[string]any)
	if summary["total"].(float64) != 2 {
		t.Errorf("summary.total = %v", summary["total"])
	}
}
