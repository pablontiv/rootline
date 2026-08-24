package proposal

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestProposalFromRepresentationRoundTrips(t *testing.T) {
	input := Proposal{
		Type: CorrectValue, Field: "ingested", Paths: []string{"a.md"},
		From: "2026-06-22", To: "2026-06-22", FromRepresentation: "timestamp",
	}
	data, err := json.Marshal(input)
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if !strings.Contains(string(data), `"from_representation":"timestamp"`) {
		t.Fatalf("missing discriminator: %s", data)
	}
	var output Proposal
	if err := json.Unmarshal(data, &output); err != nil {
		t.Fatalf("Unmarshal: %v", err)
	}
	if output.FromRepresentation != "timestamp" {
		t.Fatalf("FromRepresentation = %q", output.FromRepresentation)
	}
}

func TestProposalFromRepresentationOmittedWhenEmpty(t *testing.T) {
	data, err := json.Marshal(Proposal{Type: CorrectValue, Field: "estado", From: "Pendng", To: "Pending"})
	if err != nil {
		t.Fatalf("Marshal: %v", err)
	}
	if strings.Contains(string(data), "from_representation") {
		t.Fatalf("legacy proposal shape changed: %s", data)
	}
}
