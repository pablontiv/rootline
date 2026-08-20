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

type schemaApplyFileSnapshot struct {
	bytes []byte
	mode  os.FileMode
}

func snapshotSchemaApplyFile(t *testing.T, path string) schemaApplyFileSnapshot {
	t.Helper()
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	return schemaApplyFileSnapshot{bytes: mustReadFile(t, path), mode: info.Mode()}
}

func assertSchemaApplyFileUnchanged(t *testing.T, path string, before schemaApplyFileSnapshot) {
	t.Helper()
	got := mustReadFile(t, path)
	if string(got) != string(before.bytes) {
		t.Fatalf("%s bytes changed:\nbefore:\n%s\nafter:\n%s", path, before.bytes, got)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatalf("stat %s: %v", path, err)
	}
	if info.Mode() != before.mode {
		t.Fatalf("%s mode changed: got %v want %v", path, info.Mode(), before.mode)
	}
}

func assertSchemaApplyAbsent(t *testing.T, path string) {
	t.Helper()
	if _, statErr := os.Stat(path); !os.IsNotExist(statErr) {
		t.Fatalf("%s exists after rejected apply: %v", path, statErr)
	}
}

func assertSchemaApplyProspectiveFailure(t *testing.T, out string, err error, requiredFragments ...string) *SchemaApplyResult {
	t.Helper()
	if err == nil {
		t.Fatalf("schema apply accepted invalid prospective schema: %s", out)
	}
	result := decodeSchemaApplyResult(t, out)
	if result.Version != 1 || result.Kind != "rootline/schema-apply" || result.Complete {
		t.Fatalf("failure envelope changed contract: %+v", result)
	}
	if len(result.Errors) != 1 {
		t.Fatalf("errors = %v, want exactly one deterministic prospective validation cause", result.Errors)
	}
	if len(result.Applied) != 0 {
		t.Fatalf("invalid prospective schema published applied actions: %+v", result)
	}
	for _, want := range requiredFragments {
		if !strings.Contains(result.Errors[0], want) {
			t.Fatalf("error %q does not retain %q", result.Errors[0], want)
		}
	}
	return result
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

func TestSchemaApply_RejectsValidBeforeInvalidProposalBatchBeforePublicationOrWrite(t *testing.T) {
	validPatch := "version: 2\nroot: true\nschema:\n  estado:\n    type: string\n"
	invalidPatch := "version: 2\nschema:\n  id:\n    type: sequence\n    prefix: T\n    digits: 2\n    match:\n      \"T*\": {prefix: T, digits: 2.0}\n"

	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry_run", false: "write"}[dryRun], func(t *testing.T) {
			root := filepath.Join(t.TempDir(), "governed")
			if err := os.MkdirAll(filepath.Join(root, "existing"), 0o755); err != nil {
				t.Fatal(err)
			}
			if err := os.MkdirAll(filepath.Join(root, "absent"), 0o755); err != nil {
				t.Fatal(err)
			}
			mustWriteFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\nschema:\n  estado:\n    type: string\n"), 0o644)
			mustWriteFile(t, filepath.Join(root, "record.md"), []byte("---\nestado: Pending\n---\n# Record\n"), 0o644)

			existingTarget := filepath.Join(root, "existing", ".stem")
			existingBeforeBytes := []byte("version: 2\nschema:\n  old:\n    type: string\n")
			mustWriteFile(t, existingTarget, existingBeforeBytes, 0o600)
			if err := os.Chmod(existingTarget, 0o600); err != nil {
				t.Fatal(err)
			}
			existingBefore := snapshotSchemaApplyFile(t, existingTarget)
			absentTarget := filepath.Join(root, "absent", ".stem")
			assertSchemaApplyAbsent(t, absentTarget)

			report := SchemaProposalsReport{
				Version: 1, Kind: "rootline/schema-proposals", Path: root, Root: root,
				Proposals: []SchemaProposal{
					{ID: "valid-existing", Operation: "create_stem", Target: existingTarget, Patch: validPatch},
					{ID: "invalid-absent", Operation: "create_stem", Target: absentTarget, Patch: invalidPatch},
				},
			}
			reportPath := filepath.Join(root, "report.json")
			writeSchemaApplyPreflightReport(t, reportPath, report)
			args := []string{"--report", reportPath, "--force"}
			if dryRun {
				args = append(args, "--dry-run")
			}

			out, err := executeSchemaApply(t, args...)
			assertSchemaApplyProspectiveFailure(t, out, err, absentTarget, "id", "T*", "digits")
			assertSchemaApplyFileUnchanged(t, existingTarget, existingBefore)
			assertSchemaApplyAbsent(t, absentTarget)
		})
	}
}

