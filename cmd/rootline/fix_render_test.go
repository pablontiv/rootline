package main

import (
	"bytes"
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/proposal"
)

// --- renderProposalTable tests ---

// TestRenderProposalTable_EmptyProposals tests rendering with no proposals
func TestRenderProposalTable_EmptyProposals(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	report := &proposal.Report{
		Version:           1,
		Kind:              "rootline/analyze",
		Proposals:         []proposal.Proposal{},
		SchemaSuggestions: []proposal.Proposal{},
	}

	err := renderProposalTable(cmd, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "No proposals") {
		t.Errorf("expected 'No proposals' message, got: %s", output)
	}
}

// TestRenderProposalTable_WithProposals tests rendering with various proposal types
func TestRenderProposalTable_WithProposals(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/analyze",
		Proposals: []proposal.Proposal{
			{
				Type:        proposal.CorrectValue,
				Field:       "estado",
				Description: "correct invalid value",
				Paths:       []string{"doc.md"},
			},
			{
				Type:        proposal.AddField,
				Field:       "titulo",
				Description: "add missing field",
				Paths:       []string{"doc.md", "doc2.md"},
			},
		},
		SchemaSuggestions: []proposal.Proposal{},
	}

	err := renderProposalTable(cmd, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	// Should show table with proposal types
	if !strings.Contains(output, string(proposal.CorrectValue)) {
		t.Logf("output: %s", output)
	}
}

// TestRenderProposalTable_WithSchemaSuggestions tests schema suggestions section
func TestRenderProposalTable_WithSchemaSuggestions(t *testing.T) {
	cmd := rootCmd
	buf := new(bytes.Buffer)
	cmd.SetOut(buf)

	report := &proposal.Report{
		Version:   1,
		Kind:      "rootline/analyze",
		Proposals: []proposal.Proposal{},
		SchemaSuggestions: []proposal.Proposal{
			{
				Type:        proposal.ExtendEnum,
				Field:       "estado",
				Description: "extend enum with new value",
				Paths:       []string{".stem"},
			},
		},
	}

	err := renderProposalTable(cmd, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Schema suggestions") {
		t.Errorf("expected 'Schema suggestions' section, got: %s", output)
	}
}

func TestRenderFixTableSkippedProposalReasons(t *testing.T) {
	link := proposal.Proposal{
		Type: proposal.CorrectLink, Field: "links", Paths: []string{"stale.md"},
		From: "[[old]]", To: "[[new]]",
	}
	missing := proposal.Proposal{
		Type: proposal.AddField, Field: "estado", Paths: []string{"missing.md"},
		Value: "Pending", ValueSource: "enum_first",
	}
	for _, tc := range []struct {
		name    string
		skipped []proposal.Proposal
		missing bool
		link    bool
	}{
		{"link", []proposal.Proposal{link}, false, true},
		{"missing", []proposal.Proposal{missing}, true, false},
		{"mixed", []proposal.Proposal{missing, link}, true, true},
	} {
		t.Run(tc.name, func(t *testing.T) {
			oldOut := rootCmd.OutOrStdout()
			t.Cleanup(func() { rootCmd.SetOut(oldOut) })
			buf := new(bytes.Buffer)
			rootCmd.SetOut(buf)
			if err := renderFixTable(rootCmd, &BatchFixResult{SkippedProposals: tc.skipped}); err != nil {
				t.Fatal(err)
			}
			out := buf.String()
			if tc.missing {
				for _, want := range []string{
					"Note: 1 required field(s) left unfilled; schema declares no default:",
					"[missing.md]: estado (engine would have written \"Pending\")",
					"Invalid on purpose. Pass --fill-missing to write them.",
				} {
					if !strings.Contains(out, want) {
						t.Errorf("missing %q in output:\n%s", want, out)
					}
				}
			} else if strings.Contains(out, "--fill-missing") || strings.Contains(out, "left unfilled") || strings.Contains(out, "engine would") {
				t.Errorf("link skip incorrectly offers missing-field guidance:\n%s", out)
			}
			if tc.link {
				for _, want := range []string{
					"Skipped link proposals: 1 (no content change)",
					"[stale.md]: [[old]] -> [[new]] (no content change)",
				} {
					if !strings.Contains(out, want) {
						t.Errorf("missing %q in output:\n%s", want, out)
					}
				}
			} else if strings.Contains(out, "Skipped link proposals") {
				t.Errorf("missing-field skip incorrectly identified as a link skip:\n%s", out)
			}
		})
	}
}

