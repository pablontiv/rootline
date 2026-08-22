// Package rules implements .stem file parsing and rule loading.
//
// It handles parent-to-child inheritance of rules along the
// directory tree, merging stem definitions top-down.
package rules

import (
	"fmt"
	"os"
	"slices"

	"github.com/pablontiv/rootline/internal/extract"
	"gopkg.in/yaml.v3"
)

// StemFile represents a parsed .stem YAML file.
type StemFile struct {
	Path       string                 `yaml:"-"`
	Version    int                    `yaml:"version"`
	Root       bool                   `yaml:"root"`
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
	Styles  []string            `json:"styles,omitempty"`
	Checks  *LinkChecks         `json:"checks,omitempty"`
	Rules   map[string]LinkRule `json:"rules,omitempty"`

	// BasenameFallback opts into matching a target that names no path
	// against a uniquely-named record anywhere in the tree, the wiki
	// convention where sources link entities by bare filename.
	//
	// It is off by default because it needs a global index of every record,
	// and single-file `validate <file>` has none. With it on, that one
	// command cannot check such links and says so; with it off, every
	// command answers identically.
	BasenameFallback bool `json:"basename_fallback,omitempty"`

	// UnknownCheckKeys lists keys found under links.checks that no check
	// consumes. Diagnostic only — surfaced by stem health, never serialized.
	UnknownCheckKeys []string `json:"-"`
}

// LinkChecks enables filesystem-backed link checks (ADO code-wiki conventions).
// Cycles opts graph --check into treating link cycles as failures.
type LinkChecks struct {
	// Resolve is tri-state: unset means on. Broken-target detection is the
	// one property graph and validate both claim to check, and graph has
	// always checked it unconditionally, so leaving it opt-in kept the two
	// disagreeing by default. Setting it false opts out explicitly.
	Resolve  *bool `yaml:"resolve" json:"resolve,omitempty"`
	Anchors  bool  `yaml:"anchors" json:"anchors,omitempty"`
	Encoding bool  `yaml:"encoding" json:"encoding,omitempty"`
	Cycles   bool  `yaml:"cycles" json:"cycles,omitempty"`
}

// ShouldResolve reports whether broken-target detection runs. It is on unless
// the schema explicitly turns it off, and on when no checks block exists.
func (ls LinkSchema) ShouldResolve() bool {
	if ls.Checks == nil || ls.Checks.Resolve == nil {
		return true
	}
	return *ls.Checks.Resolve
}

// knownCheckKeys lists the keys LinkChecks consumes; keep in sync with its
// struct fields.
var knownCheckKeys = []string{"resolve", "anchors", "encoding", "cycles"}

