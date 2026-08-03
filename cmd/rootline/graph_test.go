package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestGraphJSON_Empty(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# No links\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if result["kind"] != "rootline/graph" {
		t.Errorf("kind = %v, want rootline/graph", result["kind"])
	}
	edges := result["edges"].([]any)
	if len(edges) != 0 {
		t.Errorf("edges = %d, want 0", len(edges))
	}
	cycles := result["cycles"].([]any)
	if len(cycles) != 0 {
		t.Errorf("cycles = %d, want 0", len(cycles))
	}
}

func TestGraphJSON_WithLinks(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "doc1.md"), []byte("---\n---\n# Doc1\n[[doc2.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "doc2.md"), []byte("---\n---\n# Doc2\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	edges := result["edges"].([]any)
	if len(edges) != 1 {
		t.Errorf("edges = %d, want 1", len(edges))
	}
}

func TestGraphCheck_Clean(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "doc1.md"), []byte("---\n---\n# Doc1\n[[doc2.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "doc2.md"), []byte("---\n---\n# Doc2\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No cycles or broken links") {
		t.Errorf("expected clean check, got: %s", out)
	}
}

func TestGraphCheck_WithCycle_InformationalByDefault(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Cycles found (informational): 1") {
		t.Errorf("expected informational cycle report, got: %s", out)
	}
	if strings.Contains(out, "No cycles or broken links") {
		t.Errorf("clean message printed despite cycles present: %s", out)
	}
}

func TestGraphCheck_CyclesOptInFails(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nlinks:\n  checks:\n    cycles: true\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Cycles found: 1") {
		t.Errorf("expected failing cycle report, got: %s", out)
	}
}

func TestGraphCheck_InformationalCyclesDontMaskBrokenLinks(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n[[missing.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed for broken link, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Broken links: 1") {
		t.Errorf("expected broken link report, got: %s", out)
	}
	if !strings.Contains(out, "Cycles found (informational): 1") {
		t.Errorf("expected informational cycles alongside broken links, got: %s", out)
	}
}

func TestGraphCheck_FailCyclesFlagForcesFailure(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", "--fail-cycles=true", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed with --fail-cycles=true, got: %v\noutput: %s", err, out)
	}
}

func TestGraphCheck_FailCyclesFlagForcesInformational(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nlinks:\n  checks:\n    cycles: true\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", "--fail-cycles=false", dir)
	if err != nil {
		t.Fatalf("expected success with --fail-cycles=false, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "Cycles found (informational): 1") {
		t.Errorf("expected informational cycle report, got: %s", out)
	}
}

func TestGraphCheck_WithBrokenLink(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[nonexistent.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v", err)
	}
	if !strings.Contains(out, "Broken links: 1") {
		t.Errorf("expected broken link report, got: %s", out)
	}
}

func TestGraphFormat_DOT(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "-o", "table", "--format", "dot", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.HasPrefix(out, "digraph {") {
		t.Errorf("expected DOT output starting with 'digraph {', got: %s", out)
	}
}

func TestGraphFormat_Mermaid(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "-o", "table", "--format", "mermaid", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.HasPrefix(out, "graph TD;") {
		t.Errorf("expected Mermaid output starting with 'graph TD;', got: %s", out)
	}
}

func TestGraphFormat_Invalid(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

	declareTestBoundary(t, dir)
	_, err := runCmd(t, "graph", "-o", "table", "--format", "xyz", dir)
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("expected 'unknown format' error, got: %v", err)
	}
}

func TestFilterLinksBySchema_WithRules(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "a.md",
			Links: []extract.Link{
				{Target: "T001-task.md", Type: "blocks", Line: 1},
				{Target: "something", Type: "reference", Line: 2},
			},
		},
	}
	schema := rules.LinkSchema{
		Allowed: []string{"blocks", "reference"},
		Rules:   map[string]rules.LinkRule{"blocks": {Target: "T*"}},
	}

	filterLinksBySchema(records, schema)

	if len(records[0].Links) != 1 {
		t.Fatalf("expected 1 link after filter, got %d", len(records[0].Links))
	}
	if records[0].Links[0].Type != "blocks" {
		t.Errorf("expected blocks link, got %s", records[0].Links[0].Type)
	}
}

