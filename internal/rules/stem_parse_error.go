package rules

import "fmt"

const (
	stemParseCheckYAMLValid   = "yaml-valid"
	stemParseCheckSchemaValid = "schema-valid"
)

// stemParseDiagnostic carries stem-health projection metadata for ParseStem
// failures while preserving contextual Error() output for direct consumers.
type stemParseDiagnostic struct {
	Check  string
	Field  string
	Path   string
	Reason string
}

// stemParseError classifies parsing failures with diagnostic metadata.
type stemParseError struct {
	diagnostic stemParseDiagnostic
	cause      error
}

func newStemParseError(path, check, field, reason string, cause error) *stemParseError {
	return &stemParseError{
		diagnostic: stemParseDiagnostic{
			Check:  check,
			Field:  field,
			Path:   path,
			Reason: reason,
		},
		cause: cause,
	}
}

func (e *stemParseError) Error() string {
	if e == nil {
		return ""
	}
	if e.diagnostic.Path == "" {
		return e.diagnostic.Reason
	}
	return fmt.Sprintf("parsing %s: %s", e.diagnostic.Path, e.diagnostic.Reason)
}

func (e *stemParseError) Unwrap() error {
	if e == nil {
		return nil
	}
	return e.cause
}

func (e *stemParseError) Diagnostic() stemParseDiagnostic {
	if e == nil {
		return stemParseDiagnostic{}
	}
	return e.diagnostic
}
