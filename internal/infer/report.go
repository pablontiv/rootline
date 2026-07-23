package infer

import "encoding/json"

// AnalyzeReport is the top-level output of the analyze command.
type AnalyzeReport struct {
	Version     int              `json:"version"`
	Kind        string           `json:"kind"`
	Path        string           `json:"path"`
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
	Type          string   `json:"type"`
	Source        string   `json:"source,omitempty"`
	Field         string   `json:"field,omitempty"`
	Value         string   `json:"value,omitempty"`
	From          string   `json:"from,omitempty"`
	To            string   `json:"to,omitempty"`
	Paths         []string `json:"paths,omitempty"`
	Message       string   `json:"message"`
	RequiresAgent bool     `json:"requires_agent"`
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
			Type:          inf.Type,
			Source:        inf.Source,
			Field:         inf.Field,
			Value:         inf.Value,
			Message:       inf.Message,
			RequiresAgent: agentTypes[inf.Type],
		}
	}

	r.Categories = append(r.Categories, CategoryResult{
		ID:             id,
		Name:           name,
		InferenceCount: len(inferences),
		Inferences:     reportInfs,
	})
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