func TestFilterLinksBySchema_EmptySchema(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "a.md",
			Links: []extract.Link{
				{Target: "b.md", Type: "reference", Line: 1},
				{Target: "c.md", Type: "blocks", Line: 2},
			},
		},
	}
	schema := rules.LinkSchema{}

	filterLinksBySchema(records, schema)

	if len(records[0].Links) != 2 {
		t.Errorf("expected 2 links (no filtering), got %d", len(records[0].Links))
	}
}

func TestFilterLinksBySchema_StylesOnlySchema(t *testing.T) {
	rec := &extract.Record{
		Path:  "a.md",
		Links: []extract.Link{{Target: "b.md", Type: "reference", Style: extract.StyleMarkdown}},
	}
	schema := rules.LinkSchema{Styles: []string{"markdown"}}
	filterLinksBySchema([]*extract.Record{rec}, schema)
	if len(rec.Links) != 1 {
		t.Fatalf("links = %d, want 1 (styles-only schema must not drop links)", len(rec.Links))
	}
}

func TestFilterLinksBySchema_ChecksOnlySchema(t *testing.T) {
	rec := &extract.Record{
		Path:  "a.md",
		Links: []extract.Link{{Target: "b.md", Style: extract.StyleWikilink}},
	}
	schema := rules.LinkSchema{Checks: &rules.LinkChecks{Resolve: true}}
	filterLinksBySchema([]*extract.Record{rec}, schema)
	if len(rec.Links) != 1 {
		t.Fatalf("links = %d, want 1 (checks-only schema must not drop links)", len(rec.Links))
	}
}

func TestFilterLinksBySchema_AllowedOnlySchema(t *testing.T) {
	rec := &extract.Record{
		Path:  "a.md",
		Links: []extract.Link{{Target: "b.md", Type: "reference", Style: extract.StyleWikilink}},
	}
	schema := rules.LinkSchema{Allowed: []string{"reference"}}
	filterLinksBySchema([]*extract.Record{rec}, schema)
	if len(rec.Links) != 1 {
		t.Fatalf("links = %d, want 1 (typed filtering applies only when rules are declared)", len(rec.Links))
	}
}

func TestGraphWhere_Filters(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nscope:\n  match: \"*.md\"\nschema:\n  tipo:\n    type: string\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\ntipo: test\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\ntipo: prod\n---\n# B\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", dir, "--where", "tipo == 'test'")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	nodes := result["nodes"].([]any)
	if len(nodes) != 1 {
		t.Errorf("nodes = %d, want 1 (only tipo=test)", len(nodes))
	}
}

func TestGraphWhere_InvalidExpr(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

	declareTestBoundary(t, dir)
	_, err := runCmd(t, "graph", dir, "--where", "== bad")
	if err == nil {
		t.Fatal("expected error for invalid where expression")
	}
}

func TestGraphCheck_SchemaFiltersReferenceLinks(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// .stem with links schema that only has rules for "blocks"
	stem := "version: 2\nlinks:\n  allowed: [blocks, reference]\n  blocks:\n    target: \"*.md\"\n"
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stem), 0644)
	declareTestBoundary(t, dir)
	// doc with a [[reference]] link to nonexistent target — should be filtered out
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[nonexistent]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", dir)
	if err != nil {
		t.Fatalf("expected no error (reference links filtered), got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No cycles or broken links") {
		t.Errorf("expected clean check, got: %s", out)
	}
}

func TestGraphJSON_FieldProjection_Edges(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[c.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "c.md"), []byte("---\n---\n# C\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", dir, "--field", "edges")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var edges []any
	if err := json.Unmarshal([]byte(out), &edges); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if len(edges) != 2 {
		t.Errorf("edges = %d, want 2", len(edges))
	}
}

func TestGraphJSON_FieldProjection_EdgeSources(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[c.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "c.md"), []byte("---\n---\n# C\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", dir, "--field", "edges[].source")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var sources []string
	if err := json.Unmarshal([]byte(out), &sources); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	sort.Strings(sources)
	if len(sources) != 2 || sources[0] != "a.md" || sources[1] != "b.md" {
		t.Errorf("sources = %#v, want [a.md b.md]", sources)
	}
}

