package main

import (
	"os"
	"path/filepath"
	"reflect"
	"runtime"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestSchemaApplyProposalValidatesPhysicalSymlinkTarget(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink semantics differ")
	}

	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":         "version: 2\nroot: true\nschema:\n  root_only:\n    type: string\n",
				"a/.stem":       "version: 2\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n",
				"a/deep/doc.md": "---\nestado: Pending\n---\n# Record\n",
			})
			if err := os.Symlink(filepath.Join("a", "deep"), filepath.Join(root, "link")); err != nil {
				t.Fatal(err)
			}
			if err := os.Symlink(filepath.Join("a", "deep"), filepath.Join(root, "other-link")); err != nil {
				t.Fatal(err)
			}

			logicalTarget := filepath.Join(root, "link", ".stem")
			physicalTarget := filepath.Join(root, "a", "deep", ".stem")
			otherAliasTarget := filepath.Join(root, "other-link", ".stem")
			report := SchemaProposalsReport{
				Version: 1,
				Kind:    "rootline/schema-proposals",
				Path:    filepath.Join(root, "link"),
				Root:    root,
				Proposals: []SchemaProposal{{
					ID:        "child-string",
					Operation: "create_stem",
					Target:    logicalTarget,
					Patch:     "version: 2\nschema:\n  estado:\n    type: string\n",
				}},
			}
			reportPath := filepath.Join(root, "report.json")
			writeSchemaApplyPreflightReport(t, reportPath, report)

			args := []string{"--report", reportPath}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			if err == nil {
				t.Fatalf("schema apply accepted physical symlink target conflict: %s", out)
			}
			result := decodeSchemaApplyResult(t, out)
			if result.Complete {
				t.Fatalf("complete = true, want false: %+v", result)
			}
			if len(result.Applied) != 0 {
				t.Fatalf("applied = %#v, want empty before physical hierarchy gate passes", result.Applied)
			}
			assertSchemaApplyHasStemHealth(t, result, filepath.Join("a", "deep", ".stem"), "type-consistency", "estado", rules.SeverityError)
			assertSchemaApplyAbsent(t, physicalTarget)
			assertSchemaApplyAbsent(t, logicalTarget)
			assertSchemaApplyAbsent(t, otherAliasTarget)
		})
	}
}

func TestSchemaApplyMalformedExternalAncestorBlocksProposalAndAnalyze(t *testing.T) {
	for _, reportKind := range []string{"rootline/schema-proposals", "rootline/analyze"} {
		for _, dryRun := range []bool{true, false} {
			t.Run(reportKind+"/dry_run="+map[bool]string{true: "true", false: "false"}[dryRun], func(t *testing.T) {
				project := setupValidateProject(t, map[string]string{
					".stem":               "version: 2\nroot: true\nschema:\n  title:\n    type: string\n",
					"subtree/.stem":       "version: [broken\n",
					"subtree/work/doc.md": "---\ntitle: Test\n---\n# Record\n",
				})
				workRoot := filepath.Join(project, "subtree", "work")
				malformedStem := filepath.Join(project, "subtree", ".stem")
				before := snapshotSchemaApplyFile(t, malformedStem)
				reportPath := filepath.Join(workRoot, "report.json")

				switch reportKind {
				case "rootline/schema-proposals":
					writeSchemaApplyPreflightReport(t, reportPath, SchemaProposalsReport{
						Version: 1,
						Kind:    reportKind,
						Path:    workRoot,
						Root:    workRoot,
						Proposals: []SchemaProposal{{
							ID:        "candidate",
							Operation: "create_stem",
							Target:    filepath.Join(workRoot, ".stem"),
							Patch:     "version: 2\nschema:\n  estado:\n    type: string\n",
						}},
					})
				case "rootline/analyze":
					writeSchemaApplyPreflightReport(t, reportPath, infer.AnalyzeReport{
						Version: 1,
						Kind:    reportKind,
						Path:    workRoot,
						Root:    workRoot,
						Categories: []infer.CategoryResult{{
							ID:         "field-types",
							Inferences: []infer.ReportInference{{Type: "field_type", Field: "estado", Value: "string"}},
						}},
					})
				default:
					t.Fatalf("unhandled report kind %q", reportKind)
				}

				args := []string{"--report", reportPath}
				if dryRun {
					args = append(args, "--dry-run")
				}
				out, err := executeSchemaApply(t, args...)
				if err == nil {
					t.Fatalf("schema apply accepted malformed external ancestor: %s", out)
				}
				result := decodeSchemaApplyResult(t, out)
				if len(result.StemHealth) != 1 {
					t.Fatalf("result = %+v; stem health = %+v, want one malformed external ancestor diagnostic", result, result.StemHealth)
				}
				got := result.StemHealth[0]
				if got.Path != filepath.Join("..", ".stem") || got.Check != "yaml-valid" || got.Severity != rules.SeverityError {
					t.Fatalf("stem health = %+v", result.StemHealth)
				}
				if len(result.Applied) != 0 || result.Complete {
					t.Fatalf("result = %+v", result)
				}
				assertSchemaApplyFileUnchanged(t, malformedStem, before)
				assertSchemaApplyAbsent(t, filepath.Join(workRoot, ".stem"))
			})
		}
	}
}

