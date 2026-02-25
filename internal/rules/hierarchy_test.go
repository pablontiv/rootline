package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestHierarchyLevelParsing(t *testing.T) {
	yaml := `
version: 1
scope:
  match: "*.md"
levels:
  epic:
    match: "E*"
    children: [feature]
    schema:
      id:
        type: sequence
        prefix: E
        digits: 2
      estado:
        type: enum
        values: [Pending, In Progress, Completed]
    validate:
      - field: estado
        rule: required
  feature:
    match: "F*"
    children: [story]
    schema:
      id:
        type: sequence
        prefix: F
        digits: 2
  story:
    match: "S*"
    children: [task]
  task:
    match: "T*"
    children: []
`
	stem, err := ParseStem("test.stem", []byte(yaml))
	if err != nil {
		t.Fatalf("ParseStem failed: %v", err)
	}

	if stem.Levels == nil {
		t.Fatal("expected Levels to be non-nil")
	}

	if len(stem.Levels) != 4 {
		t.Fatalf("expected 4 levels, got %d", len(stem.Levels))
	}

	// Check epic level
	epic := stem.Levels["epic"]
	if epic == nil {
		t.Fatal("expected 'epic' level")
	}
	if epic.Match != "E*" {
		t.Errorf("epic.Match = %q, want %q", epic.Match, "E*")
	}
	if len(epic.Children) != 1 || epic.Children[0] != "feature" {
		t.Errorf("epic.Children = %v, want [feature]", epic.Children)
	}
	if len(epic.Schema) != 2 {
		t.Errorf("epic.Schema has %d fields, want 2", len(epic.Schema))
	}
	idField := epic.Schema["id"]
	if idField.Type != "sequence" || idField.Prefix != "E" || idField.Digits != 2 {
		t.Errorf("epic.Schema[id] = %+v, want sequence/E/2", idField)
	}
	if len(epic.Validate) != 1 || epic.Validate[0].Field != "estado" {
		t.Errorf("epic.Validate = %+v, want [{field:estado rule:required}]", epic.Validate)
	}

	// Check feature level
	feature := stem.Levels["feature"]
	if feature == nil {
		t.Fatal("expected 'feature' level")
	}
	if feature.Match != "F*" {
		t.Errorf("feature.Match = %q, want %q", feature.Match, "F*")
	}
	if len(feature.Children) != 1 || feature.Children[0] != "story" {
		t.Errorf("feature.Children = %v, want [story]", feature.Children)
	}

	// Check task level has empty children
	task := stem.Levels["task"]
	if task == nil {
		t.Fatal("expected 'task' level")
	}
	if len(task.Children) != 0 {
		t.Errorf("task.Children = %v, want []", task.Children)
	}
}

func TestHierarchyLevelEmpty(t *testing.T) {
	yaml := `
version: 1
scope:
  match: "*.md"
schema:
  estado:
    type: string
`
	stem, err := ParseStem("test.stem", []byte(yaml))
	if err != nil {
		t.Fatalf("ParseStem failed: %v", err)
	}

	if stem.Levels != nil {
		t.Errorf("expected Levels to be nil, got %v", stem.Levels)
	}
}

// --- ExpandLevels tests ---

func TestExpandLevels(t *testing.T) {
	stem := &StemFile{
		Levels: map[string]*HierarchyLevel{
			"epic": {Match: "E*", Children: []string{"feature"},
				Schema: map[string]SchemaField{"id": {Type: "sequence", Prefix: "E", Digits: 2}}},
			"feature": {Match: "F*", Children: []string{"story"},
				Schema: map[string]SchemaField{"id": {Type: "sequence", Prefix: "F", Digits: 2}}},
			"story": {Match: "S*", Children: []string{"task"},
				Schema: map[string]SchemaField{"id": {Type: "sequence", Prefix: "S", Digits: 3}}},
			"task": {Match: "T*", Children: []string{},
				Schema:   map[string]SchemaField{"id": {Type: "sequence", Prefix: "T", Digits: 3}},
				Validate: []ValidationRule{{Field: "estado", Rule: "required"}}},
		},
	}

	entries := ExpandLevels(stem, "E01/F02/S001/T001.md")
	if len(entries) != 4 {
		t.Fatalf("expected 4 virtual entries, got %d", len(entries))
	}

	// Verify order: epic, feature, story, task (shallowest first)
	wantPrefixes := []string{"E", "F", "S", "T"}
	for i, entry := range entries {
		idField := entry.Stem.Schema["id"]
		if idField.Prefix != wantPrefixes[i] {
			t.Errorf("entry[%d] prefix = %q, want %q", i, idField.Prefix, wantPrefixes[i])
		}
	}

	// Last entry (task) should have validate rules
	if len(entries[3].Stem.Validate) != 1 {
		t.Errorf("task entry should have 1 validate rule, got %d", len(entries[3].Stem.Validate))
	}
}

