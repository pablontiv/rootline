package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

func TestFixMissingRequired(t *testing.T) {
	dir := setupTestDir(t) // .stem has estado:required+enum
	// Create file missing required estado
	target := filepath.Join(dir, "missing.md")
	mustWriteFile(t, target, []byte("---\ntipo: test\n---\n# Missing\n"), 0644)

	out, err := runCmd(t, "fix", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "added") {
		t.Errorf("expected 'added' in output, got: %s", out)
	}

	// Verify file was updated
	content := mustReadFile(t, target)
	if !strings.Contains(string(content), "estado:") {
		t.Errorf("expected estado field added to file, got: %s", string(content))
	}
}

func TestFixInvalidEnum(t *testing.T) {
	dir := setupTestDir(t) // .stem has estado: enum [Completado, Pending]
	target := filepath.Join(dir, "bad-enum.md")
	mustWriteFile(t, target, []byte("---\nestado: Compltado\n---\n# Bad\n"), 0644)

	out, err := runCmd(t, "fix", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "corrected") {
		t.Errorf("expected 'corrected' in output, got: %s", out)
	}

	content := mustReadFile(t, target)
	if !strings.Contains(string(content), "Completado") {
		t.Errorf("expected 'Completado' after fix, got: %s", string(content))
	}
}

func TestFixDryRun(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "dryrun.md")
	original := "---\ntipo: test\n---\n# Dry\n"
	mustWriteFile(t, target, []byte(original), 0644)

	out, err := runCmd(t, "fix", "--dry-run", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "would add") {
		t.Errorf("expected 'would add' in dry-run output, got: %s", out)
	}

	// Verify file was NOT modified
	content := mustReadFile(t, target)
	if string(content) != original {
		t.Error("dry-run should not modify the file")
	}
}

func TestFixNoErrors(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "doc1.md") // valid file

	out, err := runCmd(t, "fix", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "no errors") {
		t.Errorf("expected 'no errors' for valid file, got: %s", out)
	}
}

func TestFixPreservesBody(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "body.md")
	mustWriteFile(t, target, []byte("---\ntipo: test\n---\n# Important Content\n\nBody text here.\n"), 0644)

	_, err := runCmd(t, "fix", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := mustReadFile(t, target)
	if !strings.Contains(string(content), "Important Content") {
		t.Error("expected body preserved after fix")
	}
	if !strings.Contains(string(content), "Body text here.") {
		t.Error("expected body text preserved")
	}
}

func TestLevenshtein(t *testing.T) {
	tests := []struct {
		a, b string
		want int
	}{
		{"", "", 0},
		{"abc", "", 3},
		{"", "abc", 3},
		{"kitten", "sitting", 3},
		{"Completado", "Compltado", 1},
	}
	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestClosestMatch(t *testing.T) {
	candidates := []string{"Pending", "Completado"}
	got := closestMatch("Compltado", candidates)
	if got != "Completado" {
		t.Errorf("closestMatch('Compltado') = %q, want 'Completado'", got)
	}
}

func TestFixAllJSON(t *testing.T) {
	dir := setupTestDir(t) // .stem has estado:required+enum

	// Create file with missing required field
	mustWriteFile(t, filepath.Join(dir, "broken.md"), []byte("---\ntipo: test\n---\n# Broken\n"), 0644)

	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out)
	}
	if batch.Version != 1 {
		t.Errorf("expected version 1, got %d", batch.Version)
	}
	if batch.Kind != "rootline/fix-batch" {
		t.Errorf("expected kind rootline/fix-batch, got %s", batch.Kind)
	}
	if batch.Summary.Fixed == 0 {
		t.Error("expected at least 1 fixed file")
	}
	if batch.Summary.Total == 0 {
		t.Error("expected total > 0")
	}

	// Verify file was actually updated
	content := mustReadFile(t, filepath.Join(dir, "broken.md"))
	if !strings.Contains(string(content), "estado:") {
		t.Errorf("expected estado field added to file, got: %s", string(content))
	}
}

func TestFixAllTable(t *testing.T) {
	dir := setupTestDir(t)

	mustWriteFile(t, filepath.Join(dir, "broken.md"), []byte("---\ntipo: test\n---\n# Broken\n"), 0644)

	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "File") {
		t.Errorf("expected table header 'File', got: %s", out)
	}
	if !strings.Contains(out, "Fixed") {
		t.Errorf("expected table header 'Fixed', got: %s", out)
	}
	if !strings.Contains(out, "Changes") {
		t.Errorf("expected table header 'Changes', got: %s", out)
	}
}

