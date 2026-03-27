package query

import (
	"testing"
)

func TestCheckFieldNames_UnknownField(t *testing.T) {
	knownFields := []string{"estado", "tipo", "path", "body", "type"}
	warnings := CheckFieldNames("estdo == 'Pending'", knownFields)
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(warnings))
	}
	if warnings[0].Field != "estdo" {
		t.Errorf("Field = %q, want %q", warnings[0].Field, "estdo")
	}
	if warnings[0].Suggestion != "estado" {
		t.Errorf("Suggestion = %q, want %q", warnings[0].Suggestion, "estado")
	}
}

func TestCheckFieldNames_AllKnown(t *testing.T) {
	knownFields := []string{"estado", "tipo", "path", "body", "type"}
	warnings := CheckFieldNames("estado == 'Pending'", knownFields)
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0", len(warnings))
	}
}

func TestCheckFieldNames_NoSuggestion(t *testing.T) {
	knownFields := []string{"estado", "tipo"}
	warnings := CheckFieldNames("xyzabc == 'foo'", knownFields)
	if len(warnings) != 1 {
		t.Fatalf("got %d warnings, want 1", len(warnings))
	}
	if warnings[0].Field != "xyzabc" {
		t.Errorf("Field = %q, want %q", warnings[0].Field, "xyzabc")
	}
	if warnings[0].Suggestion != "" {
		t.Errorf("Suggestion = %q, want empty", warnings[0].Suggestion)
	}
}

func TestCheckFieldNames_BuiltinFields(t *testing.T) {
	knownFields := []string{"path", "body", "type"}
	warnings := CheckFieldNames("path == 'foo'", knownFields)
	if len(warnings) != 0 {
		t.Errorf("got %d warnings, want 0", len(warnings))
	}
}

func TestCheckFieldNames_MultipleUnknown(t *testing.T) {
	knownFields := []string{"estado", "tipo", "path"}
	warnings := CheckFieldNames("estdo == 'Pending' && tpo == 'foo'", knownFields)
	if len(warnings) != 2 {
		t.Fatalf("got %d warnings, want 2", len(warnings))
	}

	fields := map[string]bool{}
	for _, w := range warnings {
		fields[w.Field] = true
	}
	if !fields["estdo"] {
		t.Error("expected warning for field 'estdo'")
	}
	if !fields["tpo"] {
		t.Error("expected warning for field 'tpo'")
	}
}
