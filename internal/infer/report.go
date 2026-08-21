package infer

import (
	"cmp"
	"encoding/json"
	"slices"
)

// AnalyzeReport is the top-level output of the analyze command.
type AnalyzeReport struct {
	Version int    `json:"version"`
	Kind    string `json:"kind"`
	Path    string `json:"path"`
	// Root is the absolute scan root. Path alone is whatever the caller typed,
	// so a consumer reading the report from a different working directory has
	// no way to turn a relative Path back into the directory that was scanned.
	Root        string           `json:"root,omitempty"`
	Incremental bool             `json:"incremental,omitempty"`
	Categories  []CategoryResult `json:"categories"`
	Summary     ReportSummary    `json:"summary"`
}

// CategoryResult groups inferences by detector category.
type CategoryResult struct {
	ID             string            `json:"id"`
	Name           string            `json:"name"`
	InferenceCount int               `json:"inference_count"`
	Inferences     []ReportInference `json:"inferences"`
}

// ReportInference extends Inference with analyze-specific fields.
type ReportInference struct {
	Type            string   `json:"type"`
	Source          string   `json:"source,omitempty"`
	Field           string   `json:"field,omitempty"`
	SourceDirective string   `json:"source_directive,omitempty"`
	Value           string   `json:"value,omitempty"`
	From            string   `json:"from,omitempty"`
	To              string   `json:"to,omitempty"`
	Paths           []string `json:"paths,omitempty"`
	Message         string   `json:"message"`
	RequiresAgent   bool     `json:"requires_agent"`
}

// ReportSummary provides aggregate counts for the analyze report.
type ReportSummary struct {
	TotalInferences int `json:"total_inferences"`
	AgentRequired   int `json:"agent_required"`
	EngineResolved  int `json:"engine_resolved"`
}

// NewAnalyzeReport creates a report with version 1 and kind "rootline/analyze".
func NewAnalyzeReport(path string) *AnalyzeReport {
	return &AnalyzeReport{
		Version: 1,
		Kind:    "rootline/analyze",
		Path:    path,
	}
}

// AddCategory adds a category with its inferences to the report.
func (r *AnalyzeReport) AddCategory(id, name string, inferences []Inference, agentTypes map[string]bool) {
	reportInfs := make([]ReportInference, len(inferences))
	for i, inf := range inferences {
		reportInfs[i] = ReportInference{
			Type:            inf.Type,
			Source:          inf.Source,
			Field:           inf.Field,
			SourceDirective: inf.SourceDirective,
			Value:           inf.Value,
			Message:         inf.Message,
			RequiresAgent:   agentTypes[inf.Type],
		}
	}
	slices.SortFunc(reportInfs, compareReportInferences)

	r.Categories = append(r.Categories, CategoryResult{
		ID:             id,
		Name:           name,
		InferenceCount: len(inferences),
		Inferences:     reportInfs,
	})
}

// compareReportInferences defines the stable total order used by the analyze
// JSON contract. Fields are compared in serialized order so later additions
// must be handled deliberately rather than inheriting detector traversal order.
func compareReportInferences(a, b ReportInference) int {
	for _, result := range []int{
		cmp.Compare(a.Type, b.Type),
		cmp.Compare(a.Source, b.Source),
		cmp.Compare(a.Field, b.Field),
		cmp.Compare(a.SourceDirective, b.SourceDirective),
		cmp.Compare(a.Value, b.Value),
		cmp.Compare(a.From, b.From),
		cmp.Compare(a.To, b.To),
		slices.Compare(a.Paths, b.Paths),
		cmp.Compare(a.Message, b.Message),
	} {
		if result != 0 {
			return result
		}
	}
	if a.RequiresAgent == b.RequiresAgent {
		return 0
	}
	if !a.RequiresAgent {
		return -1
	}
	return 1
}

// Finalize computes the summary from all categories.
func (r *AnalyzeReport) Finalize() {
	total := 0
	agentReq := 0
	for _, cat := range r.Categories {
		total += cat.InferenceCount
		for _, inf := range cat.Inferences {
			if inf.RequiresAgent {
				agentReq++
			}
		}
	}
	r.Summary = ReportSummary{
		TotalInferences: total,
		AgentRequired:   agentReq,
		EngineResolved:  total - agentReq,
	}
}

// MarshalJSON implements custom JSON marshaling for AnalyzeReport.
func (r *AnalyzeReport) MarshalJSON() ([]byte, error) {
	type Alias AnalyzeReport
	return json.Marshal(&struct {
		*Alias
	}{
		Alias: (*Alias)(r),
	})
}