func TestSchemaApplyProposalRejectsEnumToStringChildBeforePublication(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":         "version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n",
				"sub/record.md": "---\nestado: Pending\n---\n# Record\n",
			})
			target := filepath.Join(root, "sub", ".stem")
			report := SchemaProposalsReport{
				Version: 1,
				Kind:    "rootline/schema-proposals",
				Path:    root,
				Root:    root,
				Proposals: []SchemaProposal{{
					ID:        "child-string",
					Operation: "create_stem",
					Target:    target,
					Patch:     "version: 2\nschema:\n  estado:\n    type: string\n",
				}},
			}
			reportPath := filepath.Join(root, "report.json")
			writeSchemaApplyPreflightReport(t, reportPath, report)

			args := []string{"--report", reportPath}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			if err == nil {
				t.Fatalf("schema apply accepted enum-to-string child proposal: %s", out)
			}
			result := decodeSchemaApplyResult(t, out)
			if result.Complete {
				t.Fatalf("complete = true, want false: %+v", result)
			}
			if len(result.Applied) != 0 {
				t.Fatalf("applied = %#v, want empty before prospective gate passes", result.Applied)
			}
			assertSchemaApplyAbsent(t, target)

			found := false
			for _, diag := range result.StemHealth {
				if diag.Path == filepath.Join("sub", ".stem") && diag.Check == "type-consistency" && diag.Field == "estado" && diag.Severity == rules.SeverityError {
					found = true
				}
			}
			if !found {
				t.Fatalf("stem_health missing type-consistency error for sub/.stem estado: %+v", result.StemHealth)
			}
			if len(result.Errors) == 0 {
				t.Fatalf("errors = empty, want blocking diagnostic converted into errors[]")
			}
			for _, want := range []string{filepath.Join("sub", ".stem"), "type-consistency", "estado"} {
				if !strings.Contains(result.Errors[0], want) {
					t.Fatalf("errors[0] = %q, want fragment %q", result.Errors[0], want)
				}
			}
		})
	}
}

