package query

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestParseSortKeys_SingleFieldDefaultAsc(t *testing.T) {
	keys, err := ParseSortKeys("prioridad")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 {
		t.Fatalf("expected 1 key, got %d", len(keys))
	}
	if keys[0].Field != "prioridad" {
		t.Errorf("field = %q, want prioridad", keys[0].Field)
	}
	if keys[0].Desc {
		t.Error("expected Desc=false for default")
	}
}

func TestParseSortKeys_SingleFieldExplicitAsc(t *testing.T) {
	keys, err := ParseSortKeys("prioridad:asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 || keys[0].Field != "prioridad" || keys[0].Desc {
		t.Errorf("got %+v, want [{prioridad false}]", keys)
	}
}

func TestParseSortKeys_SingleFieldDesc(t *testing.T) {
	keys, err := ParseSortKeys("impact_score:desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 1 || keys[0].Field != "impact_score" || !keys[0].Desc {
		t.Errorf("got %+v, want [{impact_score true}]", keys)
	}
}

func TestParseSortKeys_MultipleKeys(t *testing.T) {
	keys, err := ParseSortKeys("prioridad:asc,impact_score:desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].Field != "prioridad" || keys[0].Desc {
		t.Errorf("key[0] = %+v, want {prioridad false}", keys[0])
	}
	if keys[1].Field != "impact_score" || !keys[1].Desc {
		t.Errorf("key[1] = %+v, want {impact_score true}", keys[1])
	}
}

func TestParseSortKeys_MixedDefaults(t *testing.T) {
	keys, err := ParseSortKeys("a,b:desc,c:asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 3 {
		t.Fatalf("expected 3 keys, got %d", len(keys))
	}
	if keys[0].Field != "a" || keys[0].Desc {
		t.Errorf("key[0] = %+v", keys[0])
	}
	if keys[1].Field != "b" || !keys[1].Desc {
		t.Errorf("key[1] = %+v", keys[1])
	}
	if keys[2].Field != "c" || keys[2].Desc {
		t.Errorf("key[2] = %+v", keys[2])
	}
}

func TestParseSortKeys_EmptyString(t *testing.T) {
	keys, err := ParseSortKeys("")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 0 {
		t.Errorf("expected 0 keys for empty string, got %d", len(keys))
	}
}

func TestParseSortKeys_InvalidDirection(t *testing.T) {
	_, err := ParseSortKeys("field:invalid")
	if err == nil {
		t.Fatal("expected error for invalid direction")
	}
}

func TestParseSortKeys_TooManyColons(t *testing.T) {
	_, err := ParseSortKeys("field:asc:extra")
	if err == nil {
		t.Fatal("expected error for too many colons")
	}
}

func TestParseSortKeys_WhitespaceHandling(t *testing.T) {
	keys, err := ParseSortKeys(" prioridad : asc , impact_score : desc ")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(keys) != 2 {
		t.Fatalf("expected 2 keys, got %d", len(keys))
	}
	if keys[0].Field != "prioridad" || keys[0].Desc {
		t.Errorf("key[0] = %+v", keys[0])
	}
	if keys[1].Field != "impact_score" || !keys[1].Desc {
		t.Errorf("key[1] = %+v", keys[1])
	}
}

// makeSortRecords creates test records with varied field values.
func makeSortRecords() []*extract.Record {
	return []*extract.Record{
		{
			Path:        "B003.md",
			Type:        "markdown",
			Frontmatter: map[string]any{"prioridad": "media", "impact_score": 7},
		},
		{
			Path:        "B001.md",
			Type:        "markdown",
			Frontmatter: map[string]any{"prioridad": "alta", "impact_score": 9},
		},
		{
			Path:        "B004.md",
			Type:        "markdown",
			Frontmatter: map[string]any{"prioridad": "baja", "impact_score": 3},
		},
		{
			Path:        "B002.md",
			Type:        "markdown",
			Frontmatter: map[string]any{"prioridad": "alta", "impact_score": 5},
		},
	}
}

