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
	Target     string `yaml:"target" json:"target,omitempty"`
	Field      string `yaml:"field" json:"field,omitempty"`
	ValueField string `yaml:"value_field" json:"value_field,omitempty"`
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

// ExcludeRule defines a path-based exclusion for schema field validation.
type ExcludeRule struct {
	Match string `yaml:"match" json:"match"`
}

// FieldMatch specifies which directory/file patterns a schema field applies to.
// It supports three YAML forms:
//   - string: match: "T*"          → Patterns=["T*"]
//   - list:   match: ["F*", "T*"]  → Patterns=["F*","T*"]
//   - map:    match: {"E*": {prefix: E, digits: 2}} → Configs={"E*": {...}}
type FieldMatch struct {
	Patterns []string       `json:"patterns,omitempty"`
	Configs  map[string]any `json:"configs,omitempty"`
}

// UnmarshalYAML implements custom unmarshaling for FieldMatch.
func (fm *FieldMatch) UnmarshalYAML(value *yaml.Node) error {
	switch value.Kind {
	case yaml.ScalarNode:
		// string form: match: "T*"
		fm.Patterns = []string{value.Value}
		return nil
	case yaml.SequenceNode:
		// list form: match: ["F*", "T*"]
		var patterns []string
		if err := value.Decode(&patterns); err != nil {
			return fmt.Errorf("match: decoding list: %w", err)
		}
		fm.Patterns = patterns
		return nil
	case yaml.MappingNode:
		// map form: match: {"E*": {prefix: E, digits: 2}}
		var configs map[string]any
		if err := value.Decode(&configs); err != nil {
			return fmt.Errorf("match: decoding map: %w", err)
		}
		fm.Configs = configs
		return nil
	default:
		return fmt.Errorf("match: expected string, list, or map, got YAML kind %v", value.Kind)
	}
}

// SchemaField defines a single field in the schema.
type SchemaField struct {
	Type          string            `yaml:"type" json:"type"`
	Required      bool              `yaml:"-" json:"required"`
	Values        []string          `yaml:"values" json:"values,omitempty"`
	Default       string            `yaml:"default" json:"default,omitempty"`
	Severity      string            `yaml:"severity" json:"severity,omitempty"`
	Source        string            `yaml:"-" json:"source,omitempty"`
	Extract       string            `yaml:"source" json:"extract,omitempty"`
	Prefix        string            `yaml:"prefix" json:"prefix,omitempty"`
	Digits        int               `yaml:"digits" json:"digits,omitempty"`
	Next          string            `yaml:"-" json:"next,omitempty"`
	NextByPattern map[string]string `yaml:"-" json:"next_by_pattern,omitempty"`
	Excludes      *ExcludeRule      `yaml:"excludes" json:"excludes,omitempty"`
	Match         *FieldMatch       `yaml:"match" json:"match,omitempty"`
	RequiredMatch *FieldMatch       `yaml:"-" json:"required_match,omitempty"`
	// Section fields (type: section)
	Heading string `yaml:"heading" json:"heading,omitempty"`
	Ordered *int   `yaml:"ordered" json:"ordered,omitempty"`
}

// schemaFieldRaw is the intermediate type for YAML unmarshaling.
// It uses yaml.Node for "required" to support both bool and object forms.
type schemaFieldRaw struct {
	Type     string       `yaml:"type"`
	Required yaml.Node    `yaml:"required"`
	Values   []string     `yaml:"values"`
	Default  string       `yaml:"default"`
	Severity string       `yaml:"severity"`
	Extract  string       `yaml:"source"`
	Prefix   string       `yaml:"prefix"`
	Digits   int          `yaml:"digits"`
	Excludes *ExcludeRule `yaml:"excludes"`
	Match    *FieldMatch  `yaml:"match"`
	Heading  string       `yaml:"heading"`
	Ordered  *int         `yaml:"ordered"`
}

// UnmarshalYAML implements custom unmarshaling for SchemaField.
// The "required" field accepts either a bool (required: true) or an object
// with a match key (required: {match: ["T*"]}).
func (sf *SchemaField) UnmarshalYAML(value *yaml.Node) error {
	var raw schemaFieldRaw
	if err := value.Decode(&raw); err != nil {
		return err
	}

	sf.Type = raw.Type
	sf.Values = raw.Values
	sf.Default = raw.Default
	sf.Severity = raw.Severity
	sf.Extract = raw.Extract
	sf.Prefix = raw.Prefix
	sf.Digits = raw.Digits
	sf.Excludes = raw.Excludes
	sf.Match = raw.Match
	sf.Heading = raw.Heading
	sf.Ordered = raw.Ordered

	// Parse "required" field: bool or {match: [...]}
	switch raw.Required.Kind {
	case yaml.ScalarNode:
		var b bool
		if err := raw.Required.Decode(&b); err != nil {
			return fmt.Errorf("required: expected bool or {match: [...]}, got %q", raw.Required.Value)
		}
		sf.Required = b
	case yaml.MappingNode:
		// Object form: required: {match: ["T*"]}
		var obj struct {
			Match *FieldMatch `yaml:"match"`
		}
		if err := raw.Required.Decode(&obj); err != nil {
			return fmt.Errorf("required: %w", err)
		}
		if obj.Match == nil {
			return fmt.Errorf("required: object form requires a 'match' key")
		}
		sf.RequiredMatch = obj.Match
		sf.Required = true // default to true; resolved by FilterSchemaByMatch
	case 0:
		// Field not present in YAML — leave defaults (Required=false, RequiredMatch=nil)
	default:
		return fmt.Errorf("required: expected bool or {match: [...]}, got YAML kind %v", raw.Required.Kind)
	}

	return nil
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

	// Reject unsupported stem versions
	if stem.Version == 0 || stem.Version == 1 {
		return nil, fmt.Errorf("parsing %s: stem version %d is no longer supported — upgrade with rootline v0.x migrate --to-v2 first", path, stem.Version)
	}

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
