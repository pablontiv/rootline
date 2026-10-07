package extract

import (
	"reflect"
	"strings"
	"testing"
)

func TestParseBodySource(t *testing.T) {
	tests := []struct {
		directive string
		want      BodySource
	}{
		{
			directive: "body.h1",
			want:      BodySource{Kind: BodySourceH1},
		},
		{
			directive: `body.section["## Notes"]`,
			want: BodySource{
				Kind:     BodySourceSection,
				Heading:  "## Notes",
				Selector: SectionSelector{{Level: 2, Text: "Notes"}},
			},
		},
		{
			directive: `body.section["## Parent"]["### Notes"]`,
			want: BodySource{
				Kind:    BodySourceSection,
				Heading: "### Notes",
				Selector: SectionSelector{
					{Level: 2, Text: "Parent"},
					{Level: 3, Text: "Notes"},
				},
			},
		},
	}
	for _, tt := range tests {
		got, err := ParseBodySource(tt.directive)
		if err != nil || !reflect.DeepEqual(got, tt.want) {
			t.Fatalf("ParseBodySource(%q) = %+v, %v; want %+v", tt.directive, got, err, tt.want)
		}
	}
}

func TestParseBodySourceRejectsInvalidSelectors(t *testing.T) {
	for _, tt := range []struct{ directive, wantErr string }{
		{"body.title", "unsupported"},
		{`body.section[## Notes]`, "malformed"},
		{`body.section["Notes"]`, "heading"},
		{`body.section["## Parent"]["## Notes"]`, "increase"},
		{`body.section["### Parent"]["## Notes"]`, "increase"},
	} {
		_, err := ParseBodySource(tt.directive)
		if err == nil || !strings.Contains(err.Error(), tt.wantErr) {
			t.Fatalf("ParseBodySource(%q) error = %v; want text %q", tt.directive, err, tt.wantErr)
		}
	}
}

func TestParseBodySourceAllowsLevelJump(t *testing.T) {
	got, err := ParseBodySource(`body.section["# Root"]["#### Detail"]`)
	want := SectionSelector{{Level: 1, Text: "Root"}, {Level: 4, Text: "Detail"}}
	if err != nil || !reflect.DeepEqual(got.Selector, want) {
		t.Fatalf("selector = %+v, %v; want %+v", got.Selector, err, want)
	}
}

func TestCanonicalSectionSource(t *testing.T) {
	got, err := CanonicalSectionSource("## Notes")
	if err != nil {
		t.Fatalf("CanonicalSectionSource returned an error: %v", err)
	}
	if got != `body.section["## Notes"]` {
		t.Fatalf("CanonicalSectionSource() = %q", got)
	}

	got, err = CanonicalSectionSelectorSource(SectionSelector{
		{Level: 2, Text: "Parent"},
		{Level: 4, Text: `Quote "value"`},
	})
	if err != nil {
		t.Fatalf("CanonicalSectionSelectorSource returned an error: %v", err)
	}
	if got != `body.section["## Parent"]["#### Quote \"value\""]` {
		t.Fatalf("CanonicalSectionSelectorSource() = %q", got)
	}
	parsed, err := ParseBodySource(got)
	if err != nil || len(parsed.Selector) != 2 || parsed.Selector[1].Text != `Quote "value"` {
		t.Fatalf("serialized selector parsed as %+v, %v", parsed, err)
	}
}

func TestResolveBodyValue_PresentEmptySection(t *testing.T) {
	rec := &Record{BodySections: []Section{{Heading: "Notes", Level: 2, Content: ""}}}
	got, present, err := ResolveBodyValue(rec, `body.section["## Notes"]`)
	if err != nil || !present || got != "" {
		t.Fatalf("value=%q present=%v error=%v; want a present empty value", got, present, err)
	}
}

func TestResolveBodyValue_SelectorMatching(t *testing.T) {
	rec := &Record{Body: "# Root\n\n## Parent A\n\n### Notes\n\nfirst\n\n## Parent B\n\n### Notes\n\nsecond\n\n#### Detail\n\ndetail\n\n# Other Root\n\n## Parent B\n\n### Notes\n\nthird\n"}
	tests := []struct {
		name      string
		directive string
		want      string
		present   bool
		ambiguous bool
	}{
		{name: "simple selector is ambiguous", directive: `body.section["### Notes"]`, ambiguous: true},
		{name: "qualified selector distinguishes parents", directive: `body.section["## Parent A"]["### Notes"]`, want: "first", present: true},
		{name: "partial suffix has one match", directive: `body.section["### Notes"]["#### Detail"]`, want: "detail", present: true},
		{name: "partial suffix has no match", directive: `body.section["## Missing"]["### Notes"]`},
		{name: "partial suffix has multiple matches", directive: `body.section["## Parent B"]["### Notes"]`, ambiguous: true},
		{name: "full suffix has one match", directive: `body.section["# Root"]["## Parent B"]["### Notes"]`, want: "second", present: true},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, present, err := ResolveBodyValue(rec, tt.directive)
			if tt.ambiguous {
				if err == nil || !strings.Contains(err.Error(), "ambiguous") {
					t.Fatalf("error = %v; want ambiguity", err)
				}
				return
			}
			if err != nil || present != tt.present || got != tt.want {
				t.Fatalf("value=%q present=%v error=%v; want value=%q present=%v", got, present, err, tt.want, tt.present)
			}
		})
	}
}

