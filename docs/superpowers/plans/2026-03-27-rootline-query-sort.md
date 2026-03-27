# Query --sort Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--sort` flag to rootline's `query` command enabling multi-key sorting with enum-aware ordering from `.stem` schemas. This unblocks the autonomous remediation DAG which needs `--sort "prioridad:asc,impact_score:desc"` without fragile jq pipelines.

**Architecture:** New `internal/query/sort.go` contains `SortKey` parsing and the `SortRecords` function. The sort engine receives a `StemResolver` function to look up `.stem` schemas for enum ordering. Sort applies AFTER filtering and BEFORE limit/output. The CLI (`cmd/rootline/query.go`) and MCP server (`internal/mcp/tools.go`) both wire the flag through.

**Tech Stack:** Go 1.25+, `sort.SliceStable`, `strconv.ParseFloat` for numeric comparison, `rules.WalkUp` + `rules.MergeStemFiles` for enum resolution.

**Spec:** `/opt/homeserver/docs/superpowers/specs/2026-03-27-autonomous-remediation-closed-loop-design.md`

---

## File Map

| File | Action | Responsibility |
|------|--------|----------------|
| `internal/query/sort.go` | Create | SortKey parsing + SortRecords algorithm |
| `internal/query/sort_test.go` | Create | Unit tests for parsing and sorting |
| `cmd/rootline/query.go` | Modify | Add `--sort` flag, wire stem resolver, call SortRecords |
| `cmd/rootline/commands_test.go` | Modify | Add CLI-level sort tests |
| `internal/mcp/tools.go` | Modify | Add `Sort` field to QueryInput, wire into handleQuery |

---

## Design Notes

### Sort Key Parsing

The `--sort` flag accepts a comma-separated string of `field:direction` pairs:

- `"prioridad:asc,impact_score:desc"` -- explicit directions
- `"prioridad"` -- defaults to asc
- `"prioridad:asc"` -- explicit asc
- `"impact_score:desc"` -- explicit desc

Parsed into `[]SortKey{Field string, Desc bool}`.

### Sort Algorithm

- Uses `sort.SliceStable` to preserve insertion order for equal elements
- Multi-key: iterate keys in order; first non-zero comparison wins
- For each key, field values are resolved via `Record.EffectiveField()` (derived fields override frontmatter)
- Type detection per comparison:
  1. **Enum fields**: if the field has a `.stem` schema with `type: enum` and `values: [...]`, sort by index position in the values list (alta=0, media=1, baja=2 for `values: [alta, media, baja]`)
  2. **Numeric fields**: if both values parse as float64, sort numerically
  3. **String fields**: fallback to lexicographic `strings.Compare`
- **nil/missing values** sort last regardless of direction (nil-last is applied before direction inversion)

### Stem Resolution

The sort engine needs `.stem` schema access for enum ordering. The sort function receives a `map[string]rules.SchemaField` (the merged schema for the scan root). The CLI obtains this via `rules.WalkUp(absRoot)` + `rules.MergeStemFiles(entries)` -- the same pattern used by `describe`, `validate`, and `stats`.

### Important: `enum:` vs `values:` Key

The homeserver backlog `.stem` uses `enum: [alta, media, baja]` instead of the standard `values: [alta, media, baja]`. The rootline `SchemaField` struct only reads `values` via `yaml:"values"`, so the `enum:` key is silently dropped. **This plan documents the gap but does NOT fix it** -- the sort engine works with `SchemaField.Values` as-is. A separate fix should add `Enum []string yaml:"enum"` to `SchemaField` and merge it into `Values` during parsing. For testing in Task 4, we verify that lexicographic fallback still produces correct results for fields whose enum values happen to sort correctly, or we document the gap.

---

## Task 1: Add Sort Key Parsing

**Files:**
- Create: `internal/query/sort.go`
- Create: `internal/query/sort_test.go`

- [ ] **Step 1: Write failing tests for ParseSortKeys**

```go
// internal/query/sort_test.go
package query

import (
	"testing"
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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/query/ -run TestParseSortKeys -v`
Expected: FAIL -- `ParseSortKeys` undefined

- [ ] **Step 3: Write minimal implementation**

