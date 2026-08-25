package extract

import (
	"fmt"

	"gopkg.in/yaml.v3"
)

// FrontmatterScalar preserves the authored YAML text and the native
// representation resolved from it. It is repair evidence, not public record data.
type FrontmatterScalar struct {
	Lexeme         string
	Representation string
}

// IsRepairableScalarRepresentation reports whether quoting the exact scalar
// lexeme is an approved representation-only repair for a string field.
func IsRepairableScalarRepresentation(name string) bool {
	switch name {
	case "timestamp", "boolean", "integer":
		return true
	default:
		return false
	}
}

func representationForYAMLTag(tag string) (string, bool) {
	switch tag {
	case "!!timestamp":
		return "timestamp", true
	case "!!bool":
		return "boolean", true
	case "!!int":
		return "integer", true
	default:
		return "", false
	}
}

func decodeFrontmatter(content string) (map[string]any, map[string]FrontmatterScalar, error) {
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(content), &doc); err != nil {
		return nil, nil, err
	}

	frontmatter := make(map[string]any)
	if err := doc.Decode(&frontmatter); err != nil {
		return nil, nil, err
	}
	if frontmatter == nil {
		frontmatter = make(map[string]any)
	}

	scalars, err := collectFrontmatterScalars(&doc)
	if err != nil {
		return nil, nil, err
	}
	return frontmatter, scalars, nil
}

func collectFrontmatterScalars(doc *yaml.Node) (map[string]FrontmatterScalar, error) {
	scalars := make(map[string]FrontmatterScalar)
	if len(doc.Content) == 0 {
		return scalars, nil
	}
	if doc.Content[0].Kind != yaml.MappingNode {
		return nil, fmt.Errorf("frontmatter must be a YAML mapping")
	}
	mapping := doc.Content[0]
	for i := 0; i+1 < len(mapping.Content); i += 2 {
		key, value := mapping.Content[i], mapping.Content[i+1]
		if value.Kind != yaml.ScalarNode {
			continue
		}
		representation, ok := representationForYAMLTag(value.Tag)
		if !ok {
			continue
		}
		scalars[key.Value] = FrontmatterScalar{
			Lexeme:         value.Value,
			Representation: representation,
		}
	}
	return scalars, nil
}
