package infer

import (
	"encoding/json"
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
	if decoded.Kind != "analyze" {
		t.Errorf("expected kind 'analyze', got %s", decoded.Kind)
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
