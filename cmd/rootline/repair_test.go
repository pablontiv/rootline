package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/proposal"
)

// --- runRepairApply tests ---

func TestRunRepairApply_ValidJSON(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Setup .stem
	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  titulo:
    type: string
    required: false
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	declareTestBoundary(t, dir)

	// Setup document with missing titulo
	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Pending\n---\n# Task\n"), 0644)

	// Create a minimal repair report with add_field proposal
	report := proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:        proposal.AddField,
				Field:       "titulo",
				Value:       "New Title",
				Paths:       []string{filepath.Join(dir, "task.md")},
				Description: "add titulo field",
			},
		},
	}

	reportData, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, reportData, 0644)

	// Run repair apply
	resetFlags()
	out, err := runCmd(t, "repair", "apply", "--report", reportFile)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Output should contain success indication
	if !strings.Contains(out, "rootline/repair") && !strings.Contains(out, "repair") {
		// May be JSON or table format, just check no error
		t.Logf("repair apply output: %s", out)
	}
}

func TestRunRepairApply_DryRun(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Setup .stem
	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: InvalidValue\n---\n# Task\n"), 0644)

	// Create a report with correct_value proposal
	report := proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:        proposal.CorrectValue,
				Field:       "estado",
				From:        "InvalidValue",
				To:          "Pending",
				Paths:       []string{filepath.Join(dir, "task.md")},
				Description: "correct estado value",
			},
		},
	}

	reportData, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, reportData, 0644)

	// Save original file content
	originalContent := mustReadFile(t, filepath.Join(dir, "task.md"))

	// Run repair apply with --dry-run
	resetFlags()
	out, err := runCmd(t, "repair", "apply", "--report", reportFile, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Verify file was NOT modified
	afterContent := mustReadFile(t, filepath.Join(dir, "task.md"))
	if !bytes.Equal(originalContent, afterContent) {
		t.Error("expected --dry-run to not modify file")
	}

	// Output should mention dry-run
	if !strings.Contains(out, "[DRY RUN") && !strings.Contains(out, "DRY RUN") {
		// May be JSON output, just check it ran without error
		t.Logf("dry-run output: %s", out)
	}
}

func TestRunRepairApply_MissingReport(t *testing.T) {
	resetFlags()
	_, err := runCmd(t, "repair", "apply", "--report", "/nonexistent/report.json")
	if err == nil {
		t.Fatal("expected error for missing report file")
	}
}

func TestRunRepairApply_InvalidJSON(t *testing.T) {
	dir := t.TempDir()
	badFile := filepath.Join(dir, "bad.json")
	mustWriteFile(t, badFile, []byte("not json"), 0644)

	resetFlags()
	_, err := runCmd(t, "repair", "apply", "--report", badFile)
	if err == nil {
		t.Fatal("expected error for invalid JSON")
	}
}

func TestRunRepairApply_SchemaProposalsRejected(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Setup .stem
	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Completed]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Pending\n---\n# Task\n"), 0644)

	// Create a report with extend_enum proposal (schema surface, should be rejected)
	report := proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:        proposal.ExtendEnum,
				Field:       "estado",
				Value:       "InProgress",
				Paths:       []string{filepath.Join(dir, ".stem")},
				Description: "extend enum estado",
			},
		},
	}

	reportData, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, reportData, 0644)

	// Save original .stem
	originalStem := mustReadFile(t, filepath.Join(dir, ".stem"))

	// Run repair apply (should reject schema proposal)
	resetFlags()
	out, err := runCmd(t, "repair", "apply", "--report", reportFile)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// .stem should NOT be modified
	afterStem := mustReadFile(t, filepath.Join(dir, ".stem"))
	if !bytes.Equal(originalStem, afterStem) {
		t.Error("expected schema proposal to be rejected (not written to .stem)")
	}

	// Output should mention rejected
	if !strings.Contains(out, "Rejected") && !strings.Contains(out, "rejected") {
		t.Logf("output: %s", out)
	}
}