func TestSchemaApply_AnalyzeProspectiveValidationRejectsRetainedMatchMutationBeforePublicationOrWrite(t *testing.T) {
	stem := `version: 2
root: true
scope:
  match: "*.md"
schema:
  id:
    prefix: QX
    digits: 2
    match:
      "QXBAD*": {prefix: QX, digits: 2.0}
  estado:
    type: string
`
	for _, kind := range []string{"analyze", "rootline/analyze"} {
		for _, dryRun := range []bool{true, false} {
			t.Run(kind+"/dry_run="+map[bool]string{true: "true", false: "false"}[dryRun], func(t *testing.T) {
				root := filepath.Join(t.TempDir(), "governed")
				if err := os.MkdirAll(root, 0o755); err != nil {
					t.Fatal(err)
				}
				stemPath := filepath.Join(root, ".stem")
				mustWriteFile(t, stemPath, []byte(stem), 0o600)
				if err := os.Chmod(stemPath, 0o600); err != nil {
					t.Fatal(err)
				}
				mustWriteFile(t, filepath.Join(root, "QXBAD001-task.md"), []byte("---\nestado: Pending\n---\n# Task\n"), 0o644)
				beforeStem := snapshotSchemaApplyFile(t, stemPath)

				report := infer.AnalyzeReport{
					Version: 1, Kind: kind, Path: root, Root: root,
					Categories: []infer.CategoryResult{{
						ID: "retained-match", Inferences: []infer.ReportInference{{Type: "untyped_field", Field: "id", Value: "sequence"}},
					}},
				}
				reportPath := filepath.Join(root, "analyze.json")
				writeSchemaApplyPreflightReport(t, reportPath, report)
				args := []string{"--report", reportPath}
				if dryRun {
					args = append(args, "--dry-run")
				}

				out, err := executeSchemaApply(t, args...)
				assertSchemaApplyProspectiveFailure(t, out, err, "applying schema", "id", "QXBAD*", "digits")
				assertSchemaApplyFileUnchanged(t, stemPath, beforeStem)
			})
		}
	}
}

func TestSchemaApply_AnalyzeProspectiveValidationRejectsIncompleteTypedDeclaration(t *testing.T) {
	for _, tc := range []struct {
		name      string
		field     string
		value     string
		wantCause string
	}{
		{name: "enum_without_values", field: "estado", value: "enum", wantCause: "at least one value"},
		{name: "sequence_without_config", field: "id", value: "sequence", wantCause: "prefix and positive digits"},
	} {
		for _, dryRun := range []bool{true, false} {
			t.Run(tc.name+"/dry_run="+map[bool]string{true: "true", false: "false"}[dryRun], func(t *testing.T) {
				root := filepath.Join(t.TempDir(), "governed")
				if err := os.MkdirAll(root, 0o755); err != nil {
					t.Fatal(err)
				}
				stemPath := filepath.Join(root, ".stem")
				mustWriteFile(t, stemPath, []byte("version: 2\nroot: true\nschema:\n  title:\n    type: string\n"), 0o600)
				if err := os.Chmod(stemPath, 0o600); err != nil {
					t.Fatal(err)
				}
				mustWriteFile(t, filepath.Join(root, "record.md"), []byte("---\ntitle: Task\n---\n# Task\n"), 0o644)
				beforeStem := snapshotSchemaApplyFile(t, stemPath)
				report := infer.AnalyzeReport{
					Version: 1, Kind: "rootline/analyze", Path: root, Root: root,
					Categories: []infer.CategoryResult{{
						ID: "incomplete", Inferences: []infer.ReportInference{{Type: "field_type", Field: tc.field, Value: tc.value}},
					}},
				}
				reportPath := filepath.Join(root, "analyze.json")
				writeSchemaApplyPreflightReport(t, reportPath, report)
				args := []string{"--report", reportPath}
				if dryRun {
					args = append(args, "--dry-run")
				}

				out, err := executeSchemaApply(t, args...)
				assertSchemaApplyProspectiveFailure(t, out, err, "applying schema", tc.field, tc.wantCause)
				assertSchemaApplyFileUnchanged(t, stemPath, beforeStem)
			})
		}
	}
}