func TestExpandLevelsNoMatch(t *testing.T) {
	stem := &StemFile{
		Levels: map[string]*HierarchyLevel{
			"epic": {Match: "E*", Schema: map[string]SchemaField{"id": {Type: "string"}}},
		},
	}

	entries := ExpandLevels(stem, "unknown/path/file.md")
	if len(entries) != 0 {
		t.Errorf("expected 0 entries for non-matching path, got %d", len(entries))
	}
}

func TestExpandLevelsPartialMatch(t *testing.T) {
	stem := &StemFile{
		Levels: map[string]*HierarchyLevel{
			"epic":    {Match: "E*", Schema: map[string]SchemaField{"tipo": {Type: "string"}}},
			"feature": {Match: "F*", Schema: map[string]SchemaField{"estado": {Type: "string"}}},
		},
	}

	// Only E01 and F02 match; "unknown" doesn't
	entries := ExpandLevels(stem, "E01/unknown/F02/file.md")
	if len(entries) != 2 {
		t.Fatalf("expected 2 entries, got %d", len(entries))
	}
}

func TestExpandLevelsNilStem(t *testing.T) {
	entries := ExpandLevels(nil, "E01/F02/file.md")
	if entries != nil {
		t.Errorf("expected nil for nil stem, got %v", entries)
	}
}

func TestExpandLevelsNoLevels(t *testing.T) {
	stem := &StemFile{Version: 1}
	entries := ExpandLevels(stem, "E01/F02/file.md")
	if entries != nil {
		t.Errorf("expected nil for stem without levels, got %v", entries)
	}
}

func TestExpandLevelsWithRealChildMergeOrder(t *testing.T) {
	// Simulate: real child .stem + virtual level — real wins because it comes
	// first in the merge chain, then virtual overrides.
	// Actually per the design: real entries come first, virtual entries are appended.
	// So virtual entries override the real ones in the final merge.
	// But a real child .stem at a deeper level should still be able to override.
	stem := &StemFile{
		Schema: map[string]SchemaField{
			"base": {Type: "string", Required: true},
		},
		Levels: map[string]*HierarchyLevel{
			"epic": {Match: "E*",
				Schema: map[string]SchemaField{"epic_field": {Type: "string"}}},
		},
	}

	entries := ExpandLevels(stem, "E01/file.md")
	if len(entries) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries))
	}
	if entries[0].Stem.Schema["epic_field"].Type != "string" {
		t.Error("virtual entry should contain epic_field")
	}
	// Base schema is NOT in the virtual entry — it's on the parent stem
	if _, hasBase := entries[0].Stem.Schema["base"]; hasBase {
		t.Error("virtual entry should not contain base field")
	}
}

