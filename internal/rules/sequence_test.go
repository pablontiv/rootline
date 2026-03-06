package rules

import (
	"encoding/json"
	"testing"
)

func TestSchemaField_SequenceDeserialization(t *testing.T) {
	yaml := `
version: 2
schema:
  id:
    type: sequence
    prefix: T
    digits: 3
`
	stem, err := ParseStem("test.stem", []byte(yaml))
	if err != nil {
		t.Fatalf("ParseStem: %v", err)
	}

	field, ok := stem.Schema["id"]
	if !ok {
		t.Fatal("schema field 'id' not found")
	}

	if field.Type != "sequence" {
		t.Errorf("Type = %q, want %q", field.Type, "sequence")
	}
	if field.Prefix != "T" {
		t.Errorf("Prefix = %q, want %q", field.Prefix, "T")
	}
	if field.Digits != 3 {
		t.Errorf("Digits = %d, want %d", field.Digits, 3)
	}
	if field.Next != "" {
		t.Errorf("Next = %q, want empty (yaml:\"-\")", field.Next)
	}
}

func TestSchemaField_SequenceJSON_NextOmitted(t *testing.T) {
	f := SchemaField{Type: "sequence", Prefix: "T", Digits: 3}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(b)
	if contains(s, `"next"`) {
		t.Errorf("JSON should omit next when empty, got: %s", s)
	}
	if !contains(s, `"prefix":"T"`) {
		t.Errorf("JSON should contain prefix, got: %s", s)
	}
	if !contains(s, `"digits":3`) {
		t.Errorf("JSON should contain digits, got: %s", s)
	}
}

func TestSchemaField_SequenceJSON_NextPresent(t *testing.T) {
	f := SchemaField{Type: "sequence", Prefix: "T", Digits: 3, Next: "T004"}
	b, err := json.Marshal(f)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	s := string(b)
	if !contains(s, `"next":"T004"`) {
		t.Errorf("JSON should contain next when set, got: %s", s)
	}
}

func contains(s, sub string) bool {
	return len(s) >= len(sub) && (s == sub || len(s) > 0 && containsStr(s, sub))
}

func containsStr(s, sub string) bool {
	for i := 0; i+len(sub) <= len(s); i++ {
		if s[i:i+len(sub)] == sub {
			return true
		}
	}
	return false
}
