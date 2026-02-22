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

func TestMergeStemFiles_ArrayReplace(t *testing.T) {
	// I5 §1.3: arrays replace entirely.
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
	if len(result.Validate) != 1 {
		t.Fatalf("validate len = %d, want 1", len(result.Validate))
	}
	if result.Validate[0].Rule != "requires" {
		t.Errorf("validate[0].rule = %q, want requires", result.Validate[0].Rule)
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
	entries := []StemEntry{
		stemEntry("gp/.stem", &StemFile{
			Version: 1,
			Scope:   Scope{Match: "*.md"},
			Schema: map[string]SchemaField{
				"Fecha":  {Type: "string", Required: true},
				"Estado": {Type: "enum", Values: []string{"Activa", "Completada"}},
			},
		}),
		stemEntry("parent/.stem", &StemFile{
			Schema: map[string]SchemaField{
				"Estado":  {Type: "enum", Values: []string{"Pending", "Completado"}},
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
		t.Errorf("Estado.values = %v, want [Pending, Completado]", result.Schema["Estado"].Values)
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
				"estado": {Type: "enum", Values: []string{"Pending", "Completado"}},
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
	// Since Schema is map[string]SchemaField, null removal works via Derive/State maps.
	// For Schema, child simply omits the field — it inherits from parent.
	// This test verifies the Derive null-removal path with a Schema-style scenario.
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
			State: map[string]any{
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
			State: map[string]any{
				"visibility": map[string]any{
					"derive": map[string]any{
						"then": "private",
					},
				},
			},
		}),
	}

	result := MergeStemFiles(entries)

	vis, ok := result.State["visibility"].(map[string]any)
	if !ok {
		t.Fatalf("state.visibility is not a map")
	}
	derive, ok := vis["derive"].(map[string]any)
	if !ok {
		t.Fatalf("state.visibility.derive is not a map")
	}
	// "when" inherited from parent, "then" overridden by child.
	if derive["when"] != "published" {
		t.Errorf("derive.when = %v, want published", derive["when"])
	}
	if derive["then"] != "private" {
		t.Errorf("derive.then = %v, want private", derive["then"])
	}
	// "other" inherited.
	if result.State["other"] != "keep" {
		t.Errorf("state.other = %v, want keep", result.State["other"])
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
