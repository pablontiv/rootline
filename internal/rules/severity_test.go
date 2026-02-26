package rules

import (
	"testing"
)

func TestParseSeverityDefault(t *testing.T) {
	content := []byte(`
version: 1
schema:
  estado:
    type: string
    required: true
`)
	stem, err := ParseStem("test.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	field := stem.Schema["estado"]
	if field.Severity != "error" {
		t.Errorf("expected default severity 'error', got %q", field.Severity)
	}
}

func TestParseSeverityWarn(t *testing.T) {
	content := []byte(`
version: 1
schema:
  estado:
    type: string
    severity: warn
`)
	stem, err := ParseStem("test.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	field := stem.Schema["estado"]
	if field.Severity != "warn" {
		t.Errorf("expected severity 'warn', got %q", field.Severity)
	}
}

func TestParseSeverityOff(t *testing.T) {
	content := []byte(`
version: 1
schema:
  estado:
    type: string
    severity: "off"
`)
	stem, err := ParseStem("test.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	field := stem.Schema["estado"]
	if field.Severity != "off" {
		t.Errorf("expected severity 'off', got %q", field.Severity)
	}
}

func TestParseValidationRuleSeverityDefault(t *testing.T) {
	content := []byte(`
version: 1
validate:
  - field: estado
    rule: required
`)
	stem, err := ParseStem("test.stem", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if stem.Validate[0].Severity != "error" {
		t.Errorf("expected default severity 'error', got %q", stem.Validate[0].Severity)
	}
}

func TestMergeSeverityTightenOK(t *testing.T) {
	// Parent: warn, Child: error → result: error (tighten allowed)
	entries := []StemEntry{
		{Path: "parent/.stem", Stem: &StemFile{
			Schema: map[string]SchemaField{
				"estado": {Type: "string", Severity: "warn"},
			},
		}},
		{Path: "child/.stem", Stem: &StemFile{
			Schema: map[string]SchemaField{
				"estado": {Type: "string", Severity: "error"},
			},
		}},
	}

	result := MergeStemFiles(entries)
	field := result.Schema["estado"]
	if field.Severity != "error" {
		t.Errorf("expected severity 'error' after tighten, got %q", field.Severity)
	}
}

func TestMergeSeverityNoLoosen(t *testing.T) {
	// Parent: error, Child: warn → result: error (loosen NOT allowed)
	entries := []StemEntry{
		{Path: "parent/.stem", Stem: &StemFile{
			Schema: map[string]SchemaField{
				"estado": {Type: "string", Severity: "error"},
			},
		}},
		{Path: "child/.stem", Stem: &StemFile{
			Schema: map[string]SchemaField{
				"estado": {Type: "string", Severity: "warn"},
			},
		}},
	}

	result := MergeStemFiles(entries)
	field := result.Schema["estado"]
	if field.Severity != "error" {
		t.Errorf("expected severity 'error' (no loosen), got %q", field.Severity)
	}
}

func TestMergeSeverityEmptyDefaultsToError(t *testing.T) {
	// Parent: warn, Child: "" (unspecified) → result: error
	// Empty severity defaults to "error", which is stricter than "warn".
	entries := []StemEntry{
		{Path: "parent/.stem", Stem: &StemFile{
			Schema: map[string]SchemaField{
				"tipo": {Type: "enum", Severity: "warn"},
			},
		}},
		{Path: "child/.stem", Stem: &StemFile{
			Schema: map[string]SchemaField{
				"tipo": {Type: "enum", Severity: "", Required: true},
			},
		}},
	}

	result := MergeStemFiles(entries)
	field := result.Schema["tipo"]
	if field.Severity != "error" {
		t.Errorf("expected severity 'error' (empty defaults to error), got %q", field.Severity)
	}
	if !field.Required {
		t.Errorf("expected required=true to be preserved after merge")
	}
}

func TestMergeSeverityOffToError(t *testing.T) {
	// Parent: off, Child: error → result: error
	entries := []StemEntry{
		{Path: "parent/.stem", Stem: &StemFile{
			Schema: map[string]SchemaField{
				"estado": {Type: "string", Severity: "off"},
			},
		}},
		{Path: "child/.stem", Stem: &StemFile{
			Schema: map[string]SchemaField{
				"estado": {Type: "string", Severity: "error"},
			},
		}},
	}

	result := MergeStemFiles(entries)
	field := result.Schema["estado"]
	if field.Severity != "error" {
		t.Errorf("expected severity 'error', got %q", field.Severity)
	}
}
