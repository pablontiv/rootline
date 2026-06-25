package rules

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestExtractionErrors_NoErrors(t *testing.T) {
	rec := &extract.Record{Path: "ok.md"}
	got := ExtractionErrors(rec)
	if got != nil {
		t.Errorf("expected nil for a record without extraction errors, got %v", got)
	}
}

func TestExtractionErrors_MalformedYAML(t *testing.T) {
	rec := &extract.Record{
		Path: "broken.md",
		Errors: []extract.ExtractionError{
			{Line: 1, Message: "malformed YAML frontmatter: yaml: mapping values are not allowed in this context"},
		},
	}
	got := ExtractionErrors(rec)
	if len(got) != 1 {
		t.Fatalf("expected 1 validation error, got %d", len(got))
	}
	e := got[0]
	if e.Rule != "malformed_yaml" {
		t.Errorf("expected rule malformed_yaml, got %q", e.Rule)
	}
	if e.Severity != "error" {
		t.Errorf("expected severity error, got %q", e.Severity)
	}
	if e.Field != "_frontmatter" {
		t.Errorf("expected field _frontmatter, got %q", e.Field)
	}
	if e.Source != "broken.md" {
		t.Errorf("expected source broken.md, got %q", e.Source)
	}
	if e.Message != "malformed YAML frontmatter: yaml: mapping values are not allowed in this context" {
		t.Errorf("expected message passthrough, got %q", e.Message)
	}
	if e.Suggestion == "" {
		t.Errorf("expected a non-empty suggestion")
	}
}

func TestExtractionErrors_MultipleErrors(t *testing.T) {
	rec := &extract.Record{
		Path: "broken.md",
		Errors: []extract.ExtractionError{
			{Line: 1, Message: "err one"},
			{Line: 2, Message: "err two"},
		},
	}
	got := ExtractionErrors(rec)
	if len(got) != 2 {
		t.Fatalf("expected 2 validation errors, got %d", len(got))
	}
}