func TestSortRecords_SingleEnumAsc(t *testing.T) {
	records := makeSortRecords()
	schema := map[string]rules.SchemaField{
		"prioridad": {Type: "enum", Values: []string{"alta", "media", "baja"}},
	}
	keys := []SortKey{{Field: "prioridad", Desc: false}}

	SortRecords(records, keys, schema)

	// alta=0, media=1, baja=2 → B001, B002, B003, B004
	expected := []string{"B001.md", "B002.md", "B003.md", "B004.md"}
	for i, want := range expected {
		if records[i].Path != want {
			t.Errorf("position %d: got %s, want %s", i, records[i].Path, want)
		}
	}
}

func TestSortRecords_SingleEnumDesc(t *testing.T) {
	records := makeSortRecords()
	schema := map[string]rules.SchemaField{
		"prioridad": {Type: "enum", Values: []string{"alta", "media", "baja"}},
	}
	keys := []SortKey{{Field: "prioridad", Desc: true}}

	SortRecords(records, keys, schema)

	// baja=2 first → B004, B003, then alta: B001 or B002 (stable)
	if records[0].Path != "B004.md" {
		t.Errorf("position 0: got %s, want B004.md", records[0].Path)
	}
	if records[1].Path != "B003.md" {
		t.Errorf("position 1: got %s, want B003.md", records[1].Path)
	}
	// B001 and B002 are both alta -- stable order preserved (B001 was before B002 in input)
	if records[2].Path != "B001.md" {
		t.Errorf("position 2: got %s, want B001.md", records[2].Path)
	}
	if records[3].Path != "B002.md" {
		t.Errorf("position 3: got %s, want B002.md", records[3].Path)
	}
}

func TestSortRecords_NumericAsc(t *testing.T) {
	records := makeSortRecords()
	keys := []SortKey{{Field: "impact_score", Desc: false}}

	SortRecords(records, keys, nil)

	// 3, 5, 7, 9
	expected := []string{"B004.md", "B002.md", "B003.md", "B001.md"}
	for i, want := range expected {
		if records[i].Path != want {
			t.Errorf("position %d: got %s, want %s", i, records[i].Path, want)
		}
	}
}

func TestSortRecords_NumericDesc(t *testing.T) {
	records := makeSortRecords()
	keys := []SortKey{{Field: "impact_score", Desc: true}}

	SortRecords(records, keys, nil)

	// 9, 7, 5, 3
	expected := []string{"B001.md", "B003.md", "B002.md", "B004.md"}
	for i, want := range expected {
		if records[i].Path != want {
			t.Errorf("position %d: got %s, want %s", i, records[i].Path, want)
		}
	}
}

func TestSortRecords_MultiKeyEnumThenNumericDesc(t *testing.T) {
	records := makeSortRecords()
	schema := map[string]rules.SchemaField{
		"prioridad": {Type: "enum", Values: []string{"alta", "media", "baja"}},
	}
	// Primary: prioridad asc (alta first), secondary: impact_score desc (highest first)
	keys := []SortKey{
		{Field: "prioridad", Desc: false},
		{Field: "impact_score", Desc: true},
	}

	SortRecords(records, keys, schema)

	// alta group (impact desc): B001(9), B002(5)
	// media group: B003(7)
	// baja group: B004(3)
	expected := []string{"B001.md", "B002.md", "B003.md", "B004.md"}
	for i, want := range expected {
		if records[i].Path != want {
			t.Errorf("position %d: got %s, want %s", i, records[i].Path, want)
		}
	}
}

func TestSortRecords_NilValuesLast(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"score": 5}},
		{Path: "b.md", Frontmatter: map[string]any{}}, // missing score
		{Path: "c.md", Frontmatter: map[string]any{"score": 3}},
	}
	keys := []SortKey{{Field: "score", Desc: false}}

	SortRecords(records, keys, nil)

	// 3, 5, nil-last
	if records[0].Path != "c.md" {
		t.Errorf("position 0: got %s, want c.md", records[0].Path)
	}
	if records[1].Path != "a.md" {
		t.Errorf("position 1: got %s, want a.md", records[1].Path)
	}
	if records[2].Path != "b.md" {
		t.Errorf("position 2: got %s, want b.md (nil last)", records[2].Path)
	}
}