func TestGraphJSON_FieldProjection_BrokenLinks(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[nonexistent.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", dir, "--field", "broken_links")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var brokenLinks []any
	if err := json.Unmarshal([]byte(out), &brokenLinks); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	if len(brokenLinks) != 1 {
		t.Errorf("broken_links = %d, want 1", len(brokenLinks))
	}
}

func TestGraphJSON_Unchanged(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}

	// Verify all expected fields are present
	if _, ok := result["version"]; !ok {
		t.Error("missing version field")
	}
	if _, ok := result["kind"]; !ok {
		t.Error("missing kind field")
	}
	if _, ok := result["nodes"]; !ok {
		t.Error("missing nodes field")
	}
	if _, ok := result["edges"]; !ok {
		t.Error("missing edges field")
	}
	if _, ok := result["cycles"]; !ok {
		t.Error("missing cycles field")
	}
	if _, ok := result["broken_links"]; !ok {
		t.Error("missing broken_links field")
	}

	if result["version"].(float64) != 1 {
		t.Errorf("version = %v, want 1", result["version"])
	}
	if result["kind"] != "rootline/graph" {
		t.Errorf("kind = %v, want rootline/graph", result["kind"])
	}
}

func TestGraphCheck_MarkdownStylesClean(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nlinks:\n  styles: [markdown]\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "README.md"), []byte("---\n---\n# Root\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "docs", "overview.md"), []byte("---\n---\n# Overview\n[back](../README.md)\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No cycles or broken links") {
		t.Errorf("expected clean check, got: %s", out)
	}
}

func TestGraphJSON_MarkdownStylesProducesEdges(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs", "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nlinks:\n  styles: [markdown]\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "README.md"), []byte("---\n---\n# Root\n[a](docs/a.md)\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "docs", "a.md"), []byte("---\n---\n# A\n[b](sub/b.md)\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "docs", "sub", "b.md"), []byte("---\n---\n# B\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	edges := result["edges"].([]any)
	if len(edges) != 2 {
		t.Errorf("edges = %d, want 2", len(edges))
	}
	targets := make([]string, 0, len(edges))
	for _, e := range edges {
		targets = append(targets, e.(map[string]any)["target"].(string))
	}
	sort.Strings(targets)
	want := []string{filepath.Join("docs", "a.md"), filepath.Join("docs", "sub", "b.md")}
	if len(targets) != 2 || targets[0] != want[0] || targets[1] != want[1] {
		t.Errorf("edge targets = %v, want %v", targets, want)
	}
}

func TestGraphCheck_MarkdownBrokenLinkCanary(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nlinks:\n  styles: [markdown]\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[x](no-existe.md)\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "no-existe.md") {
		t.Errorf("expected broken link named in output, got: %s", out)
	}
}

func TestGraphCheck_QuietCycles_InformationalOnly(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", "--quiet-cycles", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	// Should have exactly one summary line and no per-cycle enumeration.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) != 1 {
		t.Errorf("expected exactly 1 output line with --quiet-cycles, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(out, "Cycles found (informational):") {
		t.Errorf("expected 'Cycles found (informational):' summary, got: %s", out)
	}
	// Verify no per-cycle lines (would look like "  1: a.md → b.md")
	if strings.Contains(out, "→") {
		t.Errorf("unexpected per-cycle enumeration with --quiet-cycles: %s", out)
	}
}

func TestGraphCheck_QuietCycles_IgnoredWhenFailing(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", "--fail-cycles", "--quiet-cycles", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v\noutput: %s", err, out)
	}
	// --quiet-cycles should be ignored when failing; full details should be shown.
	if !strings.Contains(out, "→") {
		t.Errorf("expected per-cycle details with --fail-cycles (--quiet-cycles ignored), got: %s", out)
	}
	if !strings.Contains(out, "Cycles found: 1") {
		t.Errorf("expected 'Cycles found: 1' (failing), got: %s", out)
	}
}

func TestGraphCheck_QuietCycles_WithBrokenLink(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n[[missing.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--check", "--quiet-cycles", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed for broken link, got: %v\noutput: %s", err, out)
	}
	// Broken links should still be printed in full.
	if !strings.Contains(out, "Broken links:") {
		t.Errorf("expected broken link report, got: %s", out)
	}
	// Cycles should be quiet (summary only).
	if !strings.Contains(out, "Cycles found (informational):") {
		t.Errorf("expected quiet cycle summary, got: %s", out)
	}
	if strings.Count(out, "\n") < 2 {
		t.Errorf("expected at least cycle summary + broken link line, got: %s", out)
	}
}

