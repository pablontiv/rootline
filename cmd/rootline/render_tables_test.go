package main

import (
	"bytes"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/fix"
	"github.com/spf13/cobra"
)

func newRenderTestCmd() (*cobra.Command, *bytes.Buffer) {
	buf := new(bytes.Buffer)
	cmd := &cobra.Command{}
	cmd.SetOut(buf)
	cmd.SetErr(new(bytes.Buffer))
	return cmd, buf
}

func TestRenderRepairTable_Empty(t *testing.T) {
	cmd, buf := newRenderTestCmd()
	result := &fix.RepairResult{Version: 1, Kind: "rootline/repair"}
	if err := renderRepairTable(cmd, result); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(buf.String(), "No repairs applied") {
		t.Errorf("expected 'No repairs applied', got: %s", buf.String())
	}
}

func TestRenderRepairTable_AllSections(t *testing.T) {
	cmd, buf := newRenderTestCmd()
	result := &fix.RepairResult{
		Version:  1,
		Kind:     "rootline/repair",
		DryRun:   true,
		Changed:  []string{"correct foo: a->b in x.md"},
		Skipped:  []string{"skipped y.md"},
		Rejected: []string{"schema proposal z"},
		Errors:   []string{"err one"},
	}
	if err := renderRepairTable(cmd, result); err != nil {
		t.Fatalf("err: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"DRY RUN", "Changed (1)", "Skipped (1)", "Rejected", "Errors (1)"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestRenderSchemaProposeTable_Empty(t *testing.T) {
	cmd, buf := newRenderTestCmd()
	report := &SchemaProposalsReport{Path: "docs/"}
	if err := renderSchemaProposeTable(cmd, report); err != nil {
		t.Fatalf("err: %v", err)
	}
	if !strings.Contains(buf.String(), "No schema proposals") {
		t.Errorf("expected 'No schema proposals' message")
	}
}

func TestRenderSchemaProposeTable_WithProposals(t *testing.T) {
	cmd, buf := newRenderTestCmd()
	report := &SchemaProposalsReport{
		Path: "docs/",
		Proposals: []SchemaProposal{
			{ID: "p1", Operation: "create_stem", Target: "docs/.stem", Confidence: 0.9, RequiresAgent: false},
			{ID: "p2", Operation: "update_stem", Target: "docs/sub/.stem", Confidence: 0.7, RequiresAgent: true},
		},
		Summary: ProposalsSummary{Total: 2, RequiresAgent: 1, EngineResolved: 1},
	}
	if err := renderSchemaProposeTable(cmd, report); err != nil {
		t.Fatalf("err: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"p1", "create_stem", "p2", "Summary"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestRenderSchemaApplyTable_DryRunWithItems(t *testing.T) {
	cmd, buf := newRenderTestCmd()
	result := &SchemaApplyResult{
		Version: 1,
		Kind:    "rootline/schema-apply",
		DryRun:  true,
		Applied: []string{"create docs/.stem"},
		Skipped: []string{"requires-agent proposal"},
		Errors:  []string{"some error"},
	}
	if err := renderSchemaApplyTable(cmd, result); err != nil {
		t.Fatalf("err: %v", err)
	}
	out := buf.String()
	for _, want := range []string{"Dry run", "Would apply", "Skipped", "Errors"} {
		if !strings.Contains(out, want) {
			t.Errorf("expected %q in output, got: %s", want, out)
		}
	}
}

func TestRenderSchemaApplyTable_Applied(t *testing.T) {
	cmd, buf := newRenderTestCmd()
	result := &SchemaApplyResult{
		Version: 1,
		Kind:    "rootline/schema-apply",
		Applied: []string{"updated docs/.stem"},
	}
	if err := renderSchemaApplyTable(cmd, result); err != nil {
		t.Fatalf("err: %v", err)
	}
	out := buf.String()
	if !strings.Contains(out, "Applied:") || !strings.Contains(out, "updated docs/.stem") {
		t.Errorf("expected applied items in output, got: %s", out)
	}
}
