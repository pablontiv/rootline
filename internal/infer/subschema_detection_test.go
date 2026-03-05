package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestDetectSubSchemas_TypeSpecificField(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"tipo": "test", "ejecutable_en": "1 sesion", "estado": "Pending"}},
		{Path: "b.md", Frontmatter: map[string]any{"tipo": "test", "ejecutable_en": "1 sesion", "estado": "Pending"}},
		{Path: "c.md", Frontmatter: map[string]any{"tipo": "test", "ejecutable_en": "2 sesion", "estado": "Pending"}},
		{Path: "d.md", Frontmatter: map[string]any{"tipo": "implementación", "estado": "Pending"}},
		{Path: "e.md", Frontmatter: map[string]any{"tipo": "implementación", "estado": "Pending"}},
	}

	inferences := DetectSubSchemas(records, "tipo")

	found := false
	for _, inf := range inferences {
		if inf.Type == "type_specific_field" && inf.Field == "ejecutable_en" && inf.Value == "test" {
			found = true
		}
	}
	if !found {
		t.Errorf("expected type_specific_field for ejecutable_en in test group, got %v", inferences)
	}
}

func TestDetectSubSchemas_SingleRecordGroupExcluded(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"tipo": "test", "campo": "x"}},
		{Path: "b.md", Frontmatter: map[string]any{"tipo": "impl", "estado": "y"}},
		{Path: "c.md", Frontmatter: map[string]any{"tipo": "impl", "estado": "z"}},
	}

	inferences := DetectSubSchemas(records, "tipo")

	// "test" group has only 1 record → excluded. Only "impl" remains → <2 valid groups → nil.
	if len(inferences) != 0 {
		t.Errorf("expected no inferences with single-record group, got %v", inferences)
	}
}

func TestDetectSubSchemas_NoDiscriminator(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	inferences := DetectSubSchemas(records, "tipo")
	if len(inferences) != 0 {
		t.Errorf("expected no inferences without discriminator field, got %v", inferences)
	}
}

func TestDetectSubSchemas_DefaultDiscriminator(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"tipo": "a", "x": 1}},
		{Path: "b.md", Frontmatter: map[string]any{"tipo": "a", "x": 2}},
		{Path: "c.md", Frontmatter: map[string]any{"tipo": "b", "y": 1}},
		{Path: "d.md", Frontmatter: map[string]any{"tipo": "b", "y": 2}},
	}

	// Empty discriminator defaults to "tipo"
	inferences := DetectSubSchemas(records, "")

	if len(inferences) == 0 {
		t.Fatal("expected inferences with default discriminator")
	}

	foundX := false
	foundY := false
	for _, inf := range inferences {
		if inf.Field == "x" && inf.Value == "a" {
			foundX = true
		}
		if inf.Field == "y" && inf.Value == "b" {
			foundY = true
		}
	}
	if !foundX || !foundY {
		t.Errorf("expected type-specific fields x→a and y→b, got %v", inferences)
	}
}
