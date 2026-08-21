package main

import (
	"path/filepath"
	"strings"
	"testing"

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
