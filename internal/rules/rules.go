// Package rules implements .stem file parsing and rule loading.
//
// It handles parent-to-child inheritance of rules along the
// directory tree, merging stem definitions top-down.
package rules

import (
	"fmt"
	"os"

	"gopkg.in/yaml.v3"
)

// StemFile represents a parsed .stem YAML file.
type StemFile struct {
	Path       string                 `yaml:"-"`
	Version    int                    `yaml:"version"`
	Scope      Scope                  `yaml:"scope"`
	Schema     map[string]SchemaField `yaml:"schema"`
	Validate   []ValidationRule       `yaml:"validate"`
	Derive     map[string]any         `yaml:"derive"`
	Aggregate  map[string]any         `yaml:"aggregate"`
	Links      LinkSchema             `yaml:"links"`
	Structural StructuralRules        `yaml:"structural"`
}

// StructuralRules defines directory-level validation constraints.
type StructuralRules struct {
	Subdirs SubdirRules `yaml:"subdirs" json:"subdirs,omitempty"`
}

// SubdirRules defines constraints on subdirectories.
type SubdirRules struct {
	RequireIndex string `yaml:"require_index" json:"require_index,omitempty"`
	MinChildren  int    `yaml:"min_children" json:"min_children,omitempty"`
	MaxChildren  int    `yaml:"max_children" json:"max_children,omitempty"`
	Severity     string `yaml:"severity" json:"severity,omitempty"`
}

// IsEmpty reports whether the StructuralRules have no constraints.
func (sr StructuralRules) IsEmpty() bool {
	return sr.Subdirs.RequireIndex == "" && sr.Subdirs.MinChildren == 0 && sr.Subdirs.MaxChildren == 0
}

// LinkSchema defines link constraints in a .stem file.
// YAML format:
//
//	links:
//	  allowed: [blocks, parent, reference]
//	  blocks:
//	    target: "../tasks/*.md"
//	    field: blocked_by
type LinkSchema struct {
	Allowed []string            `json:"allowed,omitempty"`
	Rules   map[string]LinkRule `json:"rules,omitempty"`
}

// LinkRule defines a constraint for a specific link type.
type LinkRule struct {
	Target string `yaml:"target" json:"target,omitempty"`
	Field  string `yaml:"field" json:"field,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for LinkSchema.
// "allowed" is parsed as a string slice; all other keys are link rules.
func (ls *LinkSchema) UnmarshalYAML(value *yaml.Node) error {
	if value.Kind != yaml.MappingNode {
		return fmt.Errorf("links: expected mapping, got %v", value.Kind)
	}

	for i := 0; i+1 < len(value.Content); i += 2 {
		key := value.Content[i].Value
		val := value.Content[i+1]

		if key == "allowed" {
			var allowed []string
			if err := val.Decode(&allowed); err != nil {
				return fmt.Errorf("links.allowed: %w", err)
			}
			ls.Allowed = allowed
			continue
		}

		var rule LinkRule
		if err := val.Decode(&rule); err != nil {
			return fmt.Errorf("links.%s: %w", key, err)
		}
		if ls.Rules == nil {
			ls.Rules = make(map[string]LinkRule)
		}
		ls.Rules[key] = rule
	}

	return nil
}

// IsEmpty reports whether the LinkSchema has no constraints.
func (ls LinkSchema) IsEmpty() bool {
	return len(ls.Allowed) == 0 && len(ls.Rules) == 0
}

// Scope defines which files a .stem applies to.
type Scope struct {
	Match string `yaml:"match" json:"match,omitempty"`
}

// SchemaField defines a single field in the schema.
type SchemaField struct {
	Type     string   `yaml:"type" json:"type"`
	Required bool     `yaml:"required" json:"required"`
	Values   []string `yaml:"values" json:"values,omitempty"`
	Default  string   `yaml:"default" json:"default,omitempty"`
	Severity string   `yaml:"severity" json:"severity,omitempty"`
	Source   string   `yaml:"-" json:"source,omitempty"`
	Prefix   string   `yaml:"prefix" json:"prefix,omitempty"`
	Digits   int      `yaml:"digits" json:"digits,omitempty"`
	Next     string   `yaml:"-" json:"next,omitempty"`
}

// ValidationRule defines a single validation constraint.
type ValidationRule struct {
	Field    string         `yaml:"field" json:"field,omitempty"`
	Rule     string         `yaml:"rule" json:"rule"`
	If       map[string]any `yaml:"if" json:"if,omitempty"`
	Then     map[string]any `yaml:"then" json:"then,omitempty"`
	Severity string         `yaml:"severity" json:"severity,omitempty"`
	Source   string         `yaml:"-" json:"source,omitempty"`
}

// severityOrder defines the ordering for severity levels.
// Higher value = stricter.
var severityOrder = map[string]int{
	"off":   0,
	"warn":  1,
	"error": 2,
}

// ParseStem parses a .stem file from raw YAML content.
// Unknown sections are silently ignored for forward compatibility.
func ParseStem(path string, content []byte) (*StemFile, error) {
	var stem StemFile
	if err := yaml.Unmarshal(content, &stem); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	stem.Path = path

	// Tag source and default severity on schema fields
	for name, field := range stem.Schema {
		field.Source = path
		if field.Severity == "" {
			field.Severity = "error"
		}
		stem.Schema[name] = field
	}

	// Tag source and default severity on validation rules
	for i := range stem.Validate {
		stem.Validate[i].Source = path
		if stem.Validate[i].Severity == "" {
			stem.Validate[i].Severity = "error"
		}
	}

	return &stem, nil
}

// ParseStemFile reads and parses a .stem file from disk.
func ParseStemFile(path string) (*StemFile, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, fmt.Errorf("reading %s: %w", path, err)
	}
	return ParseStem(path, content)
}
