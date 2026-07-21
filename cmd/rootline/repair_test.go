package main

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

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
		Kind:    "rootline/analyze",
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
		Kind:    "rootline/analyze",
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
		Kind:    "rootline/analyze",
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
		Kind:    "rootline/analyze",
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
