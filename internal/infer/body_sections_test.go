package infer

import (
	"math"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func makeRecord(body string) *extract.Record {
	source := []byte(body)
	reader := text.NewReader(source)
	node := goldmark.DefaultParser().Parse(reader)
	return &extract.Record{
		Path: "test.md",
		Body: body,
		AST:  node,
	}
}

func makeSectionRecord(sections ...extract.Section) *extract.Record {
	return &extract.Record{Path: "sections.md", Body: "section fixture", BodySections: sections}
}

func TestDetectSectionPatterns_UniversalSectionRequiredWithCanonicalSource(t *testing.T) {
	records := []*extract.Record{
		makeRecord("## Contexto\n\nSome context here.\n"),
		makeRecord("## Contexto\n\nOther context.\n"),
		makeRecord("## Contexto\n\nMore context.\n"),
	}

	inferences, err := DetectSectionPatterns(records, 0.80)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}

	inf, ok := findSectionInference(inferences, "contexto")
	if !ok {
		t.Fatalf("expected contexto inference, got %+v", inferences)
	}
	if inf.Type != "required_section" || inf.Value != "1.00" || inf.SourceDirective != `body.section["## Contexto"]` {
		t.Fatalf("unexpected inference: %+v", inf)
	}
}

func TestDetectSectionPatterns_VariableH1UsesSimpleRequiredSelector(t *testing.T) {
	records := []*extract.Record{
		makeRecord("# Alpha\n\n## Overview\nFirst.\n"),
		makeRecord("# Beta\n\n## Overview\nSecond.\n"),
	}

	inferences, err := DetectSectionPatterns(records, 1)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	inf, ok := findSectionInference(inferences, "overview")
	if !ok {
		t.Fatalf("overview inference is missing: %+v", inferences)
	}
	if inf.Type != "required_section" || inf.SourceDirective != `body.section["## Overview"]` {
		t.Fatalf("overview inference = %+v", inf)
	}
}

func TestDetectSectionPatterns_VariableParentsUseSimpleSelectorForSingleOccurrence(t *testing.T) {
	records := []*extract.Record{
		makeRecord("# Alpha\n\n## First Parent\n\n### Notes\nFirst.\n"),
		makeRecord("# Beta\n\n## Second Parent\n\n### Notes\nSecond.\n"),
	}

	inferences, err := DetectSectionPatterns(records, 1)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	inf, ok := findSectionInference(inferences, "notes")
	if !ok || inf.SourceDirective != `body.section["### Notes"]` {
		t.Fatalf("notes inference = %+v, present = %v", inf, ok)
	}
}

func TestDetectSectionPatterns_SelectsShortestCommonQualifiedSelector(t *testing.T) {
	records := []*extract.Record{
		makeRecord("# Root\n\n## First Parent\n\n### Notes\nVariable.\n\n## Shared\n\n### Notes\nStable.\n"),
		makeRecord("# Root\n\n## Second Parent\n\n### Notes\nVariable.\n\n## Shared\n\n### Notes\nStable.\n"),
	}

	inferences, err := DetectSectionPatterns(records, 1)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	inf, ok := findSectionInference(inferences, "notes")
	if !ok || inf.SourceDirective != `body.section["## Shared"]["### Notes"]` {
		t.Fatalf("notes inference = %+v, present = %v", inf, ok)
	}
}

func TestDetectSectionPatterns_IncompatibleParentsHaveNoCommonSelector(t *testing.T) {
	records := []*extract.Record{
		makeRecord("# Alpha\n\n## First A\n\n### Notes\nOne.\n\n## Second A\n\n### Notes\nTwo.\n"),
		makeRecord("# Beta\n\n## First B\n\n### Notes\nOne.\n\n## Second B\n\n### Notes\nTwo.\n"),
	}

	inferences, err := DetectSectionPatterns(records, 1)
	want := `no_common_selector: section family "### Notes" has no common selector`
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if inferences != nil {
		t.Fatalf("inferences = %+v, want nil", inferences)
	}
}

func TestDetectSectionPatterns_MultipleStableOccurrenceGroupsFail(t *testing.T) {
	records := []*extract.Record{
		makeRecord("# Alpha\n\n## First\n\n### Notes\nOne.\n\n## Second\n\n### Notes\nTwo.\n"),
		makeRecord("# Beta\n\n## First\n\n### Notes\nOne.\n\n## Second\n\n### Notes\nTwo.\n"),
	}

	_, err := DetectSectionPatterns(records, 1)
	want := `multiple_stable_groups: section family "### Notes" has multiple stable occurrence groups: body.section["## First"]["### Notes"], body.section["## Second"]["### Notes"]`
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
}

func TestDetectSectionPatterns_ThresholdCandidateOptionalUntilUniversal(t *testing.T) {
	records := []*extract.Record{
		makeRecord("## Notes\nA\n"), makeRecord("## Notes\nB\n"),
		makeRecord("## Notes\nC\n"), makeRecord("## Notes\nD\n"),
		makeRecord("# No notes\n"),
	}
	got, err := DetectSectionPatterns(records, 0.8)
	if err != nil {
		t.Fatal(err)
	}
	inf, ok := findSectionInference(got, "notes")
	if !ok {
		t.Fatalf("notes inference missing: %+v", got)
	}
	if inf.Type != "optional_section" || inf.SourceDirective != `body.section["## Notes"]` {
		t.Fatalf("unexpected inference: %+v", inf)
	}
}