func TestGraphJSON_QuietCycles_CyclesPopulated(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	declareTestBoundary(t, dir)
	out, err := runCmd(t, "graph", "--quiet-cycles", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	cycles := result["cycles"].([]any)
	if len(cycles) != 1 {
		t.Errorf("cycles = %d, want 1 (JSON unaffected by --quiet-cycles)", len(cycles))
	}
}

// schemalessLinkTree builds a tree of linked documents with no .stem anywhere.
func schemalessLinkTree(t *testing.T, target string) string {
	t.Helper()
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[["+target+"]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n"), 0644)
	return dir
}

// TestGraph_SchemalessEmitsGraph replaces TestGraphErrorPropagation_NoSchema,
// which asserted the opposite.
//
// A .stem governs which link styles graph reads and whether cycles fail a
// check. Its absence removes those restrictions; it does not remove the links.
// Refusing to draw the graph withheld the one view that helps someone decide
// where the schema should go, so a tree with no .stem now yields the whole
// extracted graph rather than a discovery error.
func TestGraph_SchemalessEmitsGraph(t *testing.T) {
	dir := schemalessLinkTree(t, "b.md")

	out, err := runCmd(t, "graph", dir)
	if err != nil {
		t.Fatalf("graph must tolerate a tree with no .stem, got: %v", err)
	}

	var result map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &result); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", jsonErr, out)
	}
	if result["kind"] != "rootline/graph" {
		t.Errorf("kind = %v, want rootline/graph", result["kind"])
	}
	edges, _ := result["edges"].([]any)
	if len(edges) != 1 {
		t.Errorf("edges = %d, want 1 (the a.md -> b.md link is still there)", len(edges))
	}
}

// setupOrderFixture writes four records whose creation order (r2, r1, r6, r3)
// deliberately differs from their lexical order, so any renderer that leaks
// Go's randomized map iteration into its output fails the fixed-order asserts.
func setupOrderFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)
	for _, name := range []string{"r2.md", "r1.md", "r6.md", "r3.md"} {
		mustWriteFile(t, filepath.Join(dir, name), []byte("---\n---\n# "+name+"\n"), 0644)
	}
	return dir
}

var wantOrderedNodes = []string{"r1.md", "r2.md", "r3.md", "r6.md"}

func TestGraphJSONNodeOrder(t *testing.T) {
	dir := setupOrderFixture(t)

	// Five consecutive renders: a single pass could match by luck, five cannot.
	for run := 1; run <= 5; run++ {
		declareTestBoundary(t, dir)
		out, err := runCmd(t, "graph", dir)
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v\noutput: %s", run, err, out)
		}
		var result struct {
			Version int      `json:"version"`
			Nodes   []string `json:"nodes"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("run %d: invalid JSON: %v\nraw: %s", run, err, out)
		}
		if result.Version != 1 {
			t.Errorf("run %d: version = %d, want 1", run, result.Version)
		}
		if strings.Join(result.Nodes, ",") != strings.Join(wantOrderedNodes, ",") {
			t.Fatalf("run %d: nodes = %v, want %v", run, result.Nodes, wantOrderedNodes)
		}
	}
}

func TestGraphDOTNodeOrder(t *testing.T) {
	dir := setupOrderFixture(t)

	for run := 1; run <= 5; run++ {
		declareTestBoundary(t, dir)
		out, err := runCmd(t, "graph", dir, "--format", "dot", "-o", "table")
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v\noutput: %s", run, err, out)
		}
		got := dotNodeDecls(out)
		if strings.Join(got, ",") != strings.Join(wantOrderedNodes, ",") {
			t.Fatalf("run %d: DOT nodes = %v, want %v\nraw: %s", run, got, wantOrderedNodes, out)
		}
	}
}

// dotNodeDecls returns the node paths declared by a DOT document, in order.
// Node declarations look like `  "r1.md";` — edges carry ` -> ` and are skipped.
func dotNodeDecls(out string) []string {
	var nodes []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		if !strings.HasPrefix(line, `"`) || strings.Contains(line, "->") {
			continue
		}
		nodes = append(nodes, strings.Trim(strings.TrimSuffix(line, ";"), `"`))
	}
	return nodes
}

