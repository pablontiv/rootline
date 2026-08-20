package main

import (
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/fix"
)

const schemaApplyPlannerPatchA = "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
const schemaApplyPlannerPatchB = "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n  estado:\n    type: string\n"

func writeSchemaApplyPlannerReport(t *testing.T, root string, proposals []SchemaProposal) string {
	t.Helper()
	report := SchemaProposalsReport{
		Version:   1,
		Kind:      "rootline/schema-proposals",
		Path:      root,
		Root:      root,
		Proposals: proposals,
	}
	path := filepath.Join(root, "report.json")
	writeSchemaApplyPreflightReport(t, path, report)
	return path
}

func setupSchemaApplyPlannerProject(t *testing.T) (root, docsDir, target string) {
	t.Helper()
	root = setupValidateProject(t, map[string]string{
		"docs/doc1.md": "---\ntitle: Test\nestado: Pending\n---\nContent",
	})
	docsDir = filepath.Join(root, "docs")
	target = filepath.Join(docsDir, ".stem")
	assertSchemaApplyAbsent(t, target)
	return root, docsDir, target
}

func assertSchemaApplyPlannerSuccess(t *testing.T, out string, err error) *SchemaApplyResult {
	t.Helper()
	if err != nil {
		t.Fatalf("schema apply returned unexpected error: %v\noutput: %s", err, out)
	}
	result := decodeSchemaApplyResult(t, out)
	if result.Version != 1 || result.Kind != "rootline/schema-apply" || !result.Complete {
		t.Fatalf("success envelope changed contract: %+v", result)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("errors = %v, want empty", result.Errors)
	}
	return result
}

func assertSchemaApplyPlannerApplied(t *testing.T, got []string, want []string) {
	t.Helper()
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("applied = %#v, want %#v", got, want)
	}
}

func assertSchemaApplyPlannerRejected(t *testing.T, got []string, want []string) {
	t.Helper()
	if len(got) == 0 && len(want) == 0 {
		return
	}
	if !reflect.DeepEqual(got, want) {
		t.Fatalf("rejected = %#v, want %#v", got, want)
	}
}

func TestPlanSchemaProposalApply_CachesCanonicalTargetStatErrorAcrossAliases(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "target", ".stem")
	aliasedTarget := root + string(filepath.Separator) + "target" + string(filepath.Separator) + ".." + string(filepath.Separator) + "target" + string(filepath.Separator) + ".stem"
	statErr := errors.New("cached stat error")
	statCalls := 0
	stat := func(path string) (os.FileInfo, error) {
		statCalls++
		if statCalls == 1 {
			if path != target {
				t.Fatalf("stat path = %q, want canonical target %q", path, target)
			}
			return nil, statErr
		}
		return nil, nil
	}
	result := &SchemaApplyResult{Errors: []string{}, Rejected: []string{}, Skipped: []string{}}
	resolved := &fix.ResolvedTargetsBreakdown{Accepted: map[string]string{}, Rejected: map[string]string{}}

	plan := planSchemaProposalApplyWithStat([]SchemaProposal{
		{ID: "first", Operation: "create_stem", Target: target, Patch: schemaApplyPlannerPatchA},
		{ID: "second", Operation: "create_stem", Target: aliasedTarget, Patch: schemaApplyPlannerPatchB},
	}, root, false, result, resolved, stat)

	if statCalls != 1 {
		t.Fatalf("stat calls = %d, want exactly one canonical target observation", statCalls)
	}
	if len(plan) != 0 {
		t.Fatalf("plan = %+v, want no actions after fatal stat error", plan)
	}
	wantErrors := []string{
		fmt.Sprintf("stat %s: %v", target, statErr),
		fmt.Sprintf("stat %s: %v", aliasedTarget, statErr),
	}
	if !reflect.DeepEqual(result.Errors, wantErrors) {
		t.Fatalf("errors = %#v, want cached stat error reused in order %#v", result.Errors, wantErrors)
	}
	if len(result.Rejected) != 0 || len(result.Skipped) != 0 {
		t.Fatalf("nonfatal lists changed after fatal stat error: rejected=%#v skipped=%#v", result.Rejected, result.Skipped)
	}
	if got := resolved.Accepted[aliasedTarget]; got != target {
		t.Fatalf("aliased target resolved to %q, want canonical %q", got, target)
	}
}

