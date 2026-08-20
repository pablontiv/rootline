package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strconv"
)

// DescribeResult is the versioned JSON output for the describe command.
// It shows the effective schema for a directory after merging all
// ancestor .stem files.
type DescribeResult struct {
	Version    int                    `json:"version"`
	Kind       string                 `json:"kind"`
	Path       string                 `json:"path"`
	Applies    []string               `json:"applies"`
	Scope      Scope                  `json:"scope"`
	Schema     map[string]SchemaField `json:"schema"`
	Validate   []ValidationRule       `json:"validate"`
	Derive     map[string]any         `json:"derive"`
	Aggregate  map[string]any         `json:"aggregate"`
	Links      LinkSchema             `json:"links"`
	Layers     []string               `json:"layers,omitempty"`
	Provenance map[string]string      `json:"provenance,omitempty"`
	Hints      []string               `json:"hints,omitempty"`
}

// NewDescribeResult builds a DescribeResult from walk-up entries and
// the effective (merged) StemFile for a given path.
func NewDescribeResult(path string, entries []StemEntry, effective *StemFile) (*DescribeResult, error) {
	applies := make([]string, len(entries))
	for i, e := range entries {
		applies[i] = e.Path
	}

	schema := copySchemaFields(effective.Schema)
	validate := append([]ValidationRule(nil), effective.Validate...)
	if validate == nil {
		validate = []ValidationRule{}
	}
	derive := copyAnyMap(effective.Derive)
	aggregate := copyAnyMap(effective.Aggregate)

	// Compute Next and NextByPattern for sequence fields on the staged schema.
	// Sorted names make the first error deterministic and avoid mutating the
	// caller's effective schema if a later field fails.
	for _, name := range sortedDescribeFieldNames(schema) {
		field := schema[name]
		if field.Type == "sequence" {
			next, err := computeNextSequence(path, name, field)
			if err != nil {
				return nil, err
			}
			field.Next = next
			nextByPattern, err := computeAllNextSequences(path, name, field)
			if err != nil {
				return nil, err
			}
			field.NextByPattern = nextByPattern
			schema[name] = field
		}
	}

	// Build layers and provenance from entries
	layers := make([]string, len(entries))
	for i, e := range entries {
		layers[i] = e.Path
	}

	// Build provenance map: field name → stem path that defined it
	provenance := make(map[string]string)
	for _, entry := range entries {
		if entry.Stem != nil {
			for name := range entry.Stem.Schema {
				// Last writer wins — later entries (closer to leaf) override earlier ones.
				provenance[name] = entry.Path
			}
		}
	}

	return &DescribeResult{
		Version:    1,
		Kind:       "rootline/describe",
		Path:       path,
		Applies:    applies,
		Scope:      effective.Scope,
		Schema:     schema,
		Validate:   validate,
		Derive:     derive,
		Aggregate:  aggregate,
		Links:      effective.Links,
		Layers:     layers,
		Provenance: provenance,
	}, nil
}

// computeNextSequence scans dirPath for files/dirs matching the prefix pattern,
// finds the highest numeric suffix, and returns prefix + next number zero-padded
// to the specified digits. Supports both top-level prefix/digits and match configs
// where prefix/digits are nested per-pattern (e.g., "E*": {prefix: E, digits: 2}).
// When multiple match config patterns are present, patterns are evaluated in
// alphabetical order to ensure deterministic results.
func copySchemaFields(in map[string]SchemaField) map[string]SchemaField {
	out := make(map[string]SchemaField, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func copyAnyMap(in map[string]any) map[string]any {
	out := make(map[string]any, len(in))
	for k, v := range in {
		out[k] = v
	}
	return out
}

func sortedDescribeFieldNames(schema map[string]SchemaField) []string {
	names := make([]string, 0, len(schema))
	for name := range schema {
		names = append(names, name)
	}
	sort.Strings(names)
	return names
}

func computeNextSequence(dirPath, fieldName string, field SchemaField) (string, error) {
	if field.Prefix != "" && field.Digits > 0 {
		return computeNextFromPrefix(dirPath, field.Prefix, field.Digits), nil
	}

	if field.Match == nil || field.Match.Configs == nil {
		return "", nil
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return "", nil
	}

	for _, globPattern := range sortedPatterns(field.Match.Configs) {
		config := field.Match.Configs[globPattern]
		cfgMap, ok := config.(map[string]any)
		if !ok || cfgMap == nil {
			return "", fmt.Errorf("invalid match config for field %q pattern %q: config must be a map", fieldName, globPattern)
		}
		resolved, err := applySequenceConfig(field, cfgMap)
		if err != nil {
			return "", fmt.Errorf("invalid match config for field %q pattern %q: %w", fieldName, globPattern, err)
		}

		for _, e := range entries {
			if matched, _ := filepath.Match(globPattern, e.Name()); matched {
				return computeNextFromPrefix(dirPath, resolved.Prefix, resolved.Digits), nil
			}
		}
	}

	return "", nil
}

// computeAllNextSequences returns the next sequence value for every pattern
// in match configs. Unlike computeNextSequence, it does not stop at the first
// match — it computes a value for each pattern independently.
func computeAllNextSequences(dirPath, fieldName string, field SchemaField) (map[string]string, error) {
	if field.Match == nil || field.Match.Configs == nil {
		return nil, nil
	}

	result := make(map[string]string, len(field.Match.Configs))
	for _, globPattern := range sortedPatterns(field.Match.Configs) {
		config := field.Match.Configs[globPattern]
		cfgMap, ok := config.(map[string]any)
		if !ok || cfgMap == nil {
			return nil, fmt.Errorf("invalid match config for field %q pattern %q: config must be a map", fieldName, globPattern)
		}
		resolved, err := applySequenceConfig(field, cfgMap)
		if err != nil {
			return nil, fmt.Errorf("invalid match config for field %q pattern %q: %w", fieldName, globPattern, err)
		}
		result[globPattern] = computeNextFromPrefix(dirPath, resolved.Prefix, resolved.Digits)
	}

	if len(result) == 0 {
		return nil, nil
	}
	return result, nil
}

// sortedPatterns returns the keys of configs sorted alphabetically.
func sortedPatterns(configs map[string]any) []string {
	patterns := make([]string, 0, len(configs))
	for p := range configs {
		patterns = append(patterns, p)
	}
	sort.Strings(patterns)
	return patterns
}

// computeNextFromPrefix scans dirPath for entries matching prefix + digits pattern
// and returns the next sequence value.
func computeNextFromPrefix(dirPath, prefix string, digits int) string {
	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(prefix) + `(\d+)`)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		// Directory doesn't exist or unreadable — start at 1
		return fmt.Sprintf("%s%0*d", prefix, digits, 1)
	}

	maxNum := 0
	for _, e := range entries {
		matches := pattern.FindStringSubmatch(e.Name())
		if matches == nil {
			continue
		}
		num, err := strconv.Atoi(matches[1])
		if err != nil {
			continue
		}
		if num > maxNum {
			maxNum = num
		}
	}

	return fmt.Sprintf("%s%0*d", prefix, digits, maxNum+1)
}

// ToJSON serializes the describe result to stable JSON.
func (r *DescribeResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