func TestResolveForRecordNoLevels(t *testing.T) {
	// Create a temp dir with a .stem and .git
	dir := t.TempDir()
	// Create .git marker
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Write a simple .stem
	stemContent := []byte("version: 1\nschema:\n  estado:\n    type: string\n")
	if err := os.WriteFile(filepath.Join(dir, ".stem"), stemContent, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ResolveForRecord(dir, "file.md")
	if err != nil {
		t.Fatalf("ResolveForRecord failed: %v", err)
	}
	if result.Schema["estado"].Type != "string" {
		t.Error("expected estado field from .stem")
	}
	if result.Levels != nil {
		t.Error("expected nil Levels")
	}
}

func TestResolveForRecordWithLevels(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := []byte(`
version: 1
scope:
  match: "*.md"
schema:
  base:
    type: string
    required: true
levels:
  epic:
    match: "E*"
    schema:
      epic_id:
        type: sequence
        prefix: E
        digits: 2
  task:
    match: "T*"
    schema:
      task_id:
        type: sequence
        prefix: T
        digits: 3
`)
	if err := os.WriteFile(filepath.Join(dir, ".stem"), stemContent, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ResolveForRecord(dir, "E01/T001.md")
	if err != nil {
		t.Fatalf("ResolveForRecord failed: %v", err)
	}

	// Should have base (from root schema) + task_id (from task level, last match)
	if result.Schema["base"].Type != "string" {
		t.Error("expected base field from root schema")
	}
	// task_id should be present from the task level virtual entry
	if result.Schema["task_id"].Type != "sequence" {
		t.Errorf("expected task_id from task level, got %+v", result.Schema["task_id"])
	}
	// epic_id should be present from the epic level virtual entry
	if result.Schema["epic_id"].Type != "sequence" {
		t.Errorf("expected epic_id from epic level, got %+v", result.Schema["epic_id"])
	}
}

// --- CheckNesting tests ---

func testLevels() map[string]*HierarchyLevel {
	return map[string]*HierarchyLevel{
		"epic":    {Match: "E*", Children: []string{"feature"}},
		"feature": {Match: "F*", Children: []string{"story"}},
		"story":   {Match: "S*", Children: []string{"task"}},
		"task":    {Match: "T*", Children: []string{}},
	}
}

func TestCheckNesting(t *testing.T) {
	errs := CheckNesting(testLevels(), "E01/F02/S001/T001.md")
	if len(errs) != 0 {
		t.Errorf("expected no errors for valid chain, got %v", errs)
	}
}

func TestCheckNestingInvalid(t *testing.T) {
	errs := CheckNesting(testLevels(), "E01/T001.md")
	if len(errs) != 1 {
		t.Fatalf("expected 1 error, got %d: %v", len(errs), errs)
	}
	if errs[0].Rule != "nesting" {
		t.Errorf("error rule = %q, want %q", errs[0].Rule, "nesting")
	}
}

func TestCheckNestingLeaf(t *testing.T) {
	// Subdir under leaf (task has children: [])
	errs := CheckNesting(testLevels(), "E01/F02/S001/T001/sub.md")
	// T001 matches task, sub.md doesn't match any level — no error
	// because unknown components are skipped
	if len(errs) != 0 {
		t.Errorf("expected no errors for unknown component under leaf, got %v", errs)
	}

	// Test a clear case: leaf level with no allowed children
	levels2 := map[string]*HierarchyLevel{
		"parent": {Match: "P*", Children: []string{"child"}},
		"child":  {Match: "C*", Children: []string{}},
		"leaf":   {Match: "L*", Children: []string{}},
	}
	errs = CheckNesting(levels2, "P01/C01/L01.md")
	if len(errs) != 1 {
		t.Fatalf("expected 1 error for leaf under child (children:[]), got %d: %v", len(errs), errs)
	}
}

func TestCheckNestingNoLevels(t *testing.T) {
	errs := CheckNesting(nil, "E01/F02/S001/T001.md")
	if len(errs) != 0 {
		t.Errorf("expected no errors for nil levels, got %v", errs)
	}
}

func TestCheckNestingUnknownComponent(t *testing.T) {
	// Path with component not matching any level is skipped
	errs := CheckNesting(testLevels(), "E01/unknown/F02/S001/T001.md")
	// "unknown" doesn't match any level, parentLevel resets to ""
	// F02 matches feature but has no parent context → OK
	// S001 under feature → OK, T001 under story → OK
	if len(errs) != 0 {
		t.Errorf("expected no errors with unknown component, got %v", errs)
	}
}
