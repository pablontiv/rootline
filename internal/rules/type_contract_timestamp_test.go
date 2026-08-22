package rules

import (
	"testing"

	"gopkg.in/yaml.v3"
)

// A YAML timestamp decodes to time.Time, which is a struct. Reporting every
// struct as "mapping" made an unquoted date indistinguishable from a real
// nested mapping, naming a shape the document does not contain.
func TestValidateFieldValue_DistinguishesTimestampFromMapping(t *testing.T) {
	tests := []struct {
		name        string
		frontmatter string
		wantActual  string
	}{
		{"unquoted date", "ingested: 2026-06-22\n", "timestamp"},
		{"unquoted timestamp", "ingested: 2026-06-22T10:00:00Z\n", "timestamp"},
		{"nested mapping", "ingested:\n  year: 2026\n", "mapping"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var fm map[string]any
			if err := yaml.Unmarshal([]byte(tt.frontmatter), &fm); err != nil {
				t.Fatalf("decoding frontmatter: %v", err)
			}

			issue := ValidateFieldValue(SchemaField{Type: "string"}, fm["ingested"])

			if issue == nil {
				t.Fatalf("want a type-mismatch issue for %q, got none", tt.frontmatter)
			}
			if issue.Actual != tt.wantActual {
				t.Errorf("Actual = %q, want %q", issue.Actual, tt.wantActual)
			}
			want := "expected string, got " + tt.wantActual
			if issue.Message != want {
				t.Errorf("Message = %q, want %q", issue.Message, want)
			}
		})
	}
}

// The control: a quoted date is a string and must stay valid, so the cases
// above are proven to reach real representation validation.
func TestValidateFieldValue_QuotedDateRemainsAString(t *testing.T) {
	var fm map[string]any
	if err := yaml.Unmarshal([]byte("ingested: \"2026-06-22\"\n"), &fm); err != nil {
		t.Fatalf("decoding frontmatter: %v", err)
	}
	if issue := ValidateFieldValue(SchemaField{Type: "string"}, fm["ingested"]); issue != nil {
		t.Fatalf("quoted date should satisfy type string, got %+v", issue)
	}
}
