package rules

import "encoding/json"

// DescribeResult is the versioned JSON output for the describe command.
// It shows the effective schema for a directory after merging all
// ancestor .stem files.
type DescribeResult struct {
	Version  int                    `json:"version"`
	Kind     string                 `json:"kind"`
	Path     string                 `json:"path"`
	Applies  []string               `json:"applies"`
	Scope    Scope                  `json:"scope"`
	Schema   map[string]SchemaField `json:"schema"`
	Validate []ValidationRule       `json:"validate"`
	Derive   map[string]any         `json:"derive"`
	State    map[string]any         `json:"state"`
	Links    map[string]any         `json:"links"`
	Hints    []string               `json:"hints,omitempty"`
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

	state := effective.State
	if state == nil {
		state = map[string]any{}
	}

	links := effective.Links
	if links == nil {
		links = map[string]any{}
	}

	return &DescribeResult{
		Version:  1,
		Kind:     "rootline/describe",
		Path:     path,
		Applies:  applies,
		Scope:    effective.Scope,
		Schema:   schema,
		Validate: validate,
		Derive:   derive,
		State:    state,
		Links:    links,
	}
}

// ToJSON serializes the describe result to stable JSON.
func (r *DescribeResult) ToJSON() ([]byte, error) {
	return json.Marshal(r)
}
