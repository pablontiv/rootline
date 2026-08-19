package rules

import (
	"path/filepath"
	"strings"
)

// FilterSchemaByMatch filters schema fields based on a record's path.
// Fields without Match apply everywhere. Fields with Match apply only when
// at least one path component matches a pattern. For map-form match (configs),
// the matching pattern's config is applied to the returned field.
func FilterSchemaByMatch(schema map[string]*SchemaField, recordPath string) map[string]*SchemaField {
	components := pathComponents(recordPath)
	result := make(map[string]*SchemaField, len(schema))

	for name, field := range schema {
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
			if matched, config := matchConfig(field.Match.Configs, components); matched {
				resolved := applyConfig(*field, config)
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

	return result
}

func filterValueSchemaByMatch(schema map[string]SchemaField, recordPath string) map[string]SchemaField {
	ptrSchema := make(map[string]*SchemaField, len(schema))
	for name, field := range schema {
		field := field
		ptrSchema[name] = &field
	}
	filtered := FilterSchemaByMatch(ptrSchema, recordPath)
	result := make(map[string]SchemaField, len(filtered))
	for name, field := range filtered {
		result[name] = *field
	}
	return result
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
		for pattern := range fm.Configs {
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
func matchConfig(configs map[string]any, components []string) (bool, map[string]any) {
	for i := len(components) - 1; i >= 0; i-- {
		for pattern, config := range configs {
			if matched, _ := filepath.Match(pattern, components[i]); matched {
				if cfgMap, ok := config.(map[string]any); ok {
					return true, cfgMap
				}
				return true, nil
			}
		}
	}
	return false, nil
}

// applyConfig creates a copy of the field with config values applied.
func applyConfig(field SchemaField, config map[string]any) *SchemaField {
	if config == nil {
		return &field
	}
	if prefix, ok := config["prefix"]; ok {
		if s, ok := prefix.(string); ok {
			field.Prefix = s
		}
	}
	if digits, ok := config["digits"]; ok {
		switch v := digits.(type) {
		case int:
			field.Digits = v
		case float64:
			field.Digits = int(v)
		}
	}
	return &field
}
