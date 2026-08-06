package infer

import (
	"encoding/json"
	"slices"
	"testing"
)

func TestAnalyzeReport_Roundtrip(t *testing.T) {
	report := NewAnalyzeReport("/some/path")

	inferences := []Inference{
		{Type: "required_section", Field: "Contexto", Value: "1.00", Message: "required"},
		{Type: "informal_dependency_candidate", Source: "task.md", Value: "F01 needed", Message: "informal"},
	}

	agentTypes := map[string]bool{
		"informal_dependency_candidate": true,
		"unverified_traceability":       true,
	}

	report.AddCategory("body_sections", "Body Section Patterns", inferences[:1], agentTypes)
	report.AddCategory("formal_deps", "Formal Dependencies", inferences[1:], agentTypes)
	report.Finalize()

	// Marshal.
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	// Unmarshal.
	var decoded AnalyzeReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Version != 1 {
		t.Errorf("expected version 1, got %d", decoded.Version)
	}
	if decoded.Kind != "rootline/analyze" {
		t.Errorf("expected kind 'rootline/analyze', got %s", decoded.Kind)
	}
	if decoded.Path != "/some/path" {
		t.Errorf("expected path '/some/path', got %s", decoded.Path)
	}
	if len(decoded.Categories) != 2 {
		t.Fatalf("expected 2 categories, got %d", len(decoded.Categories))
	}

	// Check RequiresAgent flag.
	if decoded.Categories[0].Inferences[0].RequiresAgent {
		t.Error("required_section should not require agent")
	}
	if !decoded.Categories[1].Inferences[0].RequiresAgent {
		t.Error("informal_dependency_candidate should require agent")
	}

	// Check summary.
	if decoded.Summary.TotalInferences != 2 {
		t.Errorf("expected total 2, got %d", decoded.Summary.TotalInferences)
	}
	if decoded.Summary.AgentRequired != 1 {
		t.Errorf("expected 1 agent required, got %d", decoded.Summary.AgentRequired)
	}
	if decoded.Summary.EngineResolved != 1 {
		t.Errorf("expected 1 engine resolved, got %d", decoded.Summary.EngineResolved)
	}
}

func TestAnalyzeReport_EmptyCategories(t *testing.T) {
	report := NewAnalyzeReport("/empty")
	report.Finalize()

	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal error: %v", err)
	}

	var decoded AnalyzeReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal error: %v", err)
	}

	if decoded.Summary.TotalInferences != 0 {
		t.Errorf("expected 0 total, got %d", decoded.Summary.TotalInferences)
	}
}

func TestAnalyzeReport_AddCategoryUsesDeterministicInferenceOrder(t *testing.T) {
	t.Parallel()

	ascending := []Inference{
		{Type: "field_type", Field: "owner", Value: "enum", Message: "owner"},
		{Type: "field_type", Field: "priority", Value: "enum", Message: "priority"},
		{Type: "field_type", Field: "status", Value: "enum", Message: "status"},
		{Type: "field_type", Field: "title", Value: "enum", Message: "title"},
	}
	reversed := slices.Clone(ascending)
	slices.Reverse(reversed)

	build := func(inferences []Inference) []byte {
		report := NewAnalyzeReport("/stable")
		report.AddCategory("field_types", "Field Type Inference", inferences, nil)
		data, err := json.Marshal(report)
		if err != nil {
			t.Fatalf("marshal report: %v", err)
		}
		return data
	}

	want := build(ascending)
	got := build(reversed)
	if string(got) != string(want) {
		t.Fatalf("permuted category input changed JSON\n got: %s\nwant: %s", got, want)
	}
}

func TestCompareReportInferencesUsesEveryContractField(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name string
		a    ReportInference
		b    ReportInference
	}{
		{name: "type", a: ReportInference{Type: "a"}, b: ReportInference{Type: "b"}},
		{name: "source", a: ReportInference{Type: "same", Source: "a"}, b: ReportInference{Type: "same", Source: "b"}},
		{name: "field", a: ReportInference{Type: "same", Source: "same", Field: "a"}, b: ReportInference{Type: "same", Source: "same", Field: "b"}},
		{name: "value", a: ReportInference{Type: "same", Field: "same", Value: "a"}, b: ReportInference{Type: "same", Field: "same", Value: "b"}},
		{name: "from", a: ReportInference{Type: "same", Value: "same", From: "a"}, b: ReportInference{Type: "same", Value: "same", From: "b"}},
		{name: "to", a: ReportInference{Type: "same", From: "same", To: "a"}, b: ReportInference{Type: "same", From: "same", To: "b"}},
		{name: "paths element", a: ReportInference{Type: "same", To: "same", Paths: []string{"a"}}, b: ReportInference{Type: "same", To: "same", Paths: []string{"b"}}},
		{name: "paths length", a: ReportInference{Type: "same", Paths: []string{"a"}}, b: ReportInference{Type: "same", Paths: []string{"a", "b"}}},
		{name: "message", a: ReportInference{Type: "same", Message: "a"}, b: ReportInference{Type: "same", Message: "b"}},
		{name: "requires agent", a: ReportInference{Type: "same"}, b: ReportInference{Type: "same", RequiresAgent: true}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			if got := compareReportInferences(tt.a, tt.b); got >= 0 {
				t.Fatalf("compareReportInferences(a, b) = %d, want < 0", got)
			}
			if got := compareReportInferences(tt.b, tt.a); got <= 0 {
				t.Fatalf("compareReportInferences(b, a) = %d, want > 0", got)
			}
		})
	}
}
