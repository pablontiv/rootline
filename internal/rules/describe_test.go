package rules

import (
	"encoding/json"
	"path/filepath"
	"testing"
)

func TestNewDescribeResult_FullSchema(t *testing.T) {
	entries := []StemEntry{
		{Path: "docs/.stem", Stem: &StemFile{}},
		{Path: "docs/prd/.stem", Stem: &StemFile{}},
	}
	effective := &StemFile{
		Scope: Scope{Match: "*.md"},
		Schema: map[string]SchemaField{
			"title":  {Type: "string", Required: true, Source: "docs/.stem"},
			"estado": {Type: "enum", Values: []string{"Pending", "Done"}, Source: "docs/prd/.stem"},
		},
		Validate: []ValidationRule{
			{Rule: "non_empty", Field: "title", Source: "docs/.stem"},
		},
	}

	result := NewDescribeResult("docs/prd/", entries, effective)

	if result.Version != 1 {
		t.Errorf("version = %d", result.Version)
	}
	if result.Kind != "rootline/describe" {
		t.Errorf("kind = %q", result.Kind)
	}
	if result.Path != "docs/prd/" {
		t.Errorf("path = %q", result.Path)
	}
	if len(result.Applies) != 2 {
		t.Fatalf("applies = %v, want 2 entries", result.Applies)
	}
	if result.Applies[0] != "docs/.stem" || result.Applies[1] != "docs/prd/.stem" {
		t.Errorf("applies = %v", result.Applies)
	}
	if result.Scope.Match != "*.md" {
		t.Errorf("scope.match = %q", result.Scope.Match)
	}
	if len(result.Schema) != 2 {
		t.Errorf("schema has %d fields", len(result.Schema))
	}
	if len(result.Validate) != 1 {
		t.Errorf("validate has %d rules", len(result.Validate))
	}
}

func TestNewDescribeResult_EmptySchema(t *testing.T) {
	result := NewDescribeResult("some/path/", nil, &StemFile{})

	if len(result.Applies) != 0 {
		t.Errorf("applies = %v, want empty", result.Applies)
	}
	if len(result.Schema) != 0 {
		t.Errorf("schema = %v, want empty", result.Schema)
	}
	if len(result.Validate) != 0 {
		t.Errorf("validate = %v, want empty", result.Validate)
	}
}

func TestDescribeResult_ToJSON(t *testing.T) {
	entries := []StemEntry{{Path: "root/.stem", Stem: &StemFile{}}}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"title": {Type: "string", Required: true, Source: "root/.stem"},
		},
	}
	result := NewDescribeResult("docs/", entries, effective)

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	if parsed["version"].(float64) != 1 {
		t.Errorf("version = %v", parsed["version"])
	}
	if parsed["kind"] != "rootline/describe" {
		t.Errorf("kind = %v", parsed["kind"])
	}
	schema := parsed["schema"].(map[string]any)
	titleField := schema["title"].(map[string]any)
	if titleField["source"] != "root/.stem" {
		t.Errorf("schema.title.source = %v", titleField["source"])
	}
}

func TestDescribeResult_SequenceNext(t *testing.T) {
	dir := t.TempDir()
	// Create files matching the sequence pattern
	writeTestFile(t, filepath.Join(dir, "T001-first.md"))
	writeTestFile(t, filepath.Join(dir, "T002-second.md"))

	entries := []StemEntry{{Path: filepath.Join(dir, ".stem"), Stem: &StemFile{}}}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"id": {Type: "sequence", Prefix: "T", Digits: 3},
		},
	}

	result := NewDescribeResult(dir, entries, effective)

	idField, ok := result.Schema["id"]
	if !ok {
		t.Fatal("schema field 'id' not found")
	}
	if idField.Next != "T003" {
		t.Errorf("Next = %q, want %q", idField.Next, "T003")
	}

	// Verify it appears in JSON
	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}
	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}
	schema := parsed["schema"].(map[string]any)
	idJSON := schema["id"].(map[string]any)
	if idJSON["next"] != "T003" {
		t.Errorf("JSON schema.id.next = %v, want T003", idJSON["next"])
	}
}

func TestDescribeResult_NonSequenceFieldUnchanged(t *testing.T) {
	entries := []StemEntry{{Path: "test/.stem", Stem: &StemFile{}}}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"title": {Type: "string", Required: true},
		},
	}

	result := NewDescribeResult("test/", entries, effective)

	titleField := result.Schema["title"]
	if titleField.Next != "" {
		t.Errorf("non-sequence field has Next = %q, want empty", titleField.Next)
	}
}

func TestDescribeResult_NilFieldsBecomeMaps(t *testing.T) {
	result := NewDescribeResult("x/", nil, &StemFile{})

	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON error: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse error: %v", err)
	}

	// All should be objects/arrays, never null
	for _, key := range []string{"schema", "validate", "derive", "aggregate", "links", "applies"} {
		if parsed[key] == nil {
			t.Errorf("%s is null, want empty object/array", key)
		}
	}
}
