package main

import (
	"strings"
	"testing"
)

func TestGetStagedFiles(t *testing.T) {
	// This runs in a git repo, so getStagedFiles should not error
	files, err := getStagedFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Result may be empty (no staged files) — that's OK
	_ = files
}

func TestGetStagedFiltersMarkdown(t *testing.T) {
	// getStagedFiles should only return .md files
	// We can't easily stage files in tests, but we can verify
	// the function doesn't crash
	files, err := getStagedFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	for _, f := range files {
		if !strings.HasSuffix(f, ".md") {
			t.Errorf("expected only .md files, got: %s", f)
		}
	}
}

func TestValidateStagedNoFiles(t *testing.T) {
	// With nothing staged, --staged should return exit code 0
	out, err := runCmd(t, "validate", "--staged", "--all")
	if err != nil {
		t.Fatalf("expected no error with empty staging area, got: %v", err)
	}
	// Output should be empty (nothing to validate)
	_ = out
}
