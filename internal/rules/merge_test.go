package rules

import (
	"testing"
)

func stemEntry(path string, stem *StemFile) StemEntry {
	stem.Path = path
	return StemEntry{Path: path, Stem: stem}
}

func TestMergeStemFiles_Empty(t *testing.T) {
	result := MergeStemFiles(nil)
	if result.Version != 0 {
		t.Errorf("version = %d, want 0", result.Version)
	}
	if len(result.Schema) != 0 {
		t.Errorf("schema len = %d, want 0", len(result.Schema))
	}
}

func TestMergeStemFiles_SingleEntry(t *testing.T) {
	entries := []StemEntry{
		stemEntry("root/.stem", &StemFile{
			Version: 1,
			Scope:   Scope{Match: "*.md"},
			Schema: map[string]SchemaField{
				"title": {Type: "string", Required: true},
			},
		}),
	}

	result := MergeStemFiles(entries)
	if result.Version != 1 {
		t.Errorf("version = %d, want 1", result.Version)
	}
	if result.Scope.Match != "*.md" {
		t.Errorf("scope.match = %q, want *.md", result.Scope.Match)
	}
	if len(result.Schema) != 1 {
		t.Fatalf("schema len = %d, want 1", len(result.Schema))
	}
	if result.Schema["title"].Source != "root/.stem" {
		t.Errorf("title.source = %q, want root/.stem", result.Schema["title"].Source)
	}
}

func TestMergeStemFiles_MapMerge(t *testing.T) {
	// I5 §1.3: parent {a:1, b:2} + child {b:3, c:4} = {a:1, b:3, c:4}
	// NOTE: Child redefines "b" with a different type (string → enum).
	// This demonstrates v2 merge behavior; under monotonic narrowing,
	// this would be allowed because string→enum is valid narrowing.
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Schema: map[string]SchemaField{
				"a": {Type: "string"},
				"b": {Type: "string"},
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"b": {Type: "enum", Values: []string{"x", "y"}},
				"c": {Type: "string"},
			},
		}),
	}

	result := MergeStemFiles(entries)
	if len(result.Schema) != 3 {
		t.Fatalf("schema len = %d, want 3", len(result.Schema))
	}
	// a inherited from parent.
	if result.Schema["a"].Source != "parent/.stem" {
		t.Errorf("a.source = %q, want parent/.stem", result.Schema["a"].Source)
	}
	// b overridden by child.
	if result.Schema["b"].Type != "enum" || result.Schema["b"].Source != "child/.stem" {
		t.Errorf("b = %+v, want type=enum source=child/.stem", result.Schema["b"])
	}
	// c added by child.
	if result.Schema["c"].Source != "child/.stem" {
		t.Errorf("c.source = %q, want child/.stem", result.Schema["c"].Source)
	}
}

func TestMergeSchemaFields_InheritsOmittedSource(t *testing.T) {
	entries := []StemEntry{
		stemEntry("root/.stem", &StemFile{
			Version: 1,
			Schema: map[string]SchemaField{
				"summary": {Type: "string", Extract: `body.section["## Summary"]`},
			},
		}),
		stemEntry("middle/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"summary": {Type: "string", Required: true},
			},
		}),
		stemEntry("leaf/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"summary": {Type: "enum", Values: []string{"short"}},
			},
		}),
	}

	got := MergeStemFiles(entries).Schema["summary"]
	if got.Extract != `body.section["## Summary"]` {
		t.Fatalf("summary.extract = %q, want inherited root source", got.Extract)
	}
	if got.Source != "leaf/.stem" {
		t.Fatalf("summary.source = %q, want leaf provenance", got.Source)
	}
	if got.Type != "enum" {
		t.Fatalf("summary.type = %q, want leaf override", got.Type)
	}
}

