package migrate

import (
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/gitenv"
	"github.com/pablontiv/rootline/internal/rules"
)

// ErrPreviousStemNotFound means the implicit Git baseline does not exist.
// An explicitly requested baseline never returns this sentinel.
var ErrPreviousStemNotFound = errors.New("previous .stem not found")

// LoadPreviousStem loads the previous version of a .stem file.
// If fromPath is specified, it reads from that file.
// Otherwise, it reads git HEAD, returning ErrPreviousStemNotFound only when the
// repository, HEAD, or requested path is absent. Read and parse failures are errors.
func LoadPreviousStem(stemPath, fromPath string) (*rules.StemFile, error) {
	if fromPath != "" {
		stem, err := rules.ParseStemFile(fromPath)
		if err != nil {
			return nil, fmt.Errorf("loading --from baseline %s: %w", fromPath, err)
		}
		return stem, nil
	}
	return loadFromGit(stemPath)
}

// loadFromGit retrieves the HEAD version of a .stem file from git.
func loadFromGit(stemPath string) (*rules.StemFile, error) {
	absPath, err := filepath.Abs(stemPath)
	if err != nil {
		return nil, fmt.Errorf("resolving path: %w", err)
	}

	// Find git root.
	gitRoot, err := findGitRoot(filepath.Dir(absPath))
	if err != nil {
		return nil, fmt.Errorf("finding git root: %w", err)
	}

	// Compute relative path from git root.
	relPath, err := filepath.Rel(gitRoot, absPath)
	if err != nil {
		return nil, fmt.Errorf("computing relative path: %w", err)
	}

	// Guard: relPath must not escape the git root.
	if strings.HasPrefix(relPath, "..") {
		return nil, fmt.Errorf("path %q is outside the git root %q", absPath, gitRoot)
	}

	// Use git show to read the HEAD version.
	cmd := exec.Command("git", "-C", gitRoot, "show", "HEAD:"+relPath) //nolint:gosec // relPath is derived from filesystem paths, not user input
	cmd.Env = gitenv.ClearedEnv()
	out, err := cmd.Output()
	if err != nil {
		if gitBaselineMissing(gitRoot, relPath) {
			return nil, fmt.Errorf("git HEAD:%s: %w", relPath, ErrPreviousStemNotFound)
		}
		return nil, fmt.Errorf("git show HEAD:%s: %w", relPath, err)
	}

	stem, err := rules.ParseStem(stemPath, out)
	if err != nil {
		return nil, fmt.Errorf("parsing git HEAD:%s baseline: %w", relPath, err)
	}
	return stem, nil
}

// gitBaselineMissing proves absence after git show fails. Other Git failures
// remain errors; successful baseline loads need no extra subprocesses.
func gitBaselineMissing(gitRoot, relPath string) bool {
	cmd := exec.Command("git", "--literal-pathspecs", "-C", gitRoot, "ls-tree", "-z", "HEAD", "--", relPath) //nolint:gosec // paths are derived from the target
	cmd.Env = gitenv.ClearedEnv()
	out, err := cmd.Output()
	if err == nil {
		return len(out) == 0
	}

	// An unborn branch has a symbolic HEAD but no corresponding ref.
	cmd = exec.Command("git", "-C", gitRoot, "symbolic-ref", "--quiet", "HEAD") //nolint:gosec // gitRoot is derived from the target
	cmd.Env = gitenv.ClearedEnv()
	ref, err := cmd.Output()
	if err != nil {
		return false
	}
	cmd = exec.Command("git", "-C", gitRoot, "show-ref", "--verify", "--quiet", strings.TrimSpace(string(ref))) //nolint:gosec // ref comes from Git
	cmd.Env = gitenv.ClearedEnv()
	var exitErr *exec.ExitError
	return errors.As(cmd.Run(), &exitErr) && exitErr.ExitCode() == 1
}

// findGitRoot walks up from dir to find the .git directory.
func findGitRoot(dir string) (string, error) {
	current := dir
	for {
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return current, nil
		} else if !os.IsNotExist(err) {
			return "", fmt.Errorf("checking %s: %w", gitPath, err)
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no .git directory found: %w", ErrPreviousStemNotFound)
		}
		current = parent
	}
}

// FindStemFiles locates all .stem files within a directory.
func FindStemFiles(dir string) ([]string, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}

	var stems []string
	err = filepath.Walk(absDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return nil
		}
		if info.IsDir() {
			// Skip hidden directories (except the target itself).
			base := filepath.Base(path)
			if path != absDir && strings.HasPrefix(base, ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if info.Name() == ".stem" {
			stems = append(stems, path)
		}
		return nil
	})
	return stems, err
}
