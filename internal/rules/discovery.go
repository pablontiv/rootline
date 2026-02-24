package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
)

const stemFileName = ".stem"

// StemEntry pairs a .stem file path with its parsed content.
type StemEntry struct {
	Path string
	Stem *StemFile
}

// WalkUp walks from targetPath up to the repository root (.git boundary),
// collecting and parsing .stem files along the way.
// Returns entries ordered root-to-leaf (ready for top-down merge).
func WalkUp(targetPath string) ([]StemEntry, error) {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, err
	}

	var entries []StemEntry
	dir := absTarget

	// If targetPath is a file, start from its directory.
	if info, err := os.Stat(absTarget); err == nil && !info.IsDir() {
		dir = filepath.Dir(absTarget)
	}

	for {
		// Check for .git — stop signal (check before collecting .stem
		// so we don't go above the repo root, but we DO collect .stem
		// in the same dir as .git).
		stemPath := filepath.Join(dir, stemFileName)
		if _, err := os.Stat(stemPath); err == nil {
			content, err := os.ReadFile(stemPath)
			if err != nil {
				return nil, err
			}
			stem, err := ParseStem(stemPath, content)
			if err != nil {
				return nil, err
			}
			entries = append(entries, StemEntry{Path: stemPath, Stem: stem})
		}

		// Check for .git boundary after collecting .stem in this dir.
		gitPath := filepath.Join(dir, ".git")
		if _, err := os.Stat(gitPath); err == nil {
			break
		}

		// Move up.
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root without finding .git.
			return nil, fmt.Errorf("no .git directory found — rootline must be run inside a git repository")
		}
		dir = parent
	}

	// Reverse: discovery order is leaf-to-root, we need root-to-leaf.
	slices.Reverse(entries)
	return entries, nil
}
