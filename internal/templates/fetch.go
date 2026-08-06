package templates

import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"github.com/pablontiv/rootline/internal/fsx"
	"github.com/pablontiv/rootline/internal/gitenv"
	"gopkg.in/yaml.v3"
)

// ParseRef parses "owner/repo[@tag]" or "github.com/owner/repo[@tag]".
// Returns owner, repo, tag. Returns empty strings for invalid refs.
func ParseRef(ref string) (owner, repo, tag string) {
	// Strip "github.com/" prefix if present.
	ref = strings.TrimPrefix(ref, "github.com/")

	// Split on last "@" for tag.
	if idx := strings.LastIndex(ref, "@"); idx != -1 {
		tag = ref[idx+1:]
		ref = ref[:idx]
	}

	// Split on "/" for owner/repo.
	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", ""
	}

	return parts[0], parts[1], tag
}

// FetchFromURL clones a git repo URL to a temp dir, finds .stem files,
// validates YAML, and copies them to dest preserving relative paths.
// Supports force (overwrite existing) and dryRun (list only, no copy).
// Uses GIT_TERMINAL_PROMPT=0 and a 30s timeout.
func FetchFromURL(url, tag, dest string, force, dryRun bool) ([]string, error) {
	// Verify git is available.
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git is required for --template")
	}

	// Create temp dir for clone.
	tmpDir, err := os.MkdirTemp("", "rootline-template-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer func() { _ = os.RemoveAll(tmpDir) }()

	// Build git clone args.
	cloneArgs := []string{"clone", "--depth", "1"}
	if tag != "" {
		cloneArgs = append(cloneArgs, "--branch", tag)
	}
	cloneArgs = append(cloneArgs, url, tmpDir)

	// Run git clone with 30s timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	gitCmd := exec.CommandContext(ctx, "git", cloneArgs...)
	gitCmd.Env = append(gitenv.ClearedEnv(), "GIT_TERMINAL_PROMPT=0")
	out, err := gitCmd.CombinedOutput()
	if err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("git clone timed out after 30s")
		}
		outStr := string(out)
		if strings.Contains(outStr, "Authentication failed") ||
			strings.Contains(outStr, "could not read Username") ||
			strings.Contains(outStr, "terminal prompts disabled") {
			return nil, fmt.Errorf("authentication required for %s (private repositories are not supported)", url)
		}
		return nil, fmt.Errorf("git clone failed: %w\n%s", err, outStr)
	}

	// Walk tmpDir to find .stem files, skipping .git.
	var stemFiles []string
	err = filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		// Skip .git directory.
		if d.IsDir() && d.Name() == ".git" {
			return filepath.SkipDir
		}
		if !d.IsDir() && d.Name() == ".stem" {
			stemFiles = append(stemFiles, path)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking cloned repo: %w", err)
	}

	if len(stemFiles) == 0 {
		return nil, fmt.Errorf("no .stem files found in %s", url)
	}

	// Validate YAML and compute relative paths.
	type stemEntry struct {
		absPath string
		relPath string
	}
	entries := make([]stemEntry, 0, len(stemFiles))
	for _, absPath := range stemFiles {
		data, err := os.ReadFile(absPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", absPath, err)
		}
		var v any
		if err := yaml.Unmarshal(data, &v); err != nil {
			relName, _ := filepath.Rel(tmpDir, absPath)
			return nil, fmt.Errorf("invalid .stem file %s: %w", relName, err)
		}
		relPath, err := filepath.Rel(tmpDir, absPath)
		if err != nil {
			return nil, fmt.Errorf("computing relative path: %w", err)
		}
		entries = append(entries, stemEntry{absPath: absPath, relPath: relPath})
	}

	// Build result list (relative paths from dest perspective).
	result := make([]string, 0, len(entries))
	for _, e := range entries {
		result = append(result, e.relPath)
	}

	if dryRun {
		return result, nil
	}

	// Copy files to dest.
	for _, e := range entries {
		destPath := filepath.Join(dest, e.relPath)
		if !force {
			if _, err := os.Stat(destPath); err == nil {
				return nil, fmt.Errorf("%s already exists (use --force to overwrite)", e.relPath)
			}
		}

		// Ensure destination directory exists.
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil, fmt.Errorf("creating directory for %s: %w", e.relPath, err)
		}

		if err := copyFile(e.absPath, destPath); err != nil {
			return nil, fmt.Errorf("copying %s: %w", e.relPath, err)
		}
	}

	return result, nil
}

// FetchTemplate parses ref, resolves the GitHub URL, and calls FetchFromURL.
func FetchTemplate(ref, dest string, force, dryRun bool) ([]string, error) {
	owner, repo, tag := ParseRef(ref)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid template ref %q: expected owner/repo[@tag]", ref)
	}

	url := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	return FetchFromURL(url, tag, dest, force, dryRun)
}

// copyFile atomically copies src to dst without exposing partial content.
func copyFile(src, dst string) error {
	return fsx.CopyFileAtomic(src, dst)
}
