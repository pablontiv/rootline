package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/gitenv"
)

func TestGetStagedFiles(t *testing.T) {
	dir := makeStagedRepo(t, nil)
	mustChdir(t, dir)

	files, err := getStagedFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 0 {
		t.Fatalf("expected no staged files, got: %v", files)
	}
}

func TestGetStagedFiltersMarkdown(t *testing.T) {
	dir := makeStagedRepo(t, map[string]string{
		"document.md": "# Document\n",
		"notes.txt":   "not markdown\n",
	})
	mustChdir(t, dir)

	files, err := getStagedFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "document.md" {
		t.Fatalf("expected only document.md, got: %v", files)
	}
}

func TestValidateStagedNoFiles(t *testing.T) {
	dir := makeStagedRepo(t, nil)
	mustChdir(t, dir)

	out, err := runCmd(t, "validate", "--staged", "--all")
	if err != nil {
		t.Fatalf("expected no error with empty staging area, got: %v", err)
	}
	if out != "" {
		t.Fatalf("expected empty output, got: %s", out)
	}
}

func makeStagedRepo(t *testing.T, stagedFiles map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	runFixtureGit(t, dir, "init", "--quiet")

	for path, content := range stagedFiles {
		fullPath := filepath.Join(dir, path)
		if err := os.MkdirAll(filepath.Dir(fullPath), 0o755); err != nil {
			t.Fatalf("create fixture directory: %v", err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0o644); err != nil {
			t.Fatalf("write fixture file: %v", err)
		}
		runFixtureGit(t, dir, "add", "--", path)
	}

	return dir
}

func runFixtureGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec
	cmd.Dir = dir
	cmd.Env = gitenv.ClearedEnv()
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
