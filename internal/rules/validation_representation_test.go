package rules

import (
	"encoding/json"
	"strings"
	"testing"
)

func TestValidationErrorCarriesTypeRepresentationsInternally(t *testing.T) {
	err := validationErrorFromContract("ingested", SchemaField{Type: "string"}, &FieldContractIssue{
		Code:     "type-mismatch",
		Expected: "string",
		Actual:   "timestamp",
	})
	if err.ExpectedRepresentation != "string" {
		t.Errorf("ExpectedRepresentation = %q", err.ExpectedRepresentation)
	}
	if err.ActualRepresentation != "timestamp" {
		t.Errorf("ActualRepresentation = %q", err.ActualRepresentation)
	}
}

func TestValidationErrorUnrelatedContractIssuesCarryNoRepresentationEvidence(t *testing.T) {
	for _, issue := range []FieldContractIssue{
		{Code: "invalid-enum", Expected: "Pending, Done", Actual: "Blocked"},
		{Code: "invalid-link", Expected: "wiki-link syntax", Actual: "plain text"},
		{Code: "invalid-sequence", Expected: "T plus 3 digits", Actual: "bad"},
	} {
		err := validationErrorFromContract("value", SchemaField{Type: "string"}, &issue)
		if err.ExpectedRepresentation != "" || err.ActualRepresentation != "" {
			t.Errorf("%s issue leaked representation evidence: %#v", issue.Code, err)
		}
	}
}

func TestValidationErrorRepresentationEvidenceIsNotJSON(t *testing.T) {
	err := ValidationError{
		Rule:                   "type",
		Field:                  "ingested",
		Message:                "human text",
		ExpectedRepresentation: "string",
		ActualRepresentation:   "timestamp",
	}
	data, marshalErr := json.Marshal(err)
	if marshalErr != nil {
		t.Fatalf("Marshal: %v", marshalErr)
	}
	for _, forbidden := range []string{"expected_representation", "actual_representation", "timestamp"} {
		if strings.Contains(string(data), forbidden) {
			t.Errorf("internal evidence leaked into JSON: %s", data)
		}
	}
}
