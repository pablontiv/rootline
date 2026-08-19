package rules

import (
	"context"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestValidateFieldDeclaration(t *testing.T) {
	tests := []struct {
		name     string
		field    SchemaField
		wantCode string
	}{
		{"single enum", SchemaField{Type: "enum", Values: []string{"theory"}}, ""},
		{"empty enum", SchemaField{Type: "enum"}, "incomplete-type"},
		{"boolean", SchemaField{Type: "boolean"}, ""},
		{"legacy bool", SchemaField{Type: "bool"}, "legacy-type"},
		{"integer", SchemaField{Type: "integer"}, ""},
		{"negative sequence digits", SchemaField{Type: "sequence", Prefix: "ID-", Digits: -1}, "incomplete-type"},
		{"unknown", SchemaField{Type: "number"}, "unknown-type"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateFieldDeclaration("field", tt.field)
			got := ""
			if len(issues) > 0 {
				got = issues[0].Code
			}
			if got != tt.wantCode {
				t.Fatalf("got %q want %q issues=%+v", got, tt.wantCode, issues)
			}
		})
	}
}

func TestValidateFieldValue(t *testing.T) {
	check := func(field SchemaField, value any, wantCode, wantActual string) {
		issue := ValidateFieldValue(field, value)
		gotCode, gotActual := "", ""
		if issue != nil {
			gotCode, gotActual = issue.Code, issue.Actual
		}
		if gotCode != wantCode || gotActual != wantActual {
			t.Fatalf("got code=%q actual=%q issue=%+v", gotCode, gotActual, issue)
		}
	}
	check(SchemaField{Type: "boolean"}, true, "", "")
	check(SchemaField{Type: "boolean"}, "true", "type-mismatch", "string")
	check(SchemaField{Type: "integer"}, 3, "", "")
	check(SchemaField{Type: "integer"}, "3", "type-mismatch", "string")
	check(SchemaField{Type: "integer"}, 3.0, "type-mismatch", "number")
	check(SchemaField{Type: "enum", Values: []string{"theory"}}, "theory", "", "")
}

func TestSchemaFieldYAMLMetadata_LegacyMigrations(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		contains []string
	}{
		{"legacy bool", "type: bool\n", []string{"type: boolean"}},
		{"legacy section", "type: section\nheading: \"## Notes\"\n", []string{"type: string", "source: body.section[\"## Notes\"]"}},
		{"legacy heading", "type: string\nheading: \"## Notes\"\n", []string{"heading", "source: body.section[\"## Notes\"]"}},
		{"legacy ordered without heading", "type: string\nordered: 2\n", []string{"ordered", "name the source"}},
		{"legacy ordered with heading", "type: string\nheading: \"## Notes\"\nordered: 2\n", []string{"ordered", "type: string", "source: body.section[\"## Notes\"]"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			field := mustParseField(t, tt.body)
			issues := ValidateFieldDeclaration("field", field)
			if len(issues) == 0 || issues[0].Code != "legacy-type" {
				t.Fatalf("got issues=%+v, want legacy-type", issues)
			}
			for _, want := range tt.contains {
				if !strings.Contains(issues[0].Message, want) {
					t.Fatalf("message %q does not contain %q", issues[0].Message, want)
				}
			}
		})
	}
}

func TestSchemaFieldJSONKeepsLegacyPublicFieldsOnly(t *testing.T) {
	ordered := 2
	field := SchemaField{Type: "string", Heading: "## Notes", Ordered: &ordered}

	data, err := json.Marshal(field)
	if err != nil {
		t.Fatalf("marshal SchemaField: %v", err)
	}
	jsonText := string(data)
	for _, want := range []string{`"heading":"## Notes"`, `"ordered":2`} {
		if !strings.Contains(jsonText, want) {
			t.Fatalf("JSON %s does not contain %s", jsonText, want)
		}
	}
	if strings.Contains(jsonText, "declaration") || strings.Contains(jsonText, "Present") || strings.Contains(jsonText, "Null") {
		t.Fatalf("JSON exposes private declaration metadata: %s", jsonText)
	}
}

func TestValidateStemHealth_RejectsFieldDeclarationFailures(t *testing.T) {
	dir := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
schema:
  old_bool:
    type: bool
  old_section:
    type: section
    heading: "## Notes"
  unknown:
    type: number
  empty_enum:
    type: enum
  unsupported_source:
    type: string
    source: frontmatter.title
`))

	result, err := ValidateStemHealth(context.Background(), dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	want := map[string]string{
		"old_bool":           "type: boolean",
		"old_section":        "source: body.section[\"## Notes\"]",
		"unknown":            "unknown-type",
		"empty_enum":         "incomplete-type",
		"unsupported_source": "unsupported-source",
	}
	for field, fragment := range want {
		found := false
		for _, c := range result.Checks {
			if c.Name == "field-declaration" && c.Status == "fail" && c.Field == field && strings.Contains(c.Message, fragment) {
				found = true
				break
			}
		}
		if !found {
			t.Fatalf("missing field-declaration failure for %s containing %q: %+v", field, fragment, result.Checks)
		}
	}
}

func mustParseField(t *testing.T, body string) SchemaField {
	t.Helper()
	yamlText := "version: 2\nschema:\n  field:\n    " + strings.ReplaceAll(body, "\n", "\n    ")
	stem, err := ParseStem(".stem", []byte(yamlText))
	if err != nil {
		t.Fatal(err)
	}
	return stem.Schema["field"]
}