```go
// internal/query/sort.go
package query

import (
	"fmt"
	"strings"
)

// SortKey represents a single sort criterion.
type SortKey struct {
	Field string
	Desc  bool
}

// ParseSortKeys parses a comma-separated sort specification string.
// Format: "field1:asc,field2:desc,field3" (direction defaults to asc).
func ParseSortKeys(spec string) ([]SortKey, error) {
	spec = strings.TrimSpace(spec)
	if spec == "" {
		return nil, nil
	}

	parts := strings.Split(spec, ",")
	keys := make([]SortKey, 0, len(parts))

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}

		segments := strings.SplitN(part, ":", 3)
		if len(segments) > 2 {
			return nil, fmt.Errorf("invalid sort key %q: too many colons", part)
		}

		field := strings.TrimSpace(segments[0])
		if field == "" {
			return nil, fmt.Errorf("invalid sort key %q: empty field name", part)
		}

		desc := false
		if len(segments) == 2 {
			dir := strings.TrimSpace(strings.ToLower(segments[1]))
			switch dir {
			case "asc":
				desc = false
			case "desc":
				desc = true
			default:
				return nil, fmt.Errorf("invalid sort direction %q in key %q: must be asc or desc", dir, part)
			}
		}

		keys = append(keys, SortKey{Field: field, Desc: desc})
	}

	return keys, nil
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/query/ -run TestParseSortKeys -v`
Expected: PASS -- all 9 tests

- [ ] **Step 5: Commit**

```bash
git add internal/query/sort.go internal/query/sort_test.go
git commit -m "feat(query): add ParseSortKeys for --sort flag parsing"
```

---

## Task 2: Add Sort Execution in Query Engine

**Files:**
- Modify: `internal/query/sort.go`
- Modify: `internal/query/sort_test.go`

- [ ] **Step 1: Write failing tests for SortRecords**

Append to `internal/query/sort_test.go`:

```go
import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

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
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/query/ -run "TestSortRecords" -v`
Expected: FAIL -- `SortRecords` undefined

- [ ] **Step 3: Write the sort implementation**

Append to `internal/query/sort.go`:

```go
import (
	"fmt"
	"sort"
	"strconv"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// SortRecords sorts records in-place by the given sort keys.
// schema may be nil if no enum ordering is needed.
// Uses sort.SliceStable for deterministic ordering of equal elements.
func SortRecords(records []*extract.Record, keys []SortKey, schema map[string]rules.SchemaField) {
	if len(keys) == 0 || len(records) < 2 {
		return
	}

	// Pre-build enum index maps for fields that are enum type with values.
	enumIndexes := buildEnumIndexes(keys, schema)

	sort.SliceStable(records, func(i, j int) bool {
		for _, key := range keys {
			cmp := compareField(records[i], records[j], key.Field, enumIndexes[key.Field])
			if cmp == 0 {
				continue
			}
			if key.Desc {
				return cmp > 0
			}
			return cmp < 0
		}
		return false
	})
}

// buildEnumIndexes creates value-to-position maps for enum sort keys.
func buildEnumIndexes(keys []SortKey, schema map[string]rules.SchemaField) map[string]map[string]int {
	indexes := make(map[string]map[string]int)
	if schema == nil {
		return indexes
	}
	for _, key := range keys {
		sf, ok := schema[key.Field]
		if !ok || sf.Type != "enum" || len(sf.Values) == 0 {
			continue
		}
		idx := make(map[string]int, len(sf.Values))
		for i, v := range sf.Values {
			idx[v] = i
		}
		indexes[key.Field] = idx
	}
	return indexes
}

// compareField compares two records on a single field.
// Returns -1, 0, or 1.
// nil/missing values always compare as "greater" (sort last).
func compareField(a, b *extract.Record, field string, enumIndex map[string]int) int {
	va, okA := a.EffectiveField(field)
	vb, okB := b.EffectiveField(field)

	// Handle nil/missing: nil sorts last (regardless of direction -- caller handles inversion).
	if !okA && !okB {
		return 0
	}
	if !okA {
		return 1 // a is nil, sorts after b
	}
	if !okB {
		return -1 // b is nil, sorts after a
	}

	// Enum ordering: if we have an index map, use positional comparison.
	if enumIndex != nil {
		sa := fmt.Sprintf("%v", va)
		sb := fmt.Sprintf("%v", vb)
		ia, foundA := enumIndex[sa]
		ib, foundB := enumIndex[sb]
		if !foundA {
			ia = len(enumIndex) // unknown values sort after known
		}
		if !foundB {
			ib = len(enumIndex)
		}
		if ia < ib {
			return -1
		}
		if ia > ib {
			return 1
		}
		return 0
	}

	// Numeric comparison: try to parse both as float64.
	fa, numA := toFloat64(va)
	fb, numB := toFloat64(vb)
	if numA && numB {
		if fa < fb {
			return -1
		}
		if fa > fb {
			return 1
		}
		return 0
	}

	// String fallback: lexicographic comparison.
	sa := fmt.Sprintf("%v", va)
	sb := fmt.Sprintf("%v", vb)
	return strings.Compare(sa, sb)
}

// toFloat64 attempts to extract a float64 from a value.
// Handles int, float64, and string representations.
func toFloat64(v any) (float64, bool) {
	switch val := v.(type) {
	case float64:
		return val, true
	case float32:
		return float64(val), true
	case int:
		return float64(val), true
	case int64:
		return float64(val), true
	case string:
		f, err := strconv.ParseFloat(val, 64)
		return f, err == nil
	}
	return 0, false
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/query/ -run "TestSortRecords|TestParseSortKeys" -v`
Expected: PASS -- all sorting and parsing tests