func TestSchemaApply_StatErrorAbortsBeforeActionsOrWrites(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":     "version: 2\nroot: true\nschema:\n  title:\n    type: string\n",
				"record.md": "---\ntitle: Test\n---\n# Test\n",
			})
			blockingPath := filepath.Join(root, "blocked")
			mustWriteFile(t, blockingPath, []byte("not a directory\n"), 0o600)
			if err := os.Chmod(blockingPath, 0o600); err != nil {
				t.Fatal(err)
			}
			beforeBlocker := snapshotSchemaApplyFile(t, blockingPath)
			target := filepath.Join(blockingPath, ".stem")
			_, statErr := os.Stat(target)
			if statErr == nil || os.IsNotExist(statErr) {
				t.Skipf("platform did not expose a fatal stat error for file child: %v", statErr)
			}
			reportPath := writeSchemaApplyPlannerReport(t, root, []SchemaProposal{
				{ID: "blocked", Operation: "create_stem", Target: target, Patch: schemaApplyPlannerPatchA},
			})

			args := []string{"--report", reportPath}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			if err == nil {
				t.Fatalf("schema apply accepted fatal stat error: %s", out)
			}
			result := decodeSchemaApplyResult(t, out)
			if result.Version != 1 || result.Kind != "rootline/schema-apply" || result.Complete {
				t.Fatalf("failure envelope changed contract: %+v", result)
			}
			wantErrors := []string{fmt.Sprintf("stat %s: %v", target, statErr)}
			if !reflect.DeepEqual(result.Errors, wantErrors) {
				t.Fatalf("errors = %#v, want exact ordered stat failure %#v", result.Errors, wantErrors)
			}
			if len(result.Applied) != 0 || len(result.Skipped) != 0 || len(result.Rejected) != 0 {
				t.Fatalf("actions changed before stat abort: applied=%#v skipped=%#v rejected=%#v", result.Applied, result.Skipped, result.Rejected)
			}
			assertSchemaApplyFileUnchanged(t, blockingPath, beforeBlocker)
		})
	}
}

func TestSchemaApply_DuplicateAbsentTargetUsesVirtualExistenceWithoutForce(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			root, docsDir, target := setupSchemaApplyPlannerProject(t)
			aliasedTarget := root + string(filepath.Separator) + "docs" + string(filepath.Separator) + ".." + string(filepath.Separator) + "docs" + string(filepath.Separator) + ".stem"
			reportPath := writeSchemaApplyPlannerReport(t, root, []SchemaProposal{
				{ID: "first", Operation: "create_stem", Target: target, Patch: schemaApplyPlannerPatchA},
				{ID: "second", Operation: "create_stem", Target: aliasedTarget, Patch: schemaApplyPlannerPatchB},
			})

			args := []string{"--report", reportPath}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			result := assertSchemaApplyPlannerSuccess(t, out, err)
			assertSchemaApplyPlannerApplied(t, result.Applied, []string{"create_stem: " + target})
			assertSchemaApplyPlannerRejected(t, result.Rejected, []string{fmt.Sprintf(".stem already exists in %s (use --force to overwrite)", docsDir)})
			if dryRun {
				assertSchemaApplyAbsent(t, target)
				if result.ResolvedTargets == nil {
					t.Fatal("resolved_targets is absent in dry-run")
				}
				if got := result.ResolvedTargets.Accepted[aliasedTarget]; got != target {
					t.Fatalf("aliased resolved target = %q, want canonical %q", got, target)
				}
				return
			}
			if got := string(mustReadFile(t, target)); got != schemaApplyPlannerPatchA {
				t.Fatalf("written bytes = %q, want first patch", got)
			}
			info, statErr := os.Stat(target)
			if statErr != nil {
				t.Fatalf("stat written target: %v", statErr)
			}
			if got := info.Mode().Perm(); got != 0o644 {
				t.Fatalf("written mode = %o, want 644", got)
			}
		})
	}
}