// EffectiveStyles returns the link styles governed by this schema.
// An empty declaration defaults to wikilink-only for backward compatibility.
func (ls LinkSchema) EffectiveStyles() []string {
	if len(ls.Styles) > 0 {
		return ls.Styles
	}
	return []string{extract.StyleWikilink}
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

		if key == "basename_fallback" {
			var enabled bool
			if err := val.Decode(&enabled); err != nil {
				return fmt.Errorf("links.basename_fallback: %w", err)
			}
			ls.BasenameFallback = enabled
			continue
		}

		if key == "styles" {
			var styles []string
			if err := val.Decode(&styles); err != nil {
				return fmt.Errorf("links.styles: %w", err)
			}
			ls.Styles = styles
			continue
		}

		if key == "checks" {
			var checks LinkChecks
			if err := val.Decode(&checks); err != nil {
				return fmt.Errorf("links.checks: %w", err)
			}
			ls.Checks = &checks
			for j := 0; j+1 < len(val.Content); j += 2 {
				k := val.Content[j].Value
				if !slices.Contains(knownCheckKeys, k) {
					ls.UnknownCheckKeys = append(ls.UnknownCheckKeys, k)
				}
			}
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
	return len(ls.Allowed) == 0 && len(ls.Rules) == 0 && len(ls.Styles) == 0 && ls.Checks == nil && !ls.BasenameFallback
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
	Source        string            `yaml:"-" json:"defined_in,omitempty"`
	Extract       string            `yaml:"source" json:"source,omitempty"`
	Prefix        string            `yaml:"prefix" json:"prefix,omitempty"`
	Digits        int               `yaml:"digits" json:"digits,omitempty"`
	Next          string            `yaml:"-" json:"next,omitempty"`
	NextByPattern map[string]string `yaml:"-" json:"next_by_pattern,omitempty"`
	Excludes      *ExcludeRule      `yaml:"excludes" json:"excludes,omitempty"`
	Match         *FieldMatch       `yaml:"match" json:"match,omitempty"`
	RequiredMatch *FieldMatch       `yaml:"-" json:"required_match,omitempty"`
	Heading       string            `yaml:"-" json:"heading,omitempty"`
	Ordered       *int              `yaml:"-" json:"ordered,omitempty"`
	declaration   schemaFieldDeclarationMetadata
}

type schemaFieldDeclarationMetadata struct {
	TypePresent    bool
	TypeNull       bool
	SourcePresent  bool
	SourceNull     bool
	HeadingPresent bool
	OrderedPresent bool
	NullField      bool // entire field was set to null in YAML
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

func schemaFieldNodeMap(value *yaml.Node) map[string]*yaml.Node {
	nodes := make(map[string]*yaml.Node)
	if value == nil || value.Kind != yaml.MappingNode {
		return nodes
	}
	for i := 0; i+1 < len(value.Content); i += 2 {
		nodes[value.Content[i].Value] = value.Content[i+1]
	}
	return nodes
}

// UnmarshalYAML implements custom unmarshaling for SchemaField.
// The "required" field accepts either a bool (required: true) or an object
// with a match key (required: {match: ["T*"]}).
func (sf *SchemaField) UnmarshalYAML(value *yaml.Node) error {
	var raw schemaFieldRaw
	if err := value.Decode(&raw); err != nil {
		return err
	}

	nodes := schemaFieldNodeMap(value)
	sf.declaration = schemaFieldDeclarationMetadata{
		TypePresent:    nodes["type"] != nil,
		TypeNull:       nodes["type"] != nil && nodes["type"].Tag == "!!null",
		SourcePresent:  nodes["source"] != nil,
		SourceNull:     nodes["source"] != nil && nodes["source"].Tag == "!!null",
		HeadingPresent: nodes["heading"] != nil,
		OrderedPresent: nodes["ordered"] != nil,
		NullField:      value.Tag == "!!null", // entire field was set to null
	}
	if node := nodes["type"]; node != nil && node.Tag != "!!null" {
		sf.Type = raw.Type
	}
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

// UnmarshalYAML implements custom YAML unmarshaling for StemFile to detect
// null schema fields and mark them for removal.
func (s *StemFile) UnmarshalYAML(value *yaml.Node) error {
	// Define an intermediate struct to match StemFile but without custom unmarshaling
	type stemFileRaw StemFile

	// First, do the normal unmarshaling
	if err := value.Decode((*stemFileRaw)(s)); err != nil {
		return err
	}

	// Then, walk the YAML nodes to detect null schema fields
	if value.Kind == yaml.MappingNode {
		for i := 0; i+1 < len(value.Content); i += 2 {
			key := value.Content[i]
			val := value.Content[i+1]
			if key.Value == "schema" && val.Kind == yaml.MappingNode {
				// Walk the schema map and detect null fields
				for j := 0; j+1 < len(val.Content); j += 2 {
					fieldKey := val.Content[j]
					fieldVal := val.Content[j+1]
					fieldName := fieldKey.Value
					if fieldVal.Tag == "!!null" {
						// A child cannot remove an inherited field, so a null
						// declaration has no meaning to honour. It is refused
						// here rather than carried forward: a zero-valued field
						// reaching the pipeline trips three unrelated checks
						// (incomplete-type, a required loosening, a type change
						// to "") and none of them names the real mistake.
						field := s.Schema[fieldName]
						field.declaration.NullField = true
						s.Schema[fieldName] = field
					}
				}
			}
		}
	}

	return nil
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

	// Reject a schema field set to null. A child never removes what a parent
	// declared, so there is nothing for null to mean here. Refusing it at the
	// read keeps a zero-valued declaration out of the pipeline entirely.
	for _, name := range sortedSchemaFieldNames(stem.Schema) {
		if stem.Schema[name].declaration.NullField {
			return nil, fmt.Errorf("parsing %s: schema field %q is null — a child .stem never removes an inherited field; if the field should not exist, remove it from the .stem that declares it", path, name)
		}
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
