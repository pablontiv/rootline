package rules

import "encoding/json"

// ValidationResult is the versioned JSON output for a single file validation.
type ValidationResult struct {
	Version int               `json:"version"`
	Kind    string            `json:"kind"`
	Path    string            `json:"path"`
	Valid   bool              `json:"valid"`
	Errors  []ValidationError `json:"errors"`
}

// NewValidationResult creates a ValidationResult from a path and errors.
func NewValidationResult(path string, errs []ValidationError) *ValidationResult {
	if errs == nil {
		errs = []ValidationError{}
	}
	return &ValidationResult{
		Version: 1,
		Kind:    "rootline/validate",
		Path:    path,
		Valid:   len(errs) == 0,
		Errors:  errs,
	}
}

// ToJSON serializes the result to stable JSON.
func (r *ValidationResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// BatchValidationResult is the versioned JSON output for multi-file validation.
type BatchValidationResult struct {
	Version int                 `json:"version"`
	Kind    string              `json:"kind"`
	Results []*ValidationResult `json:"results"`
	Summary BatchSummary        `json:"summary"`
}

// BatchSummary holds aggregate counts for batch validation.
type BatchSummary struct {
	Total   int `json:"total"`
	Valid   int `json:"valid"`
	Invalid int `json:"invalid"`
}

// NewBatchValidationResult creates a BatchValidationResult from individual results.
func NewBatchValidationResult(results []*ValidationResult) *BatchValidationResult {
	summary := BatchSummary{Total: len(results)}
	for _, r := range results {
		if r.Valid {
			summary.Valid++
		} else {
			summary.Invalid++
		}
	}
	return &BatchValidationResult{
		Version: 1,
		Kind:    "rootline/validate-batch",
		Results: results,
		Summary: summary,
	}
}

// ToJSON serializes the batch result to stable JSON.
func (r *BatchValidationResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
