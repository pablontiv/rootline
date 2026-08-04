package migrate

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/gitenv"
	"github.com/pablontiv/rootline/internal/rules"
)

// LoadPreviousStem loads the previous version of a .stem file.
// If fromPath is specified, it reads from that file.
// Otherwise, it attempts to read from git (HEAD version).
func LoadPreviousStem(stemPath, fromPath string) (*rules.StemFile, error) {
	if fromPath != "" {
		return rules.ParseStemFile(fromPath)
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
		// File might not exist in HEAD (new .stem).
		return nil, fmt.Errorf("git show HEAD:%s: %w (file may not exist in HEAD)", relPath, err)
	}

	return rules.ParseStem(stemPath, out)
}

// findGitRoot walks up from dir to find the .git directory.
func findGitRoot(dir string) (string, error) {
	current := dir
	for {
		gitPath := filepath.Join(current, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			return current, nil
		}
		parent := filepath.Dir(current)
		if parent == current {
			return "", fmt.Errorf("no .git directory found")
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
