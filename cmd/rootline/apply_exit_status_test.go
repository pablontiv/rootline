package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

// exitStatusStem governs a single enum field, which is enough to drive every
// outcome the exit rule distinguishes: a value inside the enum applies cleanly,
// a value outside it is written and then rolled back by post-validation.
const exitStatusStem = `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`

// newExitStatusFixture builds a governed directory holding one valid document
// and writes the given proposals as a report BESIDE that document, so the
// report resolves identically under the legacy root rule and the unified one.
// It returns the path to the report file.
func newExitStatusFixture(t *testing.T, proposals []proposal.Proposal) string {
	t.Helper()

	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(exitStatusStem), 0o644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Pending\n---\n\n# Task\n"), 0o644)

	data, err := json.Marshal(proposal.Report{
		Version:   1,
		Kind:      "rootline/proposals",
		Proposals: proposals,
	})
	if err != nil {
		t.Fatalf("marshalling report: %v", err)
	}

	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, data, 0o644)
	return reportFile
}

func correctValueTo(path, to string) proposal.Proposal {
	return proposal.Proposal{
		Type:        proposal.CorrectValue,
		Field:       "estado",
		From:        "Pending",
		To:          to,
		Paths:       []string{path},
		Description: "correct estado",
	}
}

// TestRepairApplyExitStatus pins the rule that decides `repair apply`'s exit
// status: a run fails when it could not carry through what it accepted, and
// succeeds when it merely declined or deferred work and said so.
func TestRepairApplyExitStatus(t *testing.T) {
	tests := []struct {
		name      string
		proposals []proposal.Proposal
		dryRun    bool
		wantFail  bool
	}{
		{
			name:      "a report that applies cleanly succeeds",
			proposals: []proposal.Proposal{correctValueTo("task.md", "Completed")},
		},
		{
			// The issue's headline reproduction. Since the per-file rollback
			// landed, the revert succeeds and leaves errors[] empty — so a rule
			// keyed on errors[] alone would still call this a success.
			name:      "a write reverted by post-validation fails",
			proposals: []proposal.Proposal{correctValueTo("task.md", "TOTALLY_INVALID")},
			wantFail:  true,
		},
		{
			name:      "a path that cannot be read fails",
			proposals: []proposal.Proposal{correctValueTo("nonexistent.md", "Completed")},
			wantFail:  true,
		},
		{
			name: "a schema proposal is rejected, not failed",
			proposals: []proposal.Proposal{{
				Type:        proposal.ExtendEnum,
				Field:       "estado",
				Value:       "InProgress",
				Paths:       []string{"task.md"},
				Description: "extend enum",
			}},
		},
		{
			name:      "a clean dry run succeeds",
			proposals: []proposal.Proposal{correctValueTo("task.md", "Completed")},
			dryRun:    true,
		},
		{
			// Dry run is not a carve-out. The report itself is unactionable,
			// which is exactly what a CI precondition check needs to hear.
			name:      "a dry run over an unreadable path fails",
			proposals: []proposal.Proposal{correctValueTo("nonexistent.md", "Completed")},
			dryRun:    true,
			wantFail:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			reportFile := newExitStatusFixture(t, tt.proposals)

			args := []string{"--report", reportFile, "-o", "json"}
			if tt.dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeRepairApply(t, args...)

			assertExitStatus(t, err, tt.wantFail)

			// The payload must survive the failure: a caller that sees a
			// non-zero exit still has to be able to read what happened.
			result := decodeRepairResult(t, out)
			failed := len(result.Errors) > 0 || len(result.RolledBack) > 0
			if failed != tt.wantFail {
				t.Errorf("payload says failed=%v but the exit status says %v; errors=%v rolled_back=%v",
					failed, tt.wantFail, result.Errors, result.RolledBack)
			}
		})
	}
}

// TestSchemaApplyExitStatus pins the same rule on the other report-consuming
// command, so the two cannot drift apart.
func TestSchemaApplyExitStatus(t *testing.T) {
	t.Run("a proposal with no patch content fails", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")
		reportFile := writeProposalsReport(t, docsDir, docsDir, filepath.Join(docsDir, ".stem"), "")

		out, err := executeSchemaApply(t, "--report", reportFile)
		assertExitStatus(t, err, true)

		result := decodeSchemaApplyResult(t, out)
		if len(result.Errors) == 0 {
			t.Errorf("errors = %v, want the empty-patch failure named", result.Errors)
		}
	})

	t.Run("proposals deferred to an agent are skipped, not failed", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")

		data, err := json.Marshal(SchemaProposalsReport{
			Version: 1,
			Kind:    "rootline/schema-proposals",
			Path:    docsDir,
			Proposals: []SchemaProposal{{
				ID:            "needs-a-human",
				Operation:     "create_stem",
				Target:        filepath.Join(docsDir, ".stem"),
				RequiresAgent: true,
			}},
		})
		if err != nil {
			t.Fatalf("marshalling report: %v", err)
		}
		reportFile := filepath.Join(root, "report.json")
		mustWriteFile(t, reportFile, data, 0o644)

		out, err := executeSchemaApply(t, "--report", reportFile)
		assertExitStatus(t, err, false)

		result := decodeSchemaApplyResult(t, out)
		if len(result.Skipped) != 1 {
			t.Errorf("skipped = %v, want the deferred proposal recorded", result.Skipped)
		}
	})
}

// executeRepairApply runs `repair apply` with stdout and stderr kept apart, so
// a test can decode the payload without cobra's "Error: ..." line landing in
// the same buffer. That separation is itself part of the contract: the payload
// goes to stdout and stays machine-readable even when the run exits non-zero.
func executeRepairApply(t *testing.T, args ...string) (string, error) {
	t.Helper()
	resetFlags()
	out := new(bytes.Buffer)
	rootCmd.SetOut(out)
	rootCmd.SetErr(new(bytes.Buffer))
	rootCmd.SetArgs(append([]string{"repair", "apply"}, args...))
	err := rootCmd.Execute()
	return out.String(), err
}

// assertExitStatus checks the outcome against the contract, and additionally
// requires that a failure is OUR failure. Without the errApplyFailure check a
// broken fixture — a malformed report, a missing flag — would satisfy a bare
// "an error came back" assertion and the test would pass for the wrong reason.
func assertExitStatus(t *testing.T, err error, wantFail bool) {
	t.Helper()

	if !wantFail {
		if err != nil {
			t.Fatalf("run should have exited 0, got: %v", err)
		}
		return
	}

	if err == nil {
		t.Fatal("run should have exited non-zero, got nil")
	}
	if !errors.Is(err, errApplyFailure) {
		t.Fatalf("run failed for an unrelated reason, so the exit rule was never exercised: %v", err)
	}
}
