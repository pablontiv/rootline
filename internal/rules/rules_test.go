package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestParseStem_AllSections(t *testing.T) {
	content := []byte(`
version: 2

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

aggregate:
  estado: 'all(descendants, {.estado == "Completed"}) ? "Completed" : estado'

links:
  allowed: [decision, reference]
`)

	stem, err := ParseStem("docs/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stem.Version != 2 {
		t.Errorf("version = %d, want 2", stem.Version)
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

	// Derive, Aggregate, Links (opaque maps)
	if stem.Derive == nil || len(stem.Derive) != 1 {
		t.Errorf("derive = %v, want 1 entry", stem.Derive)
	}
	if stem.Aggregate == nil || len(stem.Aggregate) != 1 {
		t.Errorf("aggregate = %v, want 1 entry", stem.Aggregate)
	}
	if len(stem.Links.Allowed) != 2 {
		t.Errorf("links.allowed = %v, want [decision, reference]", stem.Links.Allowed)
	}
}

func TestParseStem_MinimalVersionAndScope(t *testing.T) {
	content := []byte(`
version: 2
scope:
  match: "*.md"
`)

	stem, err := ParseStem("minimal/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if stem.Version != 2 {
		t.Errorf("version = %d, want 2", stem.Version)
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
version: 2
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
version: 2
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
	if stem.Version != 2 {
		t.Errorf("version = %d, want 2", stem.Version)
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
version: 2
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

func TestParseStem_ExcludesField(t *testing.T) {
	content := []byte(`version: 2
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
    excludes:
      match: "*/README.md"
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	field := stem.Schema["estado"]
	if field.Excludes == nil {
		t.Fatal("excludes is nil, want non-nil")
	}
	if field.Excludes.Match != "*/README.md" {
		t.Errorf("excludes.match = %q, want */README.md", field.Excludes.Match)
	}
}

func TestParseStem_ExcludesAbsent(t *testing.T) {
	content := []byte(`version: 2
schema:
  estado:
    type: enum
    required: true
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if stem.Schema["estado"].Excludes != nil {
		t.Error("excludes should be nil when not defined")
	}
}

func TestFieldMatch_StringForm(t *testing.T) {
	content := []byte(`version: 2
schema:
  tipo:
    type: enum
    values: [a, b]
    match: "T*"
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	field := stem.Schema["tipo"]
	if field.Match == nil {
		t.Fatal("match is nil, want non-nil")
	}
	if len(field.Match.Patterns) != 1 || field.Match.Patterns[0] != "T*" {
		t.Errorf("match.patterns = %v, want [T*]", field.Match.Patterns)
	}
	if field.Match.Configs != nil {
		t.Errorf("match.configs should be nil for string form, got %v", field.Match.Configs)
	}
}

func TestFieldMatch_ListForm(t *testing.T) {
	content := []byte(`version: 2
schema:
  tipo:
    type: enum
    values: [a, b]
    match: ["F*", "T*"]
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	field := stem.Schema["tipo"]
	if field.Match == nil {
		t.Fatal("match is nil, want non-nil")
	}
	if len(field.Match.Patterns) != 2 {
		t.Fatalf("match.patterns len = %d, want 2", len(field.Match.Patterns))
	}
	if field.Match.Patterns[0] != "F*" || field.Match.Patterns[1] != "T*" {
		t.Errorf("match.patterns = %v, want [F* T*]", field.Match.Patterns)
	}
}

func TestFieldMatch_MapForm(t *testing.T) {
	content := []byte(`version: 2
schema:
  id:
    type: sequence
    match:
      "E*": {prefix: E, digits: 2}
      "T*": {prefix: T, digits: 3}
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	field := stem.Schema["id"]
	if field.Match == nil {
		t.Fatal("match is nil, want non-nil")
	}
	if field.Match.Patterns != nil {
		t.Errorf("match.patterns should be nil for map form, got %v", field.Match.Patterns)
	}
	if len(field.Match.Configs) != 2 {
		t.Fatalf("match.configs len = %d, want 2", len(field.Match.Configs))
	}
	if _, ok := field.Match.Configs["E*"]; !ok {
		t.Error("match.configs missing key E*")
	}
	if _, ok := field.Match.Configs["T*"]; !ok {
		t.Error("match.configs missing key T*")
	}
}

func TestFieldMatch_Absent(t *testing.T) {
	content := []byte(`version: 2
schema:
  estado:
    type: enum
    required: true
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	if stem.Schema["estado"].Match != nil {
		t.Error("match should be nil when not defined")
	}
}

func TestFieldMatch_NilBehavesAsNoFilter(t *testing.T) {
	content := []byte(`version: 2
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("parse error: %v", err)
	}
	field := stem.Schema["estado"]
	if field.Match != nil {
		t.Error("match should be nil for field without match")
	}
	// nil Match means field applies everywhere (no filtering)
	if !field.Required {
		t.Error("required should still be true")
	}
}

func TestParseStemV2(t *testing.T) {
	content := []byte(`version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, "In Progress", Completed]
  id:
    type: sequence
    match:
      "E*": {prefix: E, digits: 2}
      "F*": {prefix: F, digits: 2}
      "S*": {prefix: S, digits: 3}
      "T*": {prefix: T, digits: 3}
  tipo:
    type: enum
    values: [modulo-sistema, test, documentation]
    match: ["F*", "T*"]
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stem.Version != 2 {
		t.Errorf("version = %d, want 2", stem.Version)
	}

	// estado: no match, applies everywhere
	estado := stem.Schema["estado"]
	if estado.Match != nil {
		t.Error("estado.match should be nil")
	}
	if estado.Source != "test/.stem" {
		t.Errorf("estado.source = %q, want test/.stem", estado.Source)
	}

	// id: map-form match
	id := stem.Schema["id"]
	if id.Match == nil || len(id.Match.Configs) != 4 {
		t.Errorf("id.match.configs len = %d, want 4", len(id.Match.Configs))
	}

	// tipo: list-form match
	tipo := stem.Schema["tipo"]
	if tipo.Match == nil || len(tipo.Match.Patterns) != 2 {
		t.Errorf("tipo.match.patterns len = %d, want 2", len(tipo.Match.Patterns))
	}
}

func TestParseStem_RejectsV1(t *testing.T) {
	tests := []struct {
		name    string
		content string
		wantVer int
	}{
		{
			name:    "version 1 rejected",
			content: "version: 1\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: string\n",
			wantVer: 1,
		},
		{
			name:    "version 0 rejected (no version field)",
			content: "scope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n",
			wantVer: 0,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := ParseStem("test/.stem", []byte(tt.content))
			if err == nil {
				t.Fatalf("expected error for version %d, got nil", tt.wantVer)
			}
			wantMsg := "no longer supported"
			if !strings.Contains(err.Error(), wantMsg) {
				t.Errorf("error should contain %q, got: %v", wantMsg, err)
			}
			wantUpgrade := "migrate --to-v2"
			if !strings.Contains(err.Error(), wantUpgrade) {
				t.Errorf("error should contain %q, got: %v", wantUpgrade, err)
			}
		})
	}
}

func TestParseStemV2_V2ParsesCorrectly(t *testing.T) {
	// v2 stems should parse correctly
	content := []byte(`version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Completed]
    required: true
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stem.Version != 2 {
		t.Errorf("version = %d, want 2", stem.Version)
	}
	if !stem.Schema["estado"].Required {
		t.Error("estado.required should be true")
	}
}

func TestParseStem_RequiredMatchObjectForm(t *testing.T) {
	content := []byte(`version: 2
schema:
  tipo:
    type: enum
    values: [modulo-sistema, test]
    match: ["F*", "T*"]
    required:
      match: ["T*"]
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tipo := stem.Schema["tipo"]
	if tipo.RequiredMatch == nil {
		t.Fatal("tipo.RequiredMatch is nil, want non-nil")
	}
	if len(tipo.RequiredMatch.Patterns) != 1 || tipo.RequiredMatch.Patterns[0] != "T*" {
		t.Errorf("tipo.RequiredMatch.Patterns = %v, want [T*]", tipo.RequiredMatch.Patterns)
	}
	if !tipo.Required {
		t.Error("tipo.Required should default to true when RequiredMatch is set")
	}
}

func TestParseStem_RequiredMatchMultiplePatterns(t *testing.T) {
	content := []byte(`version: 2
schema:
  tipo:
    type: enum
    values: [a, b]
    required:
      match: ["F*", "T*"]
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	tipo := stem.Schema["tipo"]
	if tipo.RequiredMatch == nil {
		t.Fatal("tipo.RequiredMatch is nil")
	}
	if len(tipo.RequiredMatch.Patterns) != 2 {
		t.Errorf("RequiredMatch patterns len = %d, want 2", len(tipo.RequiredMatch.Patterns))
	}
}

func TestParseStem_RequiredBoolBackwardCompat(t *testing.T) {
	content := []byte(`version: 2
schema:
  estado:
    type: enum
    required: true
  titulo:
    type: string
    required: false
  desc:
    type: string
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if !stem.Schema["estado"].Required {
		t.Error("estado.Required should be true")
	}
	if stem.Schema["estado"].RequiredMatch != nil {
		t.Error("estado.RequiredMatch should be nil for bool form")
	}

	if stem.Schema["titulo"].Required {
		t.Error("titulo.Required should be false")
	}

	if stem.Schema["desc"].Required {
		t.Error("desc.Required should be false when omitted")
	}
}

func TestParseStem_SectionType(t *testing.T) {
	raw := `
version: 2
schema:
    estado:
        type: enum
        values: [bloqueado, listo]
        required: true
    contexto:
        type: section
        heading: "## Contexto"
        required: true
    investigacion:
        type: section
        heading: "## Investigación"
        required: false
        default: "<!-- TODO -->"
`
	stem, err := ParseStem("test.stem", []byte(raw))
	if err != nil {
		t.Fatalf("ParseStem: %v", err)
	}
	ctx := stem.Schema["contexto"]
	if ctx.Type != "section" {
		t.Errorf("contexto type = %q, want section", ctx.Type)
	}
	if ctx.Heading != "## Contexto" {
		t.Errorf("contexto heading = %q, want '## Contexto'", ctx.Heading)
	}
	if !ctx.Required {
		t.Error("contexto should be required")
	}
	inv := stem.Schema["investigacion"]
	if inv.Heading != "## Investigación" {
		t.Errorf("investigacion heading = %q", inv.Heading)
	}
	if inv.Required {
		t.Error("investigacion should not be required")
	}
	if inv.Default != "<!-- TODO -->" {
		t.Errorf("investigacion default = %q", inv.Default)
	}
}

func TestParseStem_Domain(t *testing.T) {
	content := []byte(`
version: 2
schema:
  estado:
    type: enum
    domain: lifecycle_state
    values: [draft, active, closed]
  titulo:
    type: string
    domain: title
  custom_field:
    type: integer
    domain: acme/sprint_velocity
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	estado := stem.Schema["estado"]
	if estado.Domain != "lifecycle_state" {
		t.Errorf("estado.Domain = %q, want %q", estado.Domain, "lifecycle_state")
	}
	titulo := stem.Schema["titulo"]
	if titulo.Domain != "title" {
		t.Errorf("titulo.Domain = %q, want %q", titulo.Domain, "title")
	}
	custom := stem.Schema["custom_field"]
	if custom.Domain != "acme/sprint_velocity" {
		t.Errorf("custom_field.Domain = %q, want %q", custom.Domain, "acme/sprint_velocity")
	}
}

func TestParseStem_DomainAbsent(t *testing.T) {
	content := []byte(`
version: 2
schema:
  estado:
    type: enum
    values: [draft, active, closed]
`)
	stem, err := ParseStem("test/.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	estado := stem.Schema["estado"]
	if estado.Domain != "" {
		t.Errorf("estado.Domain = %q, want empty", estado.Domain)
	}
}

func TestParseStem_NoStructural(t *testing.T) {
	content := []byte("version: 2\n")
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
