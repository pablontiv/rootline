package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestDetectConstantFieldsAllSameValue(t *testing.T) {
	// 5 records all with estado: Specified → constant_field
	records := make([]*extract.Record, 5)
	for i := range records {
		records[i] = &extract.Record{
			Path:        "doc.md",
			Frontmatter: map[string]any{"estado": "Specified"},
		}
	}

	got := DetectConstantFields(records)
	if len(got) != 1 {
		t.Fatalf("expected 1 constant_field inference, got %d", len(got))
	}
	if got[0].Type != "constant_field" {
		t.Errorf("expected type constant_field, got %s", got[0].Type)
	}
	if got[0].Field != "estado" {
		t.Errorf("expected field estado, got %s", got[0].Field)
	}
	if got[0].Value != "Specified" {
		t.Errorf("expected value Specified, got %s", got[0].Value)
	}
	if got[0].Category != 8 {
		t.Errorf("expected category 8, got %d", got[0].Category)
	}
}

func TestDetectConstantFieldsTwoDistinctValues(t *testing.T) {
	// Field with 2 distinct values → no inference
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Specified"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "c.md", Frontmatter: map[string]any{"estado": "Specified"}},
	}

	got := DetectConstantFields(records)
	if len(got) != 0 {
		t.Errorf("expected no inferences for field with 2 distinct values, got %d", len(got))
	}
}

func TestDetectConstantFieldsTooFewRecords(t *testing.T) {
	// Less than 3 records → no inference
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Specified"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Specified"}},
	}

	got := DetectConstantFields(records)
	if len(got) != 0 {
		t.Errorf("expected no inferences with <3 records, got %d", len(got))
	}
}

func TestDetectConstantFieldsNotPresentInAll(t *testing.T) {
	// Field present in 2 of 3 records → not constant
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Specified"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Specified"}},
		{Path: "c.md", Frontmatter: map[string]any{}},
	}

	got := DetectConstantFields(records)
	if len(got) != 0 {
		t.Errorf("expected no inferences when field not in all records, got %d", len(got))
	}
}