func TestMergeStemFiles_ValidateAccumulates(t *testing.T) {
	// Validate rules accumulate from parent to child.
	parent := &StemFile{
		Version: 1,
		Validate: []ValidationRule{
			{Rule: "non_empty", Field: "title", Source: "parent/.stem"},
		},
	}
	child := &StemFile{
		Validate: []ValidationRule{
			{Rule: "requires", Source: "child/.stem"},
		},
	}

	entries := []StemEntry{
		stemEntry("parent/.stem", parent),
		stemEntry("child/.stem", child),
	}

	result := MergeStemFiles(entries)
	if len(result.Validate) != 2 {
		t.Fatalf("validate len = %d, want 2", len(result.Validate))
	}
	if result.Validate[0].Rule != "non_empty" {
		t.Errorf("validate[0].rule = %q, want non_empty (from parent)", result.Validate[0].Rule)
	}
	if result.Validate[1].Rule != "requires" {
		t.Errorf("validate[1].rule = %q, want requires (from child)", result.Validate[1].Rule)
	}
}

func TestMergeStemFiles_NullRemovesKey(t *testing.T) {
	// I5 §1.3: parent {a:1, b:2} + child {b:null} = {a:1}
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Derive: map[string]any{
				"a": map[string]any{"from": "title"},
				"b": map[string]any{"from": "status"},
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Derive: map[string]any{
				"b": nil,
			},
		}),
	}

	result := MergeStemFiles(entries)
	if len(result.Derive) != 1 {
		t.Fatalf("derive len = %d, want 1", len(result.Derive))
	}
	if _, exists := result.Derive["a"]; !exists {
		t.Error("derive[a] should exist")
	}
	if _, exists := result.Derive["b"]; exists {
		t.Error("derive[b] should have been removed by null")
	}
}

func TestMergeStemFiles_ThreeLevels(t *testing.T) {
	// Grandparent → Parent → Child merge.
	// NOTE: Parent overrides Estado's enum values from gp. Under monotonic narrowing
	// semantics, this would require either:
	// 1. Parent values are a subset of gp values (narrowing), or
	// 2. Parent does not redefine Estado (inheritance).
	// This test documents v2 legacy behavior; monotonic validation is checked
	// separately in stemhealth_test.go (monotonic-violations).
	entries := []StemEntry{
		stemEntry("gp/.stem", &StemFile{
			Version: 1,
			Scope:   Scope{Match: "*.md"},
			Schema: map[string]SchemaField{
				"Fecha":  {Type: "string", Required: true},
				"Estado": {Type: "enum", Values: []string{"Active", "Completed"}},
			},
		}),
		stemEntry("parent/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"Estado":  {Type: "enum", Values: []string{"Pending", "Completed"}},
				"Cliente": {Type: "string", Default: "Platform Owner"},
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"Tipo": {Type: "enum", Values: []string{"servicio-docker"}},
			},
		}),
	}

	result := MergeStemFiles(entries)

	// Fecha from gp, Estado from parent (overrides gp), Cliente from parent, Tipo from child.
	if len(result.Schema) != 4 {
		t.Fatalf("schema len = %d, want 4", len(result.Schema))
	}
	if result.Schema["Fecha"].Source != "gp/.stem" {
		t.Errorf("Fecha.source = %q, want gp/.stem", result.Schema["Fecha"].Source)
	}
	if result.Schema["Estado"].Source != "parent/.stem" {
		t.Errorf("Estado.source = %q, want parent/.stem", result.Schema["Estado"].Source)
	}
	if len(result.Schema["Estado"].Values) != 2 || result.Schema["Estado"].Values[0] != "Pending" {
		t.Errorf("Estado.values = %v, want [Pending, Completed]", result.Schema["Estado"].Values)
	}
	if result.Schema["Cliente"].Source != "parent/.stem" {
		t.Errorf("Cliente.source = %q, want parent/.stem", result.Schema["Cliente"].Source)
	}
	if result.Schema["Tipo"].Source != "child/.stem" {
		t.Errorf("Tipo.source = %q, want child/.stem", result.Schema["Tipo"].Source)
	}
}

