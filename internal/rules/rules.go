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
	Path     string                 `yaml:"-"`
	Version  int                    `yaml:"version"`
	Scope    Scope                  `yaml:"scope"`
	Schema   map[string]SchemaField `yaml:"schema"`
	Validate []ValidationRule       `yaml:"validate"`
	Derive   map[string]any         `yaml:"derive"`
	State    map[string]any         `yaml:"state"`
	Links    map[string]any         `yaml:"links"`
}

// Scope defines which files a .stem applies to.
type Scope struct {
	Match string `yaml:"match"`
}

// SchemaField defines a single field in the schema.
type SchemaField struct {
	Type     string   `yaml:"type"`
	Required bool     `yaml:"required"`
	Values   []string `yaml:"values"`
	Default  string   `yaml:"default"`
	Source   string   `yaml:"-"`
}

// ValidationRule defines a single validation constraint.
type ValidationRule struct {
	Field  string         `yaml:"field"`
	Rule   string         `yaml:"rule"`
	If     map[string]any `yaml:"if"`
	Then   map[string]any `yaml:"then"`
	Source string         `yaml:"-"`
}

// ParseStem parses a .stem file from raw YAML content.
// Unknown sections are silently ignored for forward compatibility.
func ParseStem(path string, content []byte) (*StemFile, error) {
	var stem StemFile
	if err := yaml.Unmarshal(content, &stem); err != nil {
		return nil, fmt.Errorf("parsing %s: %w", path, err)
	}
	stem.Path = path

	// Tag source on schema fields
	for name, field := range stem.Schema {
		field.Source = path
		stem.Schema[name] = field
	}

	// Tag source on validation rules
	for i := range stem.Validate {
		stem.Validate[i].Source = path
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
