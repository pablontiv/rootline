package infer

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
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

func TestGenerateFlatSchema_PropagatesSectionThresholdError(t *testing.T) {
	opts := DefaultInferOptions()
	opts.IncludeStructural = false
	opts.SectionThreshold = -0.1
	_, err := GenerateFlatSchema(context.Background(), ".", []*extract.Record{makeRecord("## Notes\n")}, opts)
	if err == nil || !strings.Contains(err.Error(), "threshold") {
		t.Fatalf("expected section threshold error, got %v", err)
	}
}

func TestGenerateFlatSchema_PropagatesSectionCollisionError(t *testing.T) {
	opts := DefaultInferOptions()
	opts.IncludeStructural = false
	_, err := GenerateFlatSchema(context.Background(), ".", []*extract.Record{makeSectionRecord(
		extract.Section{Level: 2, Heading: "Notes"},
		extract.Section{Level: 3, Heading: "Notes"},
	)}, opts)
	if err == nil || !strings.Contains(err.Error(), "section field name collision") || !strings.Contains(err.Error(), "## Notes") || !strings.Contains(err.Error(), "### Notes") {
		t.Fatalf("expected section collision error, got %v", err)
	}
}

func TestGenerateFlatSchema_SectionSourceUsesCanonicalDirective(t *testing.T) {
	records := []*extract.Record{
		makeSectionRecord(
			extract.Section{Level: 1, Heading: "Overview"},
			extract.Section{Level: 2, Heading: `Need "quotes" \ and [brackets]`},
			extract.Section{Level: 3, Heading: "Deep Dive"},
		),
		makeSectionRecord(
			extract.Section{Level: 1, Heading: "Overview"},
			extract.Section{Level: 2, Heading: `Need "quotes" \ and [brackets]`},
			extract.Section{Level: 3, Heading: "Deep Dive"},
		),
	}
	opts := DefaultInferOptions()
	opts.IncludeStructural = false
	stem, err := GenerateFlatSchema(context.Background(), ".", records, opts)
	if err != nil {
		t.Fatalf("GenerateFlatSchema: %v", err)
	}
	want := map[string]string{
		"overview":                 `body.section["# Overview"]`,
		"need_quotes_and_brackets": `body.section["## Need \"quotes\" \\ and [brackets]"]`,
		"deep_dive":                `body.section["### Deep Dive"]`,
	}
	for field, source := range want {
		sf, ok := stem.Schema[field]
		if !ok {
			t.Fatalf("missing section field %q in %#v", field, stem.Schema)
		}
		if sf.Type != "string" || sf.Extract != source || sf.Heading != "" || !sf.Required {
			t.Fatalf("schema[%s] = %+v, want required string source %q without heading", field, sf, source)
		}
	}
}

func TestGenerateFlatSchema_RejectsSectionCollisionWithFrontmatter(t *testing.T) {
	records := []*extract.Record{
		makeSectionRecord(extract.Section{Level: 2, Heading: "Notes"}),
		makeSectionRecord(extract.Section{Level: 2, Heading: "Notes"}),
	}
	for _, rec := range records {
		rec.Frontmatter = map[string]any{"notes": []string{"keep", "frontmatter"}}
	}
	opts := DefaultInferOptions()
	opts.IncludeStructural = false
	_, err := GenerateFlatSchema(context.Background(), ".", records, opts)
	if err == nil || !strings.Contains(err.Error(), `field "notes"`) || !strings.Contains(err.Error(), "frontmatter") || !strings.Contains(err.Error(), "body section") || !strings.Contains(err.Error(), "body.section[") || !strings.Contains(err.Error(), "## Notes") {
		t.Fatalf("expected frontmatter/body section collision error, got %v", err)
	}

	schema := Analyze(records).Schema
	before := schema["notes"]
	if err := addSectionInferenceFields(schema, []Inference{{Type: "required_section", Field: "notes", SourceDirective: `body.section["## Notes"]`}}, "frontmatter"); err == nil {
		t.Fatal("expected helper collision error")
	}
	if after := schema["notes"]; after.Type != before.Type || strings.Join(after.Values, ",") != strings.Join(before.Values, ",") || after.Extract != before.Extract {
		t.Fatalf("existing frontmatter field mutated: before=%+v after=%+v", before, after)
	}
}

