package fix

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

func mixedContainmentRepairProposal(proposalType proposal.Type) proposal.Proposal {
	p := proposal.Proposal{
		Type:        proposalType,
		Field:       "status",
		Description: "repair mixed containment targets",
		Paths:       []string{"inside.md", "../outside.md"},
		From:        "old",
		To:          "new",
		Value:       "new",
		Heading:     "## Status",
		Mode:        "replace",
	}
	if proposalType == proposal.AddField || proposalType == proposal.ExtractBody ||
		proposalType == proposal.InferFromChildren || proposalType == proposal.InferFromSiblings {
		p.Field = "added"
	}
	return p
}

func writeMixedContainmentFixture(t *testing.T) (root, inside, outside, original string, mode os.FileMode) {
	t.Helper()
	base := t.TempDir()
	root = filepath.Join(base, "root")
	if err := os.Mkdir(root, 0o755); err != nil {
		t.Fatal(err)
	}
	original = "---\nstatus: old\nallowed: old\n---\n# Document\n\nold\n\n## Status\n\nold section\n"
	inside = filepath.Join(root, "inside.md")
	outside = filepath.Join(base, "outside.md")
	mode = 0o640
	if err := os.WriteFile(inside, []byte(original), mode); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("outside stays untouched\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	return root, inside, outside, original, mode
}

func assertMixedContainmentProposalRejected(t *testing.T, result *RepairResult, inside, outside, original string, mode os.FileMode) {
	t.Helper()
	if !result.Complete || len(result.Errors) != 0 || len(result.RolledBack) != 0 {
		t.Fatalf("complete=%v errors=%v rolled_back=%v", result.Complete, result.Errors, result.RolledBack)
	}
	if len(result.Rejected) != 1 || !strings.Contains(result.Rejected[0], "../outside.md") {
		t.Fatalf("rejected=%v, want one containment rejection", result.Rejected)
	}
	if len(result.Changed) != 0 || len(result.Skipped) != 0 {
		t.Fatalf("changed=%v skipped=%v, want proposal wholly unattempted", result.Changed, result.Skipped)
	}
	got, err := os.ReadFile(inside)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("contained path was modified by rejected proposal:\n%s", got)
	}
	info, err := os.Stat(inside)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != mode.Perm() {
		t.Fatalf("mode=%#o, want %#o", info.Mode().Perm(), mode.Perm())
	}
	gotOutside, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(gotOutside) != "outside stays untouched\n" {
		t.Fatalf("outside path was modified: %q", gotOutside)
	}
}

func TestApplyRepairRejectsMixedContainmentProposalWhole(t *testing.T) {
	proposalTypes := []proposal.Type{
		proposal.CorrectValue,
		proposal.MigrateValue,
		proposal.AddField,
		proposal.ExtractBody,
		proposal.InferFromChildren,
		proposal.InferFromSiblings,
		proposal.SetField,
		proposal.SetSection,
		proposal.CorrectOutlier,
		proposal.PropagateAggregate,
		proposal.CorrectLink,
	}
	for _, proposalType := range proposalTypes {
		for _, dryRun := range []bool{false, true} {
			t.Run(fmt.Sprintf("%s/dry-run=%t", proposalType, dryRun), func(t *testing.T) {
				root, inside, outside, original, mode := writeMixedContainmentFixture(t)
				result, err := ApplyRepair([]proposal.Proposal{mixedContainmentRepairProposal(proposalType)}, dryRun, root, false)
				if err != nil {
					t.Fatalf("ApplyRepair: %v", err)
				}
				assertMixedContainmentProposalRejected(t, result, inside, outside, original, mode)
				if dryRun {
					if result.ResolvedTargets == nil || result.ResolvedTargets.Accepted["inside.md"] != inside ||
						result.ResolvedTargets.Rejected["../outside.md"] != "escapes root" {
						t.Fatalf("resolved_targets=%+v", result.ResolvedTargets)
					}
				}
			})
		}
	}
}

func TestApplyRepairMixedProposalDoesNotReadSoleContainedPath(t *testing.T) {
	base := t.TempDir()
	root := filepath.Join(base, "root")
	if err := os.MkdirAll(filepath.Join(root, "documents"), 0o755); err != nil {
		t.Fatal(err)
	}
	result, err := ApplyRepair([]proposal.Proposal{{
		Type:  proposal.AddField,
		Field: "added",
		Value: "new",
		Paths: []string{"documents", "../outside.md"},
	}}, false, root, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if !result.Complete || len(result.Errors) != 0 || len(result.Changed) != 0 || len(result.Rejected) != 1 {
		t.Fatalf("result=%+v, want policy rejection without attempting the contained directory", result)
	}
}

func TestApplyRepairMixedProposalWithRepeatedPathIsRejectedOnce(t *testing.T) {
	root, inside, outside, original, mode := writeMixedContainmentFixture(t)
	p := mixedContainmentRepairProposal(proposal.AddField)
	p.Paths = []string{"inside.md", "inside.md", "../outside.md", "../outside.md"}
	result, err := ApplyRepair([]proposal.Proposal{p}, true, root, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	assertMixedContainmentProposalRejected(t, result, inside, outside, original, mode)
	if len(result.ResolvedTargets.Accepted) != 1 || len(result.ResolvedTargets.Rejected) != 1 {
		t.Fatalf("resolved_targets=%+v, want globally deduplicated paths", result.ResolvedTargets)
	}
}

func TestApplyRepairValidProposalMaySharePathWithRejectedProposal(t *testing.T) {
	root, inside, outside, _, mode := writeMixedContainmentFixture(t)
	result, err := ApplyRepair([]proposal.Proposal{
		{Type: proposal.SetField, Field: "blocked", Value: "no", Paths: []string{"inside.md", "../outside.md"}},
		{Type: proposal.SetField, Field: "allowed", Value: "new", Paths: []string{"inside.md"}},
	}, false, root, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if !result.Complete || len(result.Errors) != 0 || len(result.Rejected) != 1 || len(result.Changed) != 1 {
		t.Fatalf("result=%+v", result)
	}
	got, err := os.ReadFile(inside)
	if err != nil {
		t.Fatal(err)
	}
	if strings.Contains(string(got), "blocked:") || !strings.Contains(string(got), "allowed: new") {
		t.Fatalf("shared target content=%q, want only the valid proposal applied", got)
	}
	info, err := os.Stat(inside)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != mode.Perm() {
		t.Fatalf("mode=%#o, want %#o", info.Mode().Perm(), mode.Perm())
	}
	gotOutside, err := os.ReadFile(outside)
	if err != nil || string(gotOutside) != "outside stays untouched\n" {
		t.Fatalf("outside content=%q err=%v", gotOutside, err)
	}
}
