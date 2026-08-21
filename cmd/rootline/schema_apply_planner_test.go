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
	"github.com/pablontiv/rootline/internal/fsx"
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

func TestPlanSchemaProposalApply_RejectsNonLiteralStemBasenameBeforeResolution(t *testing.T) {
	root := t.TempDir()
	target := filepath.Join(root, "docs", ".STEM")
	calls := 0
	resolver := func(root, target string) (*fsx.AtomicTarget, error) {
		calls++
		t.Fatalf("resolver should not be called for rejected basename: root=%s target=%s", root, target)
		return nil, nil
	}
	result := &SchemaApplyResult{Errors: []string{}, Rejected: []string{}, Skipped: []string{}}
	resolved := &fix.ResolvedTargetsBreakdown{Accepted: map[string]string{}, Rejected: map[string]string{}}

	plan := planSchemaProposalApplyWithResolver([]SchemaProposal{{ID: "case-alias", Operation: "create_stem", Target: target, Patch: schemaApplyPlannerPatchA}}, root, false, result, resolved, resolver)
	defer func() { _ = closeSchemaApplyBatch(plan) }()

	if calls != 0 {
		t.Fatalf("resolver calls = %d, want 0", calls)
	}
	if len(plan.writes) != 0 || len(plan.actionsByWrite) != 0 {
		t.Fatalf("plan = %+v, want no writes or actions", plan)
	}
	if len(result.Errors) != 0 || len(result.Skipped) != 0 {
		t.Fatalf("result = %+v, want no errors or skips", result)
	}
	wantRejected := []string{fmt.Sprintf("%s: target basename must be \".stem\"", target)}
	assertSchemaApplyPlannerRejected(t, result.Rejected, wantRejected)
	if got := resolved.Rejected[target]; got != "target basename must be \".stem\"" {
		t.Fatalf("resolved rejected = %q, want basename refusal", got)
	}
	if _, ok := resolved.Accepted[target]; ok {
		t.Fatalf("resolved accepted unexpectedly contains %s", target)
	}
}

func TestPlanSchemaProposalApply_CachesPhysicalTargetObservationAcrossAliases(t *testing.T) {
	root := t.TempDir()
	targetDir := filepath.Join(root, "target")
	if err := os.MkdirAll(targetDir, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(targetDir, ".stem")
	aliasedTarget := root + string(filepath.Separator) + "target" + string(filepath.Separator) + ".." + string(filepath.Separator) + "target" + string(filepath.Separator) + ".stem"
	var resolvedTargets []string
	resolver := func(root, target string) (*fsx.AtomicTarget, error) {
		resolvedTargets = append(resolvedTargets, target)
		return fsx.ResolveAtomicTarget(root, target)
	}
	result := &SchemaApplyResult{Errors: []string{}, Rejected: []string{}, Skipped: []string{}}
	resolved := &fix.ResolvedTargetsBreakdown{Accepted: map[string]string{}, Rejected: map[string]string{}}

	plan := planSchemaProposalApplyWithResolver([]SchemaProposal{
		{ID: "first", Operation: "create_stem", Target: target, Patch: schemaApplyPlannerPatchA},
		{ID: "second", Operation: "create_stem", Target: aliasedTarget, Patch: schemaApplyPlannerPatchB},
	}, root, false, result, resolved, resolver)
	defer func() { _ = closeSchemaApplyBatch(plan) }()

	if !reflect.DeepEqual(resolvedTargets, []string{target, target}) {
		t.Fatalf("resolved targets = %#v, want canonical lexical targets before physical classification", resolvedTargets)
	}
	if len(plan.writes) != 1 || len(plan.actionsByWrite) != 1 {
		t.Fatalf("plan = %+v, want only first alias accepted without --force", plan)
	}
	if len(result.Errors) != 0 {
		t.Fatalf("errors = %#v, want none", result.Errors)
	}
	wantRejected := []string{fmt.Sprintf(".stem already exists in %s (use --force to overwrite)", targetDir)}
	if !reflect.DeepEqual(result.Rejected, wantRejected) {
		t.Fatalf("rejected = %#v, want physical duplicate overwrite refusal %#v", result.Rejected, wantRejected)
	}
	if len(result.Skipped) != 0 {
		t.Fatalf("skipped = %#v, want none", result.Skipped)
	}
	if got := resolved.Accepted[aliasedTarget]; got != target {
		t.Fatalf("aliased target resolved to %q, want canonical %q", got, target)
	}
	physicalDir, err := filepath.EvalSymlinks(targetDir)
	if err != nil {
		t.Fatal(err)
	}
	wantPhysicalTarget := filepath.Join(physicalDir, ".stem")
	if got := plan.writes[0].target.PhysicalPath(); got != wantPhysicalTarget {
		t.Fatalf("accepted physical target = %q, want %q", got, wantPhysicalTarget)
	}
}

func TestSchemaApply_RejectsEscapingSymlinkParentBeforeDryRunOrWrite(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			parent := t.TempDir()
			root := filepath.Join(parent, "root")
			outside := filepath.Join(parent, "outside")
			if err := os.MkdirAll(root, 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(outside, 0o755); err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\nschema:\n  title:\n    type: string\n"), 0o644)
			mustWriteFile(t, filepath.Join(root, "record.md"), []byte("---\ntitle: Test\n---\n# Test\n"), 0o644)
			link := filepath.Join(root, "link")
			if err := os.Symlink(outside, link); err != nil {
				if errors.Is(err, os.ErrPermission) {
					t.Skipf("symlink unsupported: %v", err)
				}
				t.Fatal(err)
			}
			target := filepath.Join(link, ".stem")
			reportPath := writeSchemaApplyPlannerReport(t, root, []SchemaProposal{{ID: "escape", Operation: "create_stem", Target: target, Patch: schemaApplyPlannerPatchA}})
			args := []string{"--report", reportPath}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			if err == nil {
				t.Fatalf("schema apply accepted escaping symlink parent: %s", out)
			}
			result := decodeSchemaApplyResult(t, out)
			if result.Complete || len(result.Applied) != 0 || len(result.Errors) == 0 {
				t.Fatalf("unexpected envelope: %+v", result)
			}
			if _, statErr := os.Stat(filepath.Join(outside, ".stem")); !os.IsNotExist(statErr) {
				t.Fatalf("outside .stem was created: %v", statErr)
			}
		})
	}
}

