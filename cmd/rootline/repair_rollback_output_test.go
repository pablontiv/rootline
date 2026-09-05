package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/proposal"
)

func TestRenderRepairTable_RolledBack(t *testing.T) {
	for _, mixed := range []bool{false, true} {
		name := "rollback only"
		if mixed {
			name = "mixed outcomes"
		}
		t.Run(name, func(t *testing.T) {
			result := &fix.RepairResult{
				RolledBack: []fix.RolledBackFile{
					{Path: "a.md", Errors: []string{"status: invalid value", "owner: required"}},
					{Path: "b.md", Errors: []string{"title: empty"}},
				},
			}
			if mixed {
				result.Changed = []string{"correct status in kept.md"}
				result.Skipped = []string{"skip pending.md"}
				result.Rejected = []string{"schema proposal"}
				result.Errors = []string{"missing.md: unreadable"}
			}
			cmd, buf := newRenderTestCmd()
			if err := renderRepairTable(cmd, result); err != nil {
				t.Fatal(err)
			}
			out := buf.String()
			for _, want := range []string{"Rolled back (2):", "a.md", "b.md", "status: invalid value", "owner: required", "title: empty"} {
				if !strings.Contains(out, want) {
					t.Errorf("missing %q in output: %s", want, out)
				}
			}
			if strings.Contains(out, "No repairs applied") {
				t.Errorf("rollback misreported as no-op: %s", out)
			}
			if mixed {
				for _, want := range []string{"Changed (1):", "kept.md", "Skipped (1):", "pending.md", "Rejected (1):", "schema proposal", "Errors (1):", "missing.md: unreadable"} {
					if !strings.Contains(out, want) {
						t.Errorf("missing mixed outcome %q: %s", want, out)
					}
				}
			}
		})
	}
}

func TestRepairApply_RollbackOutput(t *testing.T) {
	for _, format := range []string{"table", "json"} {
		for _, dryRun := range []bool{false, true} {
			name := format + "/apply"
			if dryRun {
				name = format + "/dry-run"
			}
			t.Run(name, func(t *testing.T) {
				report := newExitStatusFixture(t, []proposal.Proposal{correctValueTo("task.md", "TOTALLY_INVALID")})
				path := filepath.Join(filepath.Dir(report), "task.md")
				before, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				args := []string{"--report", report, "-o", format}
				if dryRun {
					args = append(args, "--dry-run")
				}
				out, err := executeRepairApply(t, args...)
				assertExitStatus(t, err, !dryRun)
				if format == "table" {
					if dryRun {
						if !strings.Contains(out, "DRY RUN") || strings.Contains(out, "Rolled back") {
							t.Errorf("dry run claimed an actual rollback: %s", out)
						}
					} else {
						for _, want := range []string{"Rolled back (1):", "task.md", "estado:", "TOTALLY_INVALID", "allowed values"} {
							if !strings.Contains(out, want) {
								t.Errorf("missing rollback diagnostic %q: %s", want, out)
							}
						}
						if strings.Contains(out, "No repairs applied") || strings.Contains(out, "Changed (") {
							t.Errorf("rollback reported as no-op or successful change: %s", out)
						}
					}
				} else {
					result := decodeRepairResult(t, out)
					if result.Version != 1 || result.Kind != "rootline/repair" || result.DryRun != dryRun || result.Complete != dryRun {
						t.Errorf("unexpected JSON contract: %s", out)
					}
					if dryRun {
						if len(result.RolledBack) != 0 {
							t.Errorf("dry run contains rollback: %s", out)
						}
					} else if len(result.Changed) != 0 || len(result.RolledBack) != 1 || result.RolledBack[0].Path != "task.md" || len(result.RolledBack[0].Errors) != 1 {
						t.Errorf("unexpected rollback payload: %s", out)
					}
				}
				after, err := os.ReadFile(path)
				if err != nil {
					t.Fatal(err)
				}
				if string(after) != string(before) {
					t.Errorf("rollback or dry run changed document bytes: %q", after)
				}
			})
		}
	}
}
