package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestFixAllApplyMigrateValue(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 1
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Blocked, Completed]
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// File with a parenthesized value that triggers migrate_value.
	mustWriteFile(t, filepath.Join(dir, "task.md"), []byte("---\nestado: \"Pending (blocked by T001)\"\n---\n# Task\n\n## Context\n\nSome text\n"), 0644)

	mustChdir(t, dir)

	out, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var batch BatchFixResult
	if err := json.Unmarshal([]byte(out), &batch); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}

	// Verify file was modified: estado should be "Pending" (base value).
	content := mustReadFile(t, filepath.Join(dir, "task.md"))
	contentStr := string(content)
	if !strings.Contains(contentStr, "estado: Pending") {
		t.Errorf("expected estado: Pending after migrate_value, got:\n%s", contentStr)
	}
	// Wiki-links should have been inserted.
	if !strings.Contains(contentStr, "[[blocks:T001]]") {
		t.Errorf("expected [[blocks:T001]] wiki-link inserted, got:\n%s", contentStr)
	}

	// Check batch result reports the fix.
	var found bool
	for _, r := range batch.Results {
		if strings.Contains(r.Path, "task") {
			found = true
			if !r.Fixed {
				t.Error("expected fixed=true for task.md")
			}
		}
	}
	if !found {
		t.Error("task.md not found in results")
	}
}

func TestApplyMigrateValueWithNotes(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
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
`
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(stemContent), 0644)

	// Value with mixed targets and notes.
	mustWriteFile(t, filepath.Join(dir, "blocked.md"), []byte("---\nestado: \"Pending (blocked by E04 + human)\"\n---\n# Blocked\n\n## Details\n\nText here\n"), 0644)

	mustChdir(t, dir)

	_, err := runCmd(t, "fix", "--all", "--output", "json")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := mustReadFile(t, filepath.Join(dir, "blocked.md"))
	contentStr := string(content)
	if !strings.Contains(contentStr, "estado: Pending") {
		t.Errorf("expected estado: Pending, got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "[[blocks:E04]]") {
		t.Errorf("expected [[blocks:E04]], got:\n%s", contentStr)
	}
	if !strings.Contains(contentStr, "[[note:human]]") {
		t.Errorf("expected [[note:human]], got:\n%s", contentStr)
	}
}
