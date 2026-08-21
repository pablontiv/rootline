package rules

import (
	"testing"
)

func TestFilterSchemaByMatch(t *testing.T) {
	tests := []struct {
		name       string
		schema     map[string]*SchemaField
		recordPath string
		wantFields []string
		wantAbsent []string
	}{
		{
			name: "field without match applies everywhere",
			schema: map[string]*SchemaField{
				"estado": {Type: "enum", Values: []string{"Pending", "Done"}},
			},
			recordPath: "docs/epics/E01-name/F01-name/T001-task.md",
			wantFields: []string{"estado"},
		},
		{
			name: "field with string match applies to matching component",
			schema: map[string]*SchemaField{
				"tipo": {Type: "enum", Match: &FieldMatch{Patterns: []string{"T*"}}},
			},
			recordPath: "docs/epics/E01-name/F01-name/S001-name/T001-task.md",
			wantFields: []string{"tipo"},
		},
		{
			name: "field with string match excluded for non-matching path",
			schema: map[string]*SchemaField{
				"tipo": {Type: "enum", Match: &FieldMatch{Patterns: []string{"T*"}}},
			},
			recordPath: "docs/epics/E01-name/README.md",
			wantAbsent: []string{"tipo"},
		},
		{
			name: "field with list match applies when any pattern matches",
			schema: map[string]*SchemaField{
				"tipo": {Type: "enum", Match: &FieldMatch{Patterns: []string{"F*", "T*"}}},
			},
			recordPath: "docs/epics/E01-name/F01-name/README.md",
			wantFields: []string{"tipo"},
		},
		{
			name: "field with list match excluded when no pattern matches",
			schema: map[string]*SchemaField{
				"tipo": {Type: "enum", Match: &FieldMatch{Patterns: []string{"F*", "T*"}}},
			},
			recordPath: "docs/epics/E01-name/README.md",
			wantAbsent: []string{"tipo"},
		},
		{
			name: "map form match applies config for matching pattern",
			schema: map[string]*SchemaField{
				"id": {
					Type: "sequence",
					Match: &FieldMatch{
						Configs: map[string]any{
							"E*": map[string]any{"prefix": "E", "digits": 2},
							"T*": map[string]any{"prefix": "T", "digits": 3},
						},
					},
				},
			},
			recordPath: "docs/epics/E01-name/F01-name/S001-name/T001-task.md",
			wantFields: []string{"id"},
		},
		{
			name: "map form match resolves correct config per pattern",
			schema: map[string]*SchemaField{
				"id": {
					Type: "sequence",
					Match: &FieldMatch{
						Configs: map[string]any{
							"E*": map[string]any{"prefix": "E", "digits": 2},
							"T*": map[string]any{"prefix": "T", "digits": 3},
						},
					},
				},
			},
			recordPath: "docs/epics/E01-name/README.md",
			wantFields: []string{"id"},
		},
		{
			name: "map form match excluded when no pattern matches",
			schema: map[string]*SchemaField{
				"id": {
					Type: "sequence",
					Match: &FieldMatch{
						Configs: map[string]any{
							"E*": map[string]any{"prefix": "E", "digits": 2},
							"T*": map[string]any{"prefix": "T", "digits": 3},
						},
					},
				},
			},
			recordPath: "docs/other/data.md",
			wantAbsent: []string{"id"},
		},
		{
			name: "mixed schema with and without match",
			schema: map[string]*SchemaField{
				"estado": {Type: "enum"},
				"tipo":   {Type: "enum", Match: &FieldMatch{Patterns: []string{"F*", "T*"}}},
				"id": {
					Type: "sequence",
					Match: &FieldMatch{
						Configs: map[string]any{
							"E*": map[string]any{"prefix": "E", "digits": 2},
							"F*": map[string]any{"prefix": "F", "digits": 2},
						},
					},
				},
			},
			recordPath: "docs/epics/E01-name/F01-name/README.md",
			wantFields: []string{"estado", "tipo", "id"},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := mustFilterSchemaByMatch(t, tt.schema, tt.recordPath)

			for _, field := range tt.wantFields {
				if _, ok := result[field]; !ok {
					t.Errorf("expected field %q in result, got %v", field, fieldNames(result))
				}
			}
			for _, field := range tt.wantAbsent {
				if _, ok := result[field]; ok {
					t.Errorf("expected field %q absent from result, got %v", field, fieldNames(result))
				}
			}
		})
	}
}