func TestSortRecords_NilValuesLastEvenWithDesc(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"score": 5}},
		{Path: "b.md", Frontmatter: map[string]any{}}, // missing score
		{Path: "c.md", Frontmatter: map[string]any{"score": 3}},
	}
	keys := []SortKey{{Field: "score", Desc: true}}

	SortRecords(records, keys, nil)

	// desc: 5, 3, nil-last
	if records[0].Path != "a.md" {
		t.Errorf("position 0: got %s, want a.md", records[0].Path)
	}
	if records[1].Path != "c.md" {
		t.Errorf("position 1: got %s, want c.md", records[1].Path)
	}
	if records[2].Path != "b.md" {
		t.Errorf("position 2: got %s, want b.md (nil last)", records[2].Path)
	}
}

func TestSortRecords_StringLexicographic(t *testing.T) {
	records := []*extract.Record{
		{Path: "c.md", Frontmatter: map[string]any{"title": "Charlie"}},
		{Path: "a.md", Frontmatter: map[string]any{"title": "Alpha"}},
		{Path: "b.md", Frontmatter: map[string]any{"title": "Bravo"}},
	}
	keys := []SortKey{{Field: "title", Desc: false}}

	SortRecords(records, keys, nil)

	expected := []string{"a.md", "b.md", "c.md"}
	for i, want := range expected {
		if records[i].Path != want {
			t.Errorf("position %d: got %s, want %s", i, records[i].Path, want)
		}
	}
}

func TestSortRecords_EmptyKeysNoOp(t *testing.T) {
	records := makeSortRecords()
	original := make([]string, len(records))
	for i, r := range records {
		original[i] = r.Path
	}

	SortRecords(records, nil, nil)

	for i, r := range records {
		if r.Path != original[i] {
			t.Errorf("position %d changed: got %s, want %s", i, r.Path, original[i])
		}
	}
}

func TestSortRecords_DerivedFieldOverride(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "a.md",
			Frontmatter: map[string]any{"score": 1},
			Derived:     map[string]any{"score": 10}, // derived overrides
		},
		{
			Path:        "b.md",
			Frontmatter: map[string]any{"score": 5},
		},
	}
	keys := []SortKey{{Field: "score", Desc: false}}

	SortRecords(records, keys, nil)

	// a has effective score=10 (derived), b has score=5 → b first
	if records[0].Path != "b.md" {
		t.Errorf("position 0: got %s, want b.md (score=5)", records[0].Path)
	}
	if records[1].Path != "a.md" {
		t.Errorf("position 1: got %s, want a.md (derived score=10)", records[1].Path)
	}
}

func TestSortRecords_EnumValueNotInList(t *testing.T) {
	// Unknown enum values sort after known ones (by appending to end)
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"status": "unknown_value"}},
		{Path: "b.md", Frontmatter: map[string]any{"status": "alta"}},
		{Path: "c.md", Frontmatter: map[string]any{"status": "baja"}},
	}
	schema := map[string]rules.SchemaField{
		"status": {Type: "enum", Values: []string{"alta", "media", "baja"}},
	}
	keys := []SortKey{{Field: "status", Desc: false}}

	SortRecords(records, keys, schema)

	// alta=0, baja=2, unknown=len(values)=3
	if records[0].Path != "b.md" {
		t.Errorf("position 0: got %s, want b.md (alta)", records[0].Path)
	}
	if records[1].Path != "c.md" {
		t.Errorf("position 1: got %s, want c.md (baja)", records[1].Path)
	}
	if records[2].Path != "a.md" {
		t.Errorf("position 2: got %s, want a.md (unknown, sorts last)", records[2].Path)
	}
}

func TestSortRecords_NumericStringValues(t *testing.T) {
	// Values stored as strings that look like numbers should sort numerically
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"score": "10"}},
		{Path: "b.md", Frontmatter: map[string]any{"score": "2"}},
		{Path: "c.md", Frontmatter: map[string]any{"score": "100"}},
	}
	keys := []SortKey{{Field: "score", Desc: false}}

	SortRecords(records, keys, nil)

	// Numeric: 2, 10, 100
	expected := []string{"b.md", "a.md", "c.md"}
	for i, want := range expected {
		if records[i].Path != want {
			t.Errorf("position %d: got %s, want %s", i, records[i].Path, want)
		}
	}
}