func TestSchemaApplyProposalRejectsJointCandidatesThatOnlyConflictWhenComposed(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":              "version: 2\nroot: true\nschema:\n  title:\n    type: string\n",
				"docs/sub/record.md": "---\ntitle: Test\nestado: Pending\n---\n# Record\n",
			})
			parentTarget := filepath.Join(root, "docs", ".stem")
			childTarget := filepath.Join(root, "docs", "sub", ".stem")
			report := SchemaProposalsReport{
				Version: 1,
				Kind:    "rootline/schema-proposals",
				Path:    root,
				Root:    root,
				Proposals: []SchemaProposal{
					{
						ID:        "parent-enum",
						Operation: "create_stem",
						Target:    parentTarget,
						Patch:     "version: 2\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n",
					},
					{
						ID:        "child-string",
						Operation: "create_stem",
						Target:    childTarget,
						Patch:     "version: 2\nschema:\n  estado:\n    type: string\n",
					},
				},
			}
			reportPath := filepath.Join(root, "report.json")
			writeSchemaApplyPreflightReport(t, reportPath, report)

			args := []string{"--report", reportPath}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			if err == nil {
				t.Fatalf("schema apply accepted conflicting composed candidates: %s", out)
			}
			result := decodeSchemaApplyResult(t, out)
			if result.Complete || len(result.Applied) != 0 {
				t.Fatalf("invalid batch published or completed: %+v", result)
			}
			assertSchemaApplyAbsent(t, parentTarget)
			assertSchemaApplyAbsent(t, childTarget)
			found := false
			for _, diag := range result.StemHealth {
				if diag.Path == filepath.Join("docs", "sub", ".stem") && diag.Check == "type-consistency" && diag.Field == "estado" && diag.Severity == rules.SeverityError {
					found = true
				}
			}
			if !found {
				t.Fatalf("stem_health missing composed type-consistency error: %+v", result.StemHealth)
			}
		})
	}
}

func TestSchemaApplyAnalyzeRejectsProspectiveParentConflict(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			childRoot, childStem, reportPath := setupSchemaApplyAnalyzeParentConflict(t)
			before := snapshotSchemaApplyFile(t, childStem)

			args := []string{"--report", reportPath}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			if err == nil {
				t.Fatalf("schema apply accepted inherited-invalid analyze plan for %s: %s", childRoot, out)
			}
			result := decodeSchemaApplyResult(t, out)
			if result.Complete {
				t.Fatalf("complete = true, want false: %+v", result)
			}
			if len(result.Applied) != 0 {
				t.Fatalf("applied = %#v, want empty before prospective gate passes", result.Applied)
			}
			assertSchemaApplyFileUnchanged(t, childStem, before)
			assertSchemaApplyHasStemHealth(t, result, ".stem", "type-consistency", "estado", rules.SeverityError)
		})
	}
}

func TestSchemaApplyAnalyzeEvaluatesExternalCandidateAgainstExternalAncestor(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			grand := t.TempDir()
			parent := filepath.Join(grand, "parent")
			childRoot := filepath.Join(parent, "child")
			if err := os.MkdirAll(childRoot, 0o755); err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, filepath.Join(grand, ".stem"), []byte("version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n"), 0o644)
			parentStem := filepath.Join(parent, ".stem")
			mustWriteFile(t, parentStem, []byte("version: 2\nschema:\n  title:\n    type: string\n"), 0o644)
			mustWriteFile(t, filepath.Join(childRoot, "record.md"), []byte("---\ntitle: Test\nestado: Pending\n---\n# Record\n"), 0o644)
			before := snapshotSchemaApplyFile(t, parentStem)
			report := infer.AnalyzeReport{Version: 1, Kind: "rootline/analyze", Root: childRoot, Path: childRoot, Categories: []infer.CategoryResult{{ID: "type", Inferences: []infer.ReportInference{{Type: "field_type", Field: "estado", Value: "string"}}}}}
			reportPath := filepath.Join(childRoot, "analyze.json")
			writeSchemaApplyPreflightReport(t, reportPath, report)
			args := []string{"--report", reportPath}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			if err == nil {
				t.Fatalf("analyze accepted external candidate conflicting with external ancestor: %s", out)
			}
			result := decodeSchemaApplyResult(t, out)
			if result.Complete || len(result.Applied) != 0 {
				t.Fatalf("unexpected result: %+v", result)
			}
			assertSchemaApplyHasStemHealth(t, result, filepath.Join("..", ".stem"), "type-consistency", "estado", rules.SeverityError)
			assertSchemaApplyFileUnchanged(t, parentStem, before)
		})
	}
}

