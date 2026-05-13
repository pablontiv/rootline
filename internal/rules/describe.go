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
func NewDescribeResult(path string, entries []StemEntry, effective *StemFile) *DescribeResult {
	applies := make([]string, len(entries))
	for i, e := range entries {
		applies[i] = e.Path
	}

	schema := effective.Schema
	if schema == nil {
		schema = map[string]SchemaField{}
	}

	validate := effective.Validate
	if validate == nil {
		validate = []ValidationRule{}
	}

	derive := effective.Derive
	if derive == nil {
		derive = map[string]any{}
	}

	aggregate := effective.Aggregate
	if aggregate == nil {
		aggregate = map[string]any{}
	}

	// Compute Next and NextByPattern for sequence fields
	for name, field := range schema {
		if field.Type == "sequence" {
			field.Next = computeNextSequence(path, field)
			field.NextByPattern = computeAllNextSequences(path, field)
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
	}
}

// computeNextSequence scans dirPath for files/dirs matching the prefix pattern,
// finds the highest numeric suffix, and returns prefix + next number zero-padded
// to the specified digits. Supports both top-level prefix/digits and match configs
// where prefix/digits are nested per-pattern (e.g., "E*": {prefix: E, digits: 2}).
// When multiple match config patterns are present, patterns are evaluated in
// alphabetical order to ensure deterministic results.
func computeNextSequence(dirPath string, field SchemaField) string {
	if field.Prefix != "" && field.Digits > 0 {
		return computeNextFromPrefix(dirPath, field.Prefix, field.Digits)
	}

	if field.Match == nil || field.Match.Configs == nil {
		return ""
	}

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		return ""
	}

	for _, globPattern := range sortedPatterns(field.Match.Configs) {
		config := field.Match.Configs[globPattern]
		cfgMap, ok := config.(map[string]any)
		if !ok {
			continue
		}
		prefix, digits := extractPrefixDigits(cfgMap)
		if prefix == "" || digits <= 0 {
			continue
		}

		for _, e := range entries {
			if matched, _ := filepath.Match(globPattern, e.Name()); matched {
				return computeNextFromPrefix(dirPath, prefix, digits)
			}
		}
	}

	return ""
}

// computeAllNextSequences returns the next sequence value for every pattern
// in match configs. Unlike computeNextSequence, it does not stop at the first
// match — it computes a value for each pattern independently.
func computeAllNextSequences(dirPath string, field SchemaField) map[string]string {
	if field.Match == nil || field.Match.Configs == nil {
		return nil
	}

	result := make(map[string]string, len(field.Match.Configs))
	for _, globPattern := range sortedPatterns(field.Match.Configs) {
		config := field.Match.Configs[globPattern]
		cfgMap, ok := config.(map[string]any)
		if !ok {
			continue
		}
		prefix, digits := extractPrefixDigits(cfgMap)
		if prefix == "" || digits <= 0 {
			continue
		}
		result[globPattern] = computeNextFromPrefix(dirPath, prefix, digits)
	}

	if len(result) == 0 {
		return nil
	}
	return result
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

// extractPrefixDigits extracts prefix and digits from a match config map.
func extractPrefixDigits(cfgMap map[string]any) (string, int) {
	prefix, _ := cfgMap["prefix"].(string)
	var digits int
	switch v := cfgMap["digits"].(type) {
	case int:
		digits = v
	case float64:
		digits = int(v)
	}
	return prefix, digits
}

// ToJSON serializes the describe result to stable JSON.
func (r *DescribeResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
