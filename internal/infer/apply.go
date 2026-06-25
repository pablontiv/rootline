package infer

import (
	"fmt"
	"os"
	"strconv"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

// ApplyResult tracks what was modified during an apply operation.
type ApplyResult struct {
	Applied []string `json:"applied"`
	Skipped []string `json:"skipped"`
	DryRun  bool     `json:"dry_run,omitempty"`
}

// ApplySchemaInferences applies schema-modifying inferences to a .stem file.
// Uses yaml.Node to preserve formatting and handle fields like Required
// that have yaml:"-" tags on the struct.
// If dryRun is true, no files are written.
func ApplySchemaInferences(stemPath string, inferences []ReportInference, dryRun bool) (*ApplyResult, error) {
	data, err := os.ReadFile(stemPath)
	if err != nil {
		return nil, fmt.Errorf("reading stem: %w", err)
	}

	// Parse into struct for reading current state.
	stem, err := rules.ParseStem(stemPath, data)
	if err != nil {
		return nil, fmt.Errorf("parsing stem: %w", err)
	}

	// Parse into yaml.Node for preserving-format writes.
	var doc yaml.Node
	if err := yaml.Unmarshal(data, &doc); err != nil {
		return nil, fmt.Errorf("parsing yaml node: %w", err)
	}

	result := &ApplyResult{}
	modified := false

	for _, inf := range inferences {
		if inf.RequiresAgent {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s: %s (requires agent)", inf.Type, inf.Message))
			continue
		}

		switch inf.Type {
		case "enum_values":
			if applied, created := applyEnumExtensionNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (enum)", inf.Field))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("extend_enum: %s", inf.Field))
				}
				modified = true
			}

		case "required_field":
			if applied, created := applyRequiredFieldNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (required)", inf.Field))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("add_required: %s", inf.Field))
				}
				modified = true
			}

		case "constant_field":
			if applied, created := applyDefaultValueNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (default=%s)", inf.Field, inf.Value))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("add_default: %s=%s", inf.Field, inf.Value))
				}
				modified = true
			}

		case "field_type", "untyped_field":
			if applied, created := applyFieldTypeNode(&doc, stem, inf); applied {
				if created {
					result.Applied = append(result.Applied, fmt.Sprintf("add_field: %s (type=%s)", inf.Field, inf.Value))
				} else {
					result.Applied = append(result.Applied, fmt.Sprintf("set_type: %s=%s", inf.Field, inf.Value))
				}
				modified = true
			}

		case "sequence_incomplete":
			if applySequenceCompleteNode(&doc, inf) {
				result.Applied = append(result.Applied, fmt.Sprintf("sequence: %s completed", inf.Field))
				modified = true
			}
		}
	}

	if !modified {
		return result, nil
	}

	out, err := yaml.Marshal(&doc)
	if err != nil {
		return nil, fmt.Errorf("marshaling stem: %w", err)
	}

	// Skip write if in dry-run mode.
	if !dryRun {
		if err := os.WriteFile(stemPath, out, 0o644); err != nil {
			return nil, fmt.Errorf("writing stem: %w", err)
		}
	}

	result.DryRun = dryRun
	return result, nil
}

// findSchemaFieldNode navigates doc → schema → fieldName and returns the field's mapping node.
func findSchemaFieldNode(doc *yaml.Node, fieldName string) *yaml.Node {
	root := doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return nil
	}

	// Find "schema" key.
	var schemaNode *yaml.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "schema" {
			schemaNode = root.Content[i+1]
			break
		}
	}
	if schemaNode == nil || schemaNode.Kind != yaml.MappingNode {
		return nil
	}

	// Find field key within schema.
	for i := 0; i < len(schemaNode.Content)-1; i += 2 {
		if schemaNode.Content[i].Value == fieldName {
			return schemaNode.Content[i+1]
		}
	}
	return nil
}

// ensureSchemaFieldNode navigates doc → schema → fieldName, creating the
// schema mapping and/or the field's mapping node if absent. The returned bool
// is true when the field node was newly created. Returns (nil, false) only when
// the root or an existing schema value is not a mapping.
func ensureSchemaFieldNode(doc *yaml.Node, fieldName string) (*yaml.Node, bool) {
	root := doc
	if root.Kind == yaml.DocumentNode && len(root.Content) > 0 {
		root = root.Content[0]
	}
	if root.Kind != yaml.MappingNode {
		return nil, false
	}

	// Find or create the "schema" key.
	var schemaNode *yaml.Node
	for i := 0; i < len(root.Content)-1; i += 2 {
		if root.Content[i].Value == "schema" {
			schemaNode = root.Content[i+1]
			break
		}
	}
	if schemaNode == nil {
		schemaNode = &yaml.Node{Kind: yaml.MappingNode}
		root.Content = append(root.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "schema"},
			schemaNode,
		)
	}
	if schemaNode.Kind != yaml.MappingNode {
		return nil, false
	}

	// Find an existing field node.
	for i := 0; i < len(schemaNode.Content)-1; i += 2 {
		if schemaNode.Content[i].Value == fieldName {
			return schemaNode.Content[i+1], false
		}
	}

	// Create a new empty field mapping.
	fieldNode := &yaml.Node{Kind: yaml.MappingNode}
	schemaNode.Content = append(schemaNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: fieldName},
		fieldNode,
	)
	return fieldNode, true
}

