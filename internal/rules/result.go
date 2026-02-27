package rules

import "encoding/json"

// ValidationResult is the versioned JSON output for a single file validation.
type ValidationResult struct {
	Version  int               `json:"version"`
	Kind     string            `json:"kind"`
	Path     string            `json:"path"`
	Valid    bool              `json:"valid"`
	Errors   []ValidationError `json:"errors"`
	Warnings []ValidationError `json:"warnings"`
}

// NewValidationResult creates a ValidationResult from a path and errors.
// Errors with severity "warn" are separated into Warnings.
func NewValidationResult(path string, errs []ValidationError) *ValidationResult {
	var errors, warnings []ValidationError
	for _, e := range errs {
		if e.Severity == "warn" {
			warnings = append(warnings, e)
		} else {
			errors = append(errors, e)
		}
	}
	if errors == nil {
		errors = []ValidationError{}
	}
	if warnings == nil {
		warnings = []ValidationError{}
	}
	return &ValidationResult{
		Version:  1,
		Kind:     "rootline/validate",
		Path:     path,
		Valid:    len(errors) == 0,
		Errors:   errors,
		Warnings: warnings,
	}
}

// ToJSON serializes the result to stable JSON.
func (r *ValidationResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}

// BatchValidationResult is the versioned JSON output for multi-file validation.
type BatchValidationResult struct {
	Version       int                 `json:"version"`
	Kind          string              `json:"kind"`
	Results       []*ValidationResult `json:"results"`
	DriftWarnings []DriftWarning      `json:"drift_warnings"`
	Summary       BatchSummary        `json:"summary"`
}

// BatchSummary holds aggregate counts for batch validation.
type BatchSummary struct {
	Total              int `json:"total"`
	Valid              int `json:"valid"`
	Invalid            int `json:"invalid"`
	ErrorsCount        int `json:"errors_count"`
	WarningsCount      int `json:"warnings_count"`
	DriftWarningsCount int `json:"drift_warnings_count"`
}

// NewBatchValidationResult creates a BatchValidationResult from individual results.
func NewBatchValidationResult(results []*ValidationResult) *BatchValidationResult {
	return NewBatchValidationResultWithDrift(results, nil)
}

// NewBatchValidationResultWithDrift creates a BatchValidationResult with drift warnings.
func NewBatchValidationResultWithDrift(results []*ValidationResult, driftWarnings []DriftWarning) *BatchValidationResult {
	summary := BatchSummary{Total: len(results)}
	for _, r := range results {
		if r.Valid {
			summary.Valid++
		} else {
			summary.Invalid++
		}
		summary.ErrorsCount += len(r.Errors)
		summary.WarningsCount += len(r.Warnings)
	}
	if driftWarnings == nil {
		driftWarnings = []DriftWarning{}
	}
	summary.DriftWarningsCount = len(driftWarnings)
	return &BatchValidationResult{
		Version:       1,
		Kind:          "rootline/validate-batch",
		Results:       results,
		DriftWarnings: driftWarnings,
		Summary:       summary,
	}
}

// ToJSON serializes the batch result to stable JSON.
func (r *BatchValidationResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
