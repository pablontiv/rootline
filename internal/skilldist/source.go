package skilldist

import (
	"bytes"
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/gitenv"
)

func ResolveSource(ctx context.Context, requested string) (Source, error) {
	if requested == "" {
		return Source{}, operationError(ErrSourceNotCanonical, "", "", fmt.Errorf("source path is required"))
	}

	requestedPath, err := filepath.Abs(filepath.Clean(requested))
	if err != nil {
		return Source{}, operationError(ErrSourceNotCanonical, requested, "", fmt.Errorf("resolve source path: %w", err))
	}
	if err := rejectLinkedGitMarker(requestedPath); err != nil {
		return Source{}, err
	}

	repoRoot, err := gitOutput(ctx, requestedPath, "rev-parse", "--show-toplevel")
	if err != nil {
		return Source{}, operationError(ErrSourceNotCanonical, requestedPath, "", err)
	}
	repoRoot = filepath.Clean(repoRoot)
	if sameDirectory(requestedPath, repoRoot) {
		repoRoot = requestedPath
	}

	if err := requirePrimaryCheckout(repoRoot); err != nil {
		return Source{}, err
	}

	skillPath := filepath.Join(repoRoot, ".claude", "skills", "rootline")
	skillFile := filepath.Join(skillPath, "SKILL.md")
	info, err := os.Stat(skillFile)
	if err != nil {
		return Source{}, operationError(ErrSourceNotCanonical, skillFile, "", fmt.Errorf("canonical skill file is required"))
	}
	if !info.Mode().IsRegular() {
		return Source{}, operationError(ErrSourceNotCanonical, skillFile, "", fmt.Errorf("canonical skill file must be regular"))
	}

	status, err := gitOutput(ctx, repoRoot, "status", "--porcelain", "--", ".claude/skills/rootline")
	if err != nil {
		return Source{}, operationError(ErrSourceNotCanonical, skillPath, "", err)
	}
	if status != "" {
		return Source{}, operationError(ErrSourceNotCanonical, skillPath, "", fmt.Errorf("canonical skill tree has uncommitted changes"))
	}

	commit, err := gitOutput(ctx, repoRoot, "rev-parse", "HEAD")
	if err != nil {
		return Source{}, operationError(ErrSourceNotCanonical, repoRoot, "", err)
	}
	if commit == "" {
		return Source{}, operationError(ErrSourceNotCanonical, repoRoot, "", fmt.Errorf("source commit is empty"))
	}

	digest, err := digestCanonicalTree(skillPath)
	if err != nil {
		return Source{}, err
	}

	return Source{
		RepoRoot:  repoRoot,
		SkillPath: skillPath,
		Commit:    commit,
		Digest:    digest,
	}, nil
}

func sameDirectory(left, right string) bool {
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	return leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo)
}

func requirePrimaryCheckout(repoRoot string) error {
	gitPath := filepath.Join(repoRoot, ".git")
	info, err := os.Stat(gitPath)
	if err != nil {
		return operationError(ErrSourceNotCanonical, gitPath, "", fmt.Errorf("primary checkout metadata is required"))
	}
	if !info.IsDir() {
		return operationError(ErrLinkedWorktreeRefused, gitPath, "", fmt.Errorf("linked worktree metadata is refused"))
	}
	return nil
}

func rejectLinkedGitMarker(path string) error {
	info, err := os.Stat(filepath.Join(path, ".git"))
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return operationError(ErrSourceNotCanonical, path, "", fmt.Errorf("inspect source metadata: %w", err))
	}
	if !info.IsDir() {
		return operationError(ErrLinkedWorktreeRefused, filepath.Join(path, ".git"), "", fmt.Errorf("linked worktree metadata is refused"))
	}
	return nil
}

func gitOutput(ctx context.Context, dir string, args ...string) (string, error) {
	// #nosec G204 -- git executable is fixed; dir is passed to -C and args are package-controlled.
	cmd := exec.CommandContext(ctx, "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gitenv.ClearedEnv()
	var stdout bytes.Buffer
	var stderr bytes.Buffer
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr
	if err := cmd.Run(); err != nil {
		return "", &gitCommandError{args: append([]string(nil), args...), err: err}
	}
	return strings.TrimSpace(stdout.String()), nil
}

type gitCommandError struct {
	args []string
	err  error
}

func (e *gitCommandError) Error() string {
	return fmt.Sprintf("git %s failed", strings.Join(e.args, " "))
}

func (e *gitCommandError) Unwrap() error {
	return e.err
}
