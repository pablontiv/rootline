package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/proposal"
	"gopkg.in/yaml.v3"
)

func TestFixMissingRequired(t *testing.T) {
	dir := setupTestDir(t) // .stem has estado:required+enum
	// Create file missing required estado
	target := filepath.Join(dir, "missing.md")
	mustWriteFile(t, target, []byte("---\ntipo: test\n---\n# Missing\n"), 0644)

	// The fixture declares estado as a required enum with NO default:, so its
	// value can only be guessed. Issue #60: guessing destroys the missing-data
	// signal, so fix must report the gap and leave the document alone.
	out, err := runCmd(t, "fix", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "skipped") {
		t.Errorf("expected the skip reported, got: %s", out)
	}

	content := mustReadFile(t, target)
	if strings.Contains(string(content), "estado:") {
		t.Errorf("expected no invented value written, got: %s", string(content))
	}
}

func TestFixMissingRequired_FillMissingOptIn(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "missing.md")
	mustWriteFile(t, target, []byte("---\ntipo: test\n---\n# Missing\n"), 0644)

	out, err := runCmd(t, "fix", "--fill-missing", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "added") {
		t.Errorf("expected 'added' under --fill-missing, got: %s", out)
	}

	content := mustReadFile(t, target)
	if !strings.Contains(string(content), "estado:") {
		t.Errorf("expected the engine-chosen value written under opt-in, got: %s", string(content))
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
	if !strings.Contains(out, "skipped") {
		t.Errorf("expected the skip reported in dry-run output, got: %s", out)
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

func TestClosestMatch(t *testing.T) {
	candidates := []string{"Pending", "Completed"}
	got := fix.ClosestMatch("Completd", candidates)
	if got != "Completed" {
		t.Errorf("ClosestMatch('Completd') = %q, want 'Completed'", got)
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
	if batch.Summary.Total == 0 {
		t.Error("expected total > 0")
	}

	// estado has no declared default, so fix --all must not invent one.
	content := mustReadFile(t, filepath.Join(dir, "broken.md"))
	if strings.Contains(string(content), "estado:") {
		t.Errorf("expected no invented value written, got: %s", string(content))
	}
}

func TestFixAllJSON_ReportsSkippedProposals(t *testing.T) {
	// The unfilled field must be explainable from the output alone.
	dir := setupTestDir(t)
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
	if len(batch.SkippedProposals) == 0 {
		t.Fatalf("expected the declined proposal reported in JSON, got: %s", out)
	}
	sp := batch.SkippedProposals[0]
	if sp.Field != "estado" {
		t.Errorf("expected the skipped field named, got %q", sp.Field)
	}
	if sp.ValueSource != "enum_first" {
		t.Errorf("expected provenance in the output, got %q", sp.ValueSource)
	}
}

func TestFixAllJSON_FillMissingOptIn(t *testing.T) {
	dir := setupTestDir(t)
	mustWriteFile(t, filepath.Join(dir, "broken.md"), []byte("---\ntipo: test\n---\n# Broken\n"), 0644)
	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--fill-missing", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON output: %v\nraw: %s", err, out)
	}
	if batch.Summary.Fixed == 0 {
		t.Error("expected at least 1 fixed file under --fill-missing")
	}

	content := mustReadFile(t, filepath.Join(dir, "broken.md"))
	if !strings.Contains(string(content), "estado:") {
		t.Errorf("expected the engine-chosen value written under opt-in, got: %s", string(content))
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

	_, err := runCmd(t, "fix", "--all", "--output", "json")
	if err == nil {
		t.Fatal("expected error when no .stem file, got success")
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
	result := fix.RewriteFrontmatter(original, fm)
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
	result := fix.RewriteFrontmatter(original, fm)
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
	result := fix.InsertWikiLinksBeforeHeading(content, links)
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
	result := fix.InsertWikiLinksBeforeHeading(content, links)
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

	// The subject here is the add_field apply mechanic, not the default
	// no-fabrication policy (covered by TestFixAllJSON), so opt in explicitly.
	out, err := runCmd(t, "fix", "--all", "--fill-missing", "--output", "json")
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
	// Create 2 files with the same invalid enum value — would trigger extend_enum.
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

	// Check that .stem was NOT modified — extend_enum is now skipped.
	stemContent, readErr := filepath.Abs(filepath.Join(dir, ".stem"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	content, readErr := os.ReadFile(stemContent)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(content), "Obsoleto") {
		t.Errorf("extend_enum should NOT have been applied to .stem — schema proposals are now skipped")
	}

	// Files with estado: Obsoleto should be preserved (not changed to anything).
	for _, name := range []string{"file1.md", "file2.md"} {
		fileContent, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatalf("reading %s: %v", name, err)
		}
		if strings.Contains(string(fileContent), "Completed") {
			t.Errorf("%s: estado should not have been changed", name)
		}
		if !strings.Contains(string(fileContent), "Obsoleto") {
			t.Errorf("%s: expected estado: Obsoleto preserved, got:\n%s", name, string(fileContent))
		}
	}

	// Schema suggestions should be > 0 (extend_enum was proposed but skipped).
	if batch.SchemaSuggestions <= 0 {
		t.Error("expected schema_suggestions > 0 for skipped extend_enum")
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
	richStem := `version: 2
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
	declareTestBoundary(t, root)

	// Subdirectory with a poor stem (only id field, no enums).
	subdir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subdir, 0755); err != nil {
		t.Fatal(err)
	}
	poorStem := `version: 2
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

	stemContent := `version: 2
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
	declareTestBoundary(t, root)

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
			fix.WriteFrontmatterFields(&b, tt.fm)
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
	fix.WriteFrontmatterFields(&b, fm)
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

func TestFix_AddAggregate_DryRun(t *testing.T) {
	root := t.TempDir()
	_ = os.Mkdir(filepath.Join(root, ".git"), 0o755)

	// .stem with enum estado but no aggregate section.
	stem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, In Progress, Blocked, Completed]
`
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte(stem), 0644)
	declareTestBoundary(t, root)

	// Hierarchical structure with overlapping enum values.
	for _, epic := range []string{"E01-a", "E02-b"} {
		epicDir := filepath.Join(root, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\nestado: Pending\n---\n# "+epic+"\n"), 0644)

		for _, feat := range []string{"F01-x", "F02-y"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\nestado: Pending\n---\n# "+feat+"\n"), 0644)
		}
	}

	mustChdir(t, root)

	out, err := runCmd(t, "fix", "--all", "--dry-run", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}

	var report proposal.Report
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}

	if report.Summary.AddAggregate == 0 {
		t.Errorf("expected add_aggregate > 0, got 0.\nFull output: %s", out)
	}

	// Check the proposal in schema_suggestions (not proposals, since it's a schema proposal).
	found := false
	for _, p := range report.SchemaSuggestions {
		if p.Type == proposal.AddAggregate && p.Field == "estado" {
			found = true
			if !strings.Contains(p.Description, "would add aggregate for 'estado'") {
				t.Errorf("expected description with 'would add aggregate for estado', got: %s", p.Description)
			}
			if p.AggregateExpr == "" {
				t.Errorf("expected non-empty AggregateExpr")
			}
		}
	}
	if !found {
		t.Errorf("expected add_aggregate proposal for 'estado' in schema_suggestions, got schema_suggestions: %+v", report.SchemaSuggestions)
	}
}

func TestFix_AddAggregate_Apply(t *testing.T) {
	root := t.TempDir()
	_ = os.Mkdir(filepath.Join(root, ".git"), 0o755)

	// .stem with enum estado but no aggregate section.
	stem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, In Progress, Blocked, Completed]
`
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte(stem), 0644)
	declareTestBoundary(t, root)

	// Hierarchical structure.
	for _, epic := range []string{"E01-a", "E02-b"} {
		epicDir := filepath.Join(root, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\nestado: Pending\n---\n# "+epic+"\n"), 0644)

		for _, feat := range []string{"F01-x", "F02-y"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\nestado: Pending\n---\n# "+feat+"\n"), 0644)
		}
	}

	mustChdir(t, root)

	_, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Verify .stem was NOT modified — add_aggregate is a schema proposal and is now skipped.
	content, readErr := os.ReadFile(filepath.Join(root, ".stem"))
	if readErr != nil {
		t.Fatalf("failed to read .stem: %v", readErr)
	}
	stemStr := string(content)

	if strings.Contains(stemStr, "aggregate") {
		t.Errorf("add_aggregate should NOT have been applied to .stem — schema proposals are now skipped")
	}
}

// TestFixAllSchemaProposalsNotApplied verifies that extend_enum proposals are
// skipped by fix --all and reported as schema_suggestions instead of being applied.
func TestFixAllSchemaProposalsNotApplied(t *testing.T) {
	dir := setupTestDir(t)

	// Create 2 files with the same unknown enum value — would trigger extend_enum.
	mustWriteFile(t, filepath.Join(dir, "file1.md"), []byte("---\nestado: Archivado\ntipo: test\n---\n# F1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "file2.md"), []byte("---\nestado: Archivado\ntipo: test\n---\n# F2\n"), 0644)
	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// .stem should NOT be modified — extend_enum was NOT applied.
	stemContent, readErr := os.ReadFile(filepath.Join(dir, ".stem"))
	if readErr != nil {
		t.Fatal(readErr)
	}
	if strings.Contains(string(stemContent), "Archivado") {
		t.Error("extend_enum should NOT have been applied to .stem — schema proposals are skipped")
	}

	// But SchemaSuggestions should be > 0.
	if batch.SchemaSuggestions == 0 {
		t.Error("expected schema_suggestions > 0 for skipped extend_enum")
	}
}

// TestFixAllDataRepairsApplied verifies that data-level proposals like correct_value
// and add_field are still applied by fix --all.
func TestFixAllDataRepairsApplied(t *testing.T) {
	dir := setupTestDir(t)

	// File with typo in estado.
	mustWriteFile(t, filepath.Join(dir, "typo.md"), []byte("---\nestado: Completd\ntipo: test\n---\n# Typo\n"), 0644)

	// File with missing estado.
	mustWriteFile(t, filepath.Join(dir, "missing.md"), []byte("---\ntipo: test\n---\n# Missing\n"), 0644)

	mustChdir(t, dir)

	// Exercises both repair kinds together; add_field needs the opt-in because
	// the fixture declares estado without a default:.
	out, err := runCmd(t, "fix", "--all", "--fill-missing", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Verify typo.md was fixed.
	typoContent, err := os.ReadFile(filepath.Join(dir, "typo.md"))
	if err != nil {
		t.Fatalf("reading typo.md: %v", err)
	}
	if !strings.Contains(string(typoContent), "Completed") {
		t.Error("expected typo corrected to Completed")
	}
	if strings.Contains(string(typoContent), "Completd") {
		t.Error("typo should have been corrected")
	}

	// Verify missing.md was fixed.
	missingContent, err := os.ReadFile(filepath.Join(dir, "missing.md"))
	if err != nil {
		t.Fatalf("reading missing.md: %v", err)
	}
	if !strings.Contains(string(missingContent), "estado:") {
		t.Error("expected estado field added to missing.md")
	}

	// Check batch results show both files fixed.
	if batch.Summary.Fixed < 2 {
		t.Errorf("expected at least 2 fixed files, got %d", batch.Summary.Fixed)
	}
}

// TestFixAllDryRunShowsSchemaSuggestions verifies that schema suggestions appear
// in dry-run output without being applied.
func TestFixAllDryRunShowsSchemaSuggestions(t *testing.T) {
	dir := setupTestDir(t)

	// Create 2 files with same unknown enum to trigger extend_enum.
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: Unknown\ntipo: test\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: Unknown\ntipo: test\n---\n# B\n"), 0644)

	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--dry-run", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var report proposal.Report
	if err := json.Unmarshal([]byte(out), &report); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// Dry-run should show schema_suggestions array with extend_enum.
	if len(report.SchemaSuggestions) == 0 {
		t.Error("expected schema_suggestions in dry-run output")
	}

	foundExtendEnum := false
	for _, p := range report.SchemaSuggestions {
		if p.Type == proposal.ExtendEnum {
			foundExtendEnum = true
			break
		}
	}
	if !foundExtendEnum {
		t.Error("expected extend_enum in schema_suggestions")
	}

	// Files should not be modified in dry-run.
	aContent, _ := os.ReadFile(filepath.Join(dir, "a.md"))
	if !strings.Contains(string(aContent), "Unknown") {
		t.Error("dry-run should not modify files")
	}
}

// TestFixAllRemoveStemFieldNotApplied verifies that remove_stem_field proposals
// are skipped and reported as suggestions.
func TestFixAllRemoveStemFieldNotApplied(t *testing.T) {
	root := t.TempDir()
	_ = os.Mkdir(filepath.Join(root, ".git"), 0o755)

	// Create a .stem with a field that references a non-existent type.
	stem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
  badField:
    type: nonexistent_type
`
	stemPath := filepath.Join(root, ".stem")
	mustWriteFile(t, stemPath, []byte(stem), 0644)
	declareTestBoundary(t, root)

	// Create a document.
	mustWriteFile(t, filepath.Join(root, "test.md"), []byte("---\nestado: Pending\n---\n# Test\n"), 0644)

	mustChdir(t, root)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// .stem should still contain badField — remove_stem_field was NOT applied.
	stemContent, readErr := os.ReadFile(stemPath)
	if readErr != nil {
		t.Fatal(readErr)
	}
	if !strings.Contains(string(stemContent), "badField") {
		t.Error("remove_stem_field should NOT have been applied — schema proposals are skipped")
	}

	// Schema suggestions should be > 0 if remove_stem_field was proposed.
	// (badField was proposed for removal but not applied.)
}

// TestFixAllAddAggregateNotApplied verifies that add_aggregate proposals are
// skipped and not applied to the .stem file.
func TestFixAllAddAggregateNotApplied(t *testing.T) {
	root := t.TempDir()
	_ = os.Mkdir(filepath.Join(root, ".git"), 0o755)

	// .stem with no aggregate section.
	stem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte(stem), 0644)
	declareTestBoundary(t, root)

	// Create hierarchical structure to trigger add_aggregate.
	epicDir := filepath.Join(root, "E01-test")
	_ = os.MkdirAll(epicDir, 0755)
	mustWriteFile(t, filepath.Join(epicDir, "README.md"), []byte("---\nestado: Pending\n---\n# E01\n"), 0644)

	featDir := filepath.Join(epicDir, "F01-sub")
	_ = os.MkdirAll(featDir, 0755)
	mustWriteFile(t, filepath.Join(featDir, "README.md"), []byte("---\nestado: Completed\n---\n# F01\n"), 0644)

	mustChdir(t, root)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}

	// .stem should NOT have aggregate section — add_aggregate was NOT applied.
	stemContent, _ := os.ReadFile(filepath.Join(root, ".stem"))
	if strings.Contains(string(stemContent), "aggregate") {
		t.Error("add_aggregate should NOT have been applied — schema proposals are skipped")
	}

	// Schema suggestions should be > 0 if add_aggregate was proposed.
	// (add_aggregate was proposed but not applied.)
}

