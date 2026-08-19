package extract

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseBodySource(t *testing.T) {
	got, err := ParseBodySource("body.h1")
	if err != nil || got != (BodySource{Kind: BodySourceH1}) {
		t.Fatalf("body.h1 parsed as %+v err=%v", got, err)
	}
	got, err = ParseBodySource(`body.section["## Notes"]`)
	if err != nil || got != (BodySource{Kind: BodySourceSection, Heading: "## Notes"}) {
		t.Fatalf("section parsed as %+v err=%v", got, err)
	}
	for _, tt := range []struct{ directive, wantErr string }{
		{"body.title", "unsupported"},
		{`body.section[## Notes]`, "malformed"},
		{`body.section["Notes"]`, "heading"},
	} {
		_, err := ParseBodySource(tt.directive)
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Fatalf("%s: expected %q error, got %v", tt.directive, tt.wantErr, err)
		}
	}
}

func TestCanonicalSectionSource(t *testing.T) {
	got, err := CanonicalSectionSource("## Notes")
	if err != nil {
		t.Fatalf("CanonicalSectionSource: %v", err)
	}
	if got != `body.section["## Notes"]` {
		t.Fatalf("CanonicalSectionSource() = %q", got)
	}
}

func TestResolveBodyValue_PresentEmptySection(t *testing.T) {
	rec := &Record{BodySections: []Section{{Heading: "Notes", Level: 2, Content: ""}}}
	got, present, err := ResolveBodyValue(rec, `body.section["## Notes"]`)
	if err != nil || !present || got != "" {
		t.Fatalf("got value=%q present=%v err=%v", got, present, err)
	}
}

func TestResolveBodyValue_DuplicateSectionIsAmbiguous(t *testing.T) {
	rec := &Record{BodySections: []Section{
		{Heading: "Notes", Level: 2, Content: "first"},
		{Heading: "Notes", Level: 2, Content: "second"},
	}}
	_, _, err := ResolveBodyValue(rec, `body.section["## Notes"]`)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("expected ambiguity error, got %v", err)
	}
}

func TestMarkdownExtractor_PreservesDuplicateSectionsAndFencedCodeExclusion(t *testing.T) {
	content := []byte("---\ntitle: Fixture\n---\nTitle\n=====\n\n```\n## Fake\n```\n\nEmpty\n-----\n\n## Duplicate\n\nfirst\n\n## Duplicate\n\nsecond\n")
	parseAST := true
	extractors := map[string]*MarkdownExtractor{
		"ast":  {ParseAST: &parseAST},
		"text": {},
	}

	var baseline []Section
	for name, ext := range extractors {
		rec, err := ext.Extract(name+".md", content)
		if err != nil {
			t.Fatalf("%s Extract: %v", name, err)
		}
		if got := len(rec.BodySections); got != 4 {
			t.Fatalf("%s BodySections len = %d, want 4: %+v", name, got, rec.BodySections)
		}
		if rec.BodySections[1].Heading != "Empty" || rec.BodySections[1].Level != 2 || rec.BodySections[1].Content != "" {
			t.Fatalf("%s empty section = %+v", name, rec.BodySections[1])
		}
		for _, sec := range rec.BodySections {
			if sec.Heading == "Fake" {
				t.Fatalf("%s included fenced code heading: %+v", name, rec.BodySections)
			}
		}
		if got, present, err := ResolveBodyValue(rec, "body.h1"); err != nil || !present || got != "Title" {
			t.Fatalf("%s h1 resolution value=%q present=%v err=%v", name, got, present, err)
		}
		if _, present, err := ResolveBodyValue(rec, `body.section["## Empty"]`); err != nil || !present {
			t.Fatalf("%s empty resolution present=%v err=%v", name, present, err)
		}
		if _, _, err := ResolveBodyValue(rec, `body.section["## Duplicate"]`); err == nil || !strings.Contains(err.Error(), "ambiguous") {
			t.Fatalf("%s expected duplicate ambiguity, got %v", name, err)
		}
		if baseline == nil {
			baseline = rec.BodySections
		} else if !reflect.DeepEqual(baseline, rec.BodySections) {
			t.Fatalf("text-backed sections differ from AST: AST=%+v got=%+v", baseline, rec.BodySections)
		}
	}
}
