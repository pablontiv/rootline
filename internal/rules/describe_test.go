package rules

import (
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"
)

func mustNewDescribeResult(t *testing.T, path string, entries []StemEntry, effective *StemFile) *DescribeResult {
	t.Helper()
	result, err := NewDescribeResult(path, entries, effective)
	if err != nil {
		t.Fatal(err)
	}
	return result
}

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

	result := mustNewDescribeResult(t, "docs/prd/", entries, effective)

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
	result := mustNewDescribeResult(t, "some/path/", nil, &StemFile{})

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
			"title": {Type: "string", Required: true, Source: "root/.stem", Extract: `body.section["## Title"]`},
		},
	}
	result := mustNewDescribeResult(t, "docs/", entries, effective)

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
	if titleField["source"] != `body.section["## Title"]` {
		t.Errorf("schema.title.source = %v", titleField["source"])
	}
	if titleField["defined_in"] != "root/.stem" {
		t.Errorf("schema.title.defined_in = %v", titleField["defined_in"])
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

	result := mustNewDescribeResult(t, dir, entries, effective)

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

	result := mustNewDescribeResult(t, "test/", entries, effective)

	titleField := result.Schema["title"]
	if titleField.Next != "" {
		t.Errorf("non-sequence field has Next = %q, want empty", titleField.Next)
	}
}

func TestDescribeResult_NilFieldsBecomeMaps(t *testing.T) {
	result := mustNewDescribeResult(t, "x/", nil, &StemFile{})

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

func TestDescribeResult_ProvenanceWithNestedStems(t *testing.T) {
	// Simulate a 2-level stem chain: root/.stem defines "title" and "tipo",
	// docs/tasks/.stem narrows "tipo".
	rootStem := &StemFile{
		Schema: map[string]SchemaField{
			"title": {Type: "string", Required: true},
			"tipo":  {Type: "enum", Values: []string{"feature", "fix"}},
		},
	}
	taskStem := &StemFile{
		Schema: map[string]SchemaField{
			"tipo": {Type: "enum", Values: []string{"feature"}}, // narrowed
		},
	}

	entries := []StemEntry{
		{Path: "root/.stem", Stem: rootStem},
		{Path: "docs/tasks/.stem", Stem: taskStem},
	}

	// Effective is the merged schema
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"title": {Type: "string", Required: true, Source: "root/.stem"},
			"tipo":  {Type: "enum", Values: []string{"feature"}, Source: "docs/tasks/.stem"},
		},
	}

	result := mustNewDescribeResult(t, "docs/tasks/", entries, effective)

	// Check layers
	if len(result.Layers) != 2 {
		t.Fatalf("layers = %d, want 2", len(result.Layers))
	}
	if result.Layers[0] != "root/.stem" || result.Layers[1] != "docs/tasks/.stem" {
		t.Errorf("layers = %v, want [root/.stem docs/tasks/.stem]", result.Layers)
	}

	// Check provenance
	if result.Provenance["title"] != "root/.stem" {
		t.Errorf("provenance[title] = %q, want root/.stem", result.Provenance["title"])
	}
	if result.Provenance["tipo"] != "docs/tasks/.stem" {
		t.Errorf("provenance[tipo] = %q, want docs/tasks/.stem", result.Provenance["tipo"])
	}

	// Verify in JSON
	data, err := result.ToJSON()
	if err != nil {
		t.Fatalf("ToJSON: %v", err)
	}

	var parsed map[string]any
	if err := json.Unmarshal(data, &parsed); err != nil {
		t.Fatalf("JSON parse: %v", err)
	}

	layers, ok := parsed["layers"].([]any)
	if !ok || len(layers) != 2 {
		t.Fatalf("JSON layers = %v, want 2-element array", layers)
	}

	prov, ok := parsed["provenance"].(map[string]any)
	if !ok {
		t.Fatalf("JSON provenance = %v, want object", prov)
	}
	if prov["title"] != "root/.stem" {
		t.Errorf("JSON provenance.title = %v, want root/.stem", prov["title"])
	}
	if prov["tipo"] != "docs/tasks/.stem" {
		t.Errorf("JSON provenance.tipo = %v, want docs/tasks/.stem", prov["tipo"])
	}
}

