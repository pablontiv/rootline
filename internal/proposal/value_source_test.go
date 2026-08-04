package proposal

import (
	"encoding/json"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

// Issue #60 defect 1: an add_field proposal carried no marker separating a value
// the schema declared from one the engine picked, so a reviewer reading the JSON
// could not tell them apart and fix --all applied both alike.

func addFieldFor(t *testing.T, sf rules.SchemaField) Proposal {
	t.Helper()
	effective := &rules.StemFile{Schema: map[string]rules.SchemaField{"campo": sf}}
	errs := map[string][]rules.ValidationError{
		"a.md": {{Rule: "required", Field: "campo", Message: `required field "campo" is missing`}},
	}

	proposals := detectAddField(effective, errs)
	if len(proposals) != 1 {
		t.Fatalf("expected exactly 1 add_field proposal, got %d", len(proposals))
	}
	return proposals[0]
}

func TestDetectAddField_SchemaDefaultIsMarkedAsSuch(t *testing.T) {
	p := addFieldFor(t, rules.SchemaField{Type: "enum", Values: []string{"report", "note"}, Default: "note"})

	if p.ValueSource != ValueSourceSchemaDefault {
		t.Errorf("expected %q, got %q", ValueSourceSchemaDefault, p.ValueSource)
	}
	if p.Value != "note" {
		t.Errorf("expected the declared default, got %q", p.Value)
	}
}

func TestDetectAddField_FirstEnumMemberIsMarkedEngineChosen(t *testing.T) {
	p := addFieldFor(t, rules.SchemaField{Type: "enum", Values: []string{"report", "note"}})

	if p.ValueSource != ValueSourceEnumFirst {
		t.Errorf("expected %q for a value taken from values[0], got %q", ValueSourceEnumFirst, p.ValueSource)
	}
	if p.Value != "report" {
		t.Errorf("expected values[0], got %q", p.Value)
	}
}

func TestDetectAddField_EmptyStringIsMarkedEngineChosen(t *testing.T) {
	p := addFieldFor(t, rules.SchemaField{Type: "string"})

	if p.ValueSource != ValueSourceEmpty {
		t.Errorf("expected %q for a synthesized empty value, got %q", ValueSourceEmpty, p.ValueSource)
	}
	if p.Value != "" {
		t.Errorf("expected empty value, got %q", p.Value)
	}
}

func TestProposal_ValueSourceSerializes(t *testing.T) {
	data, err := json.Marshal(Proposal{Type: AddField, Field: "campo", ValueSource: ValueSourceEnumFirst})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if !strings.Contains(string(data), `"value_source":"enum_first"`) {
		t.Errorf("expected value_source in JSON, got %s", data)
	}

	var round Proposal
	if err := json.Unmarshal(data, &round); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	if round.ValueSource != ValueSourceEnumFirst {
		t.Errorf("expected value_source to survive a round trip, got %q", round.ValueSource)
	}
}

func TestProposal_ValueSourceOmittedWhenUnset(t *testing.T) {
	data, err := json.Marshal(Proposal{Type: CorrectValue, Field: "campo"})
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}
	if strings.Contains(string(data), "value_source") {
		t.Errorf("expected value_source omitted where provenance is meaningless, got %s", data)
	}
}

func TestIsEngineChosen(t *testing.T) {
	cases := map[string]bool{
		ValueSourceSchemaDefault: false,
		ValueSourceEnumFirst:     true,
		ValueSourceEmpty:         true,
		// An absent marker means the report predates provenance. Treating it as
		// engine-chosen would silently stop applying previously valid reports.
		"": false,
	}
	for source, want := range cases {
		if got := IsEngineChosen(source); got != want {
			t.Errorf("IsEngineChosen(%q) = %v, want %v", source, got, want)
		}
	}
}