func TestGraphMermaidNodeOrder(t *testing.T) {
	dir := setupOrderFixture(t)

	for run := 1; run <= 5; run++ {
		declareTestBoundary(t, dir)
		out, err := runCmd(t, "graph", dir, "--format", "mermaid", "-o", "table")
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v\noutput: %s", run, err, out)
		}
		got := mermaidNodeIDs(out)
		want := []string{"r1_md", "r2_md", "r3_md", "r6_md"}
		if strings.Join(got, ",") != strings.Join(want, ",") {
			t.Fatalf("run %d: mermaid nodes = %v, want %v\nraw: %s", run, got, want, out)
		}
	}
}

// mermaidNodeIDs returns the sanitized node IDs declared by a Mermaid diagram,
// in order. Declarations look like `  r1_md["r1.md"];`; edges carry ` --> `.
func mermaidNodeIDs(out string) []string {
	var ids []string
	for _, line := range strings.Split(out, "\n") {
		line = strings.TrimSpace(line)
		idx := strings.Index(line, "[")
		if idx <= 0 || strings.Contains(line, "-->") {
			continue
		}
		ids = append(ids, line[:idx])
	}
	return ids
}

func TestGraphEdgeOrder(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	declareTestBoundary(t, dir)

	// Body line numbers are 1-based after the frontmatter is stripped. Links are
	// emitted in file order (20, 15, 5) so only sorting can produce the want set.
	body := make([]string, 21)
	for i := range body {
		body[i] = ""
	}
	body[0] = "# Source"
	body[19] = "[[target.md]]" // body line 20
	body[14] = "[[other.md]]"  // body line 15
	body[4] = "[[target.md]]"  // body line 5
	mustWriteFile(t, filepath.Join(dir, "source.md"),
		[]byte("---\n---\n"+strings.Join(body, "\n")+"\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "target.md"), []byte("---\n---\n# Target\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "other.md"), []byte("---\n---\n# Other\n"), 0644)

	want := []struct {
		source, target string
		line           int
	}{
		{"source.md", "other.md", 15},
		{"source.md", "target.md", 5},
		{"source.md", "target.md", 20},
	}

	for run := 1; run <= 5; run++ {
		declareTestBoundary(t, dir)
		out, err := runCmd(t, "graph", dir)
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v\noutput: %s", run, err, out)
		}
		var result struct {
			Edges []struct {
				Source string `json:"source"`
				Target string `json:"target"`
				Line   int    `json:"line"`
			} `json:"edges"`
		}
		if err := json.Unmarshal([]byte(out), &result); err != nil {
			t.Fatalf("run %d: invalid JSON: %v\nraw: %s", run, err, out)
		}
		if len(result.Edges) != len(want) {
			t.Fatalf("run %d: edges = %d, want %d: %+v", run, len(result.Edges), len(want), result.Edges)
		}
		for i, w := range want {
			got := result.Edges[i]
			if got.Source != w.source || got.Target != w.target || got.Line != w.line {
				t.Fatalf("run %d: edge[%d] = %+v, want (%s→%s:%d)\nall: %+v",
					run, i, got, w.source, w.target, w.line, result.Edges)
			}
		}
	}
}

// TestGraph_CheckOnSchemalessReportsIntegrity keeps --check answering the
// question it was asked. Link integrity does not depend on a schema, so on a
// tree with no .stem the exit code must report broken links and nothing else.
func TestGraph_CheckOnSchemalessReportsIntegrity(t *testing.T) {
	t.Run("broken link fails the check", func(t *testing.T) {
		dir := schemalessLinkTree(t, "no-such-file")

		out, err := runCmd(t, "graph", dir, "--check")
		if err == nil {
			t.Fatalf("--check must fail on a broken link\noutput: %s", out)
		}
		if !strings.Contains(out, "Broken links: 1") {
			t.Errorf("expected the broken-link report, got: %s", out)
		}
		if strings.Contains(err.Error(), "schema") {
			t.Errorf("the failure must be about links, not schema discovery: %v", err)
		}
	})

	t.Run("valid links pass the check", func(t *testing.T) {
		dir := schemalessLinkTree(t, "b.md")

		if _, err := runCmd(t, "graph", dir, "--check"); err != nil {
			t.Fatalf("--check must pass when every link resolves, got: %v", err)
		}
	})
}
