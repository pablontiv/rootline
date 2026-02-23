package rules

import (
	"encoding/json"
	"fmt"
	"os"
	"regexp"
	"strconv"
)

// DescribeResult is the versioned JSON output for the describe command.
// It shows the effective schema for a directory after merging all
// ancestor .stem files.
type DescribeResult struct {
	Version   int                    `json:"version"`
	Kind      string                 `json:"kind"`
	Path      string                 `json:"path"`
	Applies   []string               `json:"applies"`
	Scope     Scope                  `json:"scope"`
	Schema    map[string]SchemaField `json:"schema"`
	Validate  []ValidationRule       `json:"validate"`
	Derive    map[string]any         `json:"derive"`
	Aggregate map[string]any         `json:"aggregate"`
	Links     LinkSchema             `json:"links"`
	Hints     []string               `json:"hints,omitempty"`
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

	// Compute Next for sequence fields
	for name, field := range schema {
		if field.Type == "sequence" {
			field.Next = computeNextSequence(path, field)
			schema[name] = field
		}
	}

	return &DescribeResult{
		Version:   1,
		Kind:      "rootline/describe",
		Path:      path,
		Applies:   applies,
		Scope:     effective.Scope,
		Schema:    schema,
		Validate:  validate,
		Derive:    derive,
		Aggregate: aggregate,
		Links:     effective.Links,
	}
}

// computeNextSequence scans dirPath for files/dirs matching the prefix pattern,
// finds the highest numeric suffix, and returns prefix + next number zero-padded
// to the specified digits.
func computeNextSequence(dirPath string, field SchemaField) string {
	if field.Prefix == "" || field.Digits <= 0 {
		return ""
	}

	pattern := regexp.MustCompile(`^` + regexp.QuoteMeta(field.Prefix) + `(\d+)`)

	entries, err := os.ReadDir(dirPath)
	if err != nil {
		// Directory doesn't exist or unreadable — start at 1
		return fmt.Sprintf("%s%0*d", field.Prefix, field.Digits, 1)
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

	return fmt.Sprintf("%s%0*d", field.Prefix, field.Digits, maxNum+1)
}

// ToJSON serializes the describe result to stable JSON.
func (r *DescribeResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