func TestRunRepairApply_TableOutput(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	declareTestBoundary(t, dir)
	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Pending\n---\n# Task\n"), 0644)

	report := proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:        proposal.CorrectValue,
				Field:       "estado",
				From:        "Pending",
				To:          "Completed",
				Paths:       []string{filepath.Join(dir, "task.md")},
				Description: "correct estado",
			},
		},
	}

	reportData, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	reportFile := filepath.Join(dir, "report.json")
	mustWriteFile(t, reportFile, reportData, 0644)

	// Run with table output
	resetFlags()
	out, err := runCmd(t, "repair", "apply", "--report", reportFile, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	// Should have table-like content
	_ = out
}

func TestRunRepairApply_NoReportFlag(t *testing.T) {
	resetFlags()
	_, err := runCmd(t, "repair", "apply")
	if err == nil {
		t.Fatal("expected error when --report flag is missing")
	}
}

func TestRunRepairApply_EnvelopeGuard(t *testing.T) {
	// Common setup: directory with .stem and one document
	baseSetup := func() (dir string, reportDir string) {
		dir = t.TempDir()
		if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
			t.Fatal(err)
		}

		// Setup .stem
		stemContent := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  titulo:
    type: string
    required: false
`
		mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
		declareTestBoundary(t, dir)

		// Setup document with validation error (missing titulo; required=false so title is actually present)
		mustWriteFile(t, filepath.Join(dir, "task.md"),
			[]byte("---\nestado: Pending\n---\n# Task\n"), 0644)

		return dir, dir
	}

	cases := []struct {
		name                string
		version             int
		kind                string
		proposalsPresent    bool
		expectError         bool
		errorContains       string
		expectDiskUnchanged bool
		expectDiskChanged   bool
	}{
		{
			name:                "UnsupportedVersion_0",
			version:             0,
			kind:                "rootline/proposals",
			proposalsPresent:    true,
			expectError:         true,
			errorContains:       "unsupported report version: 0",
			expectDiskUnchanged: true,
		},
		{
			name:                "UnsupportedVersion_99",
			version:             99,
			kind:                "rootline/proposals",
			proposalsPresent:    true,
			expectError:         true,
			errorContains:       "unsupported report version: 99",
			expectDiskUnchanged: true,
		},
		{
			name:                "WrongKind_Analyze",
			version:             1,
			kind:                "rootline/analyze",
			proposalsPresent:    true,
			expectError:         true,
			errorContains:       "wrong report kind",
			expectDiskUnchanged: true,
		},
		{
			name:                "WrongKind_SchemaProposals",
			version:             1,
			kind:                "rootline/schema-proposals",
			proposalsPresent:    true,
			expectError:         true,
			errorContains:       "wrong report kind",
			expectDiskUnchanged: true,
		},
		{
			name:                "WrongKind_Empty",
			version:             1,
			kind:                "",
			proposalsPresent:    true,
			expectError:         true,
			errorContains:       "wrong report kind",
			expectDiskUnchanged: true,
		},
		{
			name:                "FalsePositive_WrongKindValidProposals",
			version:             1,
			kind:                "rootline/analyze",
			proposalsPresent:    true,
			expectError:         true,
			errorContains:       "wrong report kind",
			expectDiskUnchanged: true, // FALSE-POSITIVE REGRESSION: assert both error AND disk unchanged
		},
		{
			name:              "ValidReport",
			version:           1,
			kind:              "rootline/proposals",
			proposalsPresent:  true,
			expectError:       false,
			expectDiskChanged: true,
		},
	}

	for _, tt := range cases {
		t.Run(tt.name, func(t *testing.T) {
			dir, reportDir := baseSetup()

			// Build report with proposals if requested
			report := proposal.Report{
				Version: tt.version,
				Kind:    tt.kind,
			}
			if tt.proposalsPresent {
				// Use relative paths from report directory
				taskPath := "task.md"
				if reportDir != dir {
					taskPath = filepath.Join(dir, "task.md")
				}
				report.Proposals = []proposal.Proposal{
					{
						Type:        proposal.AddField,
						Field:       "titulo",
						Value:       "New Title",
						Paths:       []string{taskPath},
						Description: "add titulo field",
					},
				}
			}

			reportData, err := json.Marshal(report)
			if err != nil {
				t.Fatalf("marshal report: %v", err)
			}

			reportFile := filepath.Join(reportDir, "report.json")
			mustWriteFile(t, reportFile, reportData, 0644)

			// Save original file content for false-positive check
			taskFile := filepath.Join(dir, "task.md")
			originalContent := mustReadFile(t, taskFile)

			// Run repair apply
			resetFlags()
			out, err := runCmd(t, "repair", "apply", "--report", reportFile)

			if tt.expectError {
				if err == nil {
					t.Fatalf("expected error, got nil; output: %s", out)
				}
				if !strings.Contains(err.Error(), tt.errorContains) {
					t.Errorf("error %q does not contain %q", err.Error(), tt.errorContains)
				}

				// FALSE-POSITIVE REGRESSION: verify disk unchanged
				if tt.expectDiskUnchanged {
					afterContent := mustReadFile(t, taskFile)
					if !bytes.Equal(originalContent, afterContent) {
						t.Fatal("false-positive regression: files were modified despite error")
					}
				}
			} else {
				if err != nil {
					t.Fatalf("unexpected error: %v\noutput: %s", err, out)
				}

				// Happy-path: verify record actually changed on disk
				if tt.expectDiskChanged {
					afterContent := mustReadFile(t, taskFile)
					if bytes.Equal(originalContent, afterContent) {
						t.Fatal("expected record to be modified on disk, but it was unchanged")
					}
				}
			}
		})
	}
}

// --- path containment (issue #69) ---

// containmentFixture builds a scan root holding the report and a governed
// record, plus a sibling file one level above the root that must stay untouched.
func containmentFixture(t *testing.T) (root, outside string) {
	t.Helper()

	base := t.TempDir()
	root = filepath.Join(base, "scan")
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	mustWriteFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`), 0644)
	declareTestBoundary(t, root)

	mustWriteFile(t, filepath.Join(root, "task.md"),
		[]byte("---\nestado: Pending\n---\n# Task\n\n## Status\n\noriginal\n"), 0644)

	outside = filepath.Join(base, "outside.md")
	mustWriteFile(t, outside,
		[]byte("---\nestado: Pending\n---\n# Outside\n\n## Status\n\nuntouched\n"), 0644)

	return root, outside
}