func TestSchemaApplyProposalRejectsCaseAliasBasenameBeforeResolution(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				"docs/.stem":  "version: 2\nroot: true\nschema:\n  title:\n    type: string\n",
				"docs/doc.md": "---\ntitle: Test\n---\n# Record\n",
			})
			skipIfCaseSensitiveSchemaApplyFixture(t, root)
			before := snapshotSchemaApplyFile(t, filepath.Join(root, "docs", ".stem"))
			report := SchemaProposalsReport{Version: 1, Kind: "rootline/schema-proposals", Path: root, Root: root, Proposals: []SchemaProposal{{ID: "case-alias", Operation: "create_stem", Target: filepath.Join(root, "Docs", ".STEM"), Patch: "version: 2\nschema:\n  title:\n    type: string\n"}}}
			reportPath := filepath.Join(root, "report.json")
			writeSchemaApplyPreflightReport(t, reportPath, report)
			args := []string{"--report", reportPath}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			if err != nil {
				t.Fatalf("schema apply rejected case alias basename: %v\noutput: %s", err, out)
			}
			result := decodeSchemaApplyResult(t, out)
			if !result.Complete || len(result.Applied) != 0 || len(result.Errors) != 0 || len(result.Skipped) != 0 {
				t.Fatalf("unexpected envelope: %+v", result)
			}
			if len(result.Rejected) != 1 || !strings.Contains(result.Rejected[0], "target basename must be \".stem\"") {
				t.Fatalf("rejected = %#v", result.Rejected)
			}
			if dryRun {
				if result.ResolvedTargets == nil {
					t.Fatalf("resolved_targets missing: %+v", result)
				}
				if got := result.ResolvedTargets.Rejected[filepath.Join(root, "Docs", ".STEM")]; got != "target basename must be \".stem\"" {
					t.Fatalf("resolved rejected = %q", got)
				}
			}
			assertSchemaApplyFileUnchanged(t, filepath.Join(root, "docs", ".stem"), before)
		})
	}
}

func TestSchemaApplyAnalyzeDryRunAndRealShareGovernanceVerdict(t *testing.T) {
	dryRoot, dryStem, dryReport := setupSchemaApplyAnalyzeParentConflict(t)
	dryBefore := snapshotSchemaApplyFile(t, dryStem)
	dryOut, dryErr := executeSchemaApply(t, "--report", dryReport, "--dry-run")
	if dryErr == nil {
		t.Fatalf("dry-run accepted inherited-invalid analyze plan for %s: %s", dryRoot, dryOut)
	}
	dryResult := decodeSchemaApplyResult(t, dryOut)
	assertSchemaApplyFileUnchanged(t, dryStem, dryBefore)

	realRoot, realStem, realReport := setupSchemaApplyAnalyzeParentConflict(t)
	realBefore := snapshotSchemaApplyFile(t, realStem)
	realOut, realErr := executeSchemaApply(t, "--report", realReport)
	if realErr == nil {
		t.Fatalf("real apply accepted inherited-invalid analyze plan for %s: %s", realRoot, realOut)
	}
	realResult := decodeSchemaApplyResult(t, realOut)
	assertSchemaApplyFileUnchanged(t, realStem, realBefore)

	if !dryResult.DryRun || realResult.DryRun {
		t.Fatalf("dry_run flags = dry:%v real:%v, want true/false", dryResult.DryRun, realResult.DryRun)
	}
	if len(dryResult.Applied) != 0 || len(realResult.Applied) != 0 {
		t.Fatalf("applied before gate: dry=%v real=%v", dryResult.Applied, realResult.Applied)
	}
	if !reflect.DeepEqual(normalizedSchemaApplyStemHealth(dryResult.StemHealth, "child"), normalizedSchemaApplyStemHealth(realResult.StemHealth, "child")) {
		t.Fatalf("dry-run and real stem_health differ:\ndry=%+v\nreal=%+v", dryResult.StemHealth, realResult.StemHealth)
	}
}