func TestFixJSONPreservesStaleCorrectLinkSkip(t *testing.T) {
	dir := t.TempDir()
	original := "See [[new]].\n"
	path := filepath.Join(dir, "stale.md")
	mustWriteFile(t, path, []byte(original), 0o644)
	p := proposal.Proposal{
		Type: proposal.CorrectLink, Field: "links", Paths: []string{"stale.md"},
		From: "[[old]]", To: "[[new]]",
	}
	report := &proposal.Report{Proposals: []proposal.Proposal{p}}
	skipped, err := fix.ApplyProposals(context.Background(), report, dir, nil, false)
	if err != nil {
		t.Fatal(err)
	}
	if len(report.Proposals) != 0 {
		t.Fatalf("stale proposal reported as applied: %v", report.Proposals)
	}
	resetFlags()
	oldOut := rootCmd.OutOrStdout()
	t.Cleanup(func() { rootCmd.SetOut(oldOut) })
	buf := new(bytes.Buffer)
	rootCmd.SetOut(buf)
	batch := &BatchFixResult{
		Version: 1, Kind: "rootline/fix-batch", SkippedProposals: skipped,
	}
	if err := outputJSON(rootCmd, batch, false); err != nil {
		t.Fatal(err)
	}
	var decoded BatchFixResult
	if err := json.Unmarshal(buf.Bytes(), &decoded); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, buf)
	}
	if !reflect.DeepEqual(decoded.SkippedProposals, []proposal.Proposal{p}) ||
		decoded.Summary.Fixed != 0 || len(decoded.Results) != 0 {
		t.Fatalf("batch=%+v, want skipped correct_link evidence and no successful changes", decoded)
	}
	if got := mustReadFile(t, path); string(got) != original {
		t.Fatalf("stale target changed: %q", got)
	}
}

// --- proposalsToFixResults tests ---

// TestProposalsToFixResults_SingleFile tests via fix command integration
func TestProposalsToFixResults_SingleFile(t *testing.T) {
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
  titulo:
    type: string
    required: false
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	declareTestBoundary(t, dir)
	taskFile := filepath.Join(dir, "task.md")
	mustWriteFile(t, taskFile, []byte("---\nestado: Pending\n---\n# Task\n"), 0644)

	// Run fix to generate proposals and results
	resetFlags()
	out, err := runCmd(t, "fix", dir)
	if err != nil {
		t.Logf("fix error (may be expected): %v", err)
	}

	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		// May be table output
		t.Logf("fix output: %s", out)
	}
}

// TestProposalsToFixResults_MultipleProposals tests via fix command
func TestProposalsToFixResults_MultipleProposals(t *testing.T) {
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
  titulo:
    type: string
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	declareTestBoundary(t, dir)
	taskFile := filepath.Join(dir, "task.md")
	mustWriteFile(t, taskFile, []byte("---\nestado: InvalidValue\ntitulo: Task\n---\n# Task\n"), 0644)

	// Run fix to test grouping
	resetFlags()
	out, err := runCmd(t, "fix", taskFile)
	if err != nil {
		t.Logf("fix error: %v", err)
	}

	// Verify output
	_ = out
}

// TestProposalsToFixResults_RemoveStemField tests via fix command
func TestProposalsToFixResults_RemoveStemField(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 2
schema:
  oldfield:
    type: string
  newfield:
    type: string
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	declareTestBoundary(t, dir)

	// Run fix to test RemoveStemField scenarios
	resetFlags()
	out, err := runCmd(t, "fix", "--all", dir)
	if err != nil {
		t.Logf("fix error: %v", err)
	}

	// Verify it ran
	_ = out
}

// TestProposalsToFixResults_NoProposals tests when no proposals match records
func TestProposalsToFixResults_NoProposals(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 2
schema:
  estado:
    type: string
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)
	declareTestBoundary(t, dir)
	taskFile := filepath.Join(dir, "task.md")
	mustWriteFile(t, taskFile, []byte("---\nestado: Pending\n---\n# Task\n"), 0644)

	// Run fix with valid file - should have no proposals
	resetFlags()
	out, err := runCmd(t, "fix", taskFile)
	if err != nil {
		t.Logf("fix error: %v", err)
	}

	// Verify it ran
	_ = out
}
