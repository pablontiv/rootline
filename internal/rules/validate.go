package rules

import (
	"fmt"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// ValidationError represents a single validation failure with full
// traceability to the .stem file that defined the rule.
type ValidationError struct {
	Rule     string `json:"rule"`
	Field    string `json:"field"`
	Message  string `json:"message"`
	Source   string `json:"source"`
	Severity string `json:"severity"`
}

// Validate checks a Record's frontmatter against the effective StemFile.
// It runs schema-level auto-checks (required, enum) and explicit
// validation rules (non_empty, enum, requires, exists).
// Returns an empty slice for a valid document.
func Validate(record *extract.Record, effective *StemFile) []ValidationError {
	if effective == nil {
		return nil
	}

	var errs []ValidationError

	// Phase 1: Schema auto-checks.
	for name, field := range effective.Schema {
		val, exists := record.Frontmatter[name]

		// Skip if severity is "off"
		if field.Severity == "off" {
			continue
		}

		// required: true → field must exist
		if field.Required && !exists {
			errs = append(errs, ValidationError{
				Rule:     "required",
				Field:    name,
				Message:  fmt.Sprintf("required field %q is missing", name),
				Source:   field.Source,
				Severity: field.Severity,
			})
			continue
		}

		// enum: if values defined and field exists, value must be in list
		if exists && len(field.Values) > 0 {
			if !enumContains(field.Values, val) {
				errs = append(errs, ValidationError{
					Rule:     "enum",
					Field:    name,
					Message:  fmt.Sprintf("value %v is not in allowed values: [%s]", val, strings.Join(field.Values, ", ")),
					Source:   field.Source,
					Severity: field.Severity,
				})
			}
		}
	}

	// Phase 2: Explicit validation rules from validate section.
	for _, rule := range effective.Validate {
		if rule.Severity == "off" {
			continue
		}
		var ruleErrs []ValidationError
		switch rule.Rule {
		case "non_empty":
			ruleErrs = checkNonEmpty(record, rule)
		case "exists":
			ruleErrs = checkExists(record, rule)
		case "requires":
			ruleErrs = checkRequires(record, rule)
		case "enum":
			ruleErrs = checkEnum(record, rule, effective)
		}
		for i := range ruleErrs {
			ruleErrs[i].Severity = rule.Severity
		}
		errs = append(errs, ruleErrs...)
	}

	return errs
}

// checkNonEmpty validates that a field exists and is not an empty string.
func checkNonEmpty(record *extract.Record, rule ValidationRule) []ValidationError {
	val, exists := record.Frontmatter[rule.Field]
	if !exists {
		return []ValidationError{{
			Rule:    "non_empty",
			Field:   rule.Field,
			Message: fmt.Sprintf("field %q does not exist", rule.Field),
			Source:  rule.Source,
		}}
	}
	if str, ok := val.(string); ok && str == "" {
		return []ValidationError{{
			Rule:    "non_empty",
			Field:   rule.Field,
			Message: fmt.Sprintf("field %q must not be empty", rule.Field),
			Source:  rule.Source,
		}}
	}
	return nil
}

// checkExists validates that a field is present in frontmatter.
func checkExists(record *extract.Record, rule ValidationRule) []ValidationError {
	if _, exists := record.Frontmatter[rule.Field]; !exists {
		return []ValidationError{{
			Rule:    "exists",
			Field:   rule.Field,
			Message: fmt.Sprintf("field %q must exist", rule.Field),
			Source:  rule.Source,
		}}
	}
	return nil
}

// checkRequires validates that if a condition matches, listed fields exist.
// Format: { rule: requires, if: { Field: Value }, then: { fields: [f1, f2] } }
func checkRequires(record *extract.Record, rule ValidationRule) []ValidationError {
	// Check if condition matches.
	if !conditionMatches(record.Frontmatter, rule.If) {
		return nil
	}

	// Extract required fields from then.fields.
	fields := extractThenFields(rule.Then)
	if len(fields) == 0 {
		return nil
	}

	var errs []ValidationError
	for _, f := range fields {
		if _, exists := record.Frontmatter[f]; !exists {
			errs = append(errs, ValidationError{
				Rule:    "requires",
				Field:   f,
				Message: fmt.Sprintf("field %q is required when %s", f, formatCondition(rule.If)),
				Source:  rule.Source,
			})
		}
	}
	return errs
}

// checkEnum validates a field value against schema values (explicit rule).
func checkEnum(record *extract.Record, rule ValidationRule, effective *StemFile) []ValidationError {
	field, ok := effective.Schema[rule.Field]
	if !ok || len(field.Values) == 0 {
		return nil
	}
	val, exists := record.Frontmatter[rule.Field]
	if !exists {
		return nil
	}
	if !enumContains(field.Values, val) {
		return []ValidationError{{
			Rule:    "enum",
			Field:   rule.Field,
			Message: fmt.Sprintf("value %v is not in allowed values: [%s]", val, strings.Join(field.Values, ", ")),
			Source:  rule.Source,
		}}
	}
	return nil
}

// conditionMatches checks if all key-value pairs in the condition
// match the frontmatter values.
func conditionMatches(frontmatter map[string]any, condition map[string]any) bool {
	for key, expected := range condition {
		actual, exists := frontmatter[key]
		if !exists {
			return false
		}
		if fmt.Sprintf("%v", actual) != fmt.Sprintf("%v", expected) {
			return false
		}
	}
	return true
}

// extractThenFields extracts the fields list from a then clause.
func extractThenFields(then map[string]any) []string {
	fieldsRaw, ok := then["fields"]
	if !ok {
		return nil
	}
	switch v := fieldsRaw.(type) {
	case []any:
		fields := make([]string, 0, len(v))
		for _, item := range v {
			if s, ok := item.(string); ok {
				fields = append(fields, s)
			}
		}
		return fields
	case []string:
		return v
	}
	return nil
}

// enumContains checks if a value is in the allowed values list.
func enumContains(values []string, val any) bool {
	s := fmt.Sprintf("%v", val)
	for _, v := range values {
		if v == s {
			return true
		}
	}
	return false
}

// formatCondition renders a condition map as a readable string.
func formatCondition(cond map[string]any) string {
	parts := make([]string, 0, len(cond))
	for k, v := range cond {
		parts = append(parts, fmt.Sprintf("%s = %v", k, v))
	}
	return strings.Join(parts, " and ")
}
