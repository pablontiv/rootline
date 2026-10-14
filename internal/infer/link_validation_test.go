package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestDetectLinkTypesNoLinks(t *testing.T) {
	records := makeRecords(map[string]any{"title": "a"})
	got := DetectLinkTypes(records, nil)
	if len(got) != 0 {
		t.Errorf("expected no inferences for records without links, got %d", len(got))
	}
}

func TestDetectLinkTypesViolation(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "a.md",
			Frontmatter: map[string]any{},
			Links: []extract.Link{
				{Target: "b.md", Type: "blocks", Line: 1},
				{Target: "c.md", Type: "invalid", Line: 2},
			},
		},
	}
	schema := rules.LinkSchema{Allowed: []string{"blocks", "reference"}}

	got := DetectLinkTypes(records, map[string]rules.LinkSchema{"a.md": schema})
	if len(got) != 1 {
		t.Fatalf("expected 1 violation, got %d", len(got))
	}
	if got[0].Type != "link_type_violation" {
		t.Errorf("expected type link_type_violation, got %s", got[0].Type)
	}
	if got[0].Value != "invalid" {
		t.Errorf("expected value 'invalid', got %s", got[0].Value)
	}
}

func TestDetectLinkTypesViolationDedup(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "a.md",
			Frontmatter: map[string]any{},
			Links: []extract.Link{
				{Target: "b.md", Type: "bad", Line: 1},
				{Target: "c.md", Type: "bad", Line: 2},
			},
		},
	}
	schema := rules.LinkSchema{Allowed: []string{"reference"}}

	got := DetectLinkTypes(records, map[string]rules.LinkSchema{"a.md": schema})
	if len(got) != 1 {
		t.Errorf("expected 1 deduped violation, got %d", len(got))
	}
}

func TestDetectLinkTypesAllAllowed(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "a.md",
			Frontmatter: map[string]any{},
			Links: []extract.Link{
				{Target: "b.md", Type: "blocks", Line: 1},
				{Target: "c.md", Type: "reference", Line: 2},
			},
		},
	}
	schema := rules.LinkSchema{Allowed: []string{"blocks", "reference"}}

	got := DetectLinkTypes(records, map[string]rules.LinkSchema{"a.md": schema})
	if len(got) != 0 {
		t.Errorf("expected no violations when all links are allowed, got %d", len(got))
	}
}

func TestDetectLinkTypesSuggestion(t *testing.T) {
	// 10 links, 9 are "reference" (90% > 80%), 1 is "blocks" (10% < 80%)
	var links []extract.Link
	for i := 0; i < 9; i++ {
		links = append(links, extract.Link{Target: "x.md", Type: "reference", Line: i + 1})
	}
	links = append(links, extract.Link{Target: "y.md", Type: "blocks", Line: 10})

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{}, Links: links},
	}
	got := DetectLinkTypes(records, nil)
	if len(got) != 1 {
		t.Fatalf("expected 1 suggestion, got %d", len(got))
	}
	if got[0].Type != "link_type_suggestion" {
		t.Errorf("expected type link_type_suggestion, got %s", got[0].Type)
	}
}

func TestDetectLinkTypesNoSuggestionBelowThreshold(t *testing.T) {
	// 4 links split evenly: 2 reference, 2 blocks — neither hits 80%
	records := []*extract.Record{
		{
			Path:        "a.md",
			Frontmatter: map[string]any{},
			Links: []extract.Link{
				{Target: "b.md", Type: "reference", Line: 1},
				{Target: "c.md", Type: "reference", Line: 2},
				{Target: "d.md", Type: "blocks", Line: 3},
				{Target: "e.md", Type: "blocks", Line: 4},
			},
		},
	}

	got := DetectLinkTypes(records, nil)
	if len(got) != 0 {
		t.Errorf("expected no suggestions below threshold, got %d", len(got))
	}
}

func TestDetectLinkTypesPerRecordSchemas(t *testing.T) {
	records := []*extract.Record{
		{Path: "allowed.md", Links: []extract.Link{{Type: "blocks"}}},
		{Path: "forbidden.md", Links: []extract.Link{{Type: "blocks"}, {Type: "blocks"}, {Type: "extends"}}},
		{Path: "bootstrap.md", Links: []extract.Link{{Type: "reference"}}},
		{Path: "empty.md", Links: []extract.Link{{Type: "reference"}}},
	}
	schemas := map[string]rules.LinkSchema{
		"allowed.md":   {Allowed: []string{"blocks"}},
		"forbidden.md": {Allowed: []string{"reference"}},
		"empty.md":     {},
	}

	got := DetectLinkTypes(records, schemas)
	if len(got) != 3 {
		t.Fatalf("expected 2 deduped violations and 1 global suggestion, got %v", got)
	}
	for i, value := range []string{"blocks", "extends"} {
		if got[i].Type != "link_type_violation" || got[i].Source != "forbidden.md" || got[i].Value != value {
			t.Errorf("violation %d = %+v, want forbidden.md %s", i, got[i], value)
		}
	}
	if got[2].Type != "link_type_suggestion" || got[2].Source != "" || got[2].Value != "[reference]" {
		t.Errorf("global suggestion = %+v, want [reference] from records without allowed lists", got[2])
	}
}
