// Package migrate implements schema diff logic for .stem files.
//
// It compares two versions of a StemFile and produces a DiffResult
// that classifies each change as breaking or non-breaking.
package migrate

import (
	"sort"

	"github.com/pablontiv/rootline/internal/rules"
)

// ChangeKind classifies a schema change.
type ChangeKind string

const (
	FieldAdded       ChangeKind = "field_added"
	FieldRemoved     ChangeKind = "field_removed"
	EnumValueAdded   ChangeKind = "enum_value_added"
	EnumValueRemoved ChangeKind = "enum_value_removed"
	TypeChanged      ChangeKind = "type_changed"
	RequiredAdded    ChangeKind = "required_added"
	RequiredRemoved  ChangeKind = "required_removed"
	DefaultChanged   ChangeKind = "default_changed"
	SeverityChanged  ChangeKind = "severity_changed"
	RuleAdded        ChangeKind = "rule_added"
	RuleRemoved      ChangeKind = "rule_removed"
)

// Change represents a single schema change between two .stem versions.
type Change struct {
	Kind          ChangeKind `json:"kind"`
	Field         string     `json:"field"`
	Breaking      bool       `json:"breaking"`
	Before        string     `json:"before,omitempty"`
	After         string     `json:"after,omitempty"`
	Message       string     `json:"message"`
	AffectedFiles int        `json:"affected_files"`
}

// DiffResult holds the complete diff between two .stem files.
type DiffResult struct {
	Version       int      `json:"version"`
	Kind          string   `json:"kind"`
	StemPath      string   `json:"stem_path"`
	Changes       []Change `json:"changes"`
	BreakingCount int      `json:"breaking_count"`
	TotalCount    int      `json:"total_count"`
}

// Diff compares before and after StemFile versions and returns classified changes.
// Either before or after may be nil (representing a new or deleted .stem).
func Diff(stemPath string, before, after *rules.StemFile) *DiffResult {
	result := &DiffResult{
		Version:  1,
		Kind:     "rootline/migrate-diff",
		StemPath: stemPath,
	}

	beforeSchema := schemaOrEmpty(before)
	afterSchema := schemaOrEmpty(after)

	// Detect field additions and removals.
	for name := range afterSchema {
		if _, existed := beforeSchema[name]; !existed {
			result.addChange(Change{
				Kind:     FieldAdded,
				Field:    name,
				Breaking: false,
				After:    afterSchema[name].Type,
				Message:  "field " + name + " added (type: " + afterSchema[name].Type + ")",
			})
		}
	}
	for name := range beforeSchema {
		if _, exists := afterSchema[name]; !exists {
			result.addChange(Change{
				Kind:     FieldRemoved,
				Field:    name,
				Breaking: true,
				Before:   beforeSchema[name].Type,
				Message:  "field " + name + " removed",
			})
		}
	}

	// Detect changes in common fields.
	for name, afterField := range afterSchema {
		beforeField, existed := beforeSchema[name]
		if !existed {
			continue
		}
		diffSchemaField(result, name, beforeField, afterField)
	}

	// Detect validation rule changes.
	beforeRules := rulesOrEmpty(before)
	afterRules := rulesOrEmpty(after)
	diffValidationRules(result, beforeRules, afterRules)

	// Sort changes by field name for deterministic output.
	sort.Slice(result.Changes, func(i, j int) bool {
		if result.Changes[i].Field != result.Changes[j].Field {
			return result.Changes[i].Field < result.Changes[j].Field
		}
		return result.Changes[i].Kind < result.Changes[j].Kind
	})

	return result
}

func (r *DiffResult) addChange(c Change) {
	r.Changes = append(r.Changes, c)
	r.TotalCount++
	if c.Breaking {
		r.BreakingCount++
	}
}

