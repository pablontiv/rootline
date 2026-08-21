package main

import (
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
)

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
