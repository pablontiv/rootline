package migrate

import (
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestConvertLevelsToMatch_Basic(t *testing.T) {
	content := []byte(`
version: 1
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, In Progress, Completado]
levels:
  epic:
    match: "E??-*"
    children: [feature]
    schema:
      id:
        type: sequence
        prefix: E
        digits: 2
  feature:
    match: "F??-*"
    children: [story]
    schema:
      id:
        type: sequence
        prefix: F
        digits: 2
      tipo:
        type: enum
        values: [servicio-docker, modulo-sistema, documentation]
  story:
    match: "S???-*"
    children: [task]
    schema:
      id:
        type: sequence
        prefix: S
        digits: 3
  task:
    match: "T???-*"
    children: []
    schema:
      id:
        type: sequence
        prefix: T
        digits: 3
      tipo:
        type: enum
        required: true
        values: [servicio-docker, modulo-sistema, documentation]
`)

	result, err := ConvertLevelsToMatch(content, "test/.stem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Version != 2 {
		t.Errorf("version = %d, want 2", result.Version)
	}

	// Root field: estado (no match)
	estado, ok := result.Schema["estado"]
	if !ok {
		t.Fatal("expected 'estado' in schema")
	}
	if estado.Match != nil {
		t.Errorf("estado.Match should be nil (root field), got %+v", estado.Match)
	}

	// Sequence id: map-form match
	id, ok := result.Schema["id"]
	if !ok {
		t.Fatal("expected 'id' in schema")
	}
	if id.Type != "sequence" {
		t.Errorf("id.type = %q, want sequence", id.Type)
	}
	if id.Match == nil || id.Match.Configs == nil {
		t.Fatal("expected id.Match.Configs non-nil")
	}
	for _, pattern := range []string{"E*", "F*", "S*", "T*"} {
		if _, ok := id.Match.Configs[pattern]; !ok {
			t.Errorf("expected id config for pattern %q", pattern)
		}
	}

	// tipo: present at feature and task levels → match: ["F*", "T*"]
	tipo, ok := result.Schema["tipo"]
	if !ok {
		t.Fatal("expected 'tipo' in schema")
	}
	if tipo.Match == nil {
		t.Fatal("expected tipo.Match non-nil")
	}
	if len(tipo.Match.Patterns) != 2 {
		t.Fatalf("tipo.Match.Patterns len = %d, want 2", len(tipo.Match.Patterns))
	}
	patterns := make(map[string]bool)
	for _, p := range tipo.Match.Patterns {
		patterns[p] = true
	}
	if !patterns["F*"] || !patterns["T*"] {
		t.Errorf("tipo.Match.Patterns = %v, want [F* T*]", tipo.Match.Patterns)
	}
	if !tipo.Required {
		t.Error("tipo.Required should be true (required at task level)")
	}
}

func TestConvertLevelsToMatch_NoLevels(t *testing.T) {
	content := []byte(`version: 1
schema:
  estado:
    type: enum
`)
	_, err := ConvertLevelsToMatch(content, "test/.stem")
	if err == nil {
		t.Fatal("expected error for stem with no levels")
	}
}

func TestConvertLevelsToMatch_PreservesRootFields(t *testing.T) {
	content := []byte(`
version: 1
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Done]
derive:
  slug: "slugify(titulo)"
aggregate:
  estado: 'all(descendants, {.estado == "Done"}) ? "Done" : estado'
levels:
  epic:
    match: "E*"
    schema:
      id:
        type: sequence
        prefix: E
        digits: 2
  feature:
    match: "F*"
    schema:
      id:
        type: sequence
        prefix: F
        digits: 2
`)

	result, err := ConvertLevelsToMatch(content, "test/.stem")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if _, ok := result.Schema["estado"]; !ok {
		t.Error("expected 'estado' preserved in schema")
	}
	if result.Derive == nil || len(result.Derive) != 1 {
		t.Error("expected derive preserved")
	}
	if result.Aggregate == nil || len(result.Aggregate) != 1 {
		t.Error("expected aggregate preserved")
	}
}

func TestNormalizeMatchPattern(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"E??-*", "E*"},
		{"F??-*", "F*"},
		{"S???-*", "S*"},
		{"T*", "T*"},
		{"E*", "E*"},
		{"*.md", "*.md"},
	}
	for _, tt := range tests {
		got := normalizeMatchPattern(tt.input)
		if got != tt.want {
			t.Errorf("normalizeMatchPattern(%q) = %q, want %q", tt.input, got, tt.want)
		}
	}
}

func TestMarshalStemV2(t *testing.T) {
	stem := &rules.StemFile{
		Version: 2,
		Scope:   rules.Scope{Match: "*.md"},
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Done"}},
		},
	}

	data, err := MarshalStemV2(stem)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	parsed, err := rules.ParseStem("test/.stem", data)
	if err != nil {
		t.Fatalf("round-trip parse error: %v", err)
	}
	if parsed.Version != 2 {
		t.Errorf("version = %d, want 2", parsed.Version)
	}
	if _, ok := parsed.Schema["estado"]; !ok {
		t.Error("expected 'estado' in parsed schema")
	}
}