func TestMergeStemFiles_FourLevelChain(t *testing.T) {
	// Root → Epic → Feature → Story: each adds a field, result has all 4.
	entries := []StemEntry{
		stemEntry("root/.stem", &StemFile{
			Version: 1,
			Schema: map[string]SchemaField{
				"estado": {Type: "enum", Values: []string{"Pending", "Completed"}},
			},
		}),
		stemEntry("epic/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"tipo": {Type: "string"},
			},
		}),
		stemEntry("feature/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"cliente": {Type: "string", Default: "Platform Owner"},
			},
		}),
		stemEntry("story/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"ejecutable_en": {Type: "string"},
			},
		}),
	}

	result := MergeStemFiles(entries)
	if len(result.Schema) != 4 {
		t.Fatalf("schema len = %d, want 4", len(result.Schema))
	}
	for _, field := range []string{"estado", "tipo", "cliente", "ejecutable_en"} {
		if _, ok := result.Schema[field]; !ok {
			t.Errorf("missing field %q in merged schema", field)
		}
	}
}

func TestMergeStemFiles_ChildOverridesRequired(t *testing.T) {
	// NOTE: v2 merge semantics allow child to loosen required to false.
	// Under monotonic narrowing semantics, this would be a violation.
	// This test documents v2 legacy behavior; monotonic validation
	// is checked separately in stemhealth_test.go (monotonic-violations).
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Schema: map[string]SchemaField{
				"campo": {Type: "string", Required: true},
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"campo": {Type: "string", Required: false},
			},
		}),
	}

	result := MergeStemFiles(entries)
	if result.Schema["campo"].Required {
		t.Error("campo.required = true, want false (child override)")
	}
}

func TestMerge_NullRemovesSchemaField(t *testing.T) {
	// parent defines campo, child nullifies it via merge.
	// This test verifies the Derive null-removal path.
	// Schema field null removal is tested separately in TestMergeStemFiles_SchemaFieldNullRemoval.
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Derive: map[string]any{
				"computed_field": "expr1",
				"removable":      "expr2",
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Derive: map[string]any{
				"removable": nil,
			},
		}),
	}

	result := MergeStemFiles(entries)
	if _, ok := result.Derive["computed_field"]; !ok {
		t.Error("computed_field should exist")
	}
	if _, ok := result.Derive["removable"]; ok {
		t.Error("removable should be removed by null")
	}
}

func TestMergeStemFiles_DeepMapMerge(t *testing.T) {
	// Nested maps merge recursively.
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Aggregate: map[string]any{
				"visibility": map[string]any{
					"derive": map[string]any{
						"when": "published",
						"then": "public",
					},
				},
				"other": "keep",
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Aggregate: map[string]any{
				"visibility": map[string]any{
					"derive": map[string]any{
						"then": "private",
					},
				},
			},
		}),
	}

	result := MergeStemFiles(entries)

	vis, ok := result.Aggregate["visibility"].(map[string]any)
	if !ok {
		t.Fatalf("aggregate.visibility is not a map")
	}
	derive, ok := vis["derive"].(map[string]any)
	if !ok {
		t.Fatalf("aggregate.visibility.derive is not a map")
	}
	// "when" inherited from parent, "then" overridden by child.
	if derive["when"] != "published" {
		t.Errorf("derive.when = %v, want published", derive["when"])
	}
	if derive["then"] != "private" {
		t.Errorf("derive.then = %v, want private", derive["then"])
	}
	// "other" inherited.
	if result.Aggregate["other"] != "keep" {
		t.Errorf("aggregate.other = %v, want keep", result.Aggregate["other"])
	}
}

func TestMergeStemFiles_ScalarReplace(t *testing.T) {
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Scope:   Scope{Match: "*.md"},
		}),
		stemEntry("child/.stem", &StemFile{
			Scope: Scope{Match: "*.yaml"},
		}),
	}

	result := MergeStemFiles(entries)
	if result.Scope.Match != "*.yaml" {
		t.Errorf("scope.match = %q, want *.yaml", result.Scope.Match)
	}
}

