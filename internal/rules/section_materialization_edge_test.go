package rules

import (
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func hasRequiredValidationError(errs []ValidationError, field string) bool {
	for _, err := range errs {
		if err.Rule == "required" && err.Field == field {
			return true
		}
	}
	return false
}

func assertMaterializationHeadings(t *testing.T, got []SectionMaterialization, want []string) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("got %+v, want headings %v", got, want)
	}
	for i := range want {
		if got[i].Heading != want[i] {
			t.Fatalf("got[%d].Heading = %q, want %q (all candidates %+v)", i, got[i].Heading, want[i], got)
		}
	}
}

func TestRequiredSectionMaterializations_InvalidDeclarationsUseStableFieldOrder(t *testing.T) {
	stem := &StemFile{Schema: map[string]SchemaField{
		"beta":  {Type: "string", Required: true, Extract: `body.section["Beta"]`},
		"alpha": {Type: "string", Required: true, Extract: `body.section["Alpha"]`},
	}}

	got, err := RequiredSectionMaterializations(emptyRecord(), stem)
	if err == nil {
		t.Fatalf("expected declaration error, got %+v", got)
	}
	want := `field "alpha": field "alpha" has unsupported source: section source heading must be an exact markdown heading, got "Alpha"`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestRequiredSectionMaterializations_SameHeadingCollisionExactErrorStableOrder(t *testing.T) {
	stem := &StemFile{Schema: map[string]SchemaField{
		"beta":  {Type: "string", Required: true, Extract: `body.section["## Shared"]`},
		"alpha": {Type: "string", Required: true, Extract: `body.section["## Shared"]`},
	}}

	got, err := RequiredSectionMaterializations(emptyRecord(), stem)
	if err == nil {
		t.Fatalf("expected same-heading conflict, got %+v", got)
	}
	want := `multiple required fields alpha, beta materialize missing section "## Shared" ambiguously`
	if err.Error() != want {
		t.Fatalf("error = %q, want %q", err.Error(), want)
	}
}

func TestRequiredSectionMaterializations_SameHeadingShadowingAndExistingSections(t *testing.T) {
	stem := &StemFile{Schema: map[string]SchemaField{
		"beta":  {Type: "string", Required: true, Extract: `body.section["## Shared"]`},
		"alpha": {Type: "string", Required: true, Extract: `body.section["## Shared"]`},
	}}

	t.Run("existing section satisfies both fields", func(t *testing.T) {
		got, err := RequiredSectionMaterializations(recordWithSection("## Shared", "body"), stem)
		if err != nil || len(got) != 0 {
			t.Fatalf("got %+v err=%v, want no materializations", got, err)
		}
	})

	t.Run("one frontmatter shadow leaves one candidate", func(t *testing.T) {
		rec := &extract.Record{Frontmatter: map[string]any{"alpha": "shadow"}}
		got, err := RequiredSectionMaterializations(rec, stem)
		if err != nil {
			t.Fatal(err)
		}
		want := []SectionMaterialization{{Field: "beta", Heading: "## Shared", Content: "<!-- TODO -->"}}
		if len(got) != 1 || got[0] != want[0] {
			t.Fatalf("got %+v, want %+v", got, want)
		}
	})

	t.Run("both frontmatter shadows include empty string", func(t *testing.T) {
		rec := &extract.Record{Frontmatter: map[string]any{"alpha": "", "beta": "shadow"}}
		got, err := RequiredSectionMaterializations(rec, stem)
		if err != nil || len(got) != 0 {
			t.Fatalf("got %+v err=%v, want no materializations", got, err)
		}
	})

	t.Run("unshadowed duplicate section error propagates", func(t *testing.T) {
		got, err := RequiredSectionMaterializations(recordWithDuplicateSection("## Shared"), stem)
		if err == nil {
			t.Fatalf("expected duplicate section error, got %+v", got)
		}
		want := `ambiguous body section source "## Shared"`
		if err.Error() != want {
			t.Fatalf("error = %q, want %q", err.Error(), want)
		}
	})
}

func TestRequiredSectionMaterializations_ExistingSpecialHeadingUsesParsedMarkerOnly(t *testing.T) {
	heading := "## Q&A: Plan #1"
	stem := &StemFile{Schema: map[string]SchemaField{
		"qa": {Type: "string", Required: true, Extract: `body.section["## Q&A: Plan #1"]`},
	}}

	got, err := RequiredSectionMaterializations(recordWithSection(heading, "answered"), stem)
	if err != nil || len(got) != 0 {
		t.Fatalf("got %+v err=%v, want existing heading %q to satisfy field", got, err, heading)
	}
}

func TestRequiredSectionMaterializations_RecordBodyDuplicateStillPropagates(t *testing.T) {
	rec := &extract.Record{
		Frontmatter: map[string]any{},
		Body:        strings.Join([]string{"## Shared", "first", "", "## Shared", "second"}, "\n"),
	}
	stem := &StemFile{Schema: map[string]SchemaField{
		"shared": {Type: "string", Required: true, Extract: `body.section["## Shared"]`},
	}}

	got, err := RequiredSectionMaterializations(rec, stem)
	if err == nil {
		t.Fatalf("expected duplicate body section error, got %+v", got)
	}
	if err.Error() != `ambiguous body section source "## Shared"` {
		t.Fatalf("error = %q", err.Error())
	}
}