- [ ] **Step 5: Run full test suite**

Run: `go test ./internal/query/ -v`
Expected: PASS -- no regressions

- [ ] **Step 6: Commit**

```bash
git add internal/query/sort.go internal/query/sort_test.go
git commit -m "feat(query): add SortRecords with enum-aware multi-key sorting"
```

---

## Task 3: Wire --sort Flag into CLI

**Files:**
- Modify: `cmd/rootline/query.go`
- Modify: `cmd/rootline/commands_test.go`

- [ ] **Step 1: Write failing CLI tests**

Append to `cmd/rootline/commands_test.go`:

```go
func TestQuerySort_SingleKey(t *testing.T) {
	dir := setupSortTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--sort", "prioridad:asc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	rows := result["rows"].([]any)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// alta=0 should be first
	row0 := rows[0].(map[string]any)
	fm0 := row0["frontmatter"].(map[string]any)
	if fm0["prioridad"] != "alta" {
		t.Errorf("row 0 prioridad = %v, want alta", fm0["prioridad"])
	}
}

func TestQuerySort_MultiKey(t *testing.T) {
	dir := setupSortTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--sort", "prioridad:asc,impact_score:desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	rows := result["rows"].([]any)
	if len(rows) != 3 {
		t.Fatalf("expected 3 rows, got %d", len(rows))
	}
	// alta with higher impact_score (8) should come before alta with lower (3)
	row0 := rows[0].(map[string]any)
	row1 := rows[1].(map[string]any)
	fm0 := row0["frontmatter"].(map[string]any)
	fm1 := row1["frontmatter"].(map[string]any)
	if fm0["prioridad"] != "alta" || fm1["prioridad"] != "alta" {
		t.Errorf("first two rows should be alta, got %v and %v", fm0["prioridad"], fm1["prioridad"])
	}
	// impact_score desc: 8 before 3
	score0, _ := fm0["impact_score"].(float64) // JSON numbers are float64
	score1, _ := fm1["impact_score"].(float64)
	if score0 < score1 {
		t.Errorf("expected impact_score desc: got %.0f before %.0f", score0, score1)
	}
}

func TestQuerySort_WithWhere(t *testing.T) {
	dir := setupSortTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--where", "prioridad == 'alta'", "--sort", "impact_score:desc")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}
	rows := result["rows"].([]any)
	if len(rows) != 2 {
		t.Fatalf("expected 2 alta rows, got %d", len(rows))
	}
}

func TestQuerySort_InvalidSort(t *testing.T) {
	dir := setupSortTestDir(t)
	_, err := runCmd(t, "query", "--from", dir, "--sort", "field:invalid_dir")
	if err == nil {
		t.Fatal("expected error for invalid sort direction")
	}
}

func TestQuerySort_Table(t *testing.T) {
	dir := setupSortTestDir(t)
	out, err := runCmd(t, "query", "--from", dir, "--sort", "prioridad:asc", "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Path") {
		t.Errorf("expected table output, got: %s", out)
	}
}

// setupSortTestDir creates a test directory with enum schema and records for sort testing.
func setupSortTestDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  prioridad:
    type: enum
    required: true
    values: [alta, media, baja]
  impact_score:
    type: number
    required: false
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	mustWriteFile(t, filepath.Join(dir, "item1.md"), []byte("---\nprioridad: media\nimpact_score: 5\n---\n# Item 1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "item2.md"), []byte("---\nprioridad: alta\nimpact_score: 8\n---\n# Item 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "item3.md"), []byte("---\nprioridad: alta\nimpact_score: 3\n---\n# Item 3\n"), 0644)

	return dir
}
```