func TestSchemaApply_DuplicateAbsentTargetWithForceUsesPlanOrder(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			root, _, target := setupSchemaApplyPlannerProject(t)
			reportPath := writeSchemaApplyPlannerReport(t, root, []SchemaProposal{
				{ID: "first", Operation: "create_stem", Target: target, Patch: schemaApplyPlannerPatchA},
				{ID: "second", Operation: "create_stem", Target: target, Patch: schemaApplyPlannerPatchB},
			})

			args := []string{"--report", reportPath, "--force"}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			result := assertSchemaApplyPlannerSuccess(t, out, err)
			assertSchemaApplyPlannerApplied(t, result.Applied, []string{"create_stem: " + target, "overwrite_stem: " + target})
			assertSchemaApplyPlannerRejected(t, result.Rejected, []string{})
			if dryRun {
				assertSchemaApplyAbsent(t, target)
				return
			}
			if got := string(mustReadFile(t, target)); got != schemaApplyPlannerPatchB {
				t.Fatalf("written bytes = %q, want second patch", got)
			}
		})
	}
}

func TestSchemaApply_ValidProposalContinuesWithPolicyRejections(t *testing.T) {
	for _, tc := range []struct {
		name         string
		proposal     func(root, validTarget string) SchemaProposal
		wantRejected func(root string) string
	}{
		{
			name: "containment_refusal",
			proposal: func(root, validTarget string) SchemaProposal {
				return SchemaProposal{ID: "outside", Operation: "create_stem", Target: filepath.Join(root, "..", "escaped.stem"), Patch: schemaApplyPlannerPatchA}
			},
			wantRejected: func(root string) string { return "escapes root" },
		},
		{
			name: "existing_target_without_force",
			proposal: func(root, validTarget string) SchemaProposal {
				existingDir := filepath.Join(root, "existing")
				if err := os.MkdirAll(existingDir, 0o755); err != nil {
					t.Fatal(err)
				}
				mustWriteFile(t, filepath.Join(existingDir, ".stem"), []byte(schemaApplyPlannerPatchB), 0o644)
				return SchemaProposal{ID: "existing", Operation: "create_stem", Target: filepath.Join(existingDir, ".stem"), Patch: schemaApplyPlannerPatchA}
			},
			wantRejected: func(root string) string {
				return fmt.Sprintf(".stem already exists in %s (use --force to overwrite)", filepath.Join(root, "existing"))
			},
		},
		{
			name: "unknown_operation",
			proposal: func(root, validTarget string) SchemaProposal {
				return SchemaProposal{ID: "unknown", Operation: "update_stem", Target: filepath.Join(root, "ignored.stem")}
			},
			wantRejected: func(root string) string { return "unknown: unknown operation \"update_stem\"" },
		},
	} {
		t.Run(tc.name, func(t *testing.T) {
			for _, dryRun := range []bool{true, false} {
				t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
					root, _, target := setupSchemaApplyPlannerProject(t)
					reportPath := writeSchemaApplyPlannerReport(t, root, []SchemaProposal{
						{ID: "valid", Operation: "create_stem", Target: target, Patch: schemaApplyPlannerPatchA},
						tc.proposal(root, target),
					})
					args := []string{"--report", reportPath}
					if dryRun {
						args = append(args, "--dry-run")
					}
					out, err := executeSchemaApply(t, args...)
					result := assertSchemaApplyPlannerSuccess(t, out, err)
					assertSchemaApplyPlannerApplied(t, result.Applied, []string{"create_stem: " + target})
					if tc.name == "containment_refusal" {
						if len(result.Rejected) != 1 || !strings.Contains(result.Rejected[0], tc.wantRejected(root)) {
							t.Fatalf("rejected = %#v, want containment reason %q", result.Rejected, tc.wantRejected(root))
						}
					} else {
						assertSchemaApplyPlannerRejected(t, result.Rejected, []string{tc.wantRejected(root)})
					}
					if dryRun {
						assertSchemaApplyAbsent(t, target)
					} else if got := string(mustReadFile(t, target)); got != schemaApplyPlannerPatchA {
						t.Fatalf("written bytes = %q, want valid patch", got)
					}
				})
			}
		})
	}
}
