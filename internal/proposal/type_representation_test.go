package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func typeError(field, expected, actual, message string) rules.ValidationError {
	return rules.ValidationError{
		Rule:                   "type",
		Field:                  field,
		Message:                message,
		ExpectedRepresentation: expected,
		ActualRepresentation:   actual,
	}
}

func TestDetectTypeRepresentationRepairsPreservesExactLexemes(t *testing.T) {
	record := &extract.Record{
		Path: "a.md",
		FrontmatterScalars: map[string]extract.FrontmatterScalar{
			"date":    {Lexeme: "2026-06-22T00:00:00Z", Representation: "timestamp"},
			"boolean": {Lexeme: "TRUE", Representation: "boolean"},
			"integer": {Lexeme: "042", Representation: "integer"},
		},
	}
	errs := map[string][]rules.ValidationError{"a.md": {
		typeError("date", "string", "timestamp", "message wording is irrelevant"),
		typeError("boolean", "string", "boolean", "another message"),
		typeError("integer", "string", "integer", "changed prose"),
	}}

	proposals, findings := detectTypeRepresentationRepairs([]*extract.Record{record}, errs)
	if len(findings) != 0 || len(proposals) != 3 {
		t.Fatalf("proposals=%#v findings=%#v", proposals, findings)
	}
	want := map[string]string{"date": "2026-06-22T00:00:00Z", "boolean": "TRUE", "integer": "042"}
	for _, p := range proposals {
		if p.Type != CorrectValue || p.From != want[p.Field] || p.To != want[p.Field] {
			t.Errorf("proposal = %#v", p)
		}
		if p.FromRepresentation != record.FrontmatterScalars[p.Field].Representation {
			t.Errorf("representation = %q for %s", p.FromRepresentation, p.Field)
		}
	}
}

func TestEveryTypeErrorBecomesOneRepairOrOneFinding(t *testing.T) {
	record := &extract.Record{
		Path: "a.md",
		FrontmatterScalars: map[string]extract.FrontmatterScalar{
			"safe":     {Lexeme: "+42", Representation: "integer"},
			"mismatch": {Lexeme: "TRUE", Representation: "boolean"},
		},
	}
	errs := map[string][]rules.ValidationError{"a.md": {
		typeError("safe", "string", "integer", "safe"),
		typeError("mapping", "string", "mapping", "mapping"),
		typeError("sequence", "string", "sequence", "sequence"),
		typeError("null", "string", "null", "null"),
		typeError("number", "string", "number", "number"),
		typeError("inverse", "boolean", "string", "inverse"),
		typeError("mismatch", "string", "integer", "metadata disagrees"),
	}}

	proposals, findings := detectTypeRepresentationRepairs([]*extract.Record{record}, errs)
	if len(proposals) != 1 || len(findings) != 6 {
		t.Fatalf("proposals=%d findings=%d", len(proposals), len(findings))
	}
	if len(proposals)+len(findings) != len(errs["a.md"]) {
		t.Fatal("a type error was duplicated or dropped")
	}
}
