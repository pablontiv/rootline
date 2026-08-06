package main

import (
	"encoding/json"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

// TestRepairCompleteTracksExitStatus is the test that keeps the envelope honest.
// `complete` and the exit status answer the same question in two places — one
// for a consumer holding a stored artifact, one for a shell — and nothing but a
// test stops them from drifting apart.
func TestRepairCompleteTracksExitStatus(t *testing.T) {
	tests := []struct {
		name         string
		proposals    []proposal.Proposal
		dryRun       bool
		wantComplete bool
	}{
		{
			name:         "a clean apply is complete",
			proposals:    []proposal.Proposal{correctValueTo("task.md", "Completed")},
			wantComplete: true,
		},
		{
			name:         "a run whose write was reverted is incomplete",
			proposals:    []proposal.Proposal{correctValueTo("task.md", "TOTALLY_INVALID")},
			wantComplete: false,
		},
		{
			name:         "a run that could not read a path is incomplete",
			proposals:    []proposal.Proposal{correctValueTo("nonexistent.md", "Completed")},
			wantComplete: false,
		},
		{
			// A refusal is not an incompletion: the command did everything it
			// agreed to do, and named what it declined.
			name: "a run that only rejected a proposal is complete",
			proposals: []proposal.Proposal{{
				Type:        proposal.ExtendEnum,
				Field:       "estado",
				Value:       "InProgress",
				Paths:       []string{"task.md"},
				Description: "extend enum",
			}},
			wantComplete: true,
		},
		{
			name:         "a clean dry run is complete",
			proposals:    []proposal.Proposal{correctValueTo("task.md", "Completed")},
			dryRun:       true,
			wantComplete: true,
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

			result := decodeRepairResult(t, out)
			if result.Complete != tt.wantComplete {
				t.Errorf("complete = %v, want %v (errors=%v rolled_back=%v)",
					result.Complete, tt.wantComplete, result.Errors, result.RolledBack)
			}

			// The binding claim: `complete` is exactly the exit rule. A consumer
			// reading a stored artifact must reach the same conclusion a shell
			// reading $? would have reached.
			if exitedZero := err == nil; exitedZero != result.Complete {
				t.Errorf("complete = %v but the command %s — the envelope and the exit status disagree",
					result.Complete, map[bool]string{true: "exited 0", false: "failed"}[exitedZero])
			}
		})
	}
}

// TestSchemaApplyCompleteTracksExitStatus makes the same guarantee on the other
// command, whose result type carries no rolled_back[].
func TestSchemaApplyCompleteTracksExitStatus(t *testing.T) {
	t.Run("a failed proposal makes the run incomplete", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")
		reportFile := writeProposalsReport(t, docsDir, docsDir, filepath.Join(docsDir, ".stem"), "")

		out, err := executeSchemaApply(t, "--report", reportFile)

		result := decodeSchemaApplyResult(t, out)
		if result.Complete {
			t.Errorf("complete = true, want false (errors=%v)", result.Errors)
		}
		if err == nil {
			t.Error("the run reported incomplete but exited 0")
		}
	})

	t.Run("a run that only deferred work is complete", func(t *testing.T) {
		root := setupValidateProject(t, map[string]string{
			"docs/doc1.md": "---\ntitle: Test\n---\nContent",
		})
		docsDir := filepath.Join(root, "docs")
		data, err := json.Marshal(SchemaProposalsReport{
			Version: 1,
			Kind:    "rootline/schema-proposals",
			Path:    docsDir,
			Root:    docsDir,
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
		reportFile := filepath.Join(root, "deferred.json")
		mustWriteFile(t, reportFile, data, 0o644)

		out, err := executeSchemaApply(t, "--report", reportFile)
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeSchemaApplyResult(t, out)
		if !result.Complete {
			t.Errorf("complete = false, want true: a deferred proposal is a skip, not a failure (skipped=%v)",
				result.Skipped)
		}
	})
}
