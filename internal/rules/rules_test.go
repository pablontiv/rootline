package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestParseStem_AllSections(t *testing.T) {
	content := []byte(`
version: 1

scope:
  match: "*.md"

schema:
  title:
    type: string
    required: true
  status:
    type: enum
    values: [draft, review, published]
    default: draft

validate:
  - field: title
    rule: non_empty
  - rule: requires
    if: { status: published }
    then: { fields: [owner] }

derive:
  slug:
    from: title
    using: slugify

state:
  visibility:
    derive:
      when: { status: published }
      then: public

links:
  allowed: [decision, reference]
`)

	stem, err := ParseStem("docs/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stem.Version != 1 {
		t.Errorf("version = %d, want 1", stem.Version)
	}
	if stem.Path != "docs/.stem" {
		t.Errorf("path = %q, want %q", stem.Path, "docs/.stem")
	}
	if stem.Scope.Match != "*.md" {
		t.Errorf("scope.match = %q, want %q", stem.Scope.Match, "*.md")
	}

	// Schema
	if len(stem.Schema) != 2 {
		t.Fatalf("schema len = %d, want 2", len(stem.Schema))
	}
	title := stem.Schema["title"]
	if title.Type != "string" || !title.Required {
		t.Errorf("title = %+v, want type=string required=true", title)
	}
	if title.Source != "docs/.stem" {
		t.Errorf("title.source = %q, want %q", title.Source, "docs/.stem")
	}
	status := stem.Schema["status"]
	if status.Type != "enum" || len(status.Values) != 3 || status.Default != "draft" {
		t.Errorf("status = %+v, want type=enum values=[3] default=draft", status)
	}

	// Validate
	if len(stem.Validate) != 2 {
		t.Fatalf("validate len = %d, want 2", len(stem.Validate))
	}
	if stem.Validate[0].Rule != "non_empty" || stem.Validate[0].Field != "title" {
		t.Errorf("validate[0] = %+v, want rule=non_empty field=title", stem.Validate[0])
	}
	if stem.Validate[1].Source != "docs/.stem" {
		t.Errorf("validate[1].source = %q, want %q", stem.Validate[1].Source, "docs/.stem")
	}

	// Derive, State, Links (opaque maps)
	if stem.Derive == nil || len(stem.Derive) != 1 {
		t.Errorf("derive = %v, want 1 entry", stem.Derive)
	}
	if stem.State == nil || len(stem.State) != 1 {
		t.Errorf("state = %v, want 1 entry", stem.State)
	}
	if len(stem.Links.Allowed) != 2 {
		t.Errorf("links.allowed = %v, want [decision, reference]", stem.Links.Allowed)
	}
}

func TestParseStem_MinimalVersionAndScope(t *testing.T) {
	content := []byte(`
version: 1
scope:
  match: "*.md"
`)

	stem, err := ParseStem("minimal/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stem.Version != 1 {
		t.Errorf("version = %d, want 1", stem.Version)
	}
	if stem.Scope.Match != "*.md" {
		t.Errorf("scope.match = %q, want %q", stem.Scope.Match, "*.md")
	}
	if len(stem.Schema) != 0 {
		t.Errorf("schema len = %d, want 0", len(stem.Schema))
	}
	if len(stem.Validate) != 0 {
		t.Errorf("validate len = %d, want 0", len(stem.Validate))
	}
}

func TestParseStem_MalformedYAML(t *testing.T) {
	content := []byte(`
version: 1
schema:
  - this is not a map
  title: broken
`)

	_, err := ParseStem("bad.stem", content)
	if err == nil {
		t.Fatal("expected error for malformed YAML, got nil")
	}
}

func TestParseStem_UnknownSectionsIgnored(t *testing.T) {
	content := []byte(`
version: 1
scope:
  match: "*.md"
future_section:
  key: value
another_unknown: 42
`)

	stem, err := ParseStem("forward/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stem.Version != 1 {
		t.Errorf("version = %d, want 1", stem.Version)
	}
	if stem.Scope.Match != "*.md" {
		t.Errorf("scope.match = %q, want %q", stem.Scope.Match, "*.md")
	}
}

func TestParseStem_RealExamples(t *testing.T) {
	tests := []struct {
		name       string
		file       string
		wantSchema int
		wantRules  int
	}{
		{"docs-root", "testdata/docs.stem", 1, 0},
		{"prd", "testdata/prd.stem", 5, 1},
		{"epics", "testdata/epics.stem", 2, 0},
		{"task-level", "testdata/task.stem", 4, 0},
		{"research", "testdata/research.stem", 2, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			content, err := os.ReadFile(filepath.Join(".", tt.file))
			if err != nil {
				t.Fatalf("reading fixture %s: %v", tt.file, err)
			}

			stem, err := ParseStem(tt.file, content)
			if err != nil {
				t.Fatalf("parsing %s: %v", tt.file, err)
			}

			if len(stem.Schema) != tt.wantSchema {
				t.Errorf("schema len = %d, want %d", len(stem.Schema), tt.wantSchema)
			}
			if len(stem.Validate) != tt.wantRules {
				t.Errorf("validate len = %d, want %d", len(stem.Validate), tt.wantRules)
			}
		})
	}
}

func TestParseStemFile_NotFound(t *testing.T) {
	_, err := ParseStemFile("/nonexistent/.stem")
	if err == nil {
		t.Fatal("expected error for nonexistent file, got nil")
	}
}

func TestParseStem_Structural(t *testing.T) {
	content := []byte(`
version: 1
structural:
  subdirs:
    require_index: README.md
    min_children: 2
    max_children: 10
    severity: warn
`)
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	stem, err := ParseStemFile(stemPath)
	if err != nil {
		t.Fatal(err)
	}

	if stem.Structural.Subdirs.RequireIndex != "README.md" {
		t.Errorf("require_index = %q, want README.md", stem.Structural.Subdirs.RequireIndex)
	}
	if stem.Structural.Subdirs.MinChildren != 2 {
		t.Errorf("min_children = %d, want 2", stem.Structural.Subdirs.MinChildren)
	}
	if stem.Structural.Subdirs.MaxChildren != 10 {
		t.Errorf("max_children = %d, want 10", stem.Structural.Subdirs.MaxChildren)
	}
	if stem.Structural.Subdirs.Severity != "warn" {
		t.Errorf("severity = %q, want warn", stem.Structural.Subdirs.Severity)
	}
}

func TestParseStem_NoStructural(t *testing.T) {
	content := []byte("version: 1\n")
	dir := t.TempDir()
	stemPath := filepath.Join(dir, ".stem")
	if err := os.WriteFile(stemPath, content, 0644); err != nil {
		t.Fatal(err)
	}

	stem, err := ParseStemFile(stemPath)
	if err != nil {
		t.Fatal(err)
	}

	if !stem.Structural.IsEmpty() {
		t.Error("structural should be empty when not defined")
	}
}
