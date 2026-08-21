package rules

import (
	"context"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestRequiredSectionMaterializations_LexicalOrder(t *testing.T) {
	stem := &StemFile{Schema: map[string]SchemaField{
		"zeta":  {Type: "string", Required: true, Extract: `body.section["## Zeta"]`},
		"alpha": {Type: "string", Required: true, Extract: `body.section["## Alpha"]`},
	}}
	rec := emptyRecord()

	got, err := RequiredSectionMaterializations(rec, stem)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 2 || got[0].Heading != "## Alpha" || got[1].Heading != "## Zeta" {
		t.Fatalf("not lexical: %+v", got)
	}
}

func TestRequiredSectionMaterializations_TableContract(t *testing.T) {
	tests := []struct {
		name        string
		fieldName   string
		field       SchemaField
		record      *extract.Record
		wantCount   int
		wantHeading string
		wantContent string
		wantErr     bool
	}{
		{"default", "notes", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`, Default: "seed"}, emptyRecord(), 1, "## Notes", "seed", false},
		{"placeholder", "notes", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, emptyRecord(), 1, "## Notes", "<!-- TODO -->", false},
		{"frontmatter override", "notes", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, recordWithFrontmatter("notes", "override"), 0, "", "", false},
		{"frontmatter nil present", "notes", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, recordWithFrontmatter("notes", nil), 0, "", "", false},
		{"empty section present", "notes", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, recordWithSection("## Notes", ""), 0, "", "", false},
		{"optional", "notes", SchemaField{Type: "string", Required: false, Extract: `body.section["## Notes"]`}, emptyRecord(), 0, "", "", false},
		{"body h1 ignored", "title", SchemaField{Type: "string", Required: true, Extract: `body.h1`}, emptyRecord(), 0, "", "", false},
		{"frontmatter source ignored", "title", SchemaField{Type: "string", Required: true}, emptyRecord(), 0, "", "", false},
		{"h1 exact heading", "summary", SchemaField{Type: "string", Required: true, Extract: `body.section["# Summary"]`}, emptyRecord(), 1, "# Summary", "<!-- TODO -->", false},
		{"h3 exact heading", "detail", SchemaField{Type: "string", Required: true, Extract: `body.section["### Detail"]`}, emptyRecord(), 1, "### Detail", "<!-- TODO -->", false},
		{"special chars exact heading", "qa", SchemaField{Type: "string", Required: true, Extract: `body.section["## Q&A: Plan #1"]`}, emptyRecord(), 1, "## Q&A: Plan #1", "<!-- TODO -->", false},
		{"duplicate", "notes", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, recordWithDuplicateSection("## Notes"), 0, "", "", true},
		{"legacy section type invalid before no-op", "legacy", SchemaField{Type: "section", Required: false}, emptyRecord(), 0, "", "", true},
		{"legacy heading invalid before no-op", "legacy", SchemaField{Type: "string", Heading: "Notes", Required: false}, emptyRecord(), 0, "", "", true},
		{"invalid source invalid before no-op", "notes", SchemaField{Type: "string", Required: false, Extract: `body.section["Notes"]`}, emptyRecord(), 0, "", "", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stem := &StemFile{Schema: map[string]SchemaField{tt.fieldName: tt.field}}

			got, err := RequiredSectionMaterializations(tt.record, stem)
			if tt.wantErr {
				if err == nil {
					t.Fatalf("expected error, got candidates %+v", got)
				}
				return
			}
			if err != nil {
				t.Fatal(err)
			}
			if len(got) != tt.wantCount {
				t.Fatalf("got %d candidates %+v, want %d", len(got), got, tt.wantCount)
			}
			if tt.wantCount == 0 {
				return
			}
			if got[0].Field != tt.fieldName || got[0].Heading != tt.wantHeading || got[0].Content != tt.wantContent {
				t.Fatalf("candidate = %+v, want field=%q heading=%q content=%q", got[0], tt.fieldName, tt.wantHeading, tt.wantContent)
			}
		})
	}
}

func TestRequiredSectionMaterializations_OrdersByHeadingThenField(t *testing.T) {
	stem := &StemFile{Schema: map[string]SchemaField{
		"gamma": {Type: "string", Required: true, Extract: `body.section["## Shared"]`, Default: "same"},
		"alpha": {Type: "string", Required: true, Extract: `body.section["## Later"]`},
		"beta":  {Type: "string", Required: true, Extract: `body.section["## Earlier"]`},
	}}

	got, err := RequiredSectionMaterializations(emptyRecord(), stem)
	if err != nil {
		t.Fatal(err)
	}
	want := []SectionMaterialization{
		{Field: "beta", Heading: "## Earlier", Content: "<!-- TODO -->"},
		{Field: "alpha", Heading: "## Later", Content: "<!-- TODO -->"},
		{Field: "gamma", Heading: "## Shared", Content: "same"},
	}
	if len(got) != len(want) {
		t.Fatalf("got %+v, want %+v", got, want)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("got[%d]=%+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestRequiredSectionMaterializations_RejectsSameMissingHeadingCollision(t *testing.T) {
	stem := &StemFile{Schema: map[string]SchemaField{
		"alpha": {Type: "string", Required: true, Extract: `body.section["## Shared"]`, Default: "alpha seed"},
		"beta":  {Type: "string", Required: true, Extract: `body.section["## Shared"]`, Default: "beta seed"},
	}}

	got, err := RequiredSectionMaterializations(emptyRecord(), stem)
	if err == nil {
		t.Fatalf("expected same-heading conflict, got %+v", got)
	}
	if !strings.Contains(err.Error(), "## Shared") || !strings.Contains(err.Error(), "alpha") || !strings.Contains(err.Error(), "beta") {
		t.Fatalf("conflict error %q does not name heading and fields", err)
	}
}

func TestRequiredSectionMaterializations_NilInputs(t *testing.T) {
	got, err := RequiredSectionMaterializations(nil, nil)
	if err != nil || len(got) != 0 {
		t.Fatalf("nil record/stem got %+v err=%v, want no candidates and no error", got, err)
	}

	stem := &StemFile{Schema: map[string]SchemaField{
		"notes": {Type: "string", Required: true, Extract: `body.section["## Notes"]`},
	}}
	got, err = RequiredSectionMaterializations(nil, stem)
	if err != nil {
		t.Fatal(err)
	}
	if len(got) != 1 || got[0].Heading != "## Notes" || got[0].Content != "<!-- TODO -->" {
		t.Fatalf("nil record got %+v, want missing section materialization", got)
	}
}

func TestRequiredSectionMaterializations_UsesRequiredApplicabilityBoundary(t *testing.T) {
	tests := []struct {
		name          string
		recordPath    string
		field         SchemaField
		aggregate     map[string]any
		wantCandidate bool
		wantRequired  bool
	}{
		{"severity off", "docs/T001-task.md", SchemaField{Type: "string", Required: true, Severity: "off", Extract: `body.section["## Notes"]`}, nil, false, false},
		{"excluded path", "docs/skip.md", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`, Excludes: &ExcludeRule{Match: "docs/skip.md"}}, nil, false, false},
		{"aggregate index", "docs/README.md", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, map[string]any{"notes": "count(children)"}, false, false},
		{"aggregate normal record", "docs/T001-task.md", SchemaField{Type: "string", Required: true, Extract: `body.section["## Notes"]`}, map[string]any{"notes": "count(children)"}, true, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := &extract.Record{Path: tt.recordPath, Frontmatter: map[string]any{}}
			stem := &StemFile{Schema: map[string]SchemaField{"notes": tt.field}, Aggregate: tt.aggregate}

			got, err := RequiredSectionMaterializations(rec, stem)
			if err != nil {
				t.Fatal(err)
			}
			if (len(got) == 1) != tt.wantCandidate {
				t.Fatalf("candidates = %+v, wantCandidate=%v", got, tt.wantCandidate)
			}

			errFound := hasRequiredValidationError(Validate(context.Background(), rec, stem), "notes")
			if errFound != tt.wantRequired {
				t.Fatalf("required validation present=%v, want %v", errFound, tt.wantRequired)
			}
		})
	}
}