func TestFilterSchemaByMatch_MapFormConfig(t *testing.T) {
	schema := map[string]*SchemaField{
		"id": {
			Type: "sequence",
			Match: &FieldMatch{
				Configs: map[string]any{
					"E*": map[string]any{"prefix": "E", "digits": 2},
					"T*": map[string]any{"prefix": "T", "digits": 3},
				},
			},
		},
	}

	t.Run("E-level gets E prefix and 2 digits", func(t *testing.T) {
		result := mustFilterSchemaByMatch(t, schema, "docs/epics/E01-name/README.md")
		id := result["id"]
		if id == nil {
			t.Fatal("expected id field in result")
		}
		if id.Prefix != "E" {
			t.Errorf("expected prefix E, got %q", id.Prefix)
		}
		if id.Digits != 2 {
			t.Errorf("expected digits 2, got %d", id.Digits)
		}
	})

	t.Run("T-level gets T prefix and 3 digits", func(t *testing.T) {
		result := mustFilterSchemaByMatch(t, schema, "docs/epics/E01-name/F01-feat/S001-story/T001-task.md")
		id := result["id"]
		if id == nil {
			t.Fatal("expected id field in result")
		}
		if id.Prefix != "T" {
			t.Errorf("expected prefix T, got %q", id.Prefix)
		}
		if id.Digits != 3 {
			t.Errorf("expected digits 3, got %d", id.Digits)
		}
	})
}

func TestFilterSchemaByMatch_RequiredMatch(t *testing.T) {
	schema := map[string]*SchemaField{
		"tipo": {
			Type:          "enum",
			Values:        []string{"modulo-sistema", "test"},
			Match:         &FieldMatch{Patterns: []string{"F*", "T*"}},
			RequiredMatch: &FieldMatch{Patterns: []string{"T*"}},
			Required:      true,
		},
	}

	t.Run("required true at T-level", func(t *testing.T) {
		result := mustFilterSchemaByMatch(t, schema, "docs/epics/E01/F01/S001/T001-task.md")
		tipo := result["tipo"]
		if tipo == nil {
			t.Fatal("expected tipo field")
		}
		if !tipo.Required {
			t.Error("expected required true at T-level")
		}
	})

	t.Run("required false at F-level", func(t *testing.T) {
		result := mustFilterSchemaByMatch(t, schema, "docs/epics/E01/F01-feat/README.md")
		tipo := result["tipo"]
		if tipo == nil {
			t.Fatal("expected tipo field")
		}
		if tipo.Required {
			t.Error("expected required false at F-level (RequiredMatch restricts to T*)")
		}
	})

	t.Run("not included at E-level", func(t *testing.T) {
		result := mustFilterSchemaByMatch(t, schema, "docs/epics/E01/README.md")
		if _, ok := result["tipo"]; ok {
			t.Error("expected tipo absent at E-level (Match restricts to F* and T*)")
		}
	})
}

func TestFilterSchemaByMatch_RequiredMatchWithoutMatch(t *testing.T) {
	schema := map[string]*SchemaField{
		"estado": {
			Type:          "enum",
			RequiredMatch: &FieldMatch{Patterns: []string{"T*"}},
			Required:      true,
		},
	}

	t.Run("required true at T-level", func(t *testing.T) {
		result := mustFilterSchemaByMatch(t, schema, "docs/epics/E01/F01/S001/T001-task.md")
		if !result["estado"].Required {
			t.Error("expected required true at T-level")
		}
	})

	t.Run("required false at E-level", func(t *testing.T) {
		result := mustFilterSchemaByMatch(t, schema, "docs/epics/E01/README.md")
		if result["estado"].Required {
			t.Error("expected required false at E-level")
		}
	})
}

func mustFilterSchemaByMatch(t *testing.T, schema map[string]*SchemaField, recordPath string) map[string]*SchemaField {
	t.Helper()
	filtered, err := FilterSchemaByMatch(schema, recordPath)
	if err != nil {
		t.Fatal(err)
	}
	return filtered
}

func fieldNames(schema map[string]*SchemaField) []string {
	names := make([]string, 0, len(schema))
	for name := range schema {
		names = append(names, name)
	}
	return names
}
