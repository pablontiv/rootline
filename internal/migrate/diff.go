// Package migrate implements schema diff logic for .stem files.
//
// It compares two versions of a StemFile and produces a DiffResult
// that classifies each change as breaking or non-breaking.
package migrate

import (
	"encoding/json"
	"reflect"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
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
	SourceChanged    ChangeKind = "source_changed"
	RuleAdded        ChangeKind = "rule_added"
	RuleRemoved      ChangeKind = "rule_removed"
)

const (
	PrefixChanged        ChangeKind = "prefix_changed"
	DigitsChanged        ChangeKind = "digits_changed"
	MatchChanged         ChangeKind = "match_changed"
	RequiredMatchChanged ChangeKind = "required_match_changed"
	ExcludesChanged      ChangeKind = "excludes_changed"
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

	// Source identity changes alter the logical field binding.
	if before.Extract != after.Extract {
		result.addChange(Change{
			Kind:     SourceChanged,
			Field:    name,
			Breaking: true,
			Before:   before.Extract,
			After:    after.Extract,
			Message:  "field " + name + " source changed",
		})
	}

	// Sequence formatting and applicability changes can invalidate existing records.
	if before.Prefix != after.Prefix {
		diffFieldAttribute(result, name, "prefix", PrefixChanged, before.Prefix, after.Prefix)
	}
	if before.Digits != after.Digits {
		diffFieldAttribute(result, name, "digits", DigitsChanged, before.Digits, after.Digits)
	}
	if !reflect.DeepEqual(before.Match, after.Match) {
		diffFieldAttribute(result, name, "match", MatchChanged, before.Match, after.Match)
	}
	if !reflect.DeepEqual(before.RequiredMatch, after.RequiredMatch) {
		diffFieldAttribute(result, name, "required_match", RequiredMatchChanged, before.RequiredMatch, after.RequiredMatch)
	}
	if !reflect.DeepEqual(before.Excludes, after.Excludes) {
		diffFieldAttribute(result, name, "excludes", ExcludesChanged, before.Excludes, after.Excludes)
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

func diffFieldAttribute(result *DiffResult, field, attribute string, kind ChangeKind, before, after any) {
	oldValue := diffValue(map[string]any{attribute: before})
	newValue := diffValue(map[string]any{attribute: after})
	if oldValue == newValue {
		return
	}
	result.addChange(Change{
		Kind:     kind,
		Field:    field,
		Breaking: true,
		Before:   oldValue,
		After:    newValue,
		Message:  "field " + field + " " + attribute + " changed from " + oldValue + " to " + newValue,
	})
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
	// Compare complete declared constraints; cache keys for ordered removals.
	beforeValues := make([]string, len(before))
	beforeKeys := make(map[string]bool, len(before))
	for i, r := range before {
		beforeValues[i] = ruleKey(r)
		beforeKeys[beforeValues[i]] = true
	}
	afterKeys := make(map[string]bool, len(after))
	for _, r := range after {
		key := ruleKey(r)
		afterKeys[key] = true
		if !beforeKeys[key] {
			result.addChange(Change{
				Kind:     RuleAdded,
				Field:    r.Field,
				Breaking: false,
				After:    key,
				Message:  "validation rule added: " + key,
			})
		}
	}
	for i, r := range before {
		if !afterKeys[beforeValues[i]] {
			result.addChange(Change{
				Kind:     RuleRemoved,
				Field:    r.Field,
				Breaking: true,
				Before:   beforeValues[i],
				Message:  "validation rule removed: " + beforeValues[i],
			})
		}
	}
}

func ruleKey(r rules.ValidationRule) string {
	// Source is provenance, not part of the declared constraint.
	return diffValue(map[string]any{
		"rule": r.Rule, "field": r.Field, "severity": r.Severity,
		"if": r.If, "then": r.Then,
	})
}

// diffValue keeps snapshots named and deterministic, including absent values.
func diffValue(value any) string {
	data, err := json.Marshal(value)
	if err != nil {
		// Parsed YAML can contain values JSON cannot represent, such as non-string
		// map keys. YAML's sorted encoding preserves those values without pointer
		// addresses or an empty comparison key.
		data, yamlErr := yaml.Marshal(value)
		if yamlErr != nil {
			panic("migrate diff: cannot encode parsed YAML value: " + yamlErr.Error())
		}
		return strings.TrimSpace(string(data))
	}
	return string(data)
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
