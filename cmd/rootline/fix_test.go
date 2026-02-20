package main

import (
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

func TestFixAll(t *testing.T) {
	dir := setupTestDir(t) // .stem has estado:required+enum

	// Create file with missing required field
	os.WriteFile(filepath.Join(dir, "broken.md"), []byte("---\ntipo: test\n---\n# Broken\n"), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	out, err := runCmd(t, "fix", "--all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "added") {
		t.Errorf("expected 'added' in output, got: %s", out)
	}

	// Verify file was updated
	content, _ := os.ReadFile(filepath.Join(dir, "broken.md"))
	if !strings.Contains(string(content), "estado:") {
		t.Errorf("expected estado field added to file, got: %s", string(content))
	}
}

func TestFixAllDryRun(t *testing.T) {
	dir := setupTestDir(t)

	original := "---\ntipo: test\n---\n# DryAll\n"
	target := filepath.Join(dir, "dryall.md")
	os.WriteFile(target, []byte(original), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	out, err := runCmd(t, "fix", "--all", "--dry-run")
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

func TestFixAllNoStem(t *testing.T) {
	dir := t.TempDir()
	// No .stem file, just a markdown file
	os.WriteFile(filepath.Join(dir, "file.md"), []byte("---\ntipo: test\n---\n# Test\n"), 0644)

	oldDir, _ := os.Getwd()
	os.Chdir(dir)
	defer os.Chdir(oldDir)

	out, err := runCmd(t, "fix", "--all")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "0 fields added, 0 values corrected") {
		t.Errorf("expected 0 fixes without .stem, got: %s", out)
	}
}

func TestFixNoArgsNoFlag(t *testing.T) {
	_, err := runCmd(t, "fix")
	if err == nil {
		t.Error("expected error when no args and no --all flag")
	}
}
