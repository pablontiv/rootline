package stemyaml

import (
	"fmt"
	"math"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
	"gopkg.in/yaml.v3"
)

// AppendSchemaField writes one canonical schema field entry.
func AppendSchemaField(b *strings.Builder, name string, field rules.SchemaField) error {
	var local strings.Builder
	fmt.Fprintf(&local, "  %s:\n", name)
	if err := appendSchemaFieldAttrs(&local, field); err != nil {
		return err
	}
	b.WriteString(local.String())
	return nil
}

// AppendSchemaFieldAttrs writes canonical schema field attributes.
func AppendSchemaFieldAttrs(b *strings.Builder, field rules.SchemaField) error {
	var local strings.Builder
	if err := appendSchemaFieldAttrs(&local, field); err != nil {
		return err
	}
	b.WriteString(local.String())
	return nil
}

func appendSchemaFieldAttrs(b *strings.Builder, field rules.SchemaField) error {
	fmt.Fprintf(b, "    type: %s\n", field.Type)
	if field.RequiredMatch != nil {
		b.WriteString("    required:\n")
		if err := appendFieldMatchAt(b, "      ", field.RequiredMatch); err != nil {
			return fmt.Errorf("serializing required match: %w", err)
		}
	} else if field.Required {
		b.WriteString("    required: true\n")
	}
	if len(field.Values) > 0 {
		values, err := yamlStringList(field.Values)
		if err != nil {
			return fmt.Errorf("serializing values: %w", err)
		}
		fmt.Fprintf(b, "    values: [%s]\n", strings.Join(values, ", "))
	}
	if field.Default != "" {
		value, err := yamlScalar(field.Default)
		if err != nil {
			return fmt.Errorf("serializing default: %w", err)
		}
		fmt.Fprintf(b, "    default: %s\n", value)
	}
	if field.Severity != "" {
		fmt.Fprintf(b, "    severity: %s\n", field.Severity)
	}
	if field.Extract != "" {
		value, err := yamlScalar(field.Extract)
		if err != nil {
			return fmt.Errorf("serializing source: %w", err)
		}
		fmt.Fprintf(b, "    source: %s\n", value)
	}
	if field.Prefix != "" {
		value, err := yamlScalar(field.Prefix)
		if err != nil {
			return fmt.Errorf("serializing prefix: %w", err)
		}
		fmt.Fprintf(b, "    prefix: %s\n", value)
	}
	if field.Digits > 0 {
		fmt.Fprintf(b, "    digits: %d\n", field.Digits)
	}
	if field.Excludes != nil {
		value, err := yamlScalar(field.Excludes.Match)
		if err != nil {
			return fmt.Errorf("serializing excludes: %w", err)
		}
		fmt.Fprintf(b, "    excludes:\n      match: %s\n", value)
	}
	if err := appendFieldMatchAt(b, "    ", field.Match); err != nil {
		return fmt.Errorf("serializing match: %w", err)
	}
	return nil
}

func appendFieldMatchAt(b *strings.Builder, indent string, match *rules.FieldMatch) error {
	if match == nil {
		return nil
	}
	switch {
	case match.Configs != nil:
		if len(match.Configs) == 0 {
			b.WriteString(indent + "match: {}\n")
			return nil
		}
		b.WriteString(indent + "match:\n")
		keys := make([]string, 0, len(match.Configs))
		for k := range match.Configs {
			keys = append(keys, k)
		}
		sort.Strings(keys)
		for _, key := range keys {
			cfgMap, ok := match.Configs[key].(map[string]any)
			if !ok || cfgMap == nil {
				return fmt.Errorf("pattern %q config must be a map", key)
			}
			keyScalar, err := yamlQuotedScalar(key)
			if err != nil {
				return fmt.Errorf("pattern %q key: %w", key, err)
			}
			if len(cfgMap) == 0 {
				fmt.Fprintf(b, "%s  %s: {}\n", indent, keyScalar)
				continue
			}
			for cfgKey := range cfgMap {
				if cfgKey != "prefix" && cfgKey != "digits" {
					return fmt.Errorf("pattern %q config key %q is unsupported", key, cfgKey)
				}
			}
			fmt.Fprintf(b, "%s  %s:\n", indent, keyScalar)
			if prefix, ok := cfgMap["prefix"]; ok {
				prefixString, ok := prefix.(string)
				if !ok {
					return fmt.Errorf("pattern %q prefix must be a string", key)
				}
				prefixScalar, err := yamlScalar(prefixString)
				if err != nil {
					return fmt.Errorf("pattern %q prefix: %w", key, err)
				}
				fmt.Fprintf(b, "%s    prefix: %s\n", indent, prefixScalar)
			}
			if digits, ok := cfgMap["digits"]; ok {
				digitsInt, err := configDigits(digits)
				if err != nil {
					return fmt.Errorf("pattern %q digits: %w", key, err)
				}
				fmt.Fprintf(b, "%s    digits: %d\n", indent, digitsInt)
			}
		}
	case match.Patterns != nil && len(match.Patterns) == 0:
		fmt.Fprintf(b, "%smatch: []\n", indent)
	case len(match.Patterns) == 1:
		pattern, err := yamlQuotedScalar(match.Patterns[0])
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "%smatch: %s\n", indent, pattern)
	case len(match.Patterns) > 1:
		patterns, err := yamlQuotedStringList(match.Patterns)
		if err != nil {
			return err
		}
		fmt.Fprintf(b, "%smatch: [%s]\n", indent, strings.Join(patterns, ", "))
	}
	return nil
}

func yamlStringList(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, value := range values {
		scalar, err := yamlScalar(value)
		if err != nil {
			return nil, err
		}
		out = append(out, scalar)
	}
	return out, nil
}

func yamlQuotedStringList(values []string) ([]string, error) {
	out := make([]string, 0, len(values))
	for _, value := range values {
		scalar, err := yamlQuotedScalar(value)
		if err != nil {
			return nil, err
		}
		out = append(out, scalar)
	}
	return out, nil
}

func configDigits(value any) (int, error) {
	switch v := value.(type) {
	case int:
		return v, nil
	case int64:
		return int(v), nil
	case float64:
		if math.Trunc(v) != v {
			return 0, fmt.Errorf("must be an integer")
		}
		return int(v), nil
	default:
		return 0, fmt.Errorf("must be an integer")
	}
}

func yamlScalar(value string) (string, error) {
	b, err := yaml.Marshal(value)
	if err != nil {
		return "", err
	}
	return singleLineScalar(value, string(b))
}

func yamlQuotedScalar(value string) (string, error) {
	node := yaml.Node{Kind: yaml.ScalarNode, Tag: "!!str", Value: value, Style: yaml.DoubleQuotedStyle}
	b, err := yaml.Marshal(&node)
	if err != nil {
		return "", err
	}
	return singleLineScalar(value, string(b))
}

func singleLineScalar(value, marshaled string) (string, error) {
	out := strings.TrimSuffix(marshaled, "\n")
	if strings.Contains(out, "\n") {
		return "", fmt.Errorf("multiline scalar %q is unsupported", value)
	}
	return out, nil
}
