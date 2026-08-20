package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func TestTask9DescribeJSONAndTableSeparateSourceFromDefinedIn(t *testing.T) {
	dir := setupTask9SourceProject(t, false)

	jsonOut, err := runCmd(t, "describe", dir)
	if err != nil {
		t.Fatalf("describe JSON failed: %v\n%s", err, jsonOut)
	}
	var desc struct {
		Schema map[string]map[string]any `json:"schema"`
	}
	if err := json.Unmarshal([]byte(jsonOut), &desc); err != nil {
		t.Fatalf("invalid describe JSON: %v\n%s", err, jsonOut)
	}
	notes := desc.Schema["notes"]
	if notes["source"] != `body.section["## Notes"]` {
		t.Fatalf("describe JSON source = %#v, want logical directive", notes["source"])
	}
	if defined, ok := notes["defined_in"].(string); !ok || !strings.HasSuffix(defined, ".stem") {
		t.Fatalf("describe JSON defined_in = %#v, want physical .stem", notes["defined_in"])
	}

	tableOut, err := runCmd(t, "describe", dir, "-o", "table")
	if err != nil {
		t.Fatalf("describe table failed: %v\n%s", err, tableOut)
	}
	for _, want := range []string{"Source", "Defined In", `body.section["## Notes"]`, ".stem"} {
		if !strings.Contains(tableOut, want) {
			t.Fatalf("describe table missing %q:\n%s", want, tableOut)
		}
	}
}

func TestTask9ExplainOverrideEmitsOneLogicalFieldMatchingQuery(t *testing.T) {
	dir := setupTask9SourceProject(t, false)

	queryOut, err := runCmd(t, "query", dir, "--where", "path == 'override.md'", "--select", "path,notes")
	if err != nil {
		t.Fatalf("query override failed: %v\n%s", err, queryOut)
	}
	var query struct {
		Rows []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal([]byte(queryOut), &query); err != nil {
		t.Fatalf("invalid query JSON: %v\n%s", err, queryOut)
	}
	if len(query.Rows) != 1 || query.Rows[0]["notes"] != "fm value" {
		t.Fatalf("query rows = %#v, want one fm override value", query.Rows)
	}

	explainOut, err := runCmd(t, "explain", filepath.Join(dir, "override.md"))
	if err != nil {
		t.Fatalf("explain override failed: %v\n%s", err, explainOut)
	}
	var explain struct {
		Fields []map[string]any `json:"fields"`
	}
	if err := json.Unmarshal([]byte(explainOut), &explain); err != nil {
		t.Fatalf("invalid explain JSON: %v\n%s", err, explainOut)
	}
	var notesRows []map[string]any
	for _, field := range explain.Fields {
		if field["name"] == "notes" {
			notesRows = append(notesRows, field)
		}
	}
	if len(notesRows) != 1 {
		t.Fatalf("explain notes rows = %#v, want exactly one logical field row", notesRows)
	}
	notes := notesRows[0]
	if notes["value"] != query.Rows[0]["notes"] || notes["origin"] != "frontmatter" {
		t.Fatalf("explain notes = %#v, want query value with frontmatter origin", notes)
	}
	if notes["source"] != `body.section["## Notes"]` {
		t.Fatalf("explain notes.source = %#v, want logical directive", notes["source"])
	}
	if defined, ok := notes["defined_in"].(string); !ok || !strings.HasSuffix(defined, ".stem") {
		t.Fatalf("explain notes.defined_in = %#v, want physical .stem", notes["defined_in"])
	}
	if notes["source"] == notes["origin"] {
		t.Fatalf("explain serialized value origin as logical source: %#v", notes)
	}
}

func TestTask9ExplainTableOverrideContainsOneNotesRow(t *testing.T) {
	dir := setupTask9SourceProject(t, false)

	out, err := runCmd(t, "explain", filepath.Join(dir, "override.md"), "-o", "table")
	if err != nil {
		t.Fatalf("explain table override failed: %v\n%s", err, out)
	}
	if got := strings.Count(out, "notes"); got != 1 {
		t.Fatalf("explain table notes row count = %d, want 1\n%s", got, out)
	}
	if !strings.Contains(out, `body.section["## Notes"]`) || !strings.Contains(out, "Defined In") {
		t.Fatalf("explain table missing logical source or physical defined-in label:\n%s", out)
	}
}