func TestResolveBodyValue_SelectorCannotOmitDirectParent(t *testing.T) {
	rec := &Record{Body: "# Root\n\n## Parent\n\n### Intermediate\n\n#### Notes\n\nvalue\n"}
	_, present, err := ResolveBodyValue(rec, `body.section["## Parent"]["#### Notes"]`)
	if err != nil || present {
		t.Fatalf("present=%v error=%v; want no match", present, err)
	}
	got, present, err := ResolveBodyValue(rec, `body.section["### Intermediate"]["#### Notes"]`)
	if err != nil || !present || got != "value" {
		t.Fatalf("value=%q present=%v error=%v; want the direct-parent match", got, present, err)
	}
}

func TestResolveBodyValue_DuplicateFullPathIsAmbiguous(t *testing.T) {
	rec := &Record{Body: "# Root\n\n## Parent\n\n### Notes\n\nfirst\n\n### Notes\n\nsecond\n"}
	_, _, err := ResolveBodyValue(rec, `body.section["# Root"]["## Parent"]["### Notes"]`)
	if err == nil || !strings.Contains(err.Error(), "ambiguous") {
		t.Fatalf("error = %v; want ambiguity", err)
	}
}

func TestMarkdownExtractor_PreservesPathsAndFencedCodeExclusion(t *testing.T) {
	content := []byte("---\ntitle: Fixture\n---\nRoot\n====\n\n```\n## Fake\n```\n\n~~~~\n## Fake Two\n~~~~ not a closing fence\n## Fake Three\n~~~~\n\n## Empty\n\n## Parent *A*\n\n### Duplicate\n\nfirst\n\n## Parent B\n\n### Duplicate\n\nsecond\n")
	parseAST := true
	extractors := map[string]*MarkdownExtractor{
		"ast":  {ParseAST: &parseAST},
		"text": {},
	}

	var baseline []Section
	for name, ext := range extractors {
		rec, err := ext.Extract(name+".md", content)
		if err != nil {
			t.Fatalf("%s extraction failed: %v", name, err)
		}
		if got := len(rec.BodySections); got != 6 {
			t.Fatalf("%s section count = %d; want 6: %+v", name, got, rec.BodySections)
		}
		if rec.BodySections[1].Heading != "Empty" || rec.BodySections[1].Content != "" {
			t.Fatalf("%s empty section = %+v", name, rec.BodySections[1])
		}
		wantLastPath := SectionPath{
			{Level: 1, Text: "Root"},
			{Level: 2, Text: "Parent B"},
			{Level: 3, Text: "Duplicate"},
		}
		if !reflect.DeepEqual(rec.BodySections[5].Path, wantLastPath) {
			t.Fatalf("%s final path = %+v; want %+v", name, rec.BodySections[5].Path, wantLastPath)
		}
		for _, sec := range rec.BodySections {
			if strings.HasPrefix(sec.Heading, "Fake") {
				t.Fatalf("%s included a fenced heading: %+v", name, rec.BodySections)
			}
		}
		if baseline == nil {
			baseline = rec.BodySections
		} else if !reflect.DeepEqual(baseline, rec.BodySections) {
			t.Fatalf("text sections differ from AST sections: AST=%+v text=%+v", baseline, rec.BodySections)
		}
	}
}

func TestSectionPathsCloseBranches(t *testing.T) {
	sections := ExtractSectionsFromText("# First\n\n### Jump\n\n#### Leaf\n\n## Sibling\n\n# Second\n\n### Other\n")
	want := []SectionPath{
		{{Level: 1, Text: "First"}},
		{{Level: 1, Text: "First"}, {Level: 3, Text: "Jump"}},
		{{Level: 1, Text: "First"}, {Level: 3, Text: "Jump"}, {Level: 4, Text: "Leaf"}},
		{{Level: 1, Text: "First"}, {Level: 2, Text: "Sibling"}},
		{{Level: 1, Text: "Second"}},
		{{Level: 1, Text: "Second"}, {Level: 3, Text: "Other"}},
	}
	if len(sections) != len(want) {
		t.Fatalf("section count = %d; want %d", len(sections), len(want))
	}
	for i := range sections {
		if !reflect.DeepEqual(sections[i].Path, want[i]) {
			t.Fatalf("path %d = %+v; want %+v", i, sections[i].Path, want[i])
		}
	}
}
