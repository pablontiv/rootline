package infer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/yuin/goldmark"
	gmast "github.com/yuin/goldmark/ast"
	"github.com/yuin/goldmark/text"
)

func TestGenerateFlatSchema_Basic(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "test.md",
			Frontmatter: map[string]any{
				"title":  "Test 1",
				"status": "done",
			},
		},
		{
			Path: "test2.md",
			Frontmatter: map[string]any{
				"title":  "Test 2",
				"status": "done",
			},
		},
	}

	stem, err := GenerateFlatSchema(context.Background(), ".", records, DefaultInferOptions())
	if err != nil {
		t.Fatalf("GenerateFlatSchema failed: %v", err)
	}

	if stem == nil {
		t.Fatal("expected stem to be non-nil")
	}

	if stem.Version != 2 {
		t.Errorf("expected version 2, got %d", stem.Version)
	}

	if stem.Scope.Match != "*.md" {
		t.Errorf("expected scope match '*.md', got %q", stem.Scope.Match)
	}

	// Should have title and status fields
	if _, ok := stem.Schema["title"]; !ok {
		t.Error("expected 'title' field in schema")
	}
	if _, ok := stem.Schema["status"]; !ok {
		t.Error("expected 'status' field in schema")
	}
}

func TestGenerateFlatSchema_EmptyRecords(t *testing.T) {
	records := []*extract.Record{}

	_, err := GenerateFlatSchema(context.Background(), ".", records, DefaultInferOptions())
	if err == nil {
		t.Fatal("expected error for empty records")
	}
}

func TestGenerateFlatSchema_WithSections(t *testing.T) {
	// Helper to parse markdown to AST
	parseAST := func(body string) gmast.Node {
		source := []byte(body)
		reader := text.NewReader(source)
		return goldmark.DefaultParser().Parse(reader)
	}

	// Create records with AST (body awareness)
	markdown1 := `# Doc 1

## Overview
Content here

## Implementation
More content
`

	markdown2 := `# Doc 2

## Overview
Content here

## Implementation
More content
`

	records := []*extract.Record{
		{
			Path: "test1.md",
			Body: markdown1,
			AST:  parseAST(markdown1),
			Frontmatter: map[string]any{
				"title": "Doc 1",
			},
		},
		{
			Path: "test2.md",
			Body: markdown2,
			AST:  parseAST(markdown2),
			Frontmatter: map[string]any{
				"title": "Doc 2",
			},
		},
	}

	opts := DefaultInferOptions()
	opts.SectionThreshold = 0.80 // Both sections appear in 100%, should be required

	stem, err := GenerateFlatSchema(context.Background(), ".", records, opts)
	if err != nil {
		t.Fatalf("GenerateFlatSchema failed: %v", err)
	}

	// Should detect overview and implementation as required sections
	if overview, ok := stem.Schema["overview"]; ok {
		if !overview.Required {
			t.Error("expected 'overview' section to be required")
		}
	} else {
		t.Error("expected 'overview' field in schema")
	}

	if impl, ok := stem.Schema["implementation"]; ok {
		if !impl.Required {
			t.Error("expected 'implementation' section to be required")
		}
	} else {
		t.Error("expected 'implementation' field in schema")
	}
}

func TestGenerateHierarchicalSchema_NoHierarchy(t *testing.T) {
	// Records without hierarchical structure
	records := []*extract.Record{
		{
			Path: "doc1.md",
			Frontmatter: map[string]any{
				"title": "Test",
			},
		},
		{
			Path: "doc2.md",
			Frontmatter: map[string]any{
				"title": "Test",
			},
		},
	}

	_, err := GenerateHierarchicalSchema(context.Background(), ".", records, DefaultInferOptions())
	if err == nil {
		t.Fatal("expected error when hierarchy is not detected")
	}
}

