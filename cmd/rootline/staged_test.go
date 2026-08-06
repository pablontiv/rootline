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

	// An empty index is still a validated corpus of size zero. Writing nothing
	// breaks `rootline validate --staged | jq -e '.summary.invalid == 0'`,
	// which is exactly the pre-commit hook this flag exists for.
	env := decodeEnvelope(t, out)
	if env["kind"] != "rootline/validate-batch" {
		t.Errorf("kind = %v, want rootline/validate-batch", env["kind"])
	}
	summary := env["summary"].(map[string]any)
	if summary["total"].(float64) != 0 {
		t.Errorf("summary.total = %v, want 0", summary["total"])
	}
	if summary["invalid"].(float64) != 0 {
		t.Errorf("summary.invalid = %v, want 0", summary["invalid"])
	}
}

// TestGetStagedFilesIgnoresAmbientGitScope pins the fixture against an inherited git
// scope. `getStagedFiles` runs `git diff --cached` with the process environment, which
// is correct in production: a pre-commit hook wants the index git handed it. But git
// exports GIT_DIR / GIT_INDEX_FILE into every hook it runs, so the same suite executed
// from `.githooks/pre-push` read the outer repository's index instead of the fixture's
// and reported zero staged files. Issue #121 cleared the environment for the fixture's
// own git writes; the production reader was still inheriting it.
func TestGetStagedFilesIgnoresAmbientGitScope(t *testing.T) {
	foreign := t.TempDir()
	runFixtureGit(t, foreign, "init", "--quiet")
	t.Setenv("GIT_DIR", filepath.Join(foreign, ".git"))
	t.Setenv("GIT_INDEX_FILE", filepath.Join(foreign, ".git", "index"))

	dir := makeStagedRepo(t, map[string]string{"document.md": "# Document\n"})
	mustChdir(t, dir)

	files, err := getStagedFiles()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(files) != 1 || files[0] != "document.md" {
		t.Fatalf("expected only document.md, got: %v", files)
	}
}

func makeStagedRepo(t *testing.T, stagedFiles map[string]string) string {
	t.Helper()
	isolateAmbientGitScope(t)
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

// isolateAmbientGitScope removes every repo-scoping git variable from the test process
// for the duration of the test, restoring the previous values afterwards. Production
// code deliberately inherits these — see TestGetStagedFilesIgnoresAmbientGitScope — so
// a fixture repository is only authoritative once the ambient scope is gone.
func isolateAmbientGitScope(t *testing.T) {
	t.Helper()
	for _, name := range gitenv.ScopingVars() {
		previous, present := os.LookupEnv(name)
		if !present {
			continue
		}
		if err := os.Unsetenv(name); err != nil {
			t.Fatalf("unset %s: %v", name, err)
		}
		t.Cleanup(func() {
			if err := os.Setenv(name, previous); err != nil {
				t.Fatalf("restore %s: %v", name, err)
			}
		})
	}
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