func TestMergeStemFiles_StructuralPreserved(t *testing.T) {
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Structural: StructuralRules{
				Subdirs: SubdirRules{RequireIndex: "README.md", MinChildren: 2, Severity: "warn"},
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Schema: map[string]SchemaField{"tipo": {Type: "string"}},
		}),
	}

	result := MergeStemFiles(entries)
	if result.Structural.Subdirs.RequireIndex != "README.md" {
		t.Errorf("require_index = %q, want README.md", result.Structural.Subdirs.RequireIndex)
	}
	if result.Structural.Subdirs.MinChildren != 2 {
		t.Errorf("min_children = %d, want 2", result.Structural.Subdirs.MinChildren)
	}
}

func TestMergeStemFiles_StructuralOverridden(t *testing.T) {
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Structural: StructuralRules{
				Subdirs: SubdirRules{RequireIndex: "README.md", MinChildren: 2},
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Structural: StructuralRules{
				Subdirs: SubdirRules{RequireIndex: "index.md", MinChildren: 1},
			},
		}),
	}

	result := MergeStemFiles(entries)
	if result.Structural.Subdirs.RequireIndex != "index.md" {
		t.Errorf("require_index = %q, want index.md (child override)", result.Structural.Subdirs.RequireIndex)
	}
	if result.Structural.Subdirs.MinChildren != 1 {
		t.Errorf("min_children = %d, want 1 (child override)", result.Structural.Subdirs.MinChildren)
	}
}

func TestMergeStemFiles_ExcludesChildOverridesParent(t *testing.T) {
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Schema: map[string]SchemaField{
				"estado": {Type: "enum", Required: true, Excludes: &ExcludeRule{Match: "*/README.md"}},
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"estado": {Type: "enum", Required: true},
			},
		}),
	}

	result := MergeStemFiles(entries)
	if result.Schema["estado"].Excludes != nil {
		t.Error("expected child to clear parent's excludes")
	}
}

func TestMergeStemFiles_ExcludesChildSetsNew(t *testing.T) {
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Schema: map[string]SchemaField{
				"estado": {Type: "enum", Required: true},
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"estado": {Type: "enum", Required: true, Excludes: &ExcludeRule{Match: "*/index.md"}},
			},
		}),
	}

	result := MergeStemFiles(entries)
	if result.Schema["estado"].Excludes == nil {
		t.Fatal("expected child to set excludes")
	}
	if result.Schema["estado"].Excludes.Match != "*/index.md" {
		t.Errorf("excludes.match = %q, want */index.md", result.Schema["estado"].Excludes.Match)
	}
}

func TestMergeStemFiles_ChildValidateNilDoesNotReplaceParent(t *testing.T) {
	// If child has no validate section (nil), parent's validate survives.
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 1,
			Validate: []ValidationRule{
				{Rule: "non_empty", Field: "title"},
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"extra": {Type: "string"},
			},
		}),
	}

	result := MergeStemFiles(entries)
	if len(result.Validate) != 1 {
		t.Errorf("validate len = %d, want 1 (inherited from parent)", len(result.Validate))
	}
}

func TestMergeLinkSchema_StylesAndChecksChildReplace(t *testing.T) {
	parent := LinkSchema{Styles: []string{"wikilink"}, Checks: &LinkChecks{Resolve: boolPtr(true)}}
	child := LinkSchema{Styles: []string{"markdown"}}
	got := mergeLinkSchema(parent, child)
	if len(got.Styles) != 1 || got.Styles[0] != "markdown" {
		t.Errorf("Styles = %v, want child's [markdown]", got.Styles)
	}
	if got.Checks == nil || got.Checks.Resolve == nil || !*got.Checks.Resolve {
		t.Errorf("Checks = %+v, want inherited from parent", got.Checks)
	}
	// The child replaces the checks block, except that an undeclared resolve
	// inherits: it is tri-state and defaults to on, so wholesale replacement
	// would let a child declaring only encoding switch a parent's opt-out back
	// on without saying so.
	child2 := LinkSchema{Checks: &LinkChecks{Encoding: true}}
	got2 := mergeLinkSchema(parent, child2)
	if got2.Checks == nil || !got2.Checks.Encoding {
		t.Errorf("Checks = %+v, want the child's encoding", got2.Checks)
	}
	if got2.Checks.Resolve == nil || !*got2.Checks.Resolve {
		t.Errorf("Checks = %+v, want the parent's resolve inherited", got2.Checks)
	}
}