func TestSchemaApplyAnalyzeNoOpSkipsProspectiveHealth(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":       "version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n",
		"child/.stem": "version: 2\nschema:\n  estado:\n    type: string\n",
	})
	childRoot := filepath.Join(root, "child")
	childStem := filepath.Join(childRoot, ".stem")
	before := snapshotSchemaApplyFile(t, childStem)
	report := infer.AnalyzeReport{
		Version: 1,
		Kind:    "rootline/analyze",
		Path:    childRoot,
		Root:    childRoot,
		Categories: []infer.CategoryResult{{
			ID: "data-only", Inferences: []infer.ReportInference{{Type: "migrate_value", Field: "estado"}},
		}},
	}
	reportPath := filepath.Join(childRoot, "analyze.json")
	writeSchemaApplyPreflightReport(t, reportPath, report)

	out, err := executeSchemaApply(t, "--report", reportPath, "--dry-run")
	if err != nil {
		t.Fatalf("no-op analyze dry-run evaluated unrelated prospective health: %v\noutput: %s", err, out)
	}
	result := decodeSchemaApplyResult(t, out)
	if !result.Complete || len(result.Applied) != 0 || len(result.StemHealth) != 0 {
		t.Fatalf("no-op analyze envelope = %+v, want success without applied actions or prospective diagnostics", result)
	}
	assertSchemaApplyFileUnchanged(t, childStem, before)
}

func TestSchemaApplyAnalyzePreservesModeAfterAtomicWrite(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":        "version: 2\nroot: true\nschema:\n  title:\n    type: string\n",
		"child/.stem":  "version: 2\nschema:\n  estado:\n    type: string\n",
		"child/doc.md": "---\ntitle: Test\nestado: Pending\n---\n# Record\n",
	})
	childRoot := filepath.Join(root, "child")
	childStem := filepath.Join(childRoot, ".stem")
	if err := os.Chmod(childStem, 0o600); err != nil {
		t.Fatal(err)
	}
	report := infer.AnalyzeReport{
		Version: 1,
		Kind:    "rootline/analyze",
		Path:    childRoot,
		Root:    childRoot,
		Categories: []infer.CategoryResult{{
			ID: "required", Inferences: []infer.ReportInference{{Type: "required_field", Field: "estado"}},
		}},
	}
	reportPath := filepath.Join(childRoot, "analyze.json")
	writeSchemaApplyPreflightReport(t, reportPath, report)

	out, err := executeSchemaApply(t, "--report", reportPath)
	if err != nil {
		t.Fatalf("schema apply rejected valid analyze plan: %v\noutput: %s", err, out)
	}
	result := decodeSchemaApplyResult(t, out)
	if !result.Complete || len(result.Errors) != 0 {
		t.Fatalf("success envelope changed contract: %+v", result)
	}
	assertSchemaApplyPlannerApplied(t, result.Applied, []string{"add_required: estado"})
	info, err := os.Stat(childStem)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("schema apply changed .stem mode: got %v want 0600", got)
	}
	if got := string(mustReadFile(t, childStem)); !strings.Contains(got, "required: true") {
		t.Fatalf("written stem missing required mutation:\n%s", got)
	}
}

func TestSchemaApplyAnalyzeAndProposalEmitEquivalentProspectiveDiagnostics(t *testing.T) {
	proposalHealth := runSchemaApplyProposalParentConflictHealth(t)
	analyzeHealth := runSchemaApplyAnalyzeParentConflictHealth(t)

	if !reflect.DeepEqual(proposalHealth, analyzeHealth) {
		t.Fatalf("proposal/analyze prospective diagnostics differ:\nproposal=%+v\nanalyze=%+v", proposalHealth, analyzeHealth)
	}
}

