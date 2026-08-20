package rules

import (
	"fmt"
	"reflect"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

type FieldContractIssue struct {
	Code     string
	Expected string
	Actual   string
	Message  string
}

func IsSupportedFieldType(typeName string) bool {
	switch typeName {
	case "string", "list", "enum", "sequence", "link", "boolean", "integer":
		return true
	}
	return false
}

func ValidateFieldDeclaration(name string, field SchemaField) []FieldContractIssue {
	if field.Type == "bool" {
		return []FieldContractIssue{issue("legacy-type", "boolean", "bool", fmt.Sprintf("field %q uses legacy type %q; migrate to:\n%s", name, field.Type, "type: boolean"))}
	}
	if field.declaration.OrderedPresent || field.Ordered != nil {
		return []FieldContractIssue{legacyOrderedIssue(name, field)}
	}
	if field.Type == "section" {
		return []FieldContractIssue{legacySectionIssue(name, field)}
	}
	if field.declaration.HeadingPresent || field.Heading != "" {
		return []FieldContractIssue{legacySectionIssue(name, field)}
	}
	if field.Type == "" {
		actual := "omitted"
		if field.declaration.TypePresent {
			actual = "empty"
		}
		if field.declaration.TypeNull {
			actual = "null"
		}
		return []FieldContractIssue{issue("incomplete-type", "supported field type", actual, fmt.Sprintf("field %q must declare one supported type", name))}
	}
	if !IsSupportedFieldType(field.Type) {
		return []FieldContractIssue{issue("unknown-type", "string, list, enum, sequence, link, boolean, integer", field.Type, fmt.Sprintf("field %q declares unsupported type %q", name, field.Type))}
	}
	if field.Type == "enum" && len(field.Values) == 0 {
		return []FieldContractIssue{issue("incomplete-type", "one or more values", "empty values", fmt.Sprintf("enum field %q must declare at least one value", name))}
	}
	if field.Type == "sequence" {
		if sequenceIssue := validateSequenceDeclaration(name, field); sequenceIssue != nil {
			return []FieldContractIssue{*sequenceIssue}
		}
	}
	if field.Extract != "" {
		if _, err := extract.ParseBodySource(field.Extract); err != nil {
			return []FieldContractIssue{issue("unsupported-source", "body.h1 or body.section[...]", field.Extract, fmt.Sprintf("field %q has unsupported source: %v", name, err))}
		}
	}
	return nil
}

func validateSequenceDeclaration(name string, field SchemaField) *FieldContractIssue {
	if field.Match == nil || len(field.Match.Configs) == 0 {
		if field.Prefix == "" || field.Digits <= 0 {
			return incompleteSequenceIssue(name, "incomplete sequence")
		}
		return nil
	}

	for _, pattern := range sortedPatterns(field.Match.Configs) {
		config, ok := field.Match.Configs[pattern].(map[string]any)
		if !ok || config == nil {
			return incompleteSequenceIssue(name, "malformed sequence config")
		}
		if _, err := applySequenceConfig(field, config); err != nil {
			if seqErr, ok := err.(sequenceConfigError); ok {
				return incompleteSequenceConfigIssue(name, pattern, seqErr)
			}
			return incompleteSequenceIssue(name, "incomplete sequence config")
		}
	}
	return nil
}

func incompleteSequenceIssue(name, actual string) *FieldContractIssue {
	issue := issue("incomplete-type", "prefix and positive digits", actual, fmt.Sprintf("sequence field %q must declare prefix and positive digits", name))
	return &issue
}

func incompleteSequenceConfigIssue(name, pattern string, configErr sequenceConfigError) *FieldContractIssue {
	issue := issue(
		"incomplete-type",
		"prefix and positive digits",
		configErr.actual,
		fmt.Sprintf("sequence field %q must declare prefix and positive digits: match %q: %s", name, pattern, configErr.detail),
	)
	return &issue
}

func ValidateFieldValue(field SchemaField, value any) *FieldContractIssue {
	if issue := validateRepresentation(field.Type, value); issue != nil {
		return issue
	}
	return validateTypeSemantics(field, value)
}

func validateRepresentation(typeName string, value any) *FieldContractIssue {
	actual := representationName(value)
	ok := false
	switch typeName {
	case "string", "enum", "sequence":
		ok = actual == "string"
	case "list":
		ok = actual == "sequence"
	case "link":
		ok = actual == "string" || actual == "sequence"
	case "boolean":
		ok = actual == "boolean"
	case "integer":
		ok = actual == "integer"
	default:
		return nil
	}
	if ok {
		return nil
	}
	return &FieldContractIssue{Code: "type-mismatch", Expected: expectedRepresentation(typeName), Actual: actual, Message: fmt.Sprintf("expected %s, got %s", expectedRepresentation(typeName), actual)}
}

func validateTypeSemantics(field SchemaField, value any) *FieldContractIssue {
	switch field.Type {
	case "enum":
		if len(field.Values) == 0 {
			return &FieldContractIssue{Code: "incomplete-type", Expected: "one or more values", Actual: "empty values", Message: "enum has no declared values"}
		}
		if !enumContains(field.Values, value) {
			return &FieldContractIssue{Code: "invalid-enum", Expected: strings.Join(field.Values, ", "), Actual: fmt.Sprint(value), Message: fmt.Sprintf("value %v is not in allowed values", value)}
		}
	case "sequence":
		s, _ := value.(string)
		if field.Prefix == "" || field.Digits <= 0 {
			return &FieldContractIssue{Code: "incomplete-type", Expected: "prefix and positive digits", Actual: "incomplete sequence", Message: "sequence requires prefix and positive digits"}
		}
		if !strings.HasPrefix(s, field.Prefix) || !decimalDigits(s[len(field.Prefix):], field.Digits) {
			return &FieldContractIssue{Code: "invalid-sequence", Expected: fmt.Sprintf("%s plus %d digits", field.Prefix, field.Digits), Actual: s, Message: "sequence value does not match declared format"}
		}
	case "link":
		if !linkValueOK(value) {
			return &FieldContractIssue{Code: "invalid-link", Expected: "wiki-link syntax", Actual: fmt.Sprint(value), Message: "link value must contain wiki-link syntax [[target]]"}
		}
	}
	return nil
}

func representationName(value any) string {
	if value == nil {
		return "null"
	}
	switch value.(type) {
	case string:
		return "string"
	case bool:
		return "boolean"
	case int, int8, int16, int32, int64, uint, uint8, uint16, uint32, uint64:
		return "integer"
	case float32, float64:
		return "number"
	}
	kind := reflect.TypeOf(value).Kind()
	if kind == reflect.Slice || kind == reflect.Array {
		return "sequence"
	}
	if kind == reflect.Map || kind == reflect.Struct {
		return "mapping"
	}
	return fmt.Sprintf("%T", value)
}

func expectedRepresentation(typeName string) string {
	switch typeName {
	case "list":
		return "sequence"
	case "link":
		return "string or sequence"
	case "boolean":
		return "boolean"
	case "integer":
		return "integer"
	default:
		return "string"
	}
}

func legacySectionIssue(name string, field SchemaField) FieldContractIssue {
	target := "type: string\nsource: body.section[\"<exact heading>\"]"
	if field.Heading != "" {
		if source, err := extract.CanonicalSectionSource(field.Heading); err == nil {
			target = "type: string\nsource: " + source
		}
	}
	return issue("legacy-type", "canonical source", "section metadata", fmt.Sprintf("field %q uses legacy section/heading metadata; migrate to:\n%s", name, target))
}

func legacyOrderedIssue(name string, field SchemaField) FieldContractIssue {
	if field.Heading != "" {
		if source, err := extract.CanonicalSectionSource(field.Heading); err == nil {
			target := "type: string\nsource: " + source
			return issue("legacy-type", "canonical source", "ordered", fmt.Sprintf("field %q uses legacy ordered section metadata with a heading; migrate to:\n%s", name, target))
		}
	}
	return issue("legacy-type", "canonical source", "ordered", fmt.Sprintf("field %q uses legacy ordered section metadata; name the source explicitly with a canonical source directive", name))
}

func issue(code, expected, actual, message string) FieldContractIssue {
	return FieldContractIssue{Code: code, Expected: expected, Actual: actual, Message: message}
}

func decimalDigits(s string, want int) bool {
	if len(s) != want {
		return false
	}
	for _, r := range s {
		if r < '0' || r > '9' {
			return false
		}
	}
	return true
}

func linkValueOK(value any) bool {
	if s, ok := value.(string); ok {
		return extract.ContainsWikilink(s)
	}
	rv := reflect.ValueOf(value)
	if !rv.IsValid() || (rv.Kind() != reflect.Slice && rv.Kind() != reflect.Array) {
		return false
	}
	for i := 0; i < rv.Len(); i++ {
		s, ok := rv.Index(i).Interface().(string)
		if !ok || !extract.ContainsWikilink(s) {
			return false
		}
	}
	return true
}
