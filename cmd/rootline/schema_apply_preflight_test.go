package main

import (
	"encoding/json"
	"github.com/pablontiv/rootline/internal/infer"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const schemaApplyMalformedOverlayStem = `version: 2
root: true
scope:
  match: "*.md"
schema:
  id:
    type: sequence
    prefix: QX
    digits: 2
    match:
      "QXBAD*": {prefix: QX, digits: 2.0}
  estado:
    type: string
`

func writeSchemaApplyPreflightReport(t *testing.T, path string, report any) {
	t.Helper()
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, path, data, 0o644)
}
func TestSchemaApply_RejectsInvalidProposedOverlayBeforeWriteOrDryRun(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "governed")
			if err := os.MkdirAll(filepath.Join(root, "candidate"), 0o755); err != nil {
				t.Fatal(err)
			}
			stemPath := filepath.Join(root, ".stem")
			mustWriteFile(t, stemPath, []byte("version: 2\nroot: true\nschema:\n  estado:\n    type: string\n"), 0o644)
			mustWriteFile(t, filepath.Join(root, "record.md"), []byte("---\nestado: Pending\n---\n# Record\n"), 0o644)
			beforeStem := mustReadFile(t, stemPath)
			target := filepath.Join(root, "candidate", ".stem")
			report := SchemaProposalsReport{
				Version: 1, Kind: "rootline/schema-proposals", Path: root, Root: root,
				Proposals: []SchemaProposal{{
					ID: "invalid-overlay", Operation: "create_stem", Target: target,
					Patch: "version: 2\nschema:\n  id:\n    type: sequence\n    prefix: T\n    digits: 2\n    match:\n      \"T*\": {prefix: T, digits: 2.0}\n",
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
				t.Fatalf("schema apply accepted invalid proposed overlay: %s", out)
			}
			result := decodeSchemaApplyResult(t, out)
			if result.Version != 1 || result.Kind != "rootline/schema-apply" || result.Complete || len(result.Errors) != 1 {
				t.Fatalf("unexpected failure envelope: %+v", result)
			}
			for _, want := range []string{target, "id", "T*", "digits"} {
				if !strings.Contains(result.Errors[0], want) {
					t.Fatalf("error %q does not retain %q", result.Errors[0], want)
				}
			}
			if len(result.Applied) != 0 || string(mustReadFile(t, stemPath)) != string(beforeStem) {
				t.Fatalf("invalid proposal published or changed an existing target: %+v", result)
			}
			if _, statErr := os.Stat(target); !os.IsNotExist(statErr) {
				t.Fatalf("invalid proposal created absent target: %v", statErr)
			}
		})
	}
}
func TestSchemaApply_PreflightsGovernedRecordsBeforeWriteOrDryRun(t *testing.T) {
	for _, reportKind := range []string{"schema-proposals", "analyze"} {
		for _, dryRun := range []bool{true, false} {
			t.Run(reportKind+"/dry_run="+map[bool]string{true: "true", false: "false"}[dryRun], func(t *testing.T) {
				root := filepath.Join(t.TempDir(), "governed")
				if err := os.MkdirAll(filepath.Join(root, "candidate"), 0o755); err != nil {
					t.Fatal(err)
				}
				stemPath := filepath.Join(root, ".stem")
				mustWriteFile(t, stemPath, []byte(schemaApplyMalformedOverlayStem), 0o644)
				mustWriteFile(t, filepath.Join(root, "QXBAD001-task.md"), []byte("---\nestado: Pending\n---\n# Task\n"), 0o644)
				beforeStem := mustReadFile(t, stemPath)
				reportPath := filepath.Join(root, "report.json")
				var absentTarget string
				if reportKind == "schema-proposals" {
					absentTarget = filepath.Join(root, "candidate", ".stem")
					report := SchemaProposalsReport{
						Version: 1, Kind: "rootline/schema-proposals", Path: root, Root: root,
						Proposals: []SchemaProposal{{
							ID: "real-bootstrap", Operation: "create_stem", Target: absentTarget,
							Patch: "version: 2\nschema:\n  estado:\n    type: string\n",
						}},
					}
					writeSchemaApplyPreflightReport(t, reportPath, report)
				} else {
					report := infer.AnalyzeReport{
						Version: 1, Kind: "rootline/analyze", Path: root, Root: root,
						Categories: []infer.CategoryResult{{
							ID: "real-operation", Inferences: []infer.ReportInference{{Type: "required_field", Field: "estado"}},
						}},
					}
					writeSchemaApplyPreflightReport(t, reportPath, report)
				}
				args := []string{"--report", reportPath}
				if dryRun {
					args = append(args, "--dry-run")
				}
				out, err := executeSchemaApply(t, args...)
				if err == nil {
					t.Fatalf("schema apply accepted invalid governance: %s", out)
				}
				result := decodeSchemaApplyResult(t, out)
				if result.Version != 1 || result.Kind != "rootline/schema-apply" || result.Complete {
					t.Fatalf("failure envelope changed contract: %+v", result)
				}
				if len(result.Errors) != 1 {
					t.Fatalf("errors = %v, want one governed-record resolution failure", result.Errors)
				}
				for _, want := range []string{"QXBAD001-task.md", "id", "QXBAD*", "digits"} {
					if !strings.Contains(result.Errors[0], want) {
						t.Fatalf("error %q does not retain %q", result.Errors[0], want)
					}
				}
				if len(result.Applied) != 0 {
					t.Fatalf("invalid governance published an operation: %+v", result)
				}
				if got := mustReadFile(t, stemPath); string(got) != string(beforeStem) {
					t.Fatalf("existing target changed before resolution failure:\nbefore:\n%s\nafter:\n%s", beforeStem, got)
				}
				if absentTarget != "" {
					if _, statErr := os.Stat(absentTarget); !os.IsNotExist(statErr) {
						t.Fatalf("absent proposal target was created: %v", statErr)
					}
				}
			})
		}
	}
}
