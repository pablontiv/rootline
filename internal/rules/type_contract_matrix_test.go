package rules

import (
	"strings"
	"testing"
)

func TestValidateFieldDeclaration_CanonicalTypeMatrix(t *testing.T) {
	tests := []struct {
		name     string
		field    SchemaField
		wantCode string
	}{
		{"string", SchemaField{Type: "string"}, ""},
		{"list", SchemaField{Type: "list"}, ""},
		{"enum with one value", SchemaField{Type: "enum", Values: []string{"draft"}}, ""},
		{"enum without values", SchemaField{Type: "enum"}, "incomplete-type"},
		{"sequence with positive digits", SchemaField{Type: "sequence", Prefix: "RL-", Digits: 3}, ""},
		{"sequence missing prefix", SchemaField{Type: "sequence", Digits: 3}, "incomplete-type"},
		{"sequence zero digits", SchemaField{Type: "sequence", Prefix: "RL-"}, "incomplete-type"},
		{"sequence negative digits", SchemaField{Type: "sequence", Prefix: "RL-", Digits: -3}, "incomplete-type"},
		{"link", SchemaField{Type: "link"}, ""},
		{"boolean", SchemaField{Type: "boolean"}, ""},
		{"integer", SchemaField{Type: "integer"}, ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateFieldDeclaration("field", tt.field)
			got := firstIssueCode(issues)
			if got != tt.wantCode {
				t.Fatalf("got %q want %q issues=%+v", got, tt.wantCode, issues)
			}
		})
	}
}

func TestValidateFieldDeclaration_SourceContract(t *testing.T) {
	tests := []struct {
		name     string
		source   string
		wantCode string
	}{
		{"omitted source", "", ""},
		{"h1 source", "body.h1", ""},
		{"section source", `body.section["## Notes"]`, ""},
		{"unsupported frontmatter source", "frontmatter.title", "unsupported-source"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issues := ValidateFieldDeclaration("field", SchemaField{Type: "string", Extract: tt.source})
			got := firstIssueCode(issues)
			if got != tt.wantCode {
				t.Fatalf("got %q want %q issues=%+v", got, tt.wantCode, issues)
			}
		})
	}
}

func TestValidateFieldValue_CanonicalTypeMatrix(t *testing.T) {
	tests := []struct {
		name       string
		field      SchemaField
		value      any
		wantCode   string
		wantActual string
	}{
		{"string accepts string", SchemaField{Type: "string"}, "notes", "", ""},
		{"string rejects sequence", SchemaField{Type: "string"}, []string{"notes"}, "type-mismatch", "sequence"},
		{"list accepts sequence", SchemaField{Type: "list"}, []any{"a", 1}, "", ""},
		{"list rejects string", SchemaField{Type: "list"}, "a", "type-mismatch", "string"},
		{"enum accepts declared string", SchemaField{Type: "enum", Values: []string{"draft"}}, "draft", "", ""},
		{"enum rejects undeclared string", SchemaField{Type: "enum", Values: []string{"draft"}}, "done", "invalid-enum", "done"},
		{"sequence accepts exact prefix and digits", SchemaField{Type: "sequence", Prefix: "RL-", Digits: 3}, "RL-007", "", ""},
		{"sequence rejects wrong prefix", SchemaField{Type: "sequence", Prefix: "RL-", Digits: 3}, "XX-007", "invalid-sequence", "XX-007"},
		{"sequence rejects too few digits", SchemaField{Type: "sequence", Prefix: "RL-", Digits: 3}, "RL-07", "invalid-sequence", "RL-07"},
		{"sequence rejects non-decimal suffix", SchemaField{Type: "sequence", Prefix: "RL-", Digits: 3}, "RL-00A", "invalid-sequence", "RL-00A"},
		{"link accepts string containing wikilink", SchemaField{Type: "link"}, "see [[Target]] now", "", ""},
		{"link rejects string without wikilink", SchemaField{Type: "link"}, "Target", "invalid-link", "Target"},
		{"link accepts list of wikilink strings", SchemaField{Type: "link"}, []string{"[[A]]", "see [[B]]"}, "", ""},
		{"link accepts empty list", SchemaField{Type: "link"}, []string{}, "", ""},
		{"link rejects list item without wikilink", SchemaField{Type: "link"}, []string{"[[A]]", "B"}, "invalid-link", "[[[A]] B]"},
		{"link rejects non-string list item", SchemaField{Type: "link"}, []any{"[[A]]", 3}, "invalid-link", "[[[A]] 3]"},
		{"boolean accepts bool", SchemaField{Type: "boolean"}, true, "", ""},
		{"boolean rejects quoted bool", SchemaField{Type: "boolean"}, "true", "type-mismatch", "string"},
		{"integer accepts int", SchemaField{Type: "integer"}, int64(3), "", ""},
		{"integer rejects quoted int", SchemaField{Type: "integer"}, "3", "type-mismatch", "string"},
		{"integer rejects float", SchemaField{Type: "integer"}, 3.0, "type-mismatch", "number"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			issue := ValidateFieldValue(tt.field, tt.value)
			gotCode, gotActual := "", ""
			if issue != nil {
				gotCode, gotActual = issue.Code, issue.Actual
			}
			if gotCode != tt.wantCode || gotActual != tt.wantActual {
				t.Fatalf("got code=%q actual=%q, want code=%q actual=%q issue=%+v", gotCode, gotActual, tt.wantCode, tt.wantActual, issue)
			}
		})
	}
}

func firstIssueCode(issues []FieldContractIssue) string {
	if len(issues) == 0 {
		return ""
	}
	return issues[0].Code
}

func TestLegacyOrderedMigrationGuidance(t *testing.T) {
	orderedOnly := mustParseField(t, "type: string\nordered: 1\n")
	orderedOnlyIssue := ValidateFieldDeclaration("field", orderedOnly)[0]
	if strings.Contains(orderedOnlyIssue.Message, "body.section[") || !strings.Contains(orderedOnlyIssue.Message, "name the source") {
		t.Fatalf("ordered-only guidance should require an explicit source without guessing: %q", orderedOnlyIssue.Message)
	}

	withHeading := mustParseField(t, "type: string\nheading: \"## Notes\"\nordered: 1\n")
	withHeadingIssue := ValidateFieldDeclaration("field", withHeading)[0]
	for _, want := range []string{"type: string", `source: body.section["## Notes"]`} {
		if !strings.Contains(withHeadingIssue.Message, want) {
			t.Fatalf("ordered+heading guidance %q does not contain %q", withHeadingIssue.Message, want)
		}
	}
}
