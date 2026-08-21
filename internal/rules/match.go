package rules

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"
)

// FilterSchemaByMatch filters schema fields based on a record's path.
// Fields without Match apply everywhere. Fields with Match apply only when
// at least one path component matches a pattern. For map-form match (configs),
// the matching pattern's config is applied to the returned field.
func FilterSchemaByMatch(schema map[string]*SchemaField, recordPath string) (map[string]*SchemaField, error) {
	components := pathComponents(recordPath)
	result := make(map[string]*SchemaField, len(schema))

	for _, name := range sortedPointerSchemaNames(schema) {
		field := schema[name]
		if field.Match == nil {
			// No match restriction — applies everywhere
			resolved := *field
			// Resolve RequiredMatch scoping
			if field.RequiredMatch != nil {
				resolved.Required = matchesAny(field.RequiredMatch, components)
			}
			result[name] = &resolved
			continue
		}

		if field.Match.Configs != nil {
			// Map form: match configs keyed by pattern
			if matched, pattern, config, err := matchConfig(field.Match.Configs, components); err != nil {
				return nil, fmt.Errorf("invalid match config for field %q pattern %q: %w", name, pattern, err)
			} else if matched {
				resolved, err := applyConfig(name, pattern, *field, config)
				if err != nil {
					return nil, err
				}
				// Resolve RequiredMatch scoping
				if field.RequiredMatch != nil {
					resolved.Required = matchesAny(field.RequiredMatch, components)
				}
				result[name] = resolved
			}
			continue
		}

		// List/string form: check if any component matches any pattern
		if matchesAny(field.Match, components) {
			resolved := *field
			// Resolve RequiredMatch scoping
			if field.RequiredMatch != nil {
				resolved.Required = matchesAny(field.RequiredMatch, components)
			}
			result[name] = &resolved
		}
	}

	return result, nil
}

func filterValueSchemaByMatch(schema map[string]SchemaField, recordPath string) (map[string]SchemaField, error) {
	ptrSchema := make(map[string]*SchemaField, len(schema))
	for name, field := range schema {
		field := field
		ptrSchema[name] = &field
	}
	filtered, err := FilterSchemaByMatch(ptrSchema, recordPath)
	if err != nil {
		return nil, err
	}
	result := make(map[string]SchemaField, len(filtered))
	for name, field := range filtered {
		result[name] = *field
	}
	return result, nil
}

// pathComponents splits a path into its directory/file name components.
func pathComponents(recordPath string) []string {
	cleaned := filepath.Clean(recordPath)
	return strings.Split(cleaned, string(filepath.Separator))
}

// matchesAny returns true if any path component matches any pattern in the FieldMatch.
func matchesAny(fm *FieldMatch, components []string) bool {
	for _, pattern := range fm.Patterns {
		for _, comp := range components {
			if matched, _ := filepath.Match(pattern, comp); matched {
				return true
			}
		}
	}
	if fm.Configs != nil {
		for _, pattern := range sortedPatterns(fm.Configs) {
			for _, comp := range components {
				if matched, _ := filepath.Match(pattern, comp); matched {
					return true
				}
			}
		}
	}
	return false
}

// matchConfig finds the deepest matching config pattern and returns it.
// Components are checked from deepest to shallowest so that a record at
// E01/F01/S001/T001-task.md matches "T*" config, not "E*".
func matchConfig(configs map[string]any, components []string) (bool, string, map[string]any, error) {
	patterns := sortedPatterns(configs)
	for i := len(components) - 1; i >= 0; i-- {
		for _, pattern := range patterns {
			config := configs[pattern]
			if matched, _ := filepath.Match(pattern, components[i]); matched {
				cfgMap, ok := config.(map[string]any)
				if !ok || cfgMap == nil {
					return false, pattern, nil, sequenceConfigError{actual: "malformed sequence config", detail: "config must be a map"}
				}
				return true, pattern, cfgMap, nil
			}
		}
	}
	return false, "", nil, nil
}

// applyConfig creates a copy of the field with config values applied.
func applyConfig(fieldName, pattern string, field SchemaField, config map[string]any) (*SchemaField, error) {
	if config == nil {
		return nil, fmt.Errorf("invalid match config for field %q pattern %q: config must be a map", fieldName, pattern)
	}
	if field.Type == "sequence" {
		resolved, err := applySequenceConfig(field, config)
		if err != nil {
			return nil, fmt.Errorf("invalid match config for field %q pattern %q: %w", fieldName, pattern, err)
		}
		return &resolved, nil
	}
	if prefix, ok := config["prefix"]; ok {
		if s, ok := prefix.(string); ok {
			field.Prefix = s
		}
	}
	if digits, ok := config["digits"]; ok {
		if v, ok := strictConfigDigits(digits); ok {
			field.Digits = v
		}
	}
	return &field, nil
}

func sortedPointerSchemaNames(schema map[string]*SchemaField) []string {
	names := make([]string, 0, len(schema))
	for name := range schema {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}
