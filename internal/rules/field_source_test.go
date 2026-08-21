package rules

import (
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestResolveFieldValue_FrontmatterPresenceOverridesBodySource(t *testing.T) {
	for _, value := range []any{"", nil} {
		rec := &extract.Record{
			Frontmatter: map[string]any{"notes": value},
			BodySections: []extract.Section{
				{Heading: "Notes", Level: 2, Content: "first"},
				{Heading: "Notes", Level: 2, Content: "second"},
			},
		}

		got, present, err := ResolveFieldValue(rec, "notes", SchemaField{Extract: `body.section["## Notes"]`})
		if err != nil || !present || got != value {
			t.Fatalf("got value=%#v present=%v err=%v; want frontmatter value %#v", got, present, err, value)
		}
	}
}

func TestResolveFieldValue_PresentEmptyBodySection(t *testing.T) {
	rec := &extract.Record{
		Frontmatter:  map[string]any{},
		BodySections: []extract.Section{{Heading: "Notes", Level: 2, Content: ""}},
	}

	got, present, err := ResolveFieldValue(rec, "notes", SchemaField{Extract: `body.section["## Notes"]`})
	if err != nil || !present || got != "" {
		t.Fatalf("got value=%#v present=%v err=%v; want present empty body section", got, present, err)
	}
}

func TestResolveFieldValue_BodySourceAmbiguityPropagates(t *testing.T) {
	rec := &extract.Record{
		Frontmatter: map[string]any{},
		BodySections: []extract.Section{
			{Heading: "Notes", Level: 2, Content: "first"},
			{Heading: "Notes", Level: 2, Content: "second"},
		},
	}

	_, _, err := ResolveFieldValue(rec, "notes", SchemaField{Extract: `body.section["## Notes"]`})
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
}

func TestResolveEffectiveField_SourceBackedFieldUsesSchemaContract(t *testing.T) {
	rec := &extract.Record{
		Frontmatter: map[string]any{"title": "frontmatter"},
		Body:        "# Body Title\n",
		Derived:     map[string]any{"title": "derived"},
	}
	effective := &StemFile{Schema: map[string]SchemaField{"title": {Extract: "body.h1"}}}

	got, present, err := ResolveEffectiveField(rec, effective, "title")
	if err != nil || !present || got != "frontmatter" {
		t.Fatalf("got value=%#v present=%v err=%v; want frontmatter", got, present, err)
	}
}

func TestResolveEffectiveField_FieldWithoutSourcePreservesDerivedPrecedence(t *testing.T) {
	rec := &extract.Record{
		Frontmatter: map[string]any{"status": "frontmatter"},
		Derived:     map[string]any{"status": "derived"},
	}
	effective := &StemFile{Schema: map[string]SchemaField{"status": {Type: "string"}}}

	got, present, err := ResolveEffectiveField(rec, effective, "status")
	if err != nil || !present || got != "derived" {
		t.Fatalf("got value=%#v present=%v err=%v; want derived", got, present, err)
	}
}
