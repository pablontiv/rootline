package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// --- majorityValue tests ---

func TestMajorityValue_Clear(t *testing.T) {
	val, count, total := majorityValue([]string{"a", "a", "a", "b"})
	if val != "a" {
		t.Errorf("val = %q, want %q", val, "a")
	}
	if count != 3 {
		t.Errorf("count = %d, want 3", count)
	}
	if total != 4 {
		t.Errorf("total = %d, want 4", total)
	}
}

func TestMajorityValue_Tie(t *testing.T) {
	val, count, total := majorityValue([]string{"a", "b"})
	if val != "" {
		t.Errorf("val = %q, want empty on tie", val)
	}
	if count != 1 {
		t.Errorf("count = %d, want 1", count)
	}
	if total != 2 {
		t.Errorf("total = %d, want 2", total)
	}
}

func TestMajorityValue_Empty(t *testing.T) {
	val, count, total := majorityValue(nil)
	if val != "" || count != 0 || total != 0 {
		t.Errorf("got (%q, %d, %d), want (\"\", 0, 0)", val, count, total)
	}
}

// --- detectInferFromSiblings tests ---

func TestInferFromSiblings_MajorityPresent(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"a", "b", "c"}, Required: true},
		},
	}

	records := []*extract.Record{
		{Path: "dir/f1.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f2.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f3.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f4.md", Frontmatter: map[string]any{}},
	}

	errs := map[string][]rules.ValidationError{
		"dir/f4.md": {{Rule: "required", Field: "tipo", Message: `required field "tipo" is missing`}},
	}

	proposals := detectInferFromSiblings(records, stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Type != InferFromSiblings {
		t.Errorf("type = %q, want infer_from_siblings", proposals[0].Type)
	}
	if proposals[0].Value != "a" {
		t.Errorf("value = %q, want %q", proposals[0].Value, "a")
	}
	if proposals[0].Paths[0] != "dir/f4.md" {
		t.Errorf("path = %q, want dir/f4.md", proposals[0].Paths[0])
	}
}

func TestInferFromSiblings_BelowThreshold(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"a", "b", "c"}, Required: true},
		},
	}

	// 2 with "a", 2 with "b", 1 missing — 50% < 60% threshold
	records := []*extract.Record{
		{Path: "dir/f1.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f2.md", Frontmatter: map[string]any{"tipo": "b"}},
		{Path: "dir/f3.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f4.md", Frontmatter: map[string]any{"tipo": "b"}},
		{Path: "dir/f5.md", Frontmatter: map[string]any{}},
	}

	errs := map[string][]rules.ValidationError{
		"dir/f5.md": {{Rule: "required", Field: "tipo", Message: `required field "tipo" is missing`}},
	}

	proposals := detectInferFromSiblings(records, stem, errs)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (below threshold)", len(proposals))
	}
}

func TestInferFromSiblings_SkipsREADME(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"a", "b"}, Required: true},
		},
	}

	// README has "b" but should be excluded from grouping
	records := []*extract.Record{
		{Path: "dir/README.md", Frontmatter: map[string]any{"tipo": "b"}},
		{Path: "dir/f1.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f2.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f3.md", Frontmatter: map[string]any{}},
	}

	errs := map[string][]rules.ValidationError{
		"dir/f3.md": {{Rule: "required", Field: "tipo", Message: `required field "tipo" is missing`}},
	}

	proposals := detectInferFromSiblings(records, stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Value != "a" {
		t.Errorf("value = %q, want %q (README excluded)", proposals[0].Value, "a")
	}
}

func TestInferFromSiblings_OnlyEnumFields(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"title": {Type: "string", Required: true},
		},
	}

	records := []*extract.Record{
		{Path: "dir/f1.md", Frontmatter: map[string]any{"title": "foo"}},
		{Path: "dir/f2.md", Frontmatter: map[string]any{"title": "foo"}},
		{Path: "dir/f3.md", Frontmatter: map[string]any{}},
	}

	errs := map[string][]rules.ValidationError{
		"dir/f3.md": {{Rule: "required", Field: "title", Message: `required field "title" is missing`}},
	}

	proposals := detectInferFromSiblings(records, stem, errs)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (non-enum field)", len(proposals))
	}
}