func TestRequiredSectionMaterializations_NormalizesRecordSpecificSchemaWithoutMutation(t *testing.T) {
	stem := &StemFile{Schema: map[string]SchemaField{
		"empty_required_match":       {Type: "string", Required: true, RequiredMatch: &FieldMatch{}, Extract: `body.section["## Empty RequiredMatch"]`},
		"included_field_match":       {Type: "string", Required: true, Match: &FieldMatch{Patterns: []string{"T*"}}, Extract: `body.section["## Included Field Match"]`},
		"included_required_match":    {Type: "string", Required: true, RequiredMatch: &FieldMatch{Patterns: []string{"T*"}}, Extract: `body.section["## Included RequiredMatch"]`},
		"nonmatching_field_match":    {Type: "string", Required: true, Match: &FieldMatch{Patterns: []string{"Z*"}}, Extract: `body.section["## Excluded Field Match"]`},
		"nonmatching_required_match": {Type: "string", Required: true, RequiredMatch: &FieldMatch{Patterns: []string{"Z*"}}, Extract: `body.section["## Excluded RequiredMatch"]`},
	}}
	rec := &extract.Record{Path: "docs/E01/F01/S001/T001-task.md", Frontmatter: map[string]any{}}

	got, err := RequiredSectionMaterializations(rec, stem)
	if err != nil {
		t.Fatal(err)
	}
	assertMaterializationHeadings(t, got, []string{"## Included Field Match", "## Included RequiredMatch"})
	if !stem.Schema["nonmatching_required_match"].Required {
		t.Fatal("materialization mutated caller schema Required flag")
	}
	if stem.Schema["nonmatching_field_match"].Match == nil {
		t.Fatal("materialization mutated caller schema Match")
	}

	filtered := mustFilterSchemaByMatch(t, pointerSchema(stem.Schema), rec.Path)
	alreadyEffective := &StemFile{Schema: valueSchema(filtered)}
	got, err = RequiredSectionMaterializations(rec, alreadyEffective)
	if err != nil {
		t.Fatal(err)
	}
	assertMaterializationHeadings(t, got, []string{"## Included Field Match", "## Included RequiredMatch"})
}

