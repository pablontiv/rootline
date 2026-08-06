package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/proposal"
)

const reportRootStem = `version: 2
root: true
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Done]
`

// chdir moves the process into dir for the duration of the test. The whole
// point of this file is that the working directory must NOT decide the outcome,
// so the tests have to actually run from somewhere else.
func chdir(t *testing.T, dir string) {
	t.Helper()

	previous, err := os.Getwd()
	if err != nil {
		t.Fatalf("reading working directory: %v", err)
	}
	if err := os.Chdir(dir); err != nil {
		t.Fatalf("entering %s: %v", dir, err)
	}
	t.Cleanup(func() {
		if err := os.Chdir(previous); err != nil {
			t.Fatalf("restoring working directory: %v", err)
		}
	})
}

// newSplitLayout builds the shape the issue describes and CI actually uses: a
// governed docs/ directory, a sibling reports/ directory holding the report,
// and a third directory that is neither, to run the commands from.
func newSplitLayout(t *testing.T) (docsDir, reportsDir, elsewhere string) {
	t.Helper()

	base := t.TempDir()
	docsDir = filepath.Join(base, "proj", "docs")
	reportsDir = filepath.Join(base, "reports")
	elsewhere = filepath.Join(base, "elsewhere")

	for _, dir := range []string{docsDir, reportsDir, elsewhere} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatalf("creating %s: %v", dir, err)
		}
	}

	mustWriteFile(t, filepath.Join(docsDir, ".stem"), []byte(reportRootStem), 0o644)
	mustWriteFile(t, filepath.Join(docsDir, "a.md"),
		[]byte("---\nestado: Pendiente\n---\n\n# A\n"), 0o644)

	return docsDir, reportsDir, elsewhere
}

// TestApplyCommandsAgreeOnReportRoot is the acceptance test for issue #61's
// third sub-defect. It generates a report the documented way, stores it in a
// sibling directory, and runs BOTH report-consuming commands from a working
// directory that is neither the report's nor the scanned one — then requires
// that they resolved the same root.
//
// Before this change `repair apply` resolved against the report FILE's
// directory and `schema apply` against the recorded scan root, so this exact
// layout made `repair apply` a silent no-op while `schema apply` worked.
func TestApplyCommandsAgreeOnReportRoot(t *testing.T) {
	docsDir, reportsDir, elsewhere := newSplitLayout(t)

	// Produce the report exactly as `repair apply --report`'s help text says to.
	out, err := runCmd(t, "fix", "--all", docsDir, "--dry-run", "-o", "json")
	if err != nil {
		t.Fatalf("generating the report: %v\noutput: %s", err, out)
	}
	var generated proposal.Report
	if err := json.Unmarshal([]byte(out), &generated); err != nil {
		t.Fatalf("decoding the generated report: %v\noutput: %s", err, out)
	}
	if generated.Root == "" {
		t.Fatal("fix --all recorded no root, so a consumer has nothing to resolve against")
	}
	if len(generated.Proposals) == 0 {
		t.Fatal("fix --all proposed nothing, so this fixture cannot prove anything")
	}

	repairReport := filepath.Join(reportsDir, "repairs.json")
	mustWriteFile(t, repairReport, []byte(out), 0o644)

	// Build a schema report for the same scan root, stored in the same place.
	schemaReport := filepath.Join(reportsDir, "schema.json")
	schemaData, err := json.Marshal(SchemaProposalsReport{
		Version: 1,
		Kind:    "rootline/schema-proposals",
		Path:    docsDir,
		Root:    docsDir,
		Proposals: []SchemaProposal{{
			ID:            "deferred",
			Operation:     "create_stem",
			Target:        filepath.Join(docsDir, ".stem"),
			RequiresAgent: true,
		}},
	})
	if err != nil {
		t.Fatalf("marshalling the schema report: %v", err)
	}
	mustWriteFile(t, schemaReport, schemaData, 0o644)

	// Run from a third directory. Neither command may consult it.
	chdir(t, elsewhere)

	repairOut, err := executeRepairApply(t, "--report", repairReport, "-o", "json")
	if err != nil {
		t.Fatalf("repair apply failed: %v\noutput: %s", err, repairOut)
	}
	repairResult := decodeRepairResult(t, repairOut)

	schemaOut, err := executeSchemaApply(t, "--report", schemaReport)
	if err != nil {
		t.Fatalf("schema apply failed: %v\noutput: %s", err, schemaOut)
	}
	schemaResult := decodeSchemaApplyResult(t, schemaOut)

	// The claim under test: one rule, one answer.
	if repairResult.Root != schemaResult.Root {
		t.Fatalf("the two commands resolved different roots from the same layout:\n  repair apply: %q\n  schema apply: %q",
			repairResult.Root, schemaResult.Root)
	}
	if repairResult.Root != docsDir {
		t.Errorf("resolved root = %q, want the scanned directory %q (not the report's directory %q, nor the working directory %q)",
			repairResult.Root, docsDir, reportsDir, elsewhere)
	}

	// And the resolution must be real, not merely reported: the document the
	// report named has to have actually been repaired.
	if len(repairResult.Changed) == 0 {
		t.Fatalf("repair apply changed nothing, so it resolved to a directory with no documents: %+v", repairResult)
	}
	content, err := os.ReadFile(filepath.Join(docsDir, "a.md"))
	if err != nil {
		t.Fatalf("re-reading the document: %v", err)
	}
	if !strings.Contains(string(content), "estado: Pending") {
		t.Errorf("a.md was not repaired:\n%s", content)
	}
}