// setFieldProperty sets or adds a scalar property on a mapping node.
func setFieldProperty(node *yaml.Node, key, value string) {
	if node.Kind != yaml.MappingNode {
		return
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			node.Content[i+1].Value = value
			return
		}
	}
	// Add new key-value pair.
	node.Content = append(node.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: key},
		&yaml.Node{Kind: yaml.ScalarNode, Value: value},
	)
}

// nodeProperty returns the scalar value of a property key on a mapping node.
// The yaml.Node is the source of truth during a run; the parsed stem.Schema
// struct is stale for fields created earlier in the same loop.
func nodeProperty(node *yaml.Node, key string) (string, bool) {
	if node.Kind != yaml.MappingNode {
		return "", false
	}
	for i := 0; i < len(node.Content)-1; i += 2 {
		if node.Content[i].Value == key {
			return node.Content[i+1].Value, true
		}
	}
	return "", false
}

func applyEnumExtensionNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool) {
	inferredValues := parseValueList(inf.Value)
	if len(inferredValues) == 0 {
		return false, false
	}

	// Zero value when the field is absent → empty existing set → all inferred are new.
	sf := stem.Schema[inf.Field]
	existing := make(map[string]bool)
	for _, v := range sf.Values {
		existing[v] = true
	}

	var newValues []string
	for _, v := range inferredValues {
		if !existing[v] {
			newValues = append(newValues, v)
		}
	}
	if len(newValues) == 0 {
		return false, false
	}

	fieldNode, created := ensureSchemaFieldNode(doc, inf.Field)
	if fieldNode == nil {
		return false, false
	}

	if created {
		// New field: declare it as an enum with a fresh values sequence.
		setFieldProperty(fieldNode, "type", "enum")
		valuesNode := &yaml.Node{Kind: yaml.SequenceNode}
		for _, v := range newValues {
			valuesNode.Content = append(valuesNode.Content,
				&yaml.Node{Kind: yaml.ScalarNode, Value: v},
			)
		}
		fieldNode.Content = append(fieldNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: "values"},
			valuesNode,
		)
		return true, true
	}

	// Existing field (possibly created earlier this run by field_type).
	// Refuse to attach enum values to a field already typed as non-enum.
	if t, has := nodeProperty(fieldNode, "type"); has && t != "enum" {
		return false, false
	}

	// Append to an existing values sequence if present.
	for i := 0; i < len(fieldNode.Content)-1; i += 2 {
		if fieldNode.Content[i].Value == "values" {
			valuesNode := fieldNode.Content[i+1]
			if valuesNode.Kind == yaml.SequenceNode {
				for _, v := range newValues {
					valuesNode.Content = append(valuesNode.Content,
						&yaml.Node{Kind: yaml.ScalarNode, Value: v},
					)
				}
				return true, false
			}
		}
	}

	// No values sequence yet (e.g. field_type created {type: enum} first):
	// ensure type:enum and create the values sequence.
	setFieldProperty(fieldNode, "type", "enum")
	valuesNode := &yaml.Node{Kind: yaml.SequenceNode}
	for _, v := range newValues {
		valuesNode.Content = append(valuesNode.Content,
			&yaml.Node{Kind: yaml.ScalarNode, Value: v},
		)
	}
	fieldNode.Content = append(fieldNode.Content,
		&yaml.Node{Kind: yaml.ScalarNode, Value: "values"},
		valuesNode,
	)
	return true, false
}

func applyRequiredFieldNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool) {
	sf, ok := stem.Schema[inf.Field]
	if ok && sf.Required {
		return false, false
	}

	fieldNode, created := ensureSchemaFieldNode(doc, inf.Field)
	if fieldNode == nil {
		return false, false
	}

	setFieldProperty(fieldNode, "required", "true")
	return true, created
}

func applyDefaultValueNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool) {
	sf, ok := stem.Schema[inf.Field]
	if ok && sf.Default != "" {
		return false, false
	}

	fieldNode, created := ensureSchemaFieldNode(doc, inf.Field)
	if fieldNode == nil {
		return false, false
	}

	setFieldProperty(fieldNode, "default", inf.Value)
	return true, created
}

func applyFieldTypeNode(doc *yaml.Node, stem *rules.StemFile, inf ReportInference) (bool, bool) {
	fieldNode, created := ensureSchemaFieldNode(doc, inf.Field)
	if fieldNode == nil {
		return false, false
	}

	// Source of truth is the live node, not the stale stem.Schema struct:
	// never overwrite a type already set (e.g. type:enum from enum_values
	// for this same field earlier in the run).
	if _, has := nodeProperty(fieldNode, "type"); has {
		return false, false
	}

	setFieldProperty(fieldNode, "type", inf.Value)
	return true, created
}

func parseValueList(s string) []string {
	s = strings.TrimPrefix(s, "[")
	s = strings.TrimSuffix(s, "]")
	s = strings.TrimSpace(s)
	if s == "" {
		return nil
	}
	return strings.Fields(s)
}

func applySequenceCompleteNode(doc *yaml.Node, inf ReportInference) bool {
	parts := strings.SplitN(inf.Value, ":", 2)
	if len(parts) != 2 {
		return false
	}
	prefix := parts[0]
	digits, err := strconv.Atoi(parts[1])
	if err != nil || digits <= 0 {
		return false
	}

	fieldNode := findSchemaFieldNode(doc, inf.Field)
	if fieldNode == nil {
		return false
	}

	// setFieldProperty adds the key-value pair if missing, or updates if present.
	setFieldProperty(fieldNode, "prefix", prefix)
	setFieldProperty(fieldNode, "digits", strconv.Itoa(digits))
	return true
}
