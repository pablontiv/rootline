package rules

import (
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
)

const stemFileName = ".stem"

// ErrNoSchemaFound is returned when no .stem file is found in the directory chain.
//
// The remediation hint is part of the message on purpose. Commands used to
// answer this condition with a successful, empty result carrying a "run
// rootline init" hint; now that it is a hard error, the guidance travels with
// the error so the help is not lost.
var ErrNoSchemaFound = fmt.Errorf("no .stem file found in the directory chain; run 'rootline init' to create one")

// StemEntry pairs a .stem file path with its parsed content.
type StemEntry struct {
	Path string
	Stem *StemFile
}

// WalkUp walks from targetPath upward, collecting every .stem file it finds,
// and terminates at a .stem that declares `root: true`.
//
// A directory without a .stem is NOT a stop condition. It contributes nothing
// to the chain and the walk continues upward. This preserves inheritance: a
// directory omits .stem precisely because it adds no rules of its own, and its
// records inherit from above unchanged.
//
// The ONLY stop conditions are a `root: true` marker (the marker's own .stem IS
// collected before stopping) and the filesystem root.
//
// Return semantics:
//   - entries (non-empty): every .stem from the boundary down to targetPath,
//     ordered root-to-leaf, ready for top-down merge
//   - error ErrNoSchemaFound: the walk found zero .stem files
//   - other errors: filesystem I/O or parse failures
//
// There is deliberately no "empty entries, no error" case. Zero .stem is exactly
// the no-schema condition and is always an error, so no caller can silently
// proceed with an empty schema — that collapse is the current false-green bug.
//
// Algorithm:
//
//	dir := absTarget; if isFile(absTarget) { dir = parentOf(absTarget) }
//	entries := []
//	for {
//	    if exists(dir / ".stem"):
//	        stem := parse(dir / ".stem")     // parse failure => hard error
//	        append stem to entries           // gaps simply append nothing
//	        if stem.Root:                    // marker collected, then stop
//	            break
//	    parent := parentOf(dir)
//	    if parent == dir:                    // filesystem root
//	        break
//	    dir = parent
//	}
//
//	reverse(entries)                         // collected leaf-to-root
//	if len(entries) == 0:
//	    return ErrNoSchemaFound
//	return entries
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
		// Check if .stem exists in this directory.
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

			// If this .stem has root: true, stop the walk.
			if stem.Root {
				break
			}
		}

		// Move up to parent directory.
		parent := filepath.Dir(dir)
		if parent == dir {
			// Reached filesystem root.
			break
		}
		dir = parent
	}

	// Reverse: discovery order is leaf-to-root, we need root-to-leaf.
	slices.Reverse(entries)

	// If no .stem was found anywhere, return ErrNoSchemaFound.
	if len(entries) == 0 {
		return nil, ErrNoSchemaFound
	}

	return entries, nil
}

// WarnIfChainCrossesProjectBoundary checks if the resolved stem chain includes
// entries that appear to be outside the project, such as in a home directory.
// It uses heuristics: home directory check and distance heuristic.
//
// Returns a non-empty string if a potential boundary crossing is detected,
// empty string otherwise.
func WarnIfChainCrossesProjectBoundary(entries []StemEntry, startPath string) string {
	if len(entries) == 0 {
		return ""
	}

	// Heuristic 1: Check if any entry is in $HOME
	homeDir := os.Getenv("HOME")
	if homeDir != "" {
		for _, entry := range entries {
			// If a stem is in HOME and startPath is NOT in HOME, we've crossed the boundary
			if strings.HasPrefix(entry.Path, homeDir) && !strings.HasPrefix(startPath, homeDir) {
				return fmt.Sprintf(
					"Warning: .stem chain includes %q (in home directory); "+
						"consider adding 'root: true' to the top-level .stem in your project",
					entry.Path,
				)
			}
		}
	}

	// Heuristic 2: Check if the root entry is very far from the start
	// (more than 5 levels up suggests we've crossed an implicit boundary)
	var startDir string
	if info, err := os.Stat(startPath); err == nil && info.IsDir() {
		startDir = startPath
	} else {
		startDir = filepath.Dir(startPath)
	}

	rootDir := filepath.Dir(entries[0].Path)
	rel, _ := filepath.Rel(rootDir, startDir)
	levelCount := strings.Count(rel, string(filepath.Separator)) + 1

	if levelCount > 5 {
		return fmt.Sprintf(
			"Warning: .stem root is %d levels above the project; "+
				"consider adding 'root: true' to a .stem closer to your project root",
			levelCount,
		)
	}

	return ""
}