// resolve is tri-state and defaults to on, so a child declaring only another
// check must not silently switch a parent's opt-out back on.
func TestMergeLinkSchema_ChildDoesNotResetParentResolveOptOut(t *testing.T) {
	off := false
	parent := LinkSchema{Checks: &LinkChecks{Resolve: &off}}
	child := LinkSchema{Checks: &LinkChecks{Encoding: true}}

	got := mergeLinkSchema(parent, child)
	if got.ShouldResolve() {
		t.Errorf("child declaring only encoding re-enabled the parent's resolve opt-out: %+v", got.Checks)
	}
	if !got.Checks.Encoding {
		t.Errorf("child's own encoding declaration was lost: %+v", got.Checks)
	}
}

// A child that explicitly turns resolve back on still wins.
func TestMergeLinkSchema_ChildCanReenableResolve(t *testing.T) {
	off, on := false, true
	parent := LinkSchema{Checks: &LinkChecks{Resolve: &off}}
	child := LinkSchema{Checks: &LinkChecks{Resolve: &on}}

	if !mergeLinkSchema(parent, child).ShouldResolve() {
		t.Error("child's explicit resolve: true must win over the parent's opt-out")
	}
}

// TestMergeStemFiles_SchemaFieldNullRemoval tests that schema: {field: null} removes the field.
func TestMergeStemFiles_SchemaFieldNullRemoval(t *testing.T) {
	// Parent defines 'removed' field, child nullifies it with NullField flag.
	// The field should be deleted from the effective schema.
	entries := []StemEntry{
		stemEntry("parent/.stem", &StemFile{
			Version: 2,
			Schema: map[string]SchemaField{
				"titulo":  {Type: "string", Required: true},
				"removed": {Type: "string", Required: true},
			},
		}),
		stemEntry("child/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"removed": {declaration: schemaFieldDeclarationMetadata{NullField: true}},
			},
		}),
	}

	result := MergeStemFiles(entries)
	if len(result.Schema) != 1 {
		t.Fatalf("schema len = %d, want 1 (removed field should be deleted)", len(result.Schema))
	}
	if _, exists := result.Schema["removed"]; exists {
		t.Error("removed field should have been deleted by null")
	}
	if _, exists := result.Schema["titulo"]; !exists {
		t.Error("titulo field should still exist")
	}
}

// TestMergeStemFiles_SchemaFieldNullRemovalWithYAMLParsing verifies end-to-end YAML parsing and null removal.
func TestMergeStemFiles_SchemaFieldNullRemovalWithYAMLParsing(t *testing.T) {
	// This test verifies the documented null-removal syntax works end-to-end.
	// Parent .stem file
	parentYAML := `version: 2
root: true
scope:
  match: "*.md"
schema:
  titulo:
    type: string
    required: true
  removed:
    type: string
    required: true
`

	// Child .stem file with null removal
	childYAML := `version: 2
scope:
  match: "*.md"
schema:
  removed: null
`

	parentStem, err := ParseStem("parent/.stem", []byte(parentYAML))
	if err != nil {
		t.Fatalf("parsing parent stem: %v", err)
	}

	childStem, err := ParseStem("child/.stem", []byte(childYAML))
	if err != nil {
		t.Fatalf("parsing child stem: %v", err)
	}

	// Merge
	result := MergeStemFiles([]StemEntry{
		{Path: "parent/.stem", Stem: parentStem},
		{Path: "child/.stem", Stem: childStem},
	})

	// Verify: only titulo should remain, removed should be gone.
	if len(result.Schema) != 1 {
		t.Fatalf("schema len = %d, want 1 (removed field should be deleted)", len(result.Schema))
	}
	if _, exists := result.Schema["removed"]; exists {
		t.Error("removed field should have been deleted by null")
	}
	if _, exists := result.Schema["titulo"]; !exists {
		t.Error("titulo field should still exist")
	}
}
