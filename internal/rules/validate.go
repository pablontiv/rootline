package rules

import (
	"context"
	"fmt"
	"path/filepath"
	"regexp"
	"strings"

	"github.com/pablontiv/picokit/fuzzy"
	"github.com/pablontiv/rootline/internal/extract"
)

// ValidationError represents a single validation failure with full
// traceability to the .stem file that defined the rule.
type ValidationError struct {
	Rule       string `json:"rule"`
	Field      string `json:"field"`
	Message    string `json:"message"`
	Source     string `json:"source"`
	Severity   string `json:"severity"`
	Suggestion string `json:"suggestion,omitempty"`
}

// resolveFieldValue resolves a field value using frontmatter-first precedence.
// Priority:
// 1. If the field exists in record.Frontmatter, return that value.
// 2. If the field has an Extract directive, try to extract from the body.
// 3. Otherwise, return (nil, false).
//
// This ensures frontmatter always takes precedence over body extraction,
// implementing the spec's "Frontmatter Precedence Over Body Extraction" requirement.
func resolveFieldValue(record *extract.Record, name string, field SchemaField) (any, bool) {
	// Frontmatter takes precedence (highest priority).
	if v, ok := record.Frontmatter[name]; ok {
		return v, true
	}

	// If no frontmatter value and field has an Extract directive, try body extraction.
	if field.Extract != "" {
		bodyVal, exists, err := extract.ResolveBodyValue(record, field.Extract)
		if err == nil && exists && bodyVal != "" {
			return bodyVal, true
		}
	}

	// Field does not exist.
	return nil, false
}