func TestFixAllDryRunJSON(t *testing.T) {
	dir := setupTestDir(t)

	original := "---\ntipo: test\n---\n# DryAll\n"
	target := filepath.Join(dir, "dryall.md")
	mustWriteFile(t, target, []byte(original), 0644)

	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--dry-run", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// In dry-run, Fixed should be false but changes should be reported
	for _, r := range batch.Results {
		if r.FieldsAdded > 0 || r.ValuesCorrected > 0 {
			if r.Fixed {
				t.Errorf("dry-run should not mark files as fixed, got fixed=true for %s", r.Path)
			}
			if len(r.Changes) == 0 {
				t.Errorf("expected changes reported for %s in dry-run", r.Path)
			}
		}
	}

	// Verify file was NOT modified
	content := mustReadFile(t, target)
	if string(content) != original {
		t.Error("dry-run should not modify the file")
	}
}

func TestFixAllNoStem(t *testing.T) {
	dir := t.TempDir()
	// No .stem file, just a markdown file
	mustWriteFile(t, filepath.Join(dir, "file.md"), []byte("---\ntipo: test\n---\n# Test\n"), 0644)

	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if batch.Summary.Fixed != 0 {
		t.Errorf("expected 0 fixes without .stem, got: %d", batch.Summary.Fixed)
	}
}

func TestFixAllNoErrors(t *testing.T) {
	dir := setupTestDir(t) // has valid doc1.md

	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}
	if batch.Summary.Fixed != 0 {
		t.Errorf("expected 0 fixed for valid files, got: %d", batch.Summary.Fixed)
	}
	if batch.Summary.Total != batch.Summary.Skipped {
		t.Errorf("expected all files skipped, got total=%d skipped=%d", batch.Summary.Total, batch.Summary.Skipped)
	}
}

func TestFixNoArgsNoFlag(t *testing.T) {
	_, err := runCmd(t, "fix")
	if err == nil {
		t.Error("expected error when no args and no --all flag")
	}
}

func TestRewriteFrontmatterNoPrior(t *testing.T) {
	// File with no frontmatter — should prepend it
	original := "# Just a heading\nSome body.\n"
	fm := map[string]any{"estado": "Pending"}
	result := rewriteFrontmatter(original, fm)
	if !strings.HasPrefix(result, "---\n") {
		t.Error("expected frontmatter prepended")
	}
	if !strings.Contains(result, "estado: Pending") {
		t.Error("expected estado field in new frontmatter")
	}
	if !strings.Contains(result, "Just a heading") {
		t.Error("expected original body preserved")
	}
}

func TestRewriteFrontmatterMalformed(t *testing.T) {
	// Frontmatter starts but never closes — should return original
	original := "---\nestado: test\n# No closing\n"
	fm := map[string]any{"estado": "Pending"}
	result := rewriteFrontmatter(original, fm)
	if result != original {
		t.Errorf("expected original returned for malformed frontmatter, got: %s", result)
	}
}

func TestFixAllWithEnumCorrection(t *testing.T) {
	dir := setupTestDir(t)

	// Create file with bad enum value
	mustWriteFile(t, filepath.Join(dir, "bad-enum.md"), []byte("---\nestado: Compltado\ntipo: test\n---\n# Bad\n"), 0644)

	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON output: %v", err)
	}

	// Find the result for bad-enum.md
	var found bool
	for _, r := range batch.Results {
		if strings.Contains(r.Path, "bad-enum") {
			found = true
			if r.ValuesCorrected == 0 {
				t.Error("expected enum correction for bad-enum.md")
			}
			if !r.Fixed {
				t.Error("expected fixed=true for bad-enum.md")
			}
			hasCorrection := false
			for _, c := range r.Changes {
				if strings.Contains(c, "correct") {
					hasCorrection = true
				}
			}
			if !hasCorrection {
				t.Errorf("expected 'correct' in changes, got: %v", r.Changes)
			}
		}
	}
	if !found {
		t.Error("bad-enum.md not found in results")
	}
}

func TestInsertWikiLinksBeforeHeading(t *testing.T) {
	content := "---\nestado: Pending\n---\n# Title\n\n## Context\n\nSome text\n"
	links := []string{"[[blocks:T001]]", "[[blocks:T002]]"}
	result := insertWikiLinksBeforeHeading(content, links)
	if !strings.Contains(result, "[[blocks:T001]]\n[[blocks:T002]]") {
		t.Errorf("expected wiki-links inserted, got: %s", result)
	}
	if !strings.Contains(result, "## Context") {
		t.Error("expected heading preserved")
	}
}

