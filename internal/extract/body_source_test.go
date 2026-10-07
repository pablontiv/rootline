package extract

import (
	"reflect"
	"strings"
	"testing"

	"github.com/yuin/goldmark"
	gmtext "github.com/yuin/goldmark/text"
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
		{"body.section[\"## First\nSecond\"]", "malformed"},
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

func TestCanonicalSectionSelectorSource_MultilineHeadingRoundTrip(t *testing.T) {
	wantSelector := SectionSelector{
		{Level: 1, Text: "Root"},
		{Level: 2, Text: "First\nSecond"},
	}
	got, err := CanonicalSectionSelectorSource(wantSelector)
	if err != nil {
		t.Fatalf("CanonicalSectionSelectorSource returned an error: %v", err)
	}
	wantSource := `body.section["# Root"]["## First\nSecond"]`
	if got != wantSource {
		t.Fatalf("source = %q, want %q", got, wantSource)
	}

	parsed, err := ParseBodySource(got)
	if err != nil {
		t.Fatalf("ParseBodySource returned an error: %v", err)
	}
	if !reflect.DeepEqual(parsed.Selector, wantSelector) {
		t.Fatalf("selector = %+v, want %+v", parsed.Selector, wantSelector)
	}
	if parsed.Heading != "## First\nSecond" {
		t.Fatalf("heading = %q, want the complete multiline heading", parsed.Heading)
	}
}

func TestResolveBodyValue_HeadingSyntaxCompatibility(t *testing.T) {
	for _, tt := range []struct {
		name      string
		body      string
		selector  SectionSelector
		wantValue string
	}{
		{
			name:      "ATX",
			body:      "## Notes\n\nATX content\n",
			selector:  SectionSelector{{Level: 2, Text: "Notes"}},
			wantValue: "ATX content",
		},
		{
			name:      "simple Setext",
			body:      "Notes\n---\n\nSetext content\n",
			selector:  SectionSelector{{Level: 2, Text: "Notes"}},
			wantValue: "Setext content",
		},
		{
			name:      "multiline Setext hierarchy",
			body:      "# Root\n\nFirst\nSecond\n---\n\nMultiline content\n",
			selector:  SectionSelector{{Level: 1, Text: "Root"}, {Level: 2, Text: "First\nSecond"}},
			wantValue: "Multiline content",
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			source, err := CanonicalSectionSelectorSource(tt.selector)
			if err != nil {
				t.Fatalf("CanonicalSectionSelectorSource returned an error: %v", err)
			}
			got, present, err := ResolveBodyValue(&Record{Body: tt.body}, source)
			if err != nil || !present || got != tt.wantValue {
				t.Fatalf("value = %q, present = %v, error = %v; want %q", got, present, err, tt.wantValue)
			}
		})
	}
}

func TestResolveBodyValue_PresentEmptySection(t *testing.T) {
	rec := &Record{BodySections: []Section{{Heading: "Notes", Level: 2, Content: ""}}}
	got, present, err := ResolveBodyValue(rec, `body.section["## Notes"]`)
	if err != nil || !present || got != "" {
		t.Fatalf("value=%q present=%v error=%v; want a present empty value", got, present, err)
	}
}

func TestResolveBodyValue_ReusesAvailableAST(t *testing.T) {
	astSource := []byte("# AST\n\nAST content\n")
	node := goldmark.DefaultParser().Parse(gmtext.NewReader(astSource))
	body := "plain\n\n# Text\n\ntext content\n"

	// The alternate AST makes AST reuse observable without parser instrumentation.
	wantSections := ExtractSections(node, []byte(body))
	if reflect.DeepEqual(wantSections, ExtractSectionsFromText(body)) {
		t.Fatal("test inputs do not distinguish AST reuse from transient parsing")
	}
	if len(wantSections) != 1 || wantSections[0].Level != 1 {
		t.Fatalf("AST-backed sections = %+v; want one H1 section", wantSections)
	}

	rec := &Record{Body: body, AST: node}
	gotSections := sectionsForRecord(rec)
	if !reflect.DeepEqual(gotSections, wantSections) {
		t.Fatalf("record sections = %+v; want AST-backed sections %+v", gotSections, wantSections)
	}
	got, present, err := ResolveBodyValue(rec, "body.h1")
	if err != nil || !present || got != wantSections[0].Heading {
		t.Fatalf("value=%q present=%v error=%v; want AST-backed H1 %q", got, present, err, wantSections[0].Heading)
	}
}

func TestResolveBodyValue_ParsesBodyWithoutAST(t *testing.T) {
	rec := &Record{Body: "# Text\n\ntext content\n"}
	got, present, err := ResolveBodyValue(rec, `body.section["# Text"]`)
	if err != nil || !present || got != "text content" {
		t.Fatalf("value=%q present=%v error=%v; want transient parsing result", got, present, err)
	}
}

func TestResolveBodyValue_ASTWithoutHeadingsHasNoHeadingResult(t *testing.T) {
	body := "paragraph only\n"
	node := goldmark.DefaultParser().Parse(gmtext.NewReader([]byte(body)))
	rec := &Record{Body: body, AST: node}

	if got, present, err := ResolveBodyValue(rec, "body.h1"); err != nil || present || got != "" {
		t.Fatalf("value=%q present=%v error=%v; want no H1", got, present, err)
	}
	if got, present, err := ResolveBodyValue(rec, `body.section["# Missing"]`); err != nil || present || got != "" {
		t.Fatalf("value=%q present=%v error=%v; want no section", got, present, err)
	}
}

func TestResolveBodyValue_PreservesInitializedEmptySections(t *testing.T) {
	body := "# Text\n\ntext content\n"
	node := goldmark.DefaultParser().Parse(gmtext.NewReader([]byte(body)))
	rec := &Record{Body: body, BodySections: []Section{}, AST: node}

	if got, present, err := ResolveBodyValue(rec, "body.h1"); err != nil || present || got != "" {
		t.Fatalf("value=%q present=%v error=%v; want initialized empty sections", got, present, err)
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