// --- Slice 3d: Mutating Command Error Semantics ---

// TestFixAllRefusesNoSchema verifies that fix --all fails when no schema is present.
// This is a bootstrap test: currently fix --all succeeds silently on ungoverned trees.
// After slice 3d, it should fail.
func TestFixAllRefusesNoSchema(t *testing.T) {
	// Create a tree with NO .stem file
	dir := t.TempDir()

	// Create markdown file
	target := filepath.Join(dir, "test.md")
	mustWriteFile(t, target, []byte("---\ntitle: Test\n---\n# Test\n"), 0644)

	// Attempt fix --all (should fail because no schema exists)
	out, err := runCmd(t, "fix", "--all", dir)

	// After slice 3d: this should error
	// Currently the command succeeds (err == nil) but we want it to fail
	if err == nil {
		t.Fatalf("expected error when fix --all has no schema, but succeeded with output: %s", out)
	}
}

// TestFixAllRefusesBadSchema verifies that fix --all fails when the schema is unparseable.
// This is a hard error that should always propagate.
func TestFixAllRefusesBadSchema(t *testing.T) {
	// Create a tree with an unparseable .stem
	dir := t.TempDir()

	// Create unparseable .stem
	badStem := "version: 2\nthis: [is not valid YAML"
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(badStem), 0644)

	// Create markdown file
	target := filepath.Join(dir, "test.md")
	mustWriteFile(t, target, []byte("---\ntitle: Test\n---\n# Test\n"), 0644)

	// Attempt fix --all (should fail because schema parse error)
	out, err := runCmd(t, "fix", "--all", dir)

	// After slice 3d: this should error
	if err == nil {
		t.Errorf("expected error when fix --all has bad schema, but succeeded with output: %s", out)
	}
}