func diffSchemaField(result *DiffResult, name string, before, after rules.SchemaField) {
	// Type changed.
	if before.Type != after.Type {
		result.addChange(Change{
			Kind:     TypeChanged,
			Field:    name,
			Breaking: true,
			Before:   before.Type,
			After:    after.Type,
			Message:  "field " + name + " type changed from " + before.Type + " to " + after.Type,
		})
	}

	// Required changed.
	if !before.Required && after.Required {
		result.addChange(Change{
			Kind:     RequiredAdded,
			Field:    name,
			Breaking: true,
			Before:   "false",
			After:    "true",
			Message:  "field " + name + " is now required",
		})
	}
	if before.Required && !after.Required {
		result.addChange(Change{
			Kind:     RequiredRemoved,
			Field:    name,
			Breaking: false,
			Before:   "true",
			After:    "false",
			Message:  "field " + name + " is no longer required",
		})
	}

	// Enum value changes (only for enum type fields).
	if before.Type == "enum" && after.Type == "enum" {
		diffEnumValues(result, name, before.Values, after.Values)
	}

	// Default changed.
	if before.Default != after.Default {
		result.addChange(Change{
			Kind:     DefaultChanged,
			Field:    name,
			Breaking: false,
			Before:   before.Default,
			After:    after.Default,
			Message:  "field " + name + " default changed",
		})
	}

	// Severity changed.
	if before.Severity != after.Severity {
		result.addChange(Change{
			Kind:     SeverityChanged,
			Field:    name,
			Breaking: false,
			Before:   before.Severity,
			After:    after.Severity,
			Message:  "field " + name + " severity changed from " + before.Severity + " to " + after.Severity,
		})
	}
}

func diffEnumValues(result *DiffResult, field string, before, after []string) {
	beforeSet := make(map[string]bool, len(before))
	for _, v := range before {
		beforeSet[v] = true
	}
	afterSet := make(map[string]bool, len(after))
	for _, v := range after {
		afterSet[v] = true
	}

	for _, v := range after {
		if !beforeSet[v] {
			result.addChange(Change{
				Kind:     EnumValueAdded,
				Field:    field,
				Breaking: false,
				After:    v,
				Message:  "enum value " + v + " added to " + field,
			})
		}
	}
	for _, v := range before {
		if !afterSet[v] {
			result.addChange(Change{
				Kind:     EnumValueRemoved,
				Field:    field,
				Breaking: true,
				Before:   v,
				Message:  "enum value " + v + " removed from " + field,
			})
		}
	}
}

func diffValidationRules(result *DiffResult, before, after []rules.ValidationRule) {
	// Simple comparison: serialize each rule to a key and compare sets.
	beforeKeys := make(map[string]bool, len(before))
	for _, r := range before {
		beforeKeys[ruleKey(r)] = true
	}
	afterKeys := make(map[string]bool, len(after))
	for _, r := range after {
		afterKeys[ruleKey(r)] = true
	}

	for _, r := range after {
		if !beforeKeys[ruleKey(r)] {
			result.addChange(Change{
				Kind:     RuleAdded,
				Field:    r.Field,
				Breaking: false,
				After:    r.Rule,
				Message:  "validation rule added: " + r.Rule,
			})
		}
	}
	for _, r := range before {
		if !afterKeys[ruleKey(r)] {
			result.addChange(Change{
				Kind:     RuleRemoved,
				Field:    r.Field,
				Breaking: true,
				Before:   r.Rule,
				Message:  "validation rule removed: " + r.Rule,
			})
		}
	}
}

func ruleKey(r rules.ValidationRule) string {
	// Build a simple unique key for comparison.
	key := r.Rule + "|" + r.Field
	if r.Severity != "" {
		key += "|" + r.Severity
	}
	return key
}

func schemaOrEmpty(s *rules.StemFile) map[string]rules.SchemaField {
	if s == nil || s.Schema == nil {
		return make(map[string]rules.SchemaField)
	}
	return s.Schema
}

func rulesOrEmpty(s *rules.StemFile) []rules.ValidationRule {
	if s == nil {
		return nil
	}
	return s.Validate
}