// writeContainmentReport writes a repair report into root and returns its path.
// The report directory is what repair apply resolves as the scan root.
func writeContainmentReport(t *testing.T, root string, proposals []proposal.Proposal) string {
	t.Helper()

	data, err := json.Marshal(proposal.Report{
		Version:   1,
		Kind:      "rootline/proposals",
		Proposals: proposals,
	})
	if err != nil {
		t.Fatalf("marshal report: %v", err)
	}

	reportFile := filepath.Join(root, "report.json")
	mustWriteFile(t, reportFile, data, 0644)
	return reportFile
}

func decodeRepairResult(t *testing.T, out string) *fix.RepairResult {
	t.Helper()

	var result fix.RepairResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("decoding repair result: %v\noutput: %s", err, out)
	}
	return &result
}

// assertRejected requires exactly one rejection naming both the offending path
// and the reason, with nothing recorded as changed and nothing reported as an
// I/O error — a containment violation is a policy rejection, not a failed write.
func assertRejected(t *testing.T, result *fix.RepairResult, path, reason string) {
	t.Helper()

	if len(result.Rejected) != 1 {
		t.Fatalf("rejected = %v, want exactly one entry", result.Rejected)
	}
	if !strings.Contains(result.Rejected[0], path) {
		t.Errorf("rejection %q does not name the path %q", result.Rejected[0], path)
	}
	if !strings.Contains(result.Rejected[0], reason) {
		t.Errorf("rejection %q does not give the reason %q", result.Rejected[0], reason)
	}
	if len(result.Changed) != 0 {
		t.Errorf("changed = %v, want empty", result.Changed)
	}
	if len(result.Errors) != 0 {
		t.Errorf("errors = %v, want empty (containment violations are not I/O errors)", result.Errors)
	}
}