func TestRequiredSectionMaterializations_OnlyStringSectionFieldsMaterialize(t *testing.T) {
	stem := &StemFile{Schema: map[string]SchemaField{
		"items": {Type: "list", Required: true, Extract: `body.section["## Items"]`},
		"notes": {Type: "string", Required: true, Extract: `body.section["## Notes"]`},
	}}

	got, err := RequiredSectionMaterializations(emptyRecord(), stem)
	if err != nil {
		t.Fatal(err)
	}
	assertMaterializationHeadings(t, got, []string{"## Notes"})
}

func TestRequiredSectionMaterializations_ValidatesNonStringDeclarationsBeforeNoOp(t *testing.T) {
	stem := &StemFile{Schema: map[string]SchemaField{
		"items": {Type: "list", Required: true, Extract: `body.section["Items"]`},
	}}

	got, err := RequiredSectionMaterializations(emptyRecord(), stem)
	if err == nil {
		t.Fatalf("expected declaration error before non-string no-op, got %+v", got)
	}
}

func emptyRecord() *extract.Record {
	return &extract.Record{Frontmatter: map[string]any{}}
}

func recordWithFrontmatter(name string, value any) *extract.Record {
	return &extract.Record{Frontmatter: map[string]any{name: value}}
}

func recordWithSection(exactHeading, content string) *extract.Record {
	body := exactHeading + "\n\n" + content
	return &extract.Record{Frontmatter: map[string]any{}, BodySections: extract.ExtractSectionsFromText(body)}
}

func recordWithDuplicateSection(exactHeading string) *extract.Record {
	rec := recordWithSection(exactHeading, "first")
	rec.BodySections = append(rec.BodySections, rec.BodySections[0])
	rec.BodySections[1].Content = "second"
	return rec
}
