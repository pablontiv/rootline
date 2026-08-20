package main

import (
	"encoding/csv"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func setupTask9QueryProject(t *testing.T) string {
	t.Helper()
	dir := setupTask9SourceProject(t, false)
	mustWriteFile(t, filepath.Join(dir, "nil.md"), []byte("---\ntitle: Nil\nnotes:\n---\n# Nil\n\n## Notes\n\nbody ignored\n"), 0o644)
	return dir
}

func TestTask9QueryFormatsPreserveSourceBackedOrderAndCounts(t *testing.T) {
	dir := setupTask9QueryProject(t)

	fullOut, err := runCmd(t, "query", dir, "--sort", "notes:desc")
	if err != nil {
		t.Fatalf("full query failed: %v\n%s", err, fullOut)
	}
	var fullEnvelope map[string]any
	if err := json.Unmarshal([]byte(fullOut), &fullEnvelope); err != nil {
		t.Fatalf("invalid full query JSON: %v\n%s", err, fullOut)
	}
	assertJSONKeys(t, fullEnvelope, []string{"version", "kind", "meta", "rows"})
	assertJSONKeys(t, fullEnvelope["meta"].(map[string]any), []string{"count"})
	for _, raw := range fullEnvelope["rows"].([]any) {
		assertJSONKeys(t, raw.(map[string]any), []string{"path", "type", "frontmatter", "body", "sections", "body_sections", "derived"})
	}
	var full struct {
		Version int                 `json:"version"`
		Kind    string              `json:"kind"`
		Meta    struct{ Count int } `json:"meta"`
		Rows    []struct {
			Path        string         `json:"path"`
			Type        string         `json:"type"`
			Frontmatter map[string]any `json:"frontmatter"`
			Body        string         `json:"body"`
			Derived     map[string]any `json:"derived"`
		} `json:"rows"`
	}
	if err := json.Unmarshal([]byte(fullOut), &full); err != nil {
		t.Fatalf("invalid typed full query JSON: %v\n%s", err, fullOut)
	}
	if full.Version != 1 || full.Kind != "rootline/query" || full.Meta.Count != 4 || len(full.Rows) != 4 {
		t.Fatalf("full identity/count = v%d %q meta %d rows %d", full.Version, full.Kind, full.Meta.Count, len(full.Rows))
	}
	wantFullOrder := []string{"override.md", "body.md", "nil.md", "empty.md"}
	if got := task9FullPaths(full.Rows); !reflect.DeepEqual(got, wantFullOrder) {
		t.Fatalf("full query order = %v, want %v", got, wantFullOrder)
	}
	fullByPath := map[string]struct {
		typeName string
		body     string
		fm       map[string]any
		derived  map[string]any
	}{}
	for _, row := range full.Rows {
		fullByPath[row.Path] = struct {
			typeName string
			body     string
			fm       map[string]any
			derived  map[string]any
		}{row.Type, row.Body, row.Frontmatter, row.Derived}
	}
	wantFullValues := map[string]struct {
		body    string
		fmNote  any
		fmHas   bool
		derived any
	}{
		"body.md":     {body: "# Body\n\n## Notes\n\nbody value\n", derived: "body value"},
		"empty.md":    {body: "# Empty\n\n## Notes\n\n## End\n\nend\n", derived: ""},
		"nil.md":      {body: "# Nil\n\n## Notes\n\nbody ignored\n", fmNote: nil, fmHas: true, derived: nil},
		"override.md": {body: "# Override\n\n## Notes\n\nbody ignored\n", fmNote: "fm value", fmHas: true, derived: "fm value"},
	}
	for path, want := range wantFullValues {
		got := fullByPath[path]
		if got.typeName != "markdown" || got.body != want.body {
			t.Fatalf("full row[%s] type/body = %q/%q, want markdown/%q", path, got.typeName, got.body, want.body)
		}
		fmNote, fmHas := got.fm["notes"]
		if fmHas != want.fmHas || fmNote != want.fmNote {
			t.Fatalf("full frontmatter notes[%s] = (%#v,%v), want (%#v,%v)", path, fmNote, fmHas, want.fmNote, want.fmHas)
		}
		if derived, ok := got.derived["notes"]; !ok || derived != want.derived {
			t.Fatalf("full derived notes[%s] = (%#v,%v), want %#v present", path, derived, ok, want.derived)
		}
	}

	selectOut, err := runCmd(t, "query", dir, "--select", "path,notes", "--sort", "notes:asc")
	if err != nil {
		t.Fatalf("selected query failed: %v\n%s", err, selectOut)
	}
	var selectedEnvelope map[string]any
	if err := json.Unmarshal([]byte(selectOut), &selectedEnvelope); err != nil {
		t.Fatalf("invalid selected query JSON: %v\n%s", err, selectOut)
	}
	assertJSONKeys(t, selectedEnvelope, []string{"version", "kind", "meta", "rows"})
	assertJSONKeys(t, selectedEnvelope["meta"].(map[string]any), []string{"count"})
	for _, raw := range selectedEnvelope["rows"].([]any) {
		assertJSONKeys(t, raw.(map[string]any), []string{"path", "notes"})
	}
	var selected struct {
		Version int                 `json:"version"`
		Kind    string              `json:"kind"`
		Meta    struct{ Count int } `json:"meta"`
		Rows    []map[string]any    `json:"rows"`
	}
	if err := json.Unmarshal([]byte(selectOut), &selected); err != nil {
		t.Fatalf("invalid typed selected query JSON: %v\n%s", err, selectOut)
	}
	if selected.Version != 1 || selected.Kind != "rootline/query" || selected.Meta.Count != 4 {
		t.Fatalf("selected identity/count = v%d %q %d", selected.Version, selected.Kind, selected.Meta.Count)
	}
	wantSelected := []map[string]any{
		{"path": "empty.md", "notes": ""},
		{"path": "nil.md", "notes": nil},
		{"path": "body.md", "notes": "body value"},
		{"path": "override.md", "notes": "fm value"},
	}
	if !reflect.DeepEqual(selected.Rows, wantSelected) {
		t.Fatalf("selected rows = %#v, want %#v", selected.Rows, wantSelected)
	}

	countOut, err := runCmd(t, "query", dir, "--where", "notes == 'body value'", "--count")
	if err != nil {
		t.Fatalf("where/count failed: %v\n%s", err, countOut)
	}
	var countEnvelope map[string]any
	if err := json.Unmarshal([]byte(countOut), &countEnvelope); err != nil {
		t.Fatalf("invalid count JSON: %v\n%s", err, countOut)
	}
	assertJSONKeys(t, countEnvelope, []string{"version", "kind", "meta", "count"})
	assertJSONKeys(t, countEnvelope["meta"].(map[string]any), []string{"count"})
	var count struct {
		Version int                 `json:"version"`
		Kind    string              `json:"kind"`
		Meta    struct{ Count int } `json:"meta"`
		Count   int                 `json:"count"`
	}
	if err := json.Unmarshal([]byte(countOut), &count); err != nil {
		t.Fatalf("invalid typed count JSON: %v\n%s", err, countOut)
	}
	if count.Version != 1 || count.Kind != "rootline/count" || count.Meta.Count != 1 || count.Count != 1 {
		t.Fatalf("count result = %#v, want exact one-row rootline/count", count)
	}

	jsonlOut, err := runCmd(t, "query", dir, "--select", "path,notes", "--sort", "path:asc", "-o", "jsonl")
	if err != nil {
		t.Fatalf("jsonl query failed: %v\n%s", err, jsonlOut)
	}
	lines := strings.Split(strings.TrimSpace(jsonlOut), "\n")
	if len(lines) != 4 {
		t.Fatalf("jsonl line count = %d lines %#v, want 4", len(lines), lines)
	}
	var jsonlRows []map[string]any
	for _, line := range lines {
		var row map[string]any
		if err := json.Unmarshal([]byte(line), &row); err != nil {
			t.Fatalf("invalid JSONL row %q: %v", line, err)
		}
		jsonlRows = append(jsonlRows, row)
	}
	wantPathSorted := []map[string]any{
		{"path": "body.md", "notes": "body value"},
		{"path": "empty.md", "notes": ""},
		{"path": "nil.md", "notes": nil},
		{"path": "override.md", "notes": "fm value"},
	}
	if !reflect.DeepEqual(jsonlRows, wantPathSorted) {
		t.Fatalf("jsonl rows = %#v, want %#v", jsonlRows, wantPathSorted)
	}

	csvOut, err := runCmd(t, "query", dir, "--select", "path,notes", "--sort", "path:asc", "-o", "csv")
	if err != nil {
		t.Fatalf("csv query failed: %v\n%s", err, csvOut)
	}
	recs, err := csv.NewReader(strings.NewReader(csvOut)).ReadAll()
	if err != nil {
		t.Fatalf("invalid csv: %v\n%s", err, csvOut)
	}
	wantCSV := [][]string{{"path", "notes"}, {"body.md", "body value"}, {"empty.md", ""}, {"nil.md", ""}, {"override.md", "fm value"}}
	if !reflect.DeepEqual(recs, wantCSV) {
		t.Fatalf("csv records = %#v, want %#v", recs, wantCSV)
	}

	tableOut, err := runCmd(t, "query", dir, "--sort", "path:asc", "-o", "table")
	if err != nil {
		t.Fatalf("table query failed: %v\n%s", err, tableOut)
	}
	wantTable := []task9TableRow{
		{Path: "body.md", IsIndex: "false", Notes: "body value", Title: "Body"},
		{Path: "empty.md", IsIndex: "false", Notes: "", Title: "Empty"},
		{Path: "nil.md", IsIndex: "false", Notes: "<nil>", Title: "Nil"},
		{Path: "override.md", IsIndex: "false", Notes: "fm value", Title: "Override"},
	}
	if got := parseTask9QueryTable(t, tableOut); !reflect.DeepEqual(got, wantTable) {
		t.Fatalf("table rows = %#v, want %#v\n%s", got, wantTable, tableOut)
	}
}

func TestTask9QueryNestedMatchScopeResolvesSourceBackedFields(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\n"), 0o644)
	child := filepath.Join(dir, "docs")
	if err := os.MkdirAll(child, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(child, ".stem"), []byte(`version: 2
scope:
  match: "*.md"
schema:
  notes:
    type: string
    source: body.section["## Notes"]
`), 0o644)
	mustWriteFile(t, filepath.Join(child, "scoped.md"), []byte("---\n---\n# Scoped\n\n## Notes\n\nnested body\n"), 0o644)
	mustWriteFile(t, filepath.Join(dir, "outside.md"), []byte("---\n---\n# Outside\n\n## Notes\n\noutside\n"), 0o644)
	declareTestBoundary(t, dir)

	out, err := runCmd(t, "query", child, "--select", "path,notes")
	if err != nil {
		t.Fatalf("nested query failed: %v\n%s", err, out)
	}
	var selected struct {
		Version int                 `json:"version"`
		Kind    string              `json:"kind"`
		Meta    struct{ Count int } `json:"meta"`
		Rows    []map[string]any    `json:"rows"`
	}
	if err := json.Unmarshal([]byte(out), &selected); err != nil {
		t.Fatalf("invalid nested JSON: %v\n%s", err, out)
	}
	want := []map[string]any{{"path": "scoped.md", "notes": "nested body"}}
	if selected.Version != 1 || selected.Kind != "rootline/query" || selected.Meta.Count != 1 || !reflect.DeepEqual(selected.Rows, want) {
		t.Fatalf("nested query = v%d %q count %d rows %#v, want %#v", selected.Version, selected.Kind, selected.Meta.Count, selected.Rows, want)
	}
}

func task9FullPaths[T interface{}](rows []T) []string {
	paths := make([]string, 0, len(rows))
	for _, row := range rows {
		b, _ := json.Marshal(row)
		var obj struct {
			Path string `json:"path"`
		}
		_ = json.Unmarshal(b, &obj)
		paths = append(paths, obj.Path)
	}
	return paths
}

type task9TableRow struct {
	Path    string
	IsIndex string
	Notes   string
	Title   string
}

func parseTask9QueryTable(t *testing.T, out string) []task9TableRow {
	t.Helper()
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) < 2 {
		t.Fatalf("table output too short:\n%s", out)
	}
	if got := strings.Fields(lines[0]); !reflect.DeepEqual(got, []string{"Path", "isIndex", "notes", "title"}) {
		t.Fatalf("table header = %v, want Path/isIndex/notes/title\n%s", got, out)
	}
	var rows []task9TableRow
	for _, line := range lines[2:] {
		fields := strings.Fields(line)
		if len(fields) < 3 {
			t.Fatalf("table row fields = %v for line %q", fields, line)
		}
		row := task9TableRow{Path: fields[0], IsIndex: fields[1], Title: fields[len(fields)-1]}
		if len(fields) > 3 {
			row.Notes = strings.Join(fields[2:len(fields)-1], " ")
		}
		rows = append(rows, row)
	}
	return rows
}