func TestDetectSectionPatterns_EmptyNilBodyRecordsCountAbsent(t *testing.T) {
	records := []*extract.Record{
		makeRecord("## Contexto\n\nText.\n"),
		{Path: "empty.md", Body: "", AST: nil},
	}

	inferences, err := DetectSectionPatterns(records, 0.80)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	if _, ok := findSectionInference(inferences, "contexto"); ok {
		t.Fatalf("contexto should be absent below threshold when empty record is denominator: %+v", inferences)
	}
}

func TestDetectSectionPatterns_EmptyHeadingBodyCountsPresent(t *testing.T) {
	records := []*extract.Record{
		makeRecord("## Notes\n"),
		makeRecord("## Notes\n"),
	}

	inferences, err := DetectSectionPatterns(records, 0.80)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	inf, ok := findSectionInference(inferences, "notes")
	if !ok {
		t.Fatalf("expected notes inference for empty section bodies, got %+v", inferences)
	}
	if inf.Type != "required_section" {
		t.Fatalf("expected universal empty sections to be required, got %+v", inf)
	}
}

func TestDetectSectionPatterns_NilRecordPointerCountsAbsent(t *testing.T) {
	inferences, err := DetectSectionPatterns([]*extract.Record{makeRecord("## Notes\n"), nil}, 0.75)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	if _, ok := findSectionInference(inferences, "notes"); ok {
		t.Fatalf("nil record should remain in denominator: %+v", inferences)
	}
}

func TestDetectSectionPatterns_AllEmptyRecordsReturnNoInferences(t *testing.T) {
	inferences, err := DetectSectionPatterns([]*extract.Record{nil, {Path: "empty.md"}, {Path: "blank.md", Body: ""}}, 0.8)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	if len(inferences) != 0 {
		t.Fatalf("expected no inferences for all-empty records, got %+v", inferences)
	}
}

func TestDetectSectionPatterns_DuplicateExactHeadingCountsOncePerRecord(t *testing.T) {
	records := []*extract.Record{
		makeSectionRecord(
			extract.Section{Level: 2, Heading: "Notes"},
			extract.Section{Level: 2, Heading: "Notes"},
		),
		{Path: "other.md", Body: "no headings"},
	}
	inferences, err := DetectSectionPatterns(records, 0.75)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	if _, ok := findSectionInference(inferences, "notes"); ok {
		t.Fatalf("duplicate exact heading in one record should count once: %+v", inferences)
	}
}

func TestMatchingOccurrencesStopsAfterSecondMatch(t *testing.T) {
	selector := extract.SectionSelector{{Level: 2, Text: "Notes"}}
	occurrences := []sectionOccurrence{
		{sectionOrdinal: 1, path: extract.SectionPath{{Level: 2, Text: "Notes"}}},
		{sectionOrdinal: 2, path: extract.SectionPath{{Level: 2, Text: "Notes"}}},
		{sectionOrdinal: 3, path: extract.SectionPath{{Level: 2, Text: "Notes"}}},
	}

	matches := matchingOccurrences(occurrences, selector)
	if len(matches) != 2 {
		t.Fatalf("match count = %d, want 2", len(matches))
	}
	if matches[0].sectionOrdinal != 1 || matches[1].sectionOrdinal != 2 {
		t.Fatalf("match ordinals = %d, %d, want 1, 2", matches[0].sectionOrdinal, matches[1].sectionOrdinal)
	}
}

func TestDetectSectionPatterns_PreservesQuotedBackslashBracketDirective(t *testing.T) {
	heading := `Need "quotes" \ and [brackets]`
	inferences, err := DetectSectionPatterns([]*extract.Record{
		makeSectionRecord(extract.Section{Level: 2, Heading: heading}),
		makeSectionRecord(extract.Section{Level: 2, Heading: heading}),
	}, 0.8)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	inf, ok := findSectionInference(inferences, "need_quotes_and_brackets")
	if !ok {
		t.Fatalf("expected special heading inference, got %+v", inferences)
	}
	want := `body.section["## Need \"quotes\" \\ and [brackets]"]`
	if inf.SourceDirective != want {
		t.Fatalf("source directive = %q, want %q", inf.SourceDirective, want)
	}
}

func TestDetectSectionPatterns_InferenceOrderingIsDeterministic(t *testing.T) {
	records := []*extract.Record{
		makeSectionRecord(
			extract.Section{Level: 2, Heading: "Gamma"},
			extract.Section{Level: 2, Heading: "Alpha"},
			extract.Section{Level: 2, Heading: "Beta"},
		),
		makeSectionRecord(
			extract.Section{Level: 2, Heading: "Beta"},
			extract.Section{Level: 2, Heading: "Gamma"},
			extract.Section{Level: 2, Heading: "Alpha"},
		),
	}
	inferences, err := DetectSectionPatterns(records, 0.5)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	fields := make([]string, 0, len(inferences))
	for _, inf := range inferences {
		fields = append(fields, inf.Field)
	}
	if got := strings.Join(fields, ","); got != "alpha,beta,gamma" {
		t.Fatalf("inference order = %s, want alpha,beta,gamma", got)
	}
}

