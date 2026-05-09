package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestGraphJSON_Empty(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# No links\n"), 0644)

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
	mustWriteFile(t, filepath.Join(dir, "doc1.md"), []byte("---\n---\n# Doc1\n[[doc2.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "doc2.md"), []byte("---\n---\n# Doc2\n"), 0644)

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
	mustWriteFile(t, filepath.Join(dir, "doc1.md"), []byte("---\n---\n# Doc1\n[[doc2.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "doc2.md"), []byte("---\n---\n# Doc2\n"), 0644)

	out, err := runCmd(t, "graph", "--check", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No cycles or broken links") {
		t.Errorf("expected clean check, got: %s", out)
	}
}

func TestGraphCheck_WithCycle(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[a.md]]\n"), 0644)

	out, err := runCmd(t, "graph", "--check", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v", err)
	}
	if !strings.Contains(out, "Cycles found: 1") {
		t.Errorf("expected cycle report, got: %s", out)
	}
}

func TestGraphCheck_WithBrokenLink(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[nonexistent.md]]\n"), 0644)

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
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

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
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

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
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

	_, err := runCmd(t, "graph", "-o", "table", "--format", "xyz", dir)
	if err == nil {
		t.Fatal("expected error for invalid format")
	}
	if !strings.Contains(err.Error(), "unknown format") {
		t.Errorf("expected 'unknown format' error, got: %v", err)
	}
}

func TestGraph_OpenWithCheck_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

	_, err := runCmd(t, "graph", "--open", "--check", dir)
	if err == nil {
		t.Fatal("expected error for --open with --check")
	}
	if !strings.Contains(err.Error(), "--open") || !strings.Contains(err.Error(), "--check") {
		t.Errorf("expected error mentioning --open and --check, got: %v", err)
	}
}

func TestGraph_OpenWithDot_ReturnsError(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

	_, err := runCmd(t, "graph", "--open", "--format", "dot", dir)
	if err == nil {
		t.Fatal("expected error for --open with --format dot")
	}
	if !strings.Contains(err.Error(), "--open") || !strings.Contains(err.Error(), "--format dot") {
		t.Errorf("expected error mentioning --open and --format dot, got: %v", err)
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

func TestGraphWhere_Filters(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nscope:\n  match: \"*.md\"\nschema:\n  tipo:\n    type: string\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\ntipo: test\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\ntipo: prod\n---\n# B\n"), 0644)

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
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

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
	// doc with a [[reference]] link to nonexistent target — should be filtered out
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[nonexistent]]\n"), 0644)

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
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n[[c.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "c.md"), []byte("---\n---\n# C\n"), 0644)

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

func TestGraphJSON_FieldProjection_BrokenLinks(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[nonexistent.md]]\n"), 0644)

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
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[[b.md]]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\n---\n# B\n"), 0644)

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