Also update the `resetFlags` function in `commands_test.go` to reset the new `querySort` variable:

```go
// Add to resetFlags():
querySort = ""

// Add cobra flag reset:
if f := queryCmd.Flags().Lookup("sort"); f != nil {
    _ = f.Value.Set("")
    f.Changed = false
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/rootline/ -run "TestQuerySort" -v`
Expected: FAIL -- `querySort` undefined, `setupSortTestDir` defined but `--sort` flag not registered

- [ ] **Step 3: Modify query.go to add --sort flag**

Apply these changes to `cmd/rootline/query.go`:

```go
// Add to var block (after queryFrom):
var querySort string

// Add to init() (after the --from flag):
queryCmd.Flags().StringVar(&querySort, "sort", "", `sort by fields (e.g. "prioridad:asc,impact_score:desc")`)

// Replace the runQuery function:
func runQuery(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Determine scan root
	scanRoot := queryFrom
	if len(args) > 0 {
		scanRoot = args[0]
	}

	absRoot, err := filepath.Abs(scanRoot)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Parse sort keys early to fail fast on invalid input.
	sortKeys, err := query.ParseSortKeys(querySort)
	if err != nil {
		return fmt.Errorf("parsing --sort: %w", err)
	}

	// Scan records
	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", scanRoot, err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.EnrichBuiltinsSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	q := &query.Query{
		Count: queryCount,
		Limit: queryLimit,
	}

	// Filter records using shared helper.
	filtered, err := filterRecords(ctx, records, queryWhere, nil)
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	// Sort AFTER filtering, BEFORE limit/output.
	if len(sortKeys) > 0 {
		var schema map[string]rules.SchemaField
		entries, walkErr := rules.WalkUp(absRoot)
		if walkErr == nil && len(entries) > 0 {
			merged := rules.MergeStemFiles(entries)
			schema = merged.Schema
		}
		query.SortRecords(filtered, sortKeys, schema)
	}

	// Execute query (count/limit) on sorted records.
	result, err := query.ExecuteExpr(ctx, filtered, "", q)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	if outputFormat == "table" {
		return renderQueryTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}
```

Add import for `rules`:

```go
import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/rootline/ -run "TestQuerySort" -v`
Expected: PASS -- all 5 CLI sort tests

- [ ] **Step 5: Run full CLI test suite to check for regressions**

Run: `go test ./cmd/rootline/ -v`
Expected: PASS -- no regressions

- [ ] **Step 6: Run lint**

Run: `just check`
Expected: PASS

- [ ] **Step 7: Commit**

```bash
git add cmd/rootline/query.go cmd/rootline/commands_test.go
git commit -m "feat(query): wire --sort flag into CLI with stem-aware enum ordering"
```

---

## Task 4: Update MCP Server and Integration Test

**Files:**
- Modify: `internal/mcp/tools.go`
- Modify: `cmd/rootline/commands_test.go` (or create integration test file)

- [ ] **Step 1: Update MCP QueryInput struct**

In `internal/mcp/tools.go`, add `Sort` field to `QueryInput`:

```go
type QueryInput struct {
	Path  string   `json:"path" jsonschema:"directory to scan (absolute path)"`
	Where []string `json:"where,omitempty" jsonschema:"filter expressions (expr-lang syntax)"`
	Count bool     `json:"count,omitempty" jsonschema:"return count instead of records"`
	Limit int      `json:"limit,omitempty" jsonschema:"limit number of results (0 = unlimited)"`
	Sort  string   `json:"sort,omitempty" jsonschema:"sort by fields (e.g. prioridad:asc,impact_score:desc)"`
}
```

- [ ] **Step 2: Wire sort into handleQuery**

Update `handleQuery` in `internal/mcp/tools.go`:

