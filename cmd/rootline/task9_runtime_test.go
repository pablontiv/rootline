package main

import (
	"bytes"
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"
)

func runTask9CmdStdoutOnly(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags()
	out := new(bytes.Buffer)
	rootCmd.SetOut(out)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(args)
	err := rootCmd.Execute()
	return out.String(), err
}

func setupTask9SourceProject(t *testing.T, duplicate bool) string {
	t.Helper()
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
root: true
scope:
  match: "*.md"
schema:
  title:
    type: string
  notes:
    type: string
    source: body.section["## Notes"]
`), 0o644)
	body := "# Body\n\n## Notes\n\nbody value\n"
	if duplicate {
		body = "# Body\n\n## Notes\n\nfirst\n\n## Notes\n\nsecond\n"
	}
	mustWriteFile(t, filepath.Join(dir, "body.md"), []byte("---\ntitle: Body\n---\n"+body), 0o644)
	mustWriteFile(t, filepath.Join(dir, "override.md"), []byte("---\ntitle: Override\nnotes: fm value\n---\n# Override\n\n## Notes\n\nbody ignored\n"), 0o644)
	mustWriteFile(t, filepath.Join(dir, "empty.md"), []byte("---\ntitle: Empty\n---\n# Empty\n\n## Notes\n\n## End\n\nend\n"), 0o644)
	declareTestBoundary(t, dir)
	return dir
}

func TestTask9Query_SourceBackedFieldsAcrossSurfaces(t *testing.T) {
	dir := setupTask9SourceProject(t, false)

	out, err := runCmd(t, "query", dir, "--select", "path,notes", "--sort", "notes:asc")
	if err != nil {
		t.Fatalf("query select/sort failed: %v\n%s", err, out)
	}
	var selected struct {
		Rows []map[string]any `json:"rows"`
	}
	if err := json.Unmarshal([]byte(out), &selected); err != nil {
		t.Fatalf("invalid query JSON: %v\n%s", err, out)
	}
	seen := map[string]any{}
	for _, row := range selected.Rows {
		seen[row["path"].(string)] = row["notes"]
	}
	if seen["body.md"] != "body value" || seen["override.md"] != "fm value" || seen["empty.md"] != "" {
		t.Fatalf("projected notes = %#v, want body fallback, fm override, empty section", seen)
	}

	out, err = runCmd(t, "query", dir, "--where", "notes == 'body value'", "--count")
	if err != nil {
		t.Fatalf("query where/count failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, `"count":1`) {
		t.Fatalf("where/count output = %s, want count 1", out)
	}
}

func TestTask9Query_DuplicateSourceFailsWithoutOutput(t *testing.T) {
	dir := setupTask9SourceProject(t, true)

	out, err := runTask9CmdStdoutOnly(t, "query", dir, "--select", "path,notes")
	if err == nil || !strings.Contains(err.Error(), "ambiguous body section source") {
		t.Fatalf("query duplicate err = %v, output=%s; want ambiguity", err, out)
	}
	if strings.TrimSpace(out) != "" {
		t.Fatalf("query emitted partial output on ambiguity: %q", out)
	}
}

func TestTask9Set_SourceBackedFieldWritesFrontmatterAndPreservesBody(t *testing.T) {
	dir := setupTask9SourceProject(t, false)
	target := filepath.Join(dir, "body.md")
	before := string(mustReadFile(t, target))
	beforeBody := before[strings.Index(before, "---\n# Body")+4:]

	out, err := runCmd(t, "set", target, "notes=frontmatter override")
	if err != nil {
		t.Fatalf("set failed: %v\n%s", err, out)
	}
	after := string(mustReadFile(t, target))
	afterBody := after[strings.Index(after, "---\n# Body")+4:]
	if beforeBody != afterBody {
		t.Fatalf("body changed after set\nbefore:%q\nafter:%q", beforeBody, afterBody)
	}
	if strings.Count(after, "## Notes") != 1 {
		t.Fatalf("set must not create or mutate body sections, content:\n%s", after)
	}
	if !strings.Contains(after, "notes: frontmatter override") {
		t.Fatalf("set did not write frontmatter override:\n%s", after)
	}

	out, err = runCmd(t, "query", dir, "--where", "notes == 'frontmatter override'", "--count")
	if err != nil || !strings.Contains(out, `"count":1`) {
		t.Fatalf("query after set = err %v output %s, want override count", err, out)
	}
	if out, err = runCmd(t, "validate", target); err != nil {
		t.Fatalf("validate after set failed: %v\n%s", err, out)
	}
}

func TestTask9DescribeAndExplain_SourceVsDefinedIn(t *testing.T) {
	dir := setupTask9SourceProject(t, false)
	out, err := runCmd(t, "describe", dir)
	if err != nil {
		t.Fatalf("describe failed: %v\n%s", err, out)
	}
	var desc struct {
		Schema map[string]map[string]any `json:"schema"`
	}
	if err := json.Unmarshal([]byte(out), &desc); err != nil {
		t.Fatalf("invalid describe JSON: %v\n%s", err, out)
	}
	notes := desc.Schema["notes"]
	if notes["source"] != `body.section["## Notes"]` {
		t.Fatalf("describe notes.source = %#v, want logical directive", notes["source"])
	}
	if defined, ok := notes["defined_in"].(string); !ok || !strings.HasSuffix(defined, ".stem") {
		t.Fatalf("describe notes.defined_in = %#v, want physical .stem", notes["defined_in"])
	}

	out, err = runCmd(t, "explain", filepath.Join(dir, "body.md"))
	if err != nil {
		t.Fatalf("explain failed: %v\n%s", err, out)
	}
	var expl struct {
		Fields []map[string]any `json:"fields"`
	}
	if err := json.Unmarshal([]byte(out), &expl); err != nil {
		t.Fatalf("invalid explain JSON: %v\n%s", err, out)
	}
	var found map[string]any
	for _, field := range expl.Fields {
		if field["name"] == "notes" {
			found = field
			break
		}
	}
	if found == nil {
		t.Fatalf("explain missing notes field: %#v", expl.Fields)
	}
	if found["value"] != "body value" || found["source"] != `body.section["## Notes"]` {
		t.Fatalf("explain notes = %#v, want body value plus logical source", found)
	}
	if _, hasPhysicalSource := found["defined_in"]; !hasPhysicalSource {
		t.Fatalf("explain notes missing defined_in provenance: %#v", found)
	}

	out, err = runTask9CmdStdoutOnly(t, "explain", filepath.Join(setupTask9SourceProject(t, true), "body.md"))
	if err == nil || !strings.Contains(err.Error(), "ambiguous body section source") || strings.TrimSpace(out) != "" {
		t.Fatalf("ambiguous explain err=%v out=%q, want failure with zero output", err, out)
	}
	if !strings.Contains(err.Error(), "resolving explain fields") || strings.Contains(err.Error(), "enriching record") {
		t.Fatalf("ambiguous explain err=%v, want builder-owned resolution error before stdout", err)
	}
}

func TestTask9ExplainPreservesBuiltinIsIndexWithoutEnrichment(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\n"), 0o644)
	target := filepath.Join(dir, "README.md")
	mustWriteFile(t, target, []byte("---\ntitle: Index\n---\n# Index\n"), 0o644)
	declareTestBoundary(t, dir)

	out, err := runCmd(t, "explain", target)
	if err != nil {
		t.Fatalf("explain index failed: %v\n%s", err, out)
	}
	var expl struct {
		Fields []map[string]any `json:"fields"`
	}
	if err := json.Unmarshal([]byte(out), &expl); err != nil {
		t.Fatalf("invalid explain JSON: %v\n%s", err, out)
	}
	for _, field := range expl.Fields {
		if field["name"] == "isIndex" {
			if field["value"] != true || field["origin"] != "derived" || field["source"] != nil || field["defined_in"] != nil {
				t.Fatalf("isIndex explain field = %#v, want derived builtin without source metadata", field)
			}
			return
		}
	}
	t.Fatalf("explain fields missing isIndex: %#v", expl.Fields)
}

func TestTask9ExplainTableLabelsAreUnambiguous(t *testing.T) {
	dir := setupTask9SourceProject(t, false)
	out, err := runCmd(t, "explain", filepath.Join(dir, "body.md"), "-o", "table")
	if err != nil {
		t.Fatalf("explain table failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "Defined In") || !strings.Contains(out, "Source") {
		t.Fatalf("table output should distinguish physical and logical source labels:\n%s", out)
	}
}
