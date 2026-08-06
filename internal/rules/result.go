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

// ValidationEnvelopeVersion is the contract version of the validate envelope.
//
// Version 2 separated `.stem` health diagnostics from document results and made
// the envelope the single shape every `validate` invocation emits — one file,
// several files, `--all`, `--staged`, and the corpus-scan failure path alike.
// Version 1 emitted a bare ValidationResult for a single file, folded stem
// health into `results`, and wrote nothing at all for an empty staging area.
const ValidationEnvelopeVersion = 2

// Notice is a command-level diagnostic that belongs to the run rather than to
// any single record: a corpus scan that failed, a scanned tree with no records.
//
// It exists so a future diagnostic (an unusable flag combination, a `--where`
// naming an unknown field) has somewhere to land without reshaping the
// envelope. Consumers switch on Code, which is stable; Message is for humans.
type Notice struct {
	Severity string `json:"severity"` // "error" or "warn"
	Code     string `json:"code"`
	Message  string `json:"message"`
}

// BatchValidationResult is the versioned JSON envelope every `validate`
// invocation emits.
//
// The four collections are disjoint populations, never merged: Results holds
// documents, StemHealth holds schema diagnostics, DriftWarnings holds
// parent/child divergence, Notices holds run-level diagnostics. Every key is
// always present, so a consumer can index into it without probing.
type BatchValidationResult struct {
	Version       int                    `json:"version"`
	Kind          string                 `json:"kind"`
	Results       []*ValidationResult    `json:"results"`
	Structural    []*ValidationResult    `json:"structural"`
	StemHealth    []StemHealthDiagnostic `json:"stem_health"`
	DriftWarnings []DriftWarning         `json:"drift_warnings"`
	Notices       []Notice               `json:"notices"`
	Summary       BatchSummary           `json:"summary"`
}

// BatchSummary holds aggregate counts for batch validation.
//
// Total, Valid and Invalid count documents only, so `summary.total` agrees with
// `query --count`, `tree --field root.total` and `stats --field total` on the
// same path. Schema hygiene is counted on its own axis.
type BatchSummary struct {
	Total                   int `json:"total"`
	Valid                   int `json:"valid"`
	Invalid                 int `json:"invalid"`
	ErrorsCount             int `json:"errors_count"`
	WarningsCount           int `json:"warnings_count"`
	DriftWarningsCount      int `json:"drift_warnings_count"`
	StructuralErrorsCount   int `json:"structural_errors_count"`
	StructuralWarningsCount int `json:"structural_warnings_count"`
	StemHealthErrorsCount   int `json:"stem_health_errors_count"`
	StemHealthWarningsCount int `json:"stem_health_warnings_count"`
	StemHealthInfoCount     int `json:"stem_health_info_count"`
}

// ValidationEnvelopeInput carries the disjoint populations one validate run
// produced. It is a struct rather than a parameter list so a future collection
// can join the envelope without rewriting every call site.
type ValidationEnvelopeInput struct {
	Results       []*ValidationResult    // documents
	Structural    []*ValidationResult    // directories checked against structural rules
	StemHealth    []StemHealthDiagnostic // .stem files
	DriftWarnings []DriftWarning         // parent/child divergence
	Notices       []Notice               // the run itself
}

// NewBatchValidationResult creates an envelope from document results alone.
func NewBatchValidationResult(results []*ValidationResult) *BatchValidationResult {
	return NewValidationEnvelope(ValidationEnvelopeInput{Results: results})
}

// NewBatchValidationResultWithDrift creates an envelope with drift warnings.
func NewBatchValidationResultWithDrift(results []*ValidationResult, driftWarnings []DriftWarning) *BatchValidationResult {
	return NewValidationEnvelope(ValidationEnvelopeInput{Results: results, DriftWarnings: driftWarnings})
}

// NewValidationEnvelope builds the full envelope and derives every summary
// count from the collections, so the counts cannot drift from the payload.
func NewValidationEnvelope(in ValidationEnvelopeInput) *BatchValidationResult {
	summary := BatchSummary{Total: len(in.Results)}
	for _, r := range in.Results {
		if r.Valid {
			summary.Valid++
		} else {
			summary.Invalid++
		}
		summary.ErrorsCount += len(r.Errors)
		summary.WarningsCount += len(r.Warnings)
	}
	for _, r := range in.Structural {
		summary.StructuralErrorsCount += len(r.Errors)
		summary.StructuralWarningsCount += len(r.Warnings)
	}
	for _, d := range in.StemHealth {
		switch d.Severity {
		case SeverityError:
			summary.StemHealthErrorsCount++
		case SeverityInfo:
			summary.StemHealthInfoCount++
		default:
			summary.StemHealthWarningsCount++
		}
	}
	results := in.Results
	if results == nil {
		results = []*ValidationResult{}
	}
	structural := in.Structural
	if structural == nil {
		structural = []*ValidationResult{}
	}
	driftWarnings := in.DriftWarnings
	if driftWarnings == nil {
		driftWarnings = []DriftWarning{}
	}
	stemHealth := in.StemHealth
	if stemHealth == nil {
		stemHealth = []StemHealthDiagnostic{}
	}
	notices := in.Notices
	if notices == nil {
		notices = []Notice{}
	}
	summary.DriftWarningsCount = len(driftWarnings)
	return &BatchValidationResult{
		Version:       ValidationEnvelopeVersion,
		Kind:          "rootline/validate-batch",
		Results:       results,
		Structural:    structural,
		StemHealth:    stemHealth,
		DriftWarnings: driftWarnings,
		Notices:       notices,
		Summary:       summary,
	}
}

// HasErrorNotice reports whether any run-level notice is an error.
func (r *BatchValidationResult) HasErrorNotice() bool {
	for _, n := range r.Notices {
		if n.Severity == SeverityError {
			return true
		}
	}
	return false
}

// ToJSON serializes the batch result to stable JSON.
func (r *BatchValidationResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