func TestSchemaApplyAnalyzeCategoryOrderDoesNotChangeDryRunEnvelope(t *testing.T) {
	run := func(t *testing.T, categories []infer.CategoryResult) *SchemaApplyResult {
		t.Helper()
		root := setupValidateProject(t, map[string]string{
			".stem":   "version: 2\nroot: true\nscope:\n  match: \"*.txt\"\nschema:\n  title:\n    type: string\n",
			"doc.md":  "---\ntitle: Test\nstatus: Pending\npriority: High\n---\n# Record\n",
			"note.md": "---\ntitle: Note\nstatus: Done\npriority: Low\n---\n# Note\n",
		})
		report := infer.AnalyzeReport{Version: 1, Kind: "rootline/analyze", Root: root, Path: root, Categories: categories}
		reportPath := filepath.Join(root, "analyze.json")
		writeSchemaApplyPreflightReport(t, reportPath, report)
		out, err := executeSchemaApply(t, "--report", reportPath, "--dry-run")
		if err != nil {
			t.Fatalf("schema apply dry-run failed: %v\n%s", err, out)
		}
		return decodeSchemaApplyResult(t, out)
	}
	ascending := []infer.CategoryResult{
		{ID: "priority", Inferences: []infer.ReportInference{{Type: "field_type", Field: "priority", Value: "enum"}, {Type: "enum_values", Field: "priority", Value: "[Low High]"}}},
		{ID: "status", Inferences: []infer.ReportInference{{Type: "field_type", Field: "status", Value: "string"}, {Type: "required_field", Field: "status"}, {Type: "constant_field", Field: "status", Value: "Pending"}}},
	}
	reversed := []infer.CategoryResult{ascending[1], ascending[0]}
	gotA := run(t, ascending)
	gotB := run(t, reversed)
	if gotA.Complete != gotB.Complete || !reflect.DeepEqual(gotA.Applied, gotB.Applied) || !reflect.DeepEqual(gotA.StemHealth, gotB.StemHealth) || !reflect.DeepEqual(gotA.Errors, gotB.Errors) {
		t.Fatalf("category order changed envelope:\nA=%+v\nB=%+v", gotA, gotB)
	}
	assertSchemaApplyHasStemHealth(t, gotA, ".stem", "scope-match", "", rules.SeverityWarn)
}

func setupSchemaApplyAnalyzeParentConflict(t *testing.T) (childRoot, childStem, reportPath string) {
	t.Helper()
	root := setupValidateProject(t, map[string]string{
		".stem":        "version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n",
		"child/.stem":  "version: 2\nschema:\n  title:\n    type: string\n",
		"child/doc.md": "---\ntitle: Test\nestado: Pending\n---\n# Record\n",
	})
	childRoot = filepath.Join(root, "child")
	childStem = filepath.Join(childRoot, ".stem")
	report := infer.AnalyzeReport{
		Version: 1,
		Kind:    "rootline/analyze",
		Path:    childRoot,
		Root:    childRoot,
		Categories: []infer.CategoryResult{{
			ID: "type-widening", Inferences: []infer.ReportInference{{Type: "field_type", Field: "estado", Value: "string"}},
		}},
	}
	reportPath = filepath.Join(childRoot, "analyze.json")
	writeSchemaApplyPreflightReport(t, reportPath, report)
	return childRoot, childStem, reportPath
}

type schemaApplyStemHealthKey struct {
	Path     string
	Check    string
	Field    string
	Severity string
}

func normalizedSchemaApplyStemHealth(diags []rules.StemHealthDiagnostic, analyzeDirName string) []schemaApplyStemHealthKey {
	keys := make([]schemaApplyStemHealthKey, 0, len(diags))
	for _, diag := range diags {
		path := diag.Path
		if analyzeDirName != "" && path == ".stem" {
			path = filepath.Join(analyzeDirName, ".stem")
		}
		keys = append(keys, schemaApplyStemHealthKey{Path: path, Check: diag.Check, Field: diag.Field, Severity: diag.Severity})
	}
	return keys
}

func assertSchemaApplyHasStemHealth(t *testing.T, result *SchemaApplyResult, path, check, field, severity string) {
	t.Helper()
	for _, diag := range result.StemHealth {
		if diag.Path == path && diag.Check == check && diag.Field == field && diag.Severity == severity {
			return
		}
	}
	t.Fatalf("stem_health missing %s/%s/%s/%s: %+v", path, check, field, severity, result.StemHealth)
}