func TestGenerateHierarchicalSchema_RejectsSectionCollisionWithHierarchyField(t *testing.T) {
	tmpDir := t.TempDir()
	var records []*extract.Record
	for _, path := range []string{
		"E01-epic/F01-feature/T001-one.md", "E01-epic/F01-feature/T002-two.md",
		"E02-epic/F02-feature/T001-three.md", "E02-epic/F02-feature/T002-four.md",
	} {
		rec := makeSectionRecord(extract.Section{Level: 2, Heading: "ID"})
		rec.Path = filepath.Join(tmpDir, path)
		records = append(records, rec)
	}
	opts := DefaultInferOptions()
	opts.IncludeStructural = false
	_, err := GenerateHierarchicalSchema(context.Background(), tmpDir, records, opts)
	if err == nil || !strings.Contains(err.Error(), `field "id"`) || !strings.Contains(err.Error(), "hierarchy") || !strings.Contains(err.Error(), "body section") || !strings.Contains(err.Error(), "body.section[") || !strings.Contains(err.Error(), "## ID") {
		t.Fatalf("expected hierarchy/body section collision error, got %v", err)
	}

	rootStem := buildRootStemFile(AnalyzeHierarchy(records, tmpDir), nil, tmpDir, opts)
	before := rootStem.Schema["id"]
	if err := addSectionInferenceFields(rootStem.Schema, []Inference{{Type: "required_section", Field: "id", SourceDirective: `body.section["## ID"]`}}, "hierarchy"); err == nil {
		t.Fatal("expected helper collision error")
	}
	if after := rootStem.Schema["id"]; after.Type != before.Type || after.Prefix != before.Prefix || after.Digits != before.Digits || after.Extract != before.Extract || after.Match == nil {
		t.Fatalf("existing hierarchy field mutated: before=%+v after=%+v", before, after)
	}
}

func TestGenerateFlatSchema_SectionSourceOptionalValidatesCorpus(t *testing.T) {
	records := []*extract.Record{
		makeRecord("## Notes\nA\n"), makeRecord("## Notes\nB\n"),
		makeRecord("## Notes\nC\n"), makeRecord("## Notes\nD\n"),
		makeRecord("# No notes\n"),
	}
	opts := DefaultInferOptions()
	opts.IncludeStructural = false
	stem, err := GenerateFlatSchema(context.Background(), ".", records, opts)
	if err != nil {
		t.Fatalf("GenerateFlatSchema: %v", err)
	}
	notes, ok := stem.Schema["notes"]
	if !ok {
		t.Fatalf("missing notes field in %#v", stem.Schema)
	}
	if notes.Required || notes.Type != "string" || notes.Extract != `body.section["## Notes"]` {
		t.Fatalf("notes field = %+v, want optional string source-backed field", notes)
	}
	for _, rec := range records {
		if errs := rules.Validate(context.Background(), rec, stem); len(errs) != 0 {
			t.Fatalf("generated schema rejected %s: %+v", rec.Path, errs)
		}
	}
}

func TestGenerateHierarchicalSchema_SectionSourceUsesCanonicalDirective(t *testing.T) {
	tmpDir := t.TempDir()
	var records []*extract.Record
	for _, path := range []string{
		"E01-epic/F01-feature/T001-one.md", "E01-epic/F01-feature/T002-two.md",
		"E02-epic/F02-feature/T001-three.md", "E02-epic/F02-feature/T002-four.md",
	} {
		records = append(records, makeSectionRecord(extract.Section{Level: 3, Heading: "Findings"}))
		records[len(records)-1].Path = filepath.Join(tmpDir, path)
	}
	opts := DefaultInferOptions()
	opts.IncludeStructural = false
	stemMap, err := GenerateHierarchicalSchema(context.Background(), tmpDir, records, opts)
	if err != nil {
		t.Fatalf("GenerateHierarchicalSchema: %v", err)
	}
	findings := stemMap["."].Schema["findings"]
	if findings.Type != "string" || findings.Extract != `body.section["### Findings"]` || !findings.Required {
		t.Fatalf("findings field = %+v, want required canonical section source", findings)
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
		{"## Test", "test"},
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
