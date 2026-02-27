package migrate

import (
	"fmt"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

// ConvertLevelsToMatch converts a v1 .stem file with `levels:` to a v2 .stem
// file with match-based field scoping. The resulting stem has version=2, no
// Levels field, and per-field Match annotations.
func ConvertLevelsToMatch(stem *rules.StemFile) (*rules.StemFile, error) {
	if len(stem.Levels) == 0 {
		return nil, fmt.Errorf("stem has no levels to convert")
	}

	result := &rules.StemFile{
		Path:       stem.Path,
		Version:    2,
		Scope:      stem.Scope,
		Schema:     make(map[string]rules.SchemaField),
		Validate:   stem.Validate,
		Derive:     stem.Derive,
		Aggregate:  stem.Aggregate,
		Links:      stem.Links,
		Structural: stem.Structural,
	}

	// Copy root-level schema fields (no match needed).
	for name, field := range stem.Schema {
		result.Schema[name] = field
	}

	// Track per-level fields to merge across levels.
	// For non-id fields: collect match patterns.
	type fieldInfo struct {
		patterns []string
		field    rules.SchemaField
	}
	perLevel := make(map[string]*fieldInfo)

	// Track sequence id configs per pattern.
	idConfigs := make(map[string]any)

	for _, level := range stem.Levels {
		pattern := normalizeMatchPattern(level.Match)

		for name, field := range level.Schema {
			if name == "id" && field.Type == "sequence" {
				// Sequence id: collect into map-form match.
				idConfigs[pattern] = map[string]any{
					"prefix": field.Prefix,
					"digits": field.Digits,
				}
				continue
			}

			if info, ok := perLevel[name]; ok {
				info.patterns = append(info.patterns, pattern)
				// Keep the more complete definition.
				if len(field.Values) > len(info.field.Values) {
					info.field = field
				}
				// If any level requires it, preserve that.
				if field.Required {
					info.field.Required = true
				}
			} else {
				perLevel[name] = &fieldInfo{
					patterns: []string{pattern},
					field:    field,
				}
			}
		}
	}

	// Check if a per-level field appears at ALL levels — if so, make it root.
	nLevels := len(stem.Levels)
	for name, info := range perLevel {
		if _, exists := result.Schema[name]; exists {
			// Already in root schema — merge match patterns if needed.
			continue
		}

		if len(info.patterns) == nLevels {
			// Present at all levels → root field (no match needed).
			result.Schema[name] = info.field
		} else {
			// Present at some levels → add match restriction.
			field := info.field
			field.Match = &rules.FieldMatch{Patterns: info.patterns}
			result.Schema[name] = field
		}
	}

	// Sequence id with map-form match.
	if len(idConfigs) > 0 {
		result.Schema["id"] = rules.SchemaField{
			Type:  "sequence",
			Match: &rules.FieldMatch{Configs: idConfigs},
		}
	}

	return result, nil
}

// normalizeMatchPattern converts a v1 match pattern (e.g., "E??-*") to a
// simpler v2 glob prefix pattern (e.g., "E*"). If the pattern is already
// simple, returns it as-is.
func normalizeMatchPattern(pattern string) string {
	// v1 patterns are like "E??-*" or "E*". Normalize to "E*" form.
	if len(pattern) >= 2 && pattern[0] >= 'A' && pattern[0] <= 'Z' {
		// Find the prefix letters.
		i := 0
		for i < len(pattern) && pattern[i] >= 'A' && pattern[i] <= 'Z' {
			i++
		}
		if i > 0 && strings.ContainsAny(pattern[i:], "?*") {
			return pattern[:i] + "*"
		}
	}
	return pattern
}

// MarshalStemV2 serializes a v2 StemFile to YAML bytes.
func MarshalStemV2(stem *rules.StemFile) ([]byte, error) {
	// Build an ordered map for clean YAML output.
	out := make(map[string]any)
	out["version"] = stem.Version

	if stem.Scope.Match != "" {
		out["scope"] = map[string]string{"match": stem.Scope.Match}
	}

	if len(stem.Schema) > 0 {
		schema := make(map[string]any)
		for name, field := range stem.Schema {
			schema[name] = schemaFieldToMap(field)
		}
		out["schema"] = schema
	}

	if len(stem.Validate) > 0 {
		out["validate"] = stem.Validate
	}
	if len(stem.Derive) > 0 {
		out["derive"] = stem.Derive
	}
	if len(stem.Aggregate) > 0 {
		out["aggregate"] = stem.Aggregate
	}
	if !stem.Links.IsEmpty() {
		out["links"] = stem.Links
	}
	if !stem.Structural.IsEmpty() {
		out["structural"] = stem.Structural
	}

	return yaml.Marshal(out)
}

// schemaFieldToMap converts a SchemaField to a map for YAML serialization,
// handling the match field specially.
func schemaFieldToMap(f rules.SchemaField) map[string]any {
	m := make(map[string]any)
	m["type"] = f.Type

	if f.Required {
		m["required"] = true
	}
	if len(f.Values) > 0 {
		m["values"] = f.Values
	}
	if f.Default != "" {
		m["default"] = f.Default
	}

	if f.Match != nil {
		switch {
		case len(f.Match.Configs) > 0:
			m["match"] = f.Match.Configs
		case len(f.Match.Patterns) == 1:
			m["match"] = f.Match.Patterns[0]
		case len(f.Match.Patterns) > 1:
			m["match"] = f.Match.Patterns
		}
	}

	if f.Prefix != "" {
		m["prefix"] = f.Prefix
	}
	if f.Digits > 0 {
		m["digits"] = f.Digits
	}

	return m
}
