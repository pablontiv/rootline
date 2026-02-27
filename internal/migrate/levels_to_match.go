package migrate

import (
	"fmt"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

// v1Level represents a single level in a v1 .stem levels: map.
// This is a local type for parsing v1 stems during migration.
type v1Level struct {
	Match    string                       `yaml:"match"`
	Children []string                     `yaml:"children,omitempty"`
	Schema   map[string]rules.SchemaField `yaml:"schema,omitempty"`
	Validate []rules.ValidationRule       `yaml:"validate,omitempty"`
}

// v1StemFile extends parsing to include the levels: field from v1 format.
type v1StemFile struct {
	Version    int                          `yaml:"version"`
	Scope      rules.Scope                  `yaml:"scope"`
	Schema     map[string]rules.SchemaField `yaml:"schema"`
	Validate   []rules.ValidationRule       `yaml:"validate"`
	Derive     map[string]any               `yaml:"derive"`
	Aggregate  map[string]any               `yaml:"aggregate"`
	Links      rules.LinkSchema             `yaml:"links"`
	Structural rules.StructuralRules        `yaml:"structural"`
	Levels     map[string]*v1Level          `yaml:"levels"`
}

// ConvertLevelsToMatch reads a v1 .stem file with `levels:` from raw YAML and
// converts it to a v2 .stem file with match-based field scoping.
func ConvertLevelsToMatch(content []byte, path string) (*rules.StemFile, error) {
	var v1 v1StemFile
	if err := yaml.Unmarshal(content, &v1); err != nil {
		return nil, fmt.Errorf("parsing v1 stem: %w", err)
	}

	if len(v1.Levels) == 0 {
		return nil, fmt.Errorf("stem has no levels to convert")
	}

	result := &rules.StemFile{
		Path:       path,
		Version:    2,
		Scope:      v1.Scope,
		Schema:     make(map[string]rules.SchemaField),
		Validate:   v1.Validate,
		Derive:     v1.Derive,
		Aggregate:  v1.Aggregate,
		Links:      v1.Links,
		Structural: v1.Structural,
	}

	// Copy root-level schema fields (no match needed).
	for name, field := range v1.Schema {
		result.Schema[name] = field
	}

	// Track per-level fields to merge across levels.
	type fieldInfo struct {
		patterns         []string
		requiredPatterns []string // patterns where field is required
		field            rules.SchemaField
	}
	perLevel := make(map[string]*fieldInfo)

	// Track sequence id configs per pattern.
	idConfigs := make(map[string]any)

	for _, level := range v1.Levels {
		pattern := normalizeMatchPattern(level.Match)

		// Migrate per-level validate rules: inject match into if clause.
		for _, rule := range level.Validate {
			migrated := rule
			if migrated.If == nil {
				migrated.If = make(map[string]any)
			}
			migrated.If["match"] = pattern
			result.Validate = append(result.Validate, migrated)
		}

		for name, field := range level.Schema {
			if name == "id" && field.Type == "sequence" {
				idConfigs[pattern] = map[string]any{
					"prefix": field.Prefix,
					"digits": field.Digits,
				}
				continue
			}

			if info, ok := perLevel[name]; ok {
				info.patterns = append(info.patterns, pattern)
				if len(field.Values) > len(info.field.Values) {
					info.field = field
				}
				if field.Required {
					info.requiredPatterns = append(info.requiredPatterns, pattern)
				}
			} else {
				var reqPatterns []string
				if field.Required {
					reqPatterns = []string{pattern}
				}
				perLevel[name] = &fieldInfo{
					patterns:         []string{pattern},
					requiredPatterns: reqPatterns,
					field:            field,
				}
			}
		}
	}

	// Check if a per-level field appears at ALL levels — if so, make it root.
	nLevels := len(v1.Levels)
	for name, info := range perLevel {
		if _, exists := result.Schema[name]; exists {
			continue
		}

		field := info.field

		// Resolve required: if required at all levels → bool true;
		// if required at some levels → RequiredMatch with patterns.
		nReq := len(info.requiredPatterns)
		nPat := len(info.patterns)
		switch nReq {
		case 0:
			field.Required = false
		case nPat:
			field.Required = true
		default:
			field.Required = true
			field.RequiredMatch = &rules.FieldMatch{Patterns: info.requiredPatterns}
		}

		if nPat == nLevels {
			result.Schema[name] = field
		} else {
			field.Match = &rules.FieldMatch{Patterns: info.patterns}
			result.Schema[name] = field
		}
	}

	if len(idConfigs) > 0 {
		result.Schema["id"] = rules.SchemaField{
			Type:  "sequence",
			Match: &rules.FieldMatch{Configs: idConfigs},
		}
	}

	return result, nil
}

// normalizeMatchPattern converts a v1 match pattern (e.g., "E??-*") to a
// simpler v2 glob prefix pattern (e.g., "E*").
func normalizeMatchPattern(pattern string) string {
	if len(pattern) >= 2 && pattern[0] >= 'A' && pattern[0] <= 'Z' {
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

// schemaFieldToMap converts a SchemaField to a map for YAML serialization.
func schemaFieldToMap(f rules.SchemaField) map[string]any {
	m := make(map[string]any)
	m["type"] = f.Type

	if f.RequiredMatch != nil {
		// Object form: required: {match: ["T*"]}
		rm := map[string]any{}
		switch {
		case len(f.RequiredMatch.Patterns) == 1:
			rm["match"] = f.RequiredMatch.Patterns[0]
		case len(f.RequiredMatch.Patterns) > 1:
			rm["match"] = f.RequiredMatch.Patterns
		}
		m["required"] = rm
	} else if f.Required {
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