func TestDetectSectionPatterns_NameCollisionFails(t *testing.T) {
	records := []*extract.Record{makeRecord("## Notes\nA\n\n### Notes\nB\n")}
	_, err := DetectSectionPatterns(records, 0.5)
	if err == nil || !strings.Contains(err.Error(), "## Notes") || !strings.Contains(err.Error(), "### Notes") {
		t.Fatalf("expected both colliding headings, got %v", err)
	}
}

func TestDetectSectionPatterns_BelowThresholdFamiliesDoNotCollide(t *testing.T) {
	records := []*extract.Record{
		makeRecord("## Notes\nA\n\n### Notes\nB\n"),
		makeRecord("# Other\n"), makeRecord("# Other\n"),
		makeRecord("# Other\n"), makeRecord("# Other\n"),
	}
	inferences, err := DetectSectionPatterns(records, 0.8)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	if _, ok := findSectionInference(inferences, "notes"); ok {
		t.Fatalf("notes inference is above the threshold: %+v", inferences)
	}
}

func TestDetectSectionPatterns_DuplicateFullPathFailsOnlyAtThreshold(t *testing.T) {
	records := []*extract.Record{
		makeRecord("## Parent\n\n### Notes\nFirst.\n\n### Notes\nSecond.\n"),
		makeRecord("# Other\n"),
	}
	records[0].Path = "a.md"
	records[1].Path = "b.md"

	inferences, err := DetectSectionPatterns(records, 0.75)
	if err != nil {
		t.Fatalf("below-threshold family returned an error: %v", err)
	}
	if _, ok := findSectionInference(inferences, "notes"); ok {
		t.Fatalf("notes inference is above the threshold: %+v", inferences)
	}

	inferences, err = DetectSectionPatterns(records, 0.5)
	want := `duplicate_full_path: duplicate body section path for family "### Notes": "a.md" (record 0), body.section["## Parent"]["### Notes"] (section ordinals 1, 2)`
	if err == nil || err.Error() != want {
		t.Fatalf("error = %v, want %q", err, want)
	}
	if inferences != nil {
		t.Fatalf("inferences = %+v, want nil", inferences)
	}
}

func TestDetectSectionPatterns_CollisionOrderingIncludesCaseSpacingAndLevelMembers(t *testing.T) {
	records := []*extract.Record{makeSectionRecord(
		extract.Section{Level: 4, Heading: "ROAD-map"},
		extract.Section{Level: 2, Heading: "Road Map"},
		extract.Section{Level: 3, Heading: "road   map"},
	)}
	_, err := DetectSectionPatterns(records, 0.8)
	want := `section field name collision: road_map: "## Road Map", "### road   map", "#### ROAD-map"`
	if err == nil || err.Error() != want {
		t.Fatalf("collision error = %v, want %q", err, want)
	}
}

func TestDetectSectionPatterns_ThresholdDomainIsFiniteInclusive(t *testing.T) {
	invalid := []struct {
		name      string
		threshold float64
	}{
		{"negative", -0.1},
		{"greater_than_one", 1.1},
		{"nan", math.NaN()},
		{"positive_infinity", math.Inf(1)},
		{"negative_infinity", math.Inf(-1)},
	}
	for _, tt := range invalid {
		for _, records := range []struct {
			name    string
			records []*extract.Record
		}{
			{"with_records", []*extract.Record{makeRecord("## Notes\n")}},
			{"zero_records", nil},
		} {
			t.Run(tt.name+"/"+records.name, func(t *testing.T) {
				_, err := DetectSectionPatterns(records.records, tt.threshold)
				if err == nil || !strings.Contains(err.Error(), "threshold") {
					t.Fatalf("expected threshold error, got %v", err)
				}
			})
		}
	}

	for _, threshold := range []float64{0, 1} {
		if _, err := DetectSectionPatterns([]*extract.Record{makeRecord("## Notes\n")}, threshold); err != nil {
			t.Fatalf("valid threshold %v with records returned error: %v", threshold, err)
		}
		if got, err := DetectSectionPatterns(nil, threshold); err != nil || got != nil {
			t.Fatalf("valid threshold %v with zero records got inferences=%v err=%v", threshold, got, err)
		}
	}
}

func TestDetectSectionPatterns_NoRecords(t *testing.T) {
	inferences, err := DetectSectionPatterns(nil, 0.80)
	if err != nil {
		t.Fatalf("DetectSectionPatterns: %v", err)
	}
	if inferences != nil {
		t.Errorf("expected nil for empty records, got %v", inferences)
	}
}

func findSectionInference(inferences []Inference, field string) (Inference, bool) {
	for _, inf := range inferences {
		if inf.Field == field {
			return inf, true
		}
	}
	return Inference{}, false
}