// Validate checks a Record's frontmatter against the effective StemFile.
// It runs schema-level auto-checks (required, enum) and explicit
// validation rules (non_empty, enum, requires, exists).
// Returns an empty slice for a valid document.
func Validate(_ context.Context, record *extract.Record, effective *StemFile) []ValidationError {
	if effective == nil {
		return nil
	}

	var errs []ValidationError

	// Phase 1: Schema auto-checks.
	for name, field := range effective.Schema {
		val, exists := resolveFieldValue(record, name, field)

		// Skip if severity is "off"
		if field.Severity == "off" {
			continue
		}

		// required: true → field must exist
		if field.Required && !exists {
			// Layer 1: Skip for aggregate-computed fields on index files.
			if _, hasAggregate := effective.Aggregate[name]; hasAggregate && IsIndexFile(record.Path, effective) {
				continue
			}
			// Layer 2: Skip if excludes.match pattern matches the record path.
			if field.Excludes != nil && field.Excludes.Match != "" {
				if matched, _ := filepath.Match(field.Excludes.Match, record.Path); matched {
					continue
				}
			}
			msg := fmt.Sprintf("required field %q is missing", name)
			suggestion := ""
			fmKeys := make([]string, 0, len(record.Frontmatter))
			for k := range record.Frontmatter {
				fmKeys = append(fmKeys, k)
			}
			if match := fuzzy.Match(name, fmKeys); match != "" {
				suggestion = match
				msg += fmt.Sprintf(" (found similar: %q)", match)
			}
			errs = append(errs, ValidationError{
				Rule:       "required",
				Field:      name,
				Message:    msg,
				Source:     field.Source,
				Severity:   field.Severity,
				Suggestion: suggestion,
			})
			continue
		}

		// enum: if values defined and field exists, value must be in list
		if exists && len(field.Values) > 0 {
			if !enumContains(field.Values, val) {
				valStr := fmt.Sprintf("%v", val)
				msg := fmt.Sprintf("value %v is not in allowed values: [%s]", val, strings.Join(field.Values, ", "))
				suggestion := fuzzy.Match(valStr, field.Values)
				if suggestion != "" && suggestion != valStr {
					msg += fmt.Sprintf(" (did you mean %q?)", suggestion)
				}
				errs = append(errs, ValidationError{
					Rule:       "enum",
					Field:      name,
					Message:    msg,
					Source:     field.Source,
					Severity:   field.Severity,
					Suggestion: suggestion,
				})
			}
		}

		// link: if type is "link" and field exists, value must contain [[target]] syntax
		if exists && field.Type == "link" {
			valid := false
			switch v := val.(type) {
			case string:
				valid = extract.ContainsWikilink(v)
			case []any:
				valid = true
				for _, item := range v {
					s, ok := item.(string)
					if !ok || !extract.ContainsWikilink(s) {
						valid = false
						break
					}
				}
			}
			if !valid {
				sev := field.Severity
				if sev == "" {
					sev = "warn"
				}
				errs = append(errs, ValidationError{
					Rule:     "link-format",
					Field:    name,
					Message:  fmt.Sprintf("field %q should contain wiki-link syntax [[target]]", name),
					Source:   "schema",
					Severity: sev,
				})
			}
		}
	}

	// Phase 1b: Aggregate consistency — if a field has an aggregate expression
	// and this is an index file, the frontmatter value must match the derived value.
	for name := range effective.Aggregate {
		if !IsIndexFile(record.Path, effective) {
			continue
		}
		fmVal, hasFM := record.Frontmatter[name]
		derivedVal, hasDerived := record.Derived[name]
		if !hasFM || !hasDerived {
			continue
		}
		fmStr := fmt.Sprintf("%v", fmVal)
		derivedStr := fmt.Sprintf("%v", derivedVal)
		if fmStr != derivedStr {
			severity := "error"
			if field, ok := effective.Schema[name]; ok && field.Severity != "" {
				severity = field.Severity
			}
			errs = append(errs, ValidationError{
				Rule:     "aggregate",
				Field:    name,
				Message:  fmt.Sprintf("field %q is %q but aggregate computes %q", name, fmStr, derivedStr),
				Source:   effective.Path,
				Severity: severity,
			})
		}
	}

	// Phase 2: Link validation against LinkSchema.
	errs = append(errs, validateLinks(record.Links, effective.Links, effective.Path)...)

	// Phase 3: Explicit validation rules from validate section.
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
// Uses EffectiveField to consider derived values.
func checkNonEmpty(record *extract.Record, rule ValidationRule) []ValidationError {
	val, exists := record.EffectiveField(rule.Field)
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

// checkExists validates that a field is present (in frontmatter or derived).
func checkExists(record *extract.Record, rule ValidationRule) []ValidationError {
	if _, exists := record.EffectiveField(rule.Field); !exists {
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
// Uses EffectiveField to consider derived values in both condition and field checks.
func checkRequires(record *extract.Record, rule ValidationRule) []ValidationError {
	// Check if condition matches (using effective fields: derived + frontmatter).
	if !conditionMatchesRecord(record, rule.If) {
		return nil
	}

	// Extract required fields from then.fields.
	fields := extractThenFields(rule.Then)
	if len(fields) == 0 {
		return nil
	}

	var errs []ValidationError
	for _, f := range fields {
		if _, exists := record.EffectiveField(f); !exists {
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
		valStr := fmt.Sprintf("%v", val)
		msg := fmt.Sprintf("value %v is not in allowed values: [%s]", val, strings.Join(field.Values, ", "))
		suggestion := fuzzy.Match(valStr, field.Values)
		if suggestion != "" && suggestion != valStr {
			msg += fmt.Sprintf(" (did you mean %q?)", suggestion)
		}
		return []ValidationError{{
			Rule:       "enum",
			Field:      rule.Field,
			Message:    msg,
			Source:     rule.Source,
			Suggestion: suggestion,
		}}
	}
	return nil
}

// conditionMatchesRecord checks if all key-value pairs in the condition
// match the record's effective field values (derived fields take precedence
// over frontmatter).
//
// The key "match" is treated specially: its value is a glob pattern matched
// against the directory name of the record (e.g., "T*" matches "T001-name").
func conditionMatchesRecord(record *extract.Record, condition map[string]any) bool {
	for key, expected := range condition {
		if key == "match" {
			pattern := fmt.Sprintf("%v", expected)
			components := pathComponents(record.Path)
			found := false
			for _, comp := range components {
				if m, _ := filepath.Match(pattern, comp); m {
					found = true
					break
				}
			}
			if !found {
				return false
			}
			continue
		}
		actual, exists := record.EffectiveField(key)
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

// validateLinks checks record links against the effective LinkSchema.
// If the schema has no link constraints, all links are allowed (permissive).
func validateLinks(links []extract.Link, schema LinkSchema, source string) []ValidationError {
	if schema.IsEmpty() {
		return nil
	}

	styles := make(map[string]bool)
	for _, s := range schema.EffectiveStyles() {
		styles[s] = true
	}

	var errs []ValidationError
	for _, link := range links {
		if !styles[linkStyle(link)] {
			continue
		}
		// Check allowed types.
		if len(schema.Allowed) > 0 && !stringSliceContains(schema.Allowed, link.Type) {
			errs = append(errs, ValidationError{
				Rule:     "link_type",
				Field:    "links",
				Message:  fmt.Sprintf("link type %q not allowed (allowed: [%s])", link.Type, strings.Join(schema.Allowed, ", ")),
				Source:   source,
				Severity: "error",
			})
		}

		// Check target pattern (regexp).
		if rule, ok := schema.Rules[link.Type]; ok && rule.Target != "" {
			matched, err := regexp.MatchString(rule.Target, link.Target)
			if err != nil {
				errs = append(errs, ValidationError{
					Rule:     "link_target",
					Field:    "links",
					Message:  fmt.Sprintf("invalid target regexp %q: %v", rule.Target, err),
					Source:   source,
					Severity: "error",
				})
			} else if !matched {
				errs = append(errs, ValidationError{
					Rule:     "link_target",
					Field:    "links",
					Message:  fmt.Sprintf("link target %q does not match pattern %q", link.Target, rule.Target),
					Source:   source,
					Severity: "error",
				})
			}
		}
	}
	return errs
}

// stringSliceContains checks if a string is in a slice.
func stringSliceContains(slice []string, s string) bool {
	for _, v := range slice {
		if v == s {
			return true
		}
	}
	return false
}

// linkStyle returns the link's style, defaulting legacy style-less links to wikilink.
func linkStyle(l extract.Link) string {
	if l.Style == "" {
		return extract.StyleWikilink
	}
	return l.Style
}

// IsIndexFile reports whether a record path is a directory index file.
// It checks structural.subdirs.require_index (default "README.md").
func IsIndexFile(path string, stem *StemFile) bool {
	indexName := "README.md"
	if stem != nil && stem.Structural.Subdirs.RequireIndex != "" {
		indexName = stem.Structural.Subdirs.RequireIndex
	}
	return filepath.Base(path) == indexName
}

// ValidateStructure checks structural integrity rules that don't require a .stem file.
// It validates that the leading frontmatter block holds exactly one YAML document.
//
// Only that leading block is inspected. Thematic breaks and fenced code blocks in
// the Markdown body are ordinary content and are never read as YAML document
// separators. An unterminated leading block is left to the extractor, which
// already reports it as a malformed-frontmatter error.
func ValidateStructure(content []byte, path string) []ValidationError {
	text := strings.TrimPrefix(string(content), "\xef\xbb\xbf")

	start, end, _, ok := extract.FrontmatterBounds(text)
	if !ok {
		return nil
	}

	count := countYAMLDocuments(text[start:end])
	if count <= 1 {
		return nil
	}

	return []ValidationError{{
		Rule:     "multiple_yaml_documents",
		Field:    "_document",
		Message:  fmt.Sprintf("frontmatter contains %d YAML documents; expected exactly 1", count),
		Source:   path,
		Severity: "error",
	}}
}

// countYAMLDocuments reports how many YAML documents a frontmatter region holds.
// A document marker at column 0 ("---" or "...") starts another document only
// when further non-blank content follows it; a region merely terminated by "..."
// is still a single document.
func countYAMLDocuments(region string) int {
	count := 1
	marked := false
	for _, raw := range strings.Split(region, "\n") {
		line := strings.TrimSuffix(raw, "\r")
		switch {
		case line == "---" || line == "...":
			marked = true
		case marked && strings.TrimSpace(line) != "":
			count++
			marked = false
		}
	}
	return count
}

// formatCondition renders a condition map as a readable string.
func formatCondition(cond map[string]any) string {
	parts := make([]string, 0, len(cond))
	for k, v := range cond {
		parts = append(parts, fmt.Sprintf("%s = %v", k, v))
	}
	return strings.Join(parts, " and ")
}