func TestGenerateHierarchicalSchema_WithHierarchy(t *testing.T) {
	// Create temp directory structure
	tmpDir := t.TempDir()

	// Create hierarchical structure: E01-*/F01-*/T001-*.md and E02-*/F02-*/
	e01Dir := filepath.Join(tmpDir, "E01-epic1")
	e02Dir := filepath.Join(tmpDir, "E02-epic2")
	f01Dir := filepath.Join(e01Dir, "F01-feature1")
	f02Dir := filepath.Join(e02Dir, "F02-feature2")
	if err := os.MkdirAll(f01Dir, 0755); err != nil {
		t.Fatalf("Failed to create f01Dir: %v", err)
	}
	if err := os.MkdirAll(f02Dir, 0755); err != nil {
		t.Fatalf("Failed to create f02Dir: %v", err)
	}

	// Create records with matching hierarchical paths that will be detected
	// We need at least 2 levels to be detected, and 2+ items per level
	records := []*extract.Record{
		{
			Path: filepath.Join(f01Dir, "T001-task1.md"),
			Frontmatter: map[string]any{
				"title":  "Task 1",
				"status": "done",
				"type":   "task",
			},
		},
		{
			Path: filepath.Join(f01Dir, "T002-task2.md"),
			Frontmatter: map[string]any{
				"title":  "Task 2",
				"status": "done",
				"type":   "task",
			},
		},
		{
			Path: filepath.Join(f02Dir, "T001-task3.md"),
			Frontmatter: map[string]any{
				"title": "Task 3",
				"type":  "task",
			},
		},
		{
			Path: filepath.Join(f02Dir, "T002-task4.md"),
			Frontmatter: map[string]any{
				"title": "Task 4",
				"type":  "task",
			},
		},
	}

	stem, err := GenerateHierarchicalSchema(context.Background(), tmpDir, records, DefaultInferOptions())
	if err != nil {
		t.Fatalf("GenerateHierarchicalSchema failed: %v", err)
	}

	if stem == nil {
		t.Fatal("expected stem map to be non-nil")
	}

	// Should have root stem at "."
	rootStem, ok := stem["."]
	if !ok {
		t.Fatal("expected root .stem at '.' key")
	}

	if rootStem == nil {
		t.Fatal("expected root stem to be non-nil")
	}

	if rootStem.Version != 2 {
		t.Errorf("expected version 2, got %d", rootStem.Version)
	}

	// Should have schema fields
	if len(rootStem.Schema) == 0 {
		t.Error("expected schema to have fields")
	}

	// Should have aggregates for enum fields (if any)
	if rootStem.Aggregate == nil {
		rootStem.Aggregate = make(map[string]any)
	}
}

func TestGenerateFlatSchema_RequiredField(t *testing.T) {
	// Create records where one field appears in >80% of them
	records := []*extract.Record{
		{
			Path: "doc1.md",
			Frontmatter: map[string]any{
				"title":       "Doc 1",
				"description": "This is doc 1",
			},
		},
		{
			Path: "doc2.md",
			Frontmatter: map[string]any{
				"title":       "Doc 2",
				"description": "This is doc 2",
			},
		},
		{
			Path: "doc3.md",
			Frontmatter: map[string]any{
				"title": "Doc 3",
				// description missing
			},
		},
	}

	stem, err := GenerateFlatSchema(context.Background(), ".", records, DefaultInferOptions())
	if err != nil {
		t.Fatalf("GenerateFlatSchema failed: %v", err)
	}

	// title appears in 3/3 records (>80%), should be required
	titleField, ok := stem.Schema["title"]
	if !ok {
		t.Fatal("expected 'title' field")
	}
	if !titleField.Required {
		t.Error("expected 'title' to be required (100% coverage)")
	}

	// description appears in 2/3 records (66%), below 80%, should not be required
	descField, ok := stem.Schema["description"]
	if !ok {
		t.Fatal("expected 'description' field")
	}
	if descField.Required {
		t.Error("expected 'description' to not be required (66% coverage)")
	}
}

func TestGenerateFlatSchema_EnumDetection(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "item1.md",
			Frontmatter: map[string]any{
				"status": "open",
			},
		},
		{
			Path: "item2.md",
			Frontmatter: map[string]any{
				"status": "closed",
			},
		},
		{
			Path: "item3.md",
			Frontmatter: map[string]any{
				"status": "open",
			},
		},
	}

	stem, err := GenerateFlatSchema(context.Background(), ".", records, DefaultInferOptions())
	if err != nil {
		t.Fatalf("GenerateFlatSchema failed: %v", err)
	}

	statusField, ok := stem.Schema["status"]
	if !ok {
		t.Fatal("expected 'status' field")
	}

	// status should be inferred as enum (2 unique values, appears in 100% of records)
	if statusField.Type != "enum" {
		t.Errorf("expected status type 'enum', got %q", statusField.Type)
	}

	// Should have both values
	if len(statusField.Values) != 2 {
		t.Errorf("expected 2 enum values, got %d", len(statusField.Values))
	}
}

func TestSectionFieldName(t *testing.T) {
	tests := []struct {
		heading string
		want    string
	}{
		{"Overview", "overview"},
		{"Implementation Details", "implementation_details"},
		{"API Reference", "api_reference"},
		{"## Test", "##_test"},
	}

	for _, tt := range tests {
		got := sectionFieldName(tt.heading)
		if got != tt.want {
			t.Errorf("sectionFieldName(%q) = %q, want %q", tt.heading, got, tt.want)
		}
	}
}

func TestGenerateFlatSchema_NoStructuralRules(t *testing.T) {
	records := []*extract.Record{
		{
			Path: "test.md",
			Frontmatter: map[string]any{
				"title": "Test",
			},
		},
	}

	// With IncludeStructural=false, structural rules should be empty
	opts := DefaultInferOptions()
	opts.IncludeStructural = false

	stem, err := GenerateFlatSchema(context.Background(), ".", records, opts)
	if err != nil {
		t.Fatalf("GenerateFlatSchema failed: %v", err)
	}

	if !stem.Structural.IsEmpty() {
		t.Error("expected empty structural rules when IncludeStructural=false")
	}
}
