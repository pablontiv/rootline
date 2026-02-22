package main

import (
	"encoding/json"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestGraphJSON_Empty(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 1\n"), 0644)
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
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 1\n"), 0644)
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
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 1\n"), 0644)
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
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 1\n"), 0644)
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
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 1\n"), 0644)
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
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 1\n"), 0644)
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
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 1\n"), 0644)
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
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n"), 0644)

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

func TestGraphCheck_SchemaFiltersReferenceLinks(t *testing.T) {
	dir := t.TempDir()
	// .stem with links schema that only has rules for "blocks"
	stem := "version: 1\nlinks:\n  allowed: [blocks, reference]\n  blocks:\n    target: \"*.md\"\n"
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
