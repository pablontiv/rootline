package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixMissingRequired(t *testing.T) {
	dir := setupTestDir(t) // .stem has estado:required+enum
	// Create file missing required estado
	target := filepath.Join(dir, "missing.md")
	os.WriteFile(target, []byte("---\ntipo: test\n---\n# Missing\n"), 0644)

	out, err := runCmd(t, "fix", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "added") {
		t.Errorf("expected 'added' in output, got: %s", out)
	}

	// Verify file was updated
	content, _ := os.ReadFile(target)
	if !strings.Contains(string(content), "estado:") {
		t.Errorf("expected estado field added to file, got: %s", string(content))
	}
}

func TestFixInvalidEnum(t *testing.T) {
	dir := setupTestDir(t) // .stem has estado: enum [Completado, Pending]
	target := filepath.Join(dir, "bad-enum.md")
	os.WriteFile(target, []byte("---\nestado: Compltado\n---\n# Bad\n"), 0644)

	out, err := runCmd(t, "fix", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "corrected") {
		t.Errorf("expected 'corrected' in output, got: %s", out)
	}

	content, _ := os.ReadFile(target)
	if !strings.Contains(string(content), "Completado") {
		t.Errorf("expected 'Completado' after fix, got: %s", string(content))
	}
}

func TestFixDryRun(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "dryrun.md")
	original := "---\ntipo: test\n---\n# Dry\n"
	os.WriteFile(target, []byte(original), 0644)

	out, err := runCmd(t, "fix", "--dry-run", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "would add") {
		t.Errorf("expected 'would add' in dry-run output, got: %s", out)
	}

	// Verify file was NOT modified
	content, _ := os.ReadFile(target)
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
	os.WriteFile(target, []byte("---\ntipo: test\n---\n# Important Content\n\nBody text here.\n"), 0644)

	runCmd(t, "fix", target)

	content, _ := os.ReadFile(target)
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
	os.WriteFile(filepath.Join(dir, "broken.md"), []byte("---\ntipo: test\n---\n# Broken\n"), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

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
	content, _ := os.ReadFile(filepath.Join(dir, "broken.md"))
	if !strings.Contains(string(content), "estado:") {
		t.Errorf("expected estado field added to file, got: %s", string(content))
	}
}

func TestFixAllTable(t *testing.T) {
	dir := setupTestDir(t)

	os.WriteFile(filepath.Join(dir, "broken.md"), []byte("---\ntipo: test\n---\n# Broken\n"), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

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
	os.WriteFile(target, []byte(original), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

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
	content, _ := os.ReadFile(target)
	if string(content) != original {
		t.Error("dry-run should not modify the file")
	}
}

func TestFixAllNoStem(t *testing.T) {
	dir := t.TempDir()
	// No .stem file, just a markdown file
	os.WriteFile(filepath.Join(dir, "file.md"), []byte("---\ntipo: test\n---\n# Test\n"), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

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

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

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
	os.WriteFile(filepath.Join(dir, "bad-enum.md"), []byte("---\nestado: Compltado\ntipo: test\n---\n# Bad\n"), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

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