func TestRunRepairApply_PathContainment(t *testing.T) {
	// Scenario 1: a relative path climbing above the scan root is rejected and
	// the file it named is never opened for writing.
	t.Run("RelativeEscapeRejected", func(t *testing.T) {
		root, outside := containmentFixture(t)
		before := mustReadFile(t, outside)

		reportFile := writeContainmentReport(t, root, []proposal.Proposal{{
			Type:        proposal.CorrectValue,
			Field:       "estado",
			From:        "Pending",
			To:          "Completed",
			Paths:       []string{"../outside.md"},
			Description: "correct estado",
		}})

		resetFlags()
		out, err := runCmd(t, "repair", "apply", "--report", reportFile, "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		assertRejected(t, decodeRepairResult(t, out), "../outside.md", "escapes root")

		if after := mustReadFile(t, outside); !bytes.Equal(before, after) {
			t.Error("file outside the scan root was modified")
		}
	})

	// Scenario 2: set_section reaches the filesystem through applySetSection
	// rather than the frontmatter rewriters, so it needs its own guard.
	t.Run("SetSectionEscapeRejected", func(t *testing.T) {
		root, outside := containmentFixture(t)
		before := mustReadFile(t, outside)

		reportFile := writeContainmentReport(t, root, []proposal.Proposal{{
			Type:        proposal.SetSection,
			Heading:     "## Status",
			Value:       "injected",
			Mode:        "replace",
			Paths:       []string{"../outside.md"},
			Description: "rewrite status section",
		}})

		resetFlags()
		out, err := runCmd(t, "repair", "apply", "--report", reportFile, "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		assertRejected(t, decodeRepairResult(t, out), "../outside.md", "escapes root")

		if after := mustReadFile(t, outside); !bytes.Equal(before, after) {
			t.Error("file outside the scan root was modified via set_section")
		}
	})

	// Scenario 3: an absolute path is refused on policy. Before this change it
	// was joined onto the root and failed as a "no such file" I/O error, which
	// hid the traversal attempt among ordinary failures.
	t.Run("AbsolutePathRejected", func(t *testing.T) {
		root, outside := containmentFixture(t)
		before := mustReadFile(t, outside)

		reportFile := writeContainmentReport(t, root, []proposal.Proposal{{
			Type:        proposal.AddField,
			Field:       "tampered",
			Value:       "true",
			Paths:       []string{outside},
			Description: "add tampered field",
		}})

		resetFlags()
		out, err := runCmd(t, "repair", "apply", "--report", reportFile, "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		assertRejected(t, decodeRepairResult(t, out), outside, "absolute paths are not allowed")

		if after := mustReadFile(t, outside); !bytes.Equal(before, after) {
			t.Error("file named by an absolute path was modified")
		}
	})

	// Scenario 4: --dry-run performs the same check and additionally reports the
	// resolved destinations so a caller can see where writes would have landed.
	t.Run("DryRunRejectsAndReportsResolvedTargets", func(t *testing.T) {
		root, outside := containmentFixture(t)
		before := mustReadFile(t, outside)

		reportFile := writeContainmentReport(t, root, []proposal.Proposal{
			{
				Type:        proposal.CorrectValue,
				Field:       "estado",
				From:        "Pending",
				To:          "Completed",
				Paths:       []string{"../outside.md"},
				Description: "correct estado outside root",
			},
			{
				Type:        proposal.CorrectValue,
				Field:       "estado",
				From:        "Pending",
				To:          "Completed",
				Paths:       []string{"task.md"},
				Description: "correct estado inside root",
			},
		})

		resetFlags()
		out, err := runCmd(t, "repair", "apply", "--report", reportFile, "--dry-run", "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeRepairResult(t, out)
		if !result.DryRun {
			t.Error("dry_run = false, want true")
		}
		if result.ResolvedTargets == nil {
			t.Fatal("resolved_targets is absent, want it populated in dry-run")
		}

		wantAccepted := filepath.Join(root, "task.md")
		if got := result.ResolvedTargets.Accepted["task.md"]; got != wantAccepted {
			t.Errorf("resolved_targets.accepted[task.md] = %q, want %q", got, wantAccepted)
		}
		if got := result.ResolvedTargets.Rejected["../outside.md"]; got != "escapes root" {
			t.Errorf("resolved_targets.rejected[../outside.md] = %q, want %q", got, "escapes root")
		}

		if after := mustReadFile(t, outside); !bytes.Equal(before, after) {
			t.Error("dry-run modified a file outside the scan root")
		}
		if !strings.Contains(strings.Join(result.Changed, "\n"), "task.md") {
			t.Errorf("changed = %v, want the contained proposal previewed", result.Changed)
		}
	})

	// resolved_targets is dry-run only: an applied run already says what it did
	// in changed[], so repeating the destinations would be noise.
	t.Run("ResolvedTargetsOmittedOutsideDryRun", func(t *testing.T) {
		root, _ := containmentFixture(t)

		reportFile := writeContainmentReport(t, root, []proposal.Proposal{{
			Type:        proposal.CorrectValue,
			Field:       "estado",
			From:        "Pending",
			To:          "Completed",
			Paths:       []string{"task.md"},
			Description: "correct estado",
		}})

		resetFlags()
		out, err := runCmd(t, "repair", "apply", "--report", reportFile, "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		if result := decodeRepairResult(t, out); result.ResolvedTargets != nil {
			t.Errorf("resolved_targets = %+v, want absent outside dry-run", result.ResolvedTargets)
		}
		if strings.Contains(out, "resolved_targets") {
			t.Error("resolved_targets key present in JSON, want it omitted")
		}
	})

	// Scenario 12: a contained set_section write is applied and recorded. It
	// used to write the file but report nothing outside dry-run.
	t.Run("ContainedSetSectionIsAppliedAndRecorded", func(t *testing.T) {
		root, _ := containmentFixture(t)
		taskFile := filepath.Join(root, "task.md")

		reportFile := writeContainmentReport(t, root, []proposal.Proposal{{
			Type:        proposal.SetSection,
			Heading:     "## Status",
			Value:       "rewritten",
			Mode:        "replace",
			Paths:       []string{"task.md"},
			Description: "rewrite status section",
		}})

		resetFlags()
		out, err := runCmd(t, "repair", "apply", "--report", reportFile, "-o", "json")
		if err != nil {
			t.Fatalf("unexpected error: %v\noutput: %s", err, out)
		}

		result := decodeRepairResult(t, out)
		if len(result.Errors) != 0 {
			t.Fatalf("errors = %v, want empty", result.Errors)
		}
		if len(result.Rejected) != 0 {
			t.Fatalf("rejected = %v, want empty", result.Rejected)
		}

		changed := strings.Join(result.Changed, "\n")
		if !strings.Contains(changed, "task.md") || !strings.Contains(changed, "## Status") {
			t.Errorf("changed = %v, want an entry naming the file and the section", result.Changed)
		}

		if content := string(mustReadFile(t, taskFile)); !strings.Contains(content, "rewritten") {
			t.Errorf("task.md was not rewritten:\n%s", content)
		}
	})
}