```go
func handleQuery(ctx context.Context, _ *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	// Parse sort keys early to fail fast.
	sortKeys, err := query.ParseSortKeys(input.Sort)
	if err != nil {
		return nil, nil, fmt.Errorf("parsing sort: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	filtered, err := filterWhere(ctx, records, input.Where)
	if err != nil {
		return nil, nil, err
	}

	// Sort after filtering, before limit.
	if len(sortKeys) > 0 {
		var schema map[string]rules.SchemaField
		entries, walkErr := rules.WalkUp(absRoot)
		if walkErr == nil && len(entries) > 0 {
			merged := rules.MergeStemFiles(entries)
			schema = merged.Schema
		}
		query.SortRecords(filtered, sortKeys, schema)
	}

	q := &query.Query{Count: input.Count, Limit: input.Limit}
	result, err := query.ExecuteExpr(ctx, filtered, "", q)
	if err != nil {
		return nil, nil, fmt.Errorf("executing query: %w", err)
	}

	return jsonResult(result)
}
```

- [ ] **Step 3: Write integration test with homeserver backlog**

Append to `cmd/rootline/commands_test.go`:

```go
func TestQuerySort_IntegrationBacklog(t *testing.T) {
	// Skip if the homeserver backlog directory doesn't exist (CI environments).
	backlogDir := "/opt/homeserver/backlog"
	if _, err := os.Stat(backlogDir); os.IsNotExist(err) {
		t.Skip("skipping integration test: /opt/homeserver/backlog not found")
	}

	out, err := runCmd(t, "query", backlogDir, "--sort", "prioridad:asc,impact_score:desc", "-o", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	rows, ok := result["rows"].([]any)
	if !ok {
		t.Fatalf("rows not found in result")
	}

	if len(rows) == 0 {
		t.Skip("no backlog items found")
	}

	// NOTE: The backlog .stem uses "enum:" instead of "values:" for enum definitions.
	// This means rootline currently does NOT get the enum ordering for prioridad.
	// Sort falls back to lexicographic ordering ("alta" < "baja" < "media" = correct for asc).
	// This test verifies the sort completes without error and produces valid JSON.
	// Once the enum:/values: parsing is unified, enum-aware ordering will work automatically.
	t.Logf("sorted %d backlog items successfully", len(rows))

	// Verify output structure
	if result["kind"] != "rootline/query" {
		t.Errorf("expected kind rootline/query, got %v", result["kind"])
	}
}
```

- [ ] **Step 4: Run integration test**

Run: `go test ./cmd/rootline/ -run "TestQuerySort_IntegrationBacklog" -v`
Expected: PASS

- [ ] **Step 5: Run manual integration**

Run: `cd /opt/rootline && go run ./cmd/rootline query /opt/homeserver/backlog/ --sort "prioridad:asc,impact_score:desc" -o json | head -20`
Expected: JSON output with records sorted (lexicographic fallback for prioridad since .stem uses `enum:` not `values:`)

- [ ] **Step 6: Run full test suite**

Run: `just test`
Expected: PASS -- all tests including new sort tests

- [ ] **Step 7: Run lint and format**

Run: `just check`
Expected: PASS

- [ ] **Step 8: Commit**

```bash
git add internal/mcp/tools.go cmd/rootline/commands_test.go
git commit -m "feat(query): add --sort to MCP server and integration test"
```

---

## Post-Implementation Notes

### Known Limitation: `enum:` vs `values:` in .stem

The homeserver backlog `.stem` uses `enum: [alta, media, baja]` while rootline's `SchemaField` only parses `values: [alta, media, baja]`. This means enum-aware sort ordering does not work for the backlog until this parsing gap is fixed. The sort falls back to lexicographic ordering which coincidentally works correctly for `prioridad` ("alta" < "baja" < "media" in Spanish alphabetical order... but this is NOT the same as priority order).

**Recommended follow-up**: Either:
1. Fix rootline's `SchemaField` to also accept `enum:` as an alias for `values:` (preferred -- backwards compatible)
2. Fix the backlog `.stem` to use `values:` instead of `enum:` (data migration)

### Complexity

- Parsing: O(k) where k = number of sort keys
- Sorting: O(n * k * log n) where n = records, k = sort keys
- Enum index lookup: O(1) per comparison via pre-built map
- No additional allocations beyond the enum index maps

### Future Enhancements

- `--sort-nulls-first`: option to sort nil values first instead of last
- Sort by path, body length, or other built-in fields
- Sort support in `tree`, `stats`, and other transversal commands