func TestDescribeResult_NextByPatternInJSON(t *testing.T) {
	dir := t.TempDir()
	// Create entries matching both O* and T* patterns
	if err := os.Mkdir(filepath.Join(dir, "O01-outcome"), 0o755); err != nil {
		t.Fatal(err)
	}
	writeTestFile(t, filepath.Join(dir, "T001-task.md"))

	entries := []StemEntry{{Path: filepath.Join(dir, ".stem"), Stem: &StemFile{}}}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"id": {
				Type: "sequence",
				Match: &FieldMatch{
					Configs: map[string]any{
						"O*": map[string]any{"prefix": "O", "digits": 2},
						"T*": map[string]any{"prefix": "T", "digits": 3},
					},
				},
			},
		},
	}

	result := mustNewDescribeResult(t, dir, entries, effective)

	idField := result.Schema["id"]
	if idField.NextByPattern == nil {
		t.Fatal("NextByPattern is nil, want map")
	}
	if idField.NextByPattern["O*"] != "O02" {
		t.Errorf("NextByPattern[O*] = %q, want O02", idField.NextByPattern["O*"])
	}
	if idField.NextByPattern["T*"] != "T002" {
		t.Errorf("NextByPattern[T*] = %q, want T002", idField.NextByPattern["T*"])
	}

	// Verify it appears in JSON output
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
	nbp, ok := idJSON["next_by_pattern"].(map[string]any)
	if !ok {
		t.Fatalf("next_by_pattern missing from JSON or wrong type: %v", idJSON)
	}
	if nbp["O*"] != "O02" {
		t.Errorf("JSON next_by_pattern[O*] = %v, want O02", nbp["O*"])
	}
}

func TestNewDescribeResultInvalidSequenceDoesNotMutateInput(t *testing.T) {
	dir := t.TempDir()
	effective := &StemFile{Schema: map[string]SchemaField{
		"alpha": {Type: "sequence", Prefix: "A", Digits: 2},
		"beta": {
			Type:   "sequence",
			Prefix: "B",
			Digits: 2,
			Match: &FieldMatch{Configs: map[string]any{
				"B*": map[string]any{"digits": 2.0},
			}},
		},
	}}
	beforeAlpha := effective.Schema["alpha"]
	beforeBeta := effective.Schema["beta"]

	result, err := NewDescribeResult(dir, nil, effective)
	if err == nil || !strings.Contains(err.Error(), "beta") || !strings.Contains(err.Error(), "digits") {
		t.Fatalf("NewDescribeResult result=%#v err=%v, want beta digits error", result, err)
	}
	if !reflect.DeepEqual(effective.Schema["alpha"], beforeAlpha) || !reflect.DeepEqual(effective.Schema["beta"], beforeBeta) {
		t.Fatalf("effective schema fields mutated: got alpha=%#v beta=%#v", effective.Schema["alpha"], effective.Schema["beta"])
	}
}

func TestNewDescribeResultInvalidSequenceFirstErrorDeterministic(t *testing.T) {
	dir := t.TempDir()
	effective := &StemFile{Schema: map[string]SchemaField{
		"zeta": {
			Type:   "sequence",
			Prefix: "Z",
			Digits: 2,
			Match:  &FieldMatch{Configs: map[string]any{"Z*": map[string]any{"digits": 2.0}}},
		},
		"alpha": {
			Type:   "sequence",
			Prefix: "A",
			Digits: 2,
			Match:  &FieldMatch{Configs: map[string]any{"A*": map[string]any{"digits": 2.0}}},
		},
	}}
	for i := 0; i < 20; i++ {
		_, err := NewDescribeResult(dir, nil, effective)
		if err == nil || !strings.Contains(err.Error(), "alpha") {
			t.Fatalf("run %d error = %v, want deterministic alpha first error", i, err)
		}
	}
}