func TestInferFromSiblings_CrossDirectoryIsolation(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"a", "b"}, Required: true},
		},
	}

	// dir1 has "a" majority, dir2 has "b" majority — should not mix
	records := []*extract.Record{
		{Path: "dir1/f1.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir1/f2.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir2/f1.md", Frontmatter: map[string]any{"tipo": "b"}},
		{Path: "dir2/f2.md", Frontmatter: map[string]any{"tipo": "b"}},
		{Path: "dir1/f3.md", Frontmatter: map[string]any{}},
	}

	errs := map[string][]rules.ValidationError{
		"dir1/f3.md": {{Rule: "required", Field: "tipo", Message: `required field "tipo" is missing`}},
	}

	proposals := detectInferFromSiblings(records, stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Value != "a" {
		t.Errorf("value = %q, want %q (isolated to dir1)", proposals[0].Value, "a")
	}
}

// --- detectOutlierValue tests ---

func TestOutlierValue_StrongConsensus(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"a", "b", "c"}, Required: true},
		},
	}

	records := []*extract.Record{
		{Path: "dir/f1.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f2.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f3.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f4.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f5.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f6.md", Frontmatter: map[string]any{"tipo": "b"}},
	}

	errs := map[string][]rules.ValidationError{} // no validation errors

	proposals := detectOutlierValue(records, stem, errs)
	if len(proposals) != 1 {
		t.Fatalf("got %d proposals, want 1", len(proposals))
	}
	if proposals[0].Type != CorrectOutlier {
		t.Errorf("type = %q, want correct_outlier", proposals[0].Type)
	}
	if proposals[0].From != "b" {
		t.Errorf("from = %q, want %q", proposals[0].From, "b")
	}
	if proposals[0].To != "a" {
		t.Errorf("to = %q, want %q", proposals[0].To, "a")
	}
}

func TestOutlierValue_WeakConsensus(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"a", "b"}, Required: true},
		},
	}

	// 3 with "a", 2 with "b" — 60% < 75% threshold
	records := []*extract.Record{
		{Path: "dir/f1.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f2.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f3.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f4.md", Frontmatter: map[string]any{"tipo": "b"}},
		{Path: "dir/f5.md", Frontmatter: map[string]any{"tipo": "b"}},
	}

	errs := map[string][]rules.ValidationError{}

	proposals := detectOutlierValue(records, stem, errs)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (weak consensus)", len(proposals))
	}
}

func TestOutlierValue_MinimumSiblings(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"a", "b"}, Required: true},
		},
	}

	// Only 2 agree — below minSiblingsForOutlier (3)
	records := []*extract.Record{
		{Path: "dir/f1.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f2.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f3.md", Frontmatter: map[string]any{"tipo": "b"}},
	}

	errs := map[string][]rules.ValidationError{}

	proposals := detectOutlierValue(records, stem, errs)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (below minimum siblings)", len(proposals))
	}
}

func TestOutlierValue_SkipsRecordsWithErrors(t *testing.T) {
	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"tipo": {Type: "enum", Values: []string{"a", "b"}, Required: true},
		},
	}

	records := []*extract.Record{
		{Path: "dir/f1.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f2.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f3.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f4.md", Frontmatter: map[string]any{"tipo": "a"}},
		{Path: "dir/f5.md", Frontmatter: map[string]any{"tipo": "b"}}, // has error — should be excluded
	}

	// f5 has an enum error for "tipo" — it should be excluded from counting
	errs := map[string][]rules.ValidationError{
		"dir/f5.md": {{Rule: "enum", Field: "tipo", Message: `value "b" not in allowed values`}},
	}

	proposals := detectOutlierValue(records, stem, errs)
	if len(proposals) != 0 {
		t.Errorf("got %d proposals, want 0 (record with errors excluded)", len(proposals))
	}
}