func TestSchemaApply_ProposalNoOpsSkipProspectiveHealthOverInvalidHierarchy(t *testing.T) {
	for _, tc := range []struct {
		name         string
		proposals    func(root string) []SchemaProposal
		wantSkipped  bool
		wantRejected string
	}{
		{name: "empty", proposals: func(root string) []SchemaProposal { return nil }},
		{name: "requires_agent_only", proposals: func(root string) []SchemaProposal {
			return []SchemaProposal{{ID: "agent", Operation: "create_stem", Target: filepath.Join(root, "child", "agent.stem"), RequiresAgent: true}}
		}, wantSkipped: true},
		{name: "containment_rejected_only", proposals: func(root string) []SchemaProposal {
			return []SchemaProposal{{ID: "outside", Operation: "create_stem", Target: filepath.Join(root, "..", "outside.stem"), Patch: schemaApplyPlannerPatchA}}
		}, wantRejected: "escapes root"},
		{name: "overwrite_refused_only", proposals: func(root string) []SchemaProposal {
			return []SchemaProposal{{ID: "existing", Operation: "create_stem", Target: filepath.Join(root, "child", ".stem"), Patch: schemaApplyPlannerPatchA}}
		}, wantRejected: "already exists"},
	} {
		t.Run(tc.name, func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":       "version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n",
				"child/.stem": "version: 2\nschema:\n  estado:\n    type: string\n",
			})
			reportPath := writeSchemaApplyPlannerReport(t, root, tc.proposals(root))
			out, err := executeSchemaApply(t, "--report", reportPath, "--dry-run")
			if err != nil {
				t.Fatalf("policy-only proposal no-op evaluated invalid hierarchy: %v\n%s", err, out)
			}
			result := decodeSchemaApplyResult(t, out)
			if !result.Complete || len(result.Applied) != 0 || len(result.StemHealth) != 0 {
				t.Fatalf("unexpected no-op result: %+v", result)
			}
			if tc.wantSkipped && len(result.Skipped) == 0 {
				t.Fatalf("skipped = %#v, want requires-agent classification", result.Skipped)
			}
			if tc.wantRejected != "" && (len(result.Rejected) == 0 || !strings.Contains(result.Rejected[0], tc.wantRejected)) {
				t.Fatalf("rejected = %#v, want %q", result.Rejected, tc.wantRejected)
			}
		})
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
			if len(result.Errors) != 1 || !strings.Contains(result.Errors[0], "stat "+target) || !strings.Contains(result.Errors[0], "not a directory") {
				t.Fatalf("errors = %#v, want rooted stat failure naming %s and not-a-directory cause", result.Errors, target)
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
