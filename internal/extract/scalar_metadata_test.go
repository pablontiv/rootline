package extract

import "testing"

func TestMarkdownExtractorPreservesScalarLexemes(t *testing.T) {
	rec, err := (&MarkdownExtractor{}).Extract("a.md", []byte(`---
date: 2026-06-22
timestamp: 2026-06-22T00:00:00Z
boolean: TRUE
octal: 042
signed: +42
quoted: "042"
---
# Probe
`))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}

	want := map[string]FrontmatterScalar{
		"date":      {Lexeme: "2026-06-22", Representation: "timestamp"},
		"timestamp": {Lexeme: "2026-06-22T00:00:00Z", Representation: "timestamp"},
		"boolean":   {Lexeme: "TRUE", Representation: "boolean"},
		"octal":     {Lexeme: "042", Representation: "integer"},
		"signed":    {Lexeme: "+42", Representation: "integer"},
	}
	if len(rec.FrontmatterScalars) != len(want) {
		t.Fatalf("FrontmatterScalars = %#v, want %#v", rec.FrontmatterScalars, want)
	}
	for field, expected := range want {
		if got := rec.FrontmatterScalars[field]; got != expected {
			t.Errorf("%s metadata = %#v, want %#v", field, got, expected)
		}
	}
	if _, ok := rec.FrontmatterScalars["quoted"]; ok {
		t.Error("quoted string must not be marked as a native scalar repair candidate")
	}
}

func TestMarkdownExtractorMalformedYAMLPublishesNoScalarEvidence(t *testing.T) {
	rec, err := (&MarkdownExtractor{}).Extract("a.md", []byte("---\ndate: [broken\n---\nbody\n"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if len(rec.Errors) == 0 {
		t.Fatal("expected malformed YAML extraction error")
	}
	if len(rec.FrontmatterScalars) != 0 {
		t.Fatalf("malformed YAML published repair evidence: %#v", rec.FrontmatterScalars)
	}
}

func TestMarkdownExtractorScalarMetadataSupportsCRLF(t *testing.T) {
	rec, err := (&MarkdownExtractor{}).Extract("a.md", []byte("---\r\ndate: 2026-06-22\r\n---\r\nbody\r\n"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	got := rec.FrontmatterScalars["date"]
	if got.Lexeme != "2026-06-22" || got.Representation != "timestamp" {
		t.Fatalf("date metadata = %#v", got)
	}
}

func TestMarkdownExtractorEmptyFrontmatterRemainsValid(t *testing.T) {
	rec, err := (&MarkdownExtractor{}).Extract("a.md", []byte("---\n---\n# Empty\n"))
	if err != nil {
		t.Fatalf("Extract: %v", err)
	}
	if rec.Frontmatter == nil {
		t.Fatal("empty frontmatter must remain a non-nil map")
	}
	if len(rec.Errors) != 0 || len(rec.Frontmatter) != 0 || len(rec.FrontmatterScalars) != 0 {
		t.Fatalf("empty frontmatter changed contract: %#v", rec)
	}
}

func TestIsRepairableScalarRepresentation(t *testing.T) {
	for _, name := range []string{"timestamp", "boolean", "integer"} {
		if !IsRepairableScalarRepresentation(name) {
			t.Errorf("%q must be repairable", name)
		}
	}
	for _, name := range []string{"", "string", "number", "mapping", "sequence", "null"} {
		if IsRepairableScalarRepresentation(name) {
			t.Errorf("%q must not be repairable", name)
		}
	}
}
