package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
	"gopkg.in/yaml.v3"
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
	dir := setupTestDir(t) // .stem has estado: enum [Completed, Pending]
	target := filepath.Join(dir, "bad-enum.md")
	mustWriteFile(t, target, []byte("---\nestado: Completd\n---\n# Bad\n"), 0644)

	out, err := runCmd(t, "fix", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "corrected") {
		t.Errorf("expected 'corrected' in output, got: %s", out)
	}

	content := mustReadFile(t, target)
	if !strings.Contains(string(content), "Completed") {
		t.Errorf("expected 'Completed' after fix, got: %s", string(content))
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
		{"Completed", "Completd", 1},
	}
	for _, tt := range tests {
		got := levenshtein(tt.a, tt.b)
		if got != tt.want {
			t.Errorf("levenshtein(%q, %q) = %d, want %d", tt.a, tt.b, got, tt.want)
		}
	}
}

func TestClosestMatch(t *testing.T) {
	candidates := []string{"Pending", "Completed"}
	got := closestMatch("Completd", candidates)
	if got != "Completed" {
		t.Errorf("closestMatch('Completd') = %q, want 'Completed'", got)
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
	mustWriteFile(t, filepath.Join(dir, "bad-enum.md"), []byte("---\nestado: Completd\ntipo: test\n---\n# Bad\n"), 0644)

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
	mustWriteFile(t, filepath.Join(dir, "bad.md"), []byte("---\nestado: Completd\ntipo: test\n---\n# Bad\n"), 0644)
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

	// T003: Verify files with estado: Obsoleto are NOT changed to Completed.
	// After extend_enum makes "Obsoleto" valid, correct_value should be skipped.
	for _, name := range []string{"file1.md", "file2.md"} {
		fileContent, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if strings.Contains(string(fileContent), "Completed") {
			t.Errorf("%s: estado was changed to Completed — correct_value should have been skipped after extend_enum", name)
		}
		if !strings.Contains(string(fileContent), "Obsoleto") {
			t.Errorf("%s: expected estado: Obsoleto preserved, got:\n%s", name, string(fileContent))
		}
	}
}

func TestFixAllSkipCorrectAfterExtendButFixTypos(t *testing.T) {
	dir := setupTestDir(t)
	// 2 files with "Obsoleto" -> triggers extend_enum (adds Obsoleto to .stem).
	mustWriteFile(t, filepath.Join(dir, "obs1.md"), []byte("---\nestado: Obsoleto\ntipo: test\n---\n# Obs1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "obs2.md"), []byte("---\nestado: Obsoleto\ntipo: test\n---\n# Obs2\n"), 0644)
	// 1 file with a typo -> should still be corrected by correct_value.
	mustWriteFile(t, filepath.Join(dir, "typo.md"), []byte("---\nestado: Completd\ntipo: test\n---\n# Typo\n"), 0644)
	mustChdir(t, dir)

	_, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Obsoleto files should be preserved (not changed to Completed).
	for _, name := range []string{"obs1.md", "obs2.md"} {
		content, readErr := os.ReadFile(filepath.Join(dir, name))
		if readErr != nil {
			t.Fatalf("reading %s: %v", name, readErr)
		}
		if !strings.Contains(string(content), "Obsoleto") {
			t.Errorf("%s: expected estado: Obsoleto preserved, got:\n%s", name, string(content))
		}
	}

	// Typo file should be corrected to Completed.
	typoContent, err := os.ReadFile(filepath.Join(dir, "typo.md"))
	if err != nil {
		t.Fatalf("reading typo.md: %v", err)
	}
	if !strings.Contains(string(typoContent), "Completed") {
		t.Errorf("typo.md: expected estado corrected to Completed, got:\n%s", string(typoContent))
	}
	if strings.Contains(string(typoContent), "Completd") {
		t.Errorf("typo.md: typo 'Completd' was not corrected")
	}
}

func TestFixAllSkipCorrectAfterExtendReportAccuracy(t *testing.T) {
	dir := setupTestDir(t)
	// 2 files with "Obsoleto" -> triggers extend_enum.
	mustWriteFile(t, filepath.Join(dir, "obs1.md"), []byte("---\nestado: Obsoleto\ntipo: test\n---\n# Obs1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "obs2.md"), []byte("---\nestado: Obsoleto\ntipo: test\n---\n# Obs2\n"), 0644)
	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// The obs1.md and obs2.md should NOT report ValuesCorrected since
	// correct_value was skipped after extend_enum made the value valid.
	for _, r := range batch.Results {
		if strings.Contains(r.Path, "obs1") || strings.Contains(r.Path, "obs2") {
			if r.ValuesCorrected > 0 {
				t.Errorf("%s: expected ValuesCorrected=0 (skipped after extend_enum), got %d", r.Path, r.ValuesCorrected)
			}
			for _, c := range r.Changes {
				if strings.Contains(c, "correct") {
					t.Errorf("%s: unexpected correction in changes: %s", r.Path, c)
				}
			}
		}
	}
}

func TestFixAllApplyExtractBody(t *testing.T) {
	dir := setupTestDir(t)
	// File with no frontmatter estado but has it in body.
	mustWriteFile(t, filepath.Join(dir, "legacy.md"), []byte("---\ntipo: test\n---\n# Legacy\n\n**Estado**: Completed\n"), 0644)
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

// TestFixAllSelectsRichestStem verifies that runFixAll picks the stem with the
// richest schema (most fields) instead of a random map entry. This prevents
// losing enum definitions like estado/tipo when multiple stems coexist.
func TestFixAllSelectsRichestStem(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// Rich stem at root: has estado enum with known values.
	richStem := `version: 1
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  tipo:
    type: string
    required: false
`
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte(richStem), 0644)

	// Subdirectory with a poor stem (only id field, no enums).
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	poorStem := `version: 1
scope:
  match: "*.md"
schema:
  id:
    type: string
`
	mustWriteFile(t, filepath.Join(subdir, ".stem"), []byte(poorStem), 0644)

	// Two files with the same unknown enum value -> triggers extend_enum
	// when the rich stem is correctly selected.
	mustWriteFile(t, filepath.Join(root, "a.md"), []byte("---\nestado: Archivado\ntipo: test\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(root, "b.md"), []byte("---\nestado: Archivado\ntipo: test\n---\n# B\n"), 0644)

	// File in subdirectory (merges rich + poor stem).
	mustWriteFile(t, filepath.Join(subdir, "c.md"), []byte("---\nid: x1\nestado: Pending\n---\n# C\n"), 0644)

	mustChdir(t, root)

	// Run multiple times to catch non-determinism from map iteration order.
	for i := 0; i < 5; i++ {
		out, err := runCmd(t, "fix", "--all", "--dry-run", "--output", "json")
		if err != nil {
			t.Fatalf("iteration %d: unexpected error: %v", i, err)
		}

		var report proposal.Report
		if err := json.Unmarshal([]byte(out), &report); err != nil {
			t.Fatalf("iteration %d: invalid JSON: %v\nraw: %s", i, err, out)
		}

		// The richest stem has estado as enum. Two files use "Archivado"
		// which is not in the enum -> extend_enum proposal.
		if report.Summary.ExtendEnum == 0 {
			t.Errorf("iteration %d: expected extend_enum > 0, got 0.\nFull output: %s", i, out)
		}
	}
}

// TestFixAllDryRunConsistentFromRootAndSubdir verifies that running fix --all
// from the repo root and from a subdirectory produces the same proposal types.
func TestFixAllDryRunConsistentFromRootAndSubdir(t *testing.T) {
	root := t.TempDir()
	if err := os.Mkdir(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 1
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  tipo:
    type: string
`
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte(stemContent), 0644)

	subdir := filepath.Join(root, "docs")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}

	// Files with unknown enum value in the subdirectory.
	mustWriteFile(t, filepath.Join(subdir, "x.md"), []byte("---\nestado: Obsoleto\ntipo: a\n---\n# X\n"), 0644)
	mustWriteFile(t, filepath.Join(subdir, "y.md"), []byte("---\nestado: Obsoleto\ntipo: b\n---\n# Y\n"), 0644)

	// Run from root.
	mustChdir(t, root)
	outRoot, err := runCmd(t, "fix", "--all", "--dry-run", "--output", "json")
	if err != nil {
		t.Fatalf("from root: unexpected error: %v", err)
	}
	var reportRoot proposal.Report
	if err := json.Unmarshal([]byte(outRoot), &reportRoot); err != nil {
		t.Fatalf("from root: invalid JSON: %v", err)
	}

	// Run from subdir.
	mustChdir(t, subdir)
	outSub, err := runCmd(t, "fix", "--all", "--dry-run", "--output", "json")
	if err != nil {
		t.Fatalf("from subdir: unexpected error: %v", err)
	}
	var reportSub proposal.Report
	if err := json.Unmarshal([]byte(outSub), &reportSub); err != nil {
		t.Fatalf("from subdir: invalid JSON: %v", err)
	}

	// Both should have extend_enum proposals.
	if reportRoot.Summary.ExtendEnum == 0 {
		t.Errorf("from root: expected extend_enum > 0, got 0.\nOutput: %s", outRoot)
	}
	if reportSub.Summary.ExtendEnum == 0 {
		t.Errorf("from subdir: expected extend_enum > 0, got 0.\nOutput: %s", outSub)
	}
	if reportRoot.Summary.ExtendEnum != reportSub.Summary.ExtendEnum {
		t.Errorf("extend_enum mismatch: root=%d subdir=%d", reportRoot.Summary.ExtendEnum, reportSub.Summary.ExtendEnum)
	}
}

func TestWriteFrontmatterFields_YAMLQuoting(t *testing.T) {
	tests := []struct {
		name string
		fm   map[string]any
		want string // fragment that must appear in output
	}{
		{
			name: "plain string",
			fm:   map[string]any{"title": "Hello"},
			want: "title: Hello\n",
		},
		{
			name: "string with colon is quoted",
			fm:   map[string]any{"title": "Hello: World"},
			want: "'Hello: World'",
		},
		{
			name: "integer value",
			fm:   map[string]any{"count": 42},
			want: "count: 42\n",
		},
		{
			name: "boolean value",
			fm:   map[string]any{"draft": true},
			want: "draft: true\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			var b strings.Builder
			writeFrontmatterFields(&b, tt.fm)
			out := b.String()
			if !strings.Contains(out, tt.want) {
				t.Errorf("output = %q\nwant fragment %q", out, tt.want)
			}
			// Must round-trip as valid YAML.
			var parsed map[string]any
			if err := yaml.Unmarshal([]byte(out), &parsed); err != nil {
				t.Errorf("output is not valid YAML: %v\noutput: %q", err, out)
			}
		})
	}
}

func TestWriteFrontmatterFields_SliceValue(t *testing.T) {
	fm := map[string]any{"tags": []any{"alpha", "beta"}}
	var b strings.Builder
	writeFrontmatterFields(&b, fm)
	out := b.String()

	// Must round-trip as valid YAML.
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid YAML: %v\noutput: %q", err, out)
	}
	tags, ok := parsed["tags"].([]any)
	if !ok {
		t.Fatalf("expected tags to be a list, got %T: %v", parsed["tags"], parsed["tags"])
	}
	if len(tags) != 2 {
		t.Errorf("expected 2 tags, got %d", len(tags))
	}
}
