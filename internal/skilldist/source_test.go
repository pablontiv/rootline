package skilldist

import (
	"context"
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/gitenv"
)

func TestResolveSourceRequiresPrimaryCheckoutAndCanonicalSkill(t *testing.T) {
	repo := initSkillRepository(t)
	source, err := ResolveSource(context.Background(), repo)
	if err != nil {
		t.Fatalf("ResolveSource: %v", err)
	}
	if source.RepoRoot != repo {
		t.Fatalf("RepoRoot = %q, want %q", source.RepoRoot, repo)
	}
	if source.SkillPath != filepath.Join(repo, ".claude", "skills", "rootline") {
		t.Fatalf("SkillPath = %q", source.SkillPath)
	}
	if source.Commit == "" || source.Digest == "" {
		t.Fatalf("incomplete source evidence: %#v", source)
	}
}

func TestResolveSourceRejectsLinkedWorktreeMarker(t *testing.T) {
	repo := t.TempDir()
	mustWriteSkillFile(t, filepath.Join(repo, ".claude", "skills", "rootline"), "SKILL.md", "---\nname: rootline\n---\n")
	if err := os.WriteFile(filepath.Join(repo, ".git"), []byte("gitdir: /tmp/common/worktrees/probe\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveSource(context.Background(), repo)
	assertOperationErrorCode(t, err, ErrLinkedWorktreeRefused)
}

func TestResolveSourceRejectsSymlinkInsideCanonicalTree(t *testing.T) {
	repo := initSkillRepository(t)
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(outside, filepath.Join(repo, ".claude", "skills", "rootline", "escape.md")); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	commitSkillRepository(t, repo, "add canonical symlink")
	_, err := ResolveSource(context.Background(), repo)
	assertOperationErrorCode(t, err, ErrSourceNotCanonical)
}

func TestResolveSourceRejectsDirtyCanonicalTree(t *testing.T) {
	repo := initSkillRepository(t)
	mustWriteSkillFile(t, filepath.Join(repo, ".claude", "skills", "rootline"), "untracked.md", "dirty")
	_, err := ResolveSource(context.Background(), repo)
	assertOperationErrorCode(t, err, ErrSourceNotCanonical)
}

func initSkillRepository(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runGit(t, repo, "init")
	runGit(t, repo, "config", "user.name", "Rootline Test")
	runGit(t, repo, "config", "user.email", "rootline-test@example.invalid")
	mustWriteSkillFile(t, filepath.Join(repo, ".claude", "skills", "rootline"), "SKILL.md", "---\nname: rootline\n---\n")
	commitSkillRepository(t, repo, "add canonical skill")
	return repo
}

func commitSkillRepository(t *testing.T, repo, message string) {
	t.Helper()
	runGit(t, repo, "add", "--all")
	runGit(t, repo, "commit", "-m", message)
}

func mustWriteSkillFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func assertOperationErrorCode(t *testing.T, err error, code ErrorCode) {
	t.Helper()
	if err == nil {
		t.Fatalf("expected %s error, got nil", code)
	}
	var opErr *OperationError
	if !errors.As(err, &opErr) {
		t.Fatalf("expected OperationError, got %T: %v", err, err)
	}
	if opErr.Code != code {
		t.Fatalf("OperationError.Code = %q, want %q (err: %v)", opErr.Code, code, err)
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	// #nosec G204 -- test fixture runs fixed git executable with temp repo paths and test-controlled args.
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gitenv.ClearedEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v: %v\n%s", args, err, output)
	}
}
