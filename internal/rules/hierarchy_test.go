package rules

import (
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