func TestInsertWikiLinksNoHeading(t *testing.T) {
	content := "---\nestado: Pending\n---\n# Title\n\nNo sub-headings here.\n"
	links := []string{"[[blocks:T001]]"}
	result := insertWikiLinksBeforeHeading(content, links)
	if !strings.Contains(result, "[[blocks:T001]]") {
		t.Errorf("expected wiki-link appended, got: %s", result)
	}
}

func TestRenderProposalTable(t *testing.T) {
	resetFlags()
	cmd := rootCmd
	buf := new(strings.Builder)
	cmd.SetOut(buf)

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{Type: proposal.CorrectValue, Field: "estado", Description: "typo fix", Paths: []string{"a.md"}},
		},
		Summary: proposal.Summary{Total: 1, CorrectValue: 1},
	}

	err := renderProposalTable(cmd, report)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	output := buf.String()
	if !strings.Contains(output, "correct_value") {
		t.Errorf("expected proposal type in table, got: %s", output)
	}
}

func TestRenderProposalTableEmpty(t *testing.T) {
	resetFlags()
	cmd := rootCmd
	buf := new(strings.Builder)
	cmd.SetOut(buf)

	report := &proposal.Report{
		Version:   1,
		Kind:      "rootline/proposals",
		Proposals: nil,
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

func TestFixAllApplyMissingField(t *testing.T) {
	dir := setupTestDir(t)
	mustWriteFile(t, filepath.Join(dir, "missing.md"), []byte("---\ntipo: test\n---\n# Missing estado\n"), 0644)
	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	var found bool
	for _, r := range batch.Results {
		if strings.Contains(r.Path, "missing") {
			found = true
			if !r.Fixed {
				t.Error("expected fixed=true for missing.md")
			}
			if r.FieldsAdded == 0 {
				t.Error("expected fields added for missing.md")
			}
		}
	}
	if !found {
		t.Error("missing.md not found in results")
	}
}

func TestFixAllDryRunTable(t *testing.T) {
	dir := setupTestDir(t)
	mustWriteFile(t, filepath.Join(dir, "bad.md"), []byte("---\nestado: Compltado\ntipo: test\n---\n# Bad\n"), 0644)
	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--dry-run", "--output", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "correct_value") {
		t.Errorf("expected correct_value in table, got: %s", out)
	}
}

func TestFixAllApplyExtendEnum(t *testing.T) {
	dir := setupTestDir(t)
	// Create 2 files with the same invalid enum value — triggers extend_enum.
	mustWriteFile(t, filepath.Join(dir, "file1.md"), []byte("---\nestado: Obsoleto\ntipo: test\n---\n# F1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "file2.md"), []byte("---\nestado: Obsoleto\ntipo: test\n---\n# F2\n"), 0644)
	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Check that .stem was modified to include "Obsoleto".
	stemContent, readErr := filepath.Abs(filepath.Join(dir, ".stem"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	content, readErr := os.ReadFile(stemContent)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(content), "Obsoleto") {
		t.Errorf("expected .stem to contain 'Obsoleto' after extend_enum, got:\n%s", string(content))
	}
}

func TestFixAllApplyExtractBody(t *testing.T) {
	dir := setupTestDir(t)
	// File with no frontmatter estado but has it in body.
	mustWriteFile(t, filepath.Join(dir, "legacy.md"), []byte("---\ntipo: test\n---\n# Legacy\n\n**Estado**: Completado\n"), 0644)
	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	var found bool
	for _, r := range batch.Results {
		if strings.Contains(r.Path, "legacy") {
			found = true
			if !r.Fixed {
				t.Error("expected fixed=true for legacy.md")
			}
		}
	}
	if !found {
		t.Error("legacy.md not found in results")
	}
}

func TestFixAllDryRunWithProposals(t *testing.T) {
	dir := setupTestDir(t)
	mustWriteFile(t, filepath.Join(dir, "missing.md"), []byte("---\ntipo: test\n---\n# Missing estado\n"), 0644)
	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--dry-run", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "rootline/proposals") {
		t.Errorf("expected proposals kind in output, got: %s", out)
	}
	if !strings.Contains(out, "add_field") {
		t.Errorf("expected add_field proposal, got: %s", out)
	}
}