func runSchemaApplyProposalParentConflictHealth(t *testing.T) []schemaApplyStemHealthKey {
	t.Helper()
	root := setupValidateProject(t, map[string]string{
		".stem":        "version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n",
		"child/.stem":  "version: 2\nschema:\n  title:\n    type: string\n",
		"child/doc.md": "---\ntitle: Test\nestado: Pending\n---\n# Record\n",
	})
	childTarget := filepath.Join(root, "child", ".stem")
	report := SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/schema-proposals",
		Path:    root,
		Root:    root,
		Proposals: []SchemaProposal{{
			ID:        "child-string",
			Operation: "create_stem",
			Target:    childTarget,
			Patch:     "version: 2\nschema:\n  title:\n    type: string\n  estado:\n    type: string\n",
		}},
	}
	reportPath := filepath.Join(root, "proposal.json")
	writeSchemaApplyPreflightReport(t, reportPath, report)
	out, err := executeSchemaApply(t, "--report", reportPath, "--force")
	if err == nil {
		t.Fatalf("proposal accepted inherited-invalid candidate: %s", out)
	}
	result := decodeSchemaApplyResult(t, out)
	return normalizedSchemaApplyStemHealth(result.StemHealth, "")
}

func runSchemaApplyAnalyzeParentConflictHealth(t *testing.T) []schemaApplyStemHealthKey {
	t.Helper()
	_, _, reportPath := setupSchemaApplyAnalyzeParentConflict(t)
	out, err := executeSchemaApply(t, "--report", reportPath)
	if err == nil {
		t.Fatalf("analyze accepted inherited-invalid candidate: %s", out)
	}
	result := decodeSchemaApplyResult(t, out)
	return normalizedSchemaApplyStemHealth(result.StemHealth, "child")
}

func TestSchemaApplyProposalPublishesWarningOnlyStemHealth(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":        "version: 2\nroot: true\nschema:\n  title:\n    type: string\n",
				"docs/doc.md":  "---\ntitle: Test\n---\n# Record\n",
				"docs/note.md": "---\ntitle: Note\n---\n# Note\n",
			})
			target := filepath.Join(root, "docs", ".stem")
			report := SchemaProposalsReport{
				Version: 1,
				Kind:    "rootline/schema-proposals",
				Path:    root,
				Root:    root,
				Proposals: []SchemaProposal{{
					ID:        "warning-only",
					Operation: "create_stem",
					Target:    target,
					Patch:     "version: 2\nscope:\n  match: \"*.txt\"\nschema:\n  title:\n    type: string\n",
				}},
			}
			reportPath := filepath.Join(root, "report.json")
			writeSchemaApplyPreflightReport(t, reportPath, report)

			args := []string{"--report", reportPath}
			if dryRun {
				args = append(args, "--dry-run")
			}
			out, err := executeSchemaApply(t, args...)
			if err != nil {
				t.Fatalf("schema apply rejected warning-only stem health: %v\noutput: %s", err, out)
			}
			result := decodeSchemaApplyResult(t, out)
			if !result.Complete || len(result.Errors) != 0 {
				t.Fatalf("warning-only run should complete without errors: %+v", result)
			}
			assertSchemaApplyPlannerApplied(t, result.Applied, []string{"create_stem: " + target})
			foundScopeWarning := false
			for _, diag := range result.StemHealth {
				if diag.Severity == rules.SeverityError {
					t.Fatalf("warning-only fixture produced blocking diagnostic: %+v", diag)
				}
				if diag.Path == filepath.Join("docs", ".stem") && diag.Check == "scope-match" && diag.Severity == rules.SeverityWarn {
					foundScopeWarning = true
				}
			}
			if !foundScopeWarning {
				t.Fatalf("stem_health missing retained scope-match warning: %+v", result.StemHealth)
			}
			if dryRun {
				assertSchemaApplyAbsent(t, target)
			} else if got := string(mustReadFile(t, target)); !strings.Contains(got, "*.txt") {
				t.Fatalf("written stem = %q, want proposal patch", got)
			}
		})
	}
}
