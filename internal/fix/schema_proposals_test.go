package fix

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

func TestApplySchemaProposals_Empty(t *testing.T) {
	report := &proposal.Report{}
	if err := ApplySchemaProposals(context.Background(), report, t.TempDir()); err != nil {
		t.Errorf("expected nil for empty report, got %v", err)
	}
}

func TestApplySchemaProposals_NoExtendEnum(t *testing.T) {
	// Report with non-extend_enum proposals — should be a no-op.
	report := &proposal.Report{
		Proposals: []proposal.Proposal{
			{Type: proposal.CorrectValue, Field: "estado", From: "x", To: "y"},
		},
	}
	if err := ApplySchemaProposals(context.Background(), report, t.TempDir()); err != nil {
		t.Errorf("expected nil for non-schema proposals, got %v", err)
	}
}
