package rules

import "github.com/pablontiv/rootline/internal/extract"

// ExtractionErrors converts a record's non-fatal extraction errors (e.g. malformed
// YAML frontmatter that fell back to the permissive line-by-line parser) into
// blocking ValidationErrors, so `validate` surfaces them instead of swallowing them.
// Returns nil when the record has no extraction errors.
func ExtractionErrors(rec *extract.Record) []ValidationError {
	if len(rec.Errors) == 0 {
		return nil
	}
	out := make([]ValidationError, 0, len(rec.Errors))
	for _, ee := range rec.Errors {
		out = append(out, ValidationError{
			Rule:       "malformed_yaml",
			Field:      "_frontmatter",
			Message:    ee.Message,
			Source:     rec.Path,
			Severity:   "error",
			Suggestion: "quote values containing ':' or other YAML-special characters",
		})
	}
	return out
}
