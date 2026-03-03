package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

// setupNestedStemProject creates a directory structure with multiple .stem files
// to test inheritance and merge behavior through the CLI.
func setupNestedStemProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Root .stem: defines estado enum.
	rootStem := `version: 1
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed, In Progress]
`
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte(rootStem), 0644)

	// Nested .stem in epics/: adds tipo field (merge from parent).
	if err := os.MkdirAll(filepath.Join(root, "epics"), 0o755); err != nil {
		t.Fatal(err)
	}
	epicsStem := `version: 1
scope:
  match: "*.md"
schema:
  tipo:
    type: enum
    required: true
    values: [feature, bug, chore]
derive:
  slug: "slugify(estado)"
`
	mustWriteFile(t, filepath.Join(root, "epics", ".stem"), []byte(epicsStem), 0644)

	// Deep nested: epics/E01/.stem adds priority.
	if err := os.MkdirAll(filepath.Join(root, "epics", "E01"), 0o755); err != nil {
		t.Fatal(err)
	}
	e01Stem := `version: 1
scope:
  match: "*.md"
schema:
  priority:
    type: enum
    values: [low, medium, high]
`
	mustWriteFile(t, filepath.Join(root, "epics", "E01", ".stem"), []byte(e01Stem), 0644)

	// Files at various levels.
	mustWriteFile(t, filepath.Join(root, "root-doc.md"), []byte("---\nestado: Pending\n---\n# Root Doc\n"), 0644)
	mustWriteFile(t, filepath.Join(root, "epics", "F01.md"), []byte("---\nestado: Completed\ntipo: feature\n---\n# F01\n"), 0644)
	mustWriteFile(t, filepath.Join(root, "epics", "E01", "T001.md"), []byte("---\nestado: Pending\ntipo: bug\npriority: high\n---\n# T001\n"), 0644)
	mustWriteFile(t, filepath.Join(root, "epics", "E01", "T002.md"), []byte("---\nestado: Completed\ntipo: chore\npriority: low\n---\n# T002\n"), 0644)

	return root
}

func TestValidateNestedStem_AllValid(t *testing.T) {
	root := setupNestedStemProject(t)

	// Validate a deep file — should inherit estado from root and tipo from epics.
	out, err := runCmd(t, "validate", filepath.Join(root, "epics", "E01", "T001.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if result["valid"] != true {
		t.Errorf("expected valid=true for T001.md, got %v", result["valid"])
	}
}

func TestValidateNestedStem_MissingInheritedField(t *testing.T) {
	root := setupNestedStemProject(t)

	// Create file in epics/ missing the inherited estado field.
	mustWriteFile(t, filepath.Join(root, "epics", "bad.md"), []byte("---\ntipo: feature\n---\n# Bad\n"), 0644)

	out, _ := runCmd(t, "validate", filepath.Join(root, "epics", "bad.md"))
	// Should report validation error for missing estado.
	if !strings.Contains(out, "estado") {
		t.Errorf("expected estado error in output, got: %s", out)
	}
}

func TestExplainNestedStem_MultipleStemChain(t *testing.T) {
	root := setupNestedStemProject(t)

	out, err := runCmd(t, "explain", filepath.Join(root, "epics", "E01", "T001.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result rules.ExplainResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Should have 3 .stem files in the chain (root, epics, E01).
	if len(result.StemChain) != 3 {
		t.Errorf("stem_chain length = %d, want 3, chain: %v", len(result.StemChain), result.StemChain)
	}

	// Should have fields from all levels: estado (root), tipo (epics), priority (E01), slug (derived).
	fieldNames := make(map[string]bool)
	for _, f := range result.Fields {
		fieldNames[f.Name] = true
	}
	for _, expected := range []string{"estado", "tipo", "priority", "slug"} {
		if !fieldNames[expected] {
			t.Errorf("missing field %q in explain output, fields: %v", expected, fieldNames)
		}
	}

	// Verify slug is derived.
	for _, f := range result.Fields {
		if f.Name == "slug" {
			if f.Origin != "derived" {
				t.Errorf("slug origin = %q, want derived", f.Origin)
			}
			if f.Expression != "slugify(estado)" {
				t.Errorf("slug expression = %q, want slugify(estado)", f.Expression)
			}
		}
	}
}

func TestTreeNestedStem_CountsPropagateCorrectly(t *testing.T) {
	root := setupNestedStemProject(t)

	out, err := runCmd(t, "tree", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	// Total: root-doc.md (Pending), F01.md (Completed), T001.md (Pending), T002.md (Completed).
	if result.Root.Total != 4 {
		t.Errorf("total = %d, want 4", result.Root.Total)
	}
	if result.Root.Completed != 2 {
		t.Errorf("completed = %d, want 2", result.Root.Completed)
	}
}

func TestStatsNestedStem_CountsAll(t *testing.T) {
	root := setupNestedStemProject(t)

	out, err := runCmd(t, "stats", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result StatsResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	if result.Total != 4 {
		t.Errorf("total = %d, want 4", result.Total)
	}
	if result.ByEstado["Pending"] != 2 {
		t.Errorf("Pending = %d, want 2", result.ByEstado["Pending"])
	}
	if result.ByEstado["Completed"] != 2 {
		t.Errorf("Completed = %d, want 2", result.ByEstado["Completed"])
	}
	if result.ByTipo["feature"] != 1 {
		t.Errorf("feature = %d, want 1", result.ByTipo["feature"])
	}
}

func TestQueryNestedStem_WithWhere(t *testing.T) {
	root := setupNestedStemProject(t)

	out, err := runCmd(t, "query", "--from", root, "--where", "tipo == 'bug'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !strings.Contains(out, "T001") {
		t.Errorf("expected T001 in results (tipo=bug), got: %s", out)
	}
	if strings.Contains(out, "F01") {
		t.Errorf("expected F01 filtered out (tipo=feature)")
	}
}

func TestFixNestedStem_InheritsParentSchema(t *testing.T) {
	root := setupNestedStemProject(t)

	// Create file in deep dir missing both inherited (estado) and local (tipo) fields.
	mustWriteFile(t, filepath.Join(root, "epics", "broken.md"), []byte("---\ntitle: Test\n---\n# Broken\n"), 0644)

	out, err := runCmd(t, "fix", filepath.Join(root, "epics", "broken.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fix both inherited estado and local tipo.
	if !strings.Contains(out, "estado") {
		t.Errorf("expected estado fix, got: %s", out)
	}
	if !strings.Contains(out, "tipo") {
		t.Errorf("expected tipo fix, got: %s", out)
	}
}

func TestExplainNestedStem_RootFileHasNoChild(t *testing.T) {
	root := setupNestedStemProject(t)

	// Root-level file only sees the root .stem.
	out, err := runCmd(t, "explain", filepath.Join(root, "root-doc.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result rules.ExplainResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Should have only 1 .stem in chain (root only).
	if len(result.StemChain) != 1 {
		t.Errorf("stem_chain = %d, want 1 for root-level file, chain: %v", len(result.StemChain), result.StemChain)
	}

	// Should only have estado field (no tipo, no priority, no slug).
	fieldNames := make(map[string]bool)
	for _, f := range result.Fields {
		fieldNames[f.Name] = true
	}
	if !fieldNames["estado"] {
		t.Error("expected estado field in root doc explain")
	}
	if fieldNames["tipo"] {
		t.Error("unexpected tipo field in root doc (not in root .stem)")
	}
	if fieldNames["priority"] {
		t.Error("unexpected priority field in root doc")
	}
}