// TestRepairApplyRootPrecedenceEndToEnd walks the rungs below the happy path:
// an explicit --root beats what the report recorded, and a report predating the
// recorded fields still resolves the way it always did.
func TestRepairApplyRootPrecedenceEndToEnd(t *testing.T) {
	writeReport := func(t *testing.T, path, reportPath, reportRoot string) {
		t.Helper()
		data, err := json.Marshal(proposal.Report{
			Version: 1,
			Kind:    "rootline/proposals",
			Path:    reportPath,
			Root:    reportRoot,
			Proposals: []proposal.Proposal{{
				Type:        proposal.CorrectValue,
				Field:       "estado",
				From:        "Pendiente",
				To:          "Pending",
				Paths:       []string{"a.md"},
				Description: "correct estado",
			}},
		})
		if err != nil {
			t.Fatalf("marshalling report: %v", err)
		}
		mustWriteFile(t, path, data, 0o644)
	}

	t.Run("--root overrides the root the report recorded", func(t *testing.T) {
		docsDir, reportsDir, elsewhere := newSplitLayout(t)
		decoy := filepath.Join(filepath.Dir(reportsDir), "decoy")
		if err := os.MkdirAll(decoy, 0o755); err != nil {
			t.Fatalf("creating decoy: %v", err)
		}

		report := filepath.Join(reportsDir, "r.json")
		writeReport(t, report, decoy, decoy)

		chdir(t, elsewhere)
		out, err := executeRepairApply(t, "--report", report, "--root", docsDir, "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeRepairResult(t, out)
		if result.Root != docsDir {
			t.Errorf("root = %q, want the --root value %q", result.Root, docsDir)
		}
		if len(result.Changed) == 0 {
			t.Errorf("nothing changed, so --root did not actually redirect the run: %+v", result)
		}
	})

	t.Run("a report with no recorded root falls back to the report's own directory", func(t *testing.T) {
		docsDir, _, elsewhere := newSplitLayout(t)

		// The legacy shape: report stored beside the documents, no path/root.
		report := filepath.Join(docsDir, "r.json")
		writeReport(t, report, "", "")

		chdir(t, elsewhere)
		out, err := executeRepairApply(t, "--report", report, "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeRepairResult(t, out)
		if result.Root != docsDir {
			t.Errorf("root = %q, want the report's own directory %q", result.Root, docsDir)
		}
		if len(result.Changed) == 0 {
			t.Errorf("the legacy fallback stopped working: %+v", result)
		}
	})
}

// TestSchemaApplyAnalyzeReportRoot covers the third report-consuming path.
// `schema apply` accepts an analyze report as well as a proposals report, and
// that branch used to resolve on its own rule, so --root did not reach it and
// its envelope never said which directory it had chosen.
func TestSchemaApplyAnalyzeReportRoot(t *testing.T) {
	newAnalyzeReport := func(t *testing.T, dir, path, root string) string {
		t.Helper()
		data, err := json.Marshal(infer.AnalyzeReport{
			Version:    1,
			Kind:       "rootline/analyze",
			Path:       path,
			Root:       root,
			Categories: []infer.CategoryResult{},
		})
		if err != nil {
			t.Fatalf("marshalling analyze report: %v", err)
		}
		reportFile := filepath.Join(dir, "analyze.json")
		mustWriteFile(t, reportFile, data, 0o644)
		return reportFile
	}

	t.Run("resolves the recorded path from a foreign working directory", func(t *testing.T) {
		docsDir, reportsDir, elsewhere := newSplitLayout(t)
		reportFile := newAnalyzeReport(t, reportsDir, docsDir, docsDir)

		chdir(t, elsewhere)
		out, err := executeSchemaApply(t, "--report", reportFile)
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeSchemaApplyResult(t, out)
		if result.Root != docsDir {
			t.Errorf("root = %q, want the scanned directory %q", result.Root, docsDir)
		}
	})

	t.Run("--root overrides the recorded path", func(t *testing.T) {
		docsDir, reportsDir, elsewhere := newSplitLayout(t)
		decoy := filepath.Join(filepath.Dir(reportsDir), "decoy")
		if err := os.MkdirAll(decoy, 0o755); err != nil {
			t.Fatalf("creating decoy: %v", err)
		}
		reportFile := newAnalyzeReport(t, reportsDir, decoy, decoy)

		chdir(t, elsewhere)
		out, err := executeSchemaApply(t, "--report", reportFile, "--root", docsDir)
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeSchemaApplyResult(t, out)
		if result.Root != docsDir {
			t.Errorf("root = %q, want the --root value %q — the flag did not reach the analyze path", result.Root, docsDir)
		}
	})

	// The reason AnalyzeReport carries an absolute Root at all. `analyze docs/`
	// records Path exactly as typed, so a consumer reading the report from a
	// different working directory would resolve "docs/" against ITS OWN
	// directory and silently scan the wrong tree — or nothing at all.
	t.Run("an absolute root beats a relative path recorded by the producer", func(t *testing.T) {
		docsDir, reportsDir, elsewhere := newSplitLayout(t)
		reportFile := newAnalyzeReport(t, reportsDir, "docs", docsDir)

		chdir(t, elsewhere)
		out, err := executeSchemaApply(t, "--report", reportFile)
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeSchemaApplyResult(t, out)
		if result.Root != docsDir {
			t.Errorf("root = %q, want %q — the relative path was resolved against the working directory instead of the recorded root",
				result.Root, docsDir)
		}
	})
}
