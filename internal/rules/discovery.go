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

// ChainHasNoDeclaredBoundary checks if the walk terminated at the filesystem root
// without a root: true marker.
// Returns true if entries[0].Stem.Root is false (walk stopped at fs root, not a marker).
// Returns false if entries[0].Stem.Root is true (walk stopped at a marker).
func ChainHasNoDeclaredBoundary(entries []StemEntry) bool {
	if len(entries) == 0 {
		return true // No entries means no boundary
	}
	// entries are ordered root-to-leaf; entries[0] is the topmost
	return !entries[0].Stem.Root
}

// ProposeRootDirectory returns the directory containing the first (topmost) .stem
// in the chain. This is the directory that should be marked with root: true.
func ProposeRootDirectory(entries []StemEntry) string {
	if len(entries) == 0 {
		return ""
	}
	// entries[0] is the topmost .stem; extract its directory
	return filepath.Dir(entries[0].Path)
}

// ApplyRootMarker adds "root: true" to a .stem file, preserving existing content.
// It is idempotent: if "root: true" is already present, no action is taken.
func ApplyRootMarker(stemPath string) error {
	content, err := os.ReadFile(stemPath)
	if err != nil {
		return err
	}

	contentStr := string(content)

	// Check if root: true already exists
	if strings.Contains(contentStr, "root: true") {
		return nil
	}

	// Insert "root: true" after the version line
	// Find the first newline after "version:"
	versionIdx := strings.Index(contentStr, "version:")
	if versionIdx == -1 {
		// No version line; prepend root: true
		contentStr = "root: true\n" + contentStr
	} else {
		// Find the end of the version line
		newlineIdx := strings.Index(contentStr[versionIdx:], "\n")
		if newlineIdx == -1 {
			// No newline after version; append at end
			contentStr += "\nroot: true"
		} else {
			// Insert after the first newline
			insertPos := versionIdx + newlineIdx + 1
			contentStr = contentStr[:insertPos] + "root: true\n" + contentStr[insertPos:]
		}
	}

	// #nosec G703 - stemPath is derived from os.ReadFile() result, not user input
	return os.WriteFile(stemPath, []byte(contentStr), 0o644)
}

// ComposeNoDeclaredBoundaryError creates a clear error message that tells the user
// what happened, which .stem was picked up, which directory should own the marker,
// and the exact line to add. This message is meant for non-interactive environments
// where the user cannot be prompted for confirmation.
func ComposeNoDeclaredBoundaryError(strayStemmPath, proposedDir string) string {
	return fmt.Sprintf(
		"Schema discovery walk reached the filesystem root without a declared boundary.\n"+
			"The .stem at %q was collected outside the project.\n"+
			"To fix this, add the following line to %s/.stem:\n\n"+
			"  root: true\n\n"+
			"This establishes a clear governance boundary and prevents ancestor .stem files from affecting this project.\n",
		strayStemmPath,
		proposedDir,
	)
}

// MigrationResult holds the outcome of a root marker migration attempt.
type MigrationResult struct {
	Applied bool   // true if root marker was applied
	Error   string // non-empty if an error occurred
}

// AttemptRootMarkerMigration handles the interactive or non-interactive root marker
// migration. It checks if a TTY is available and prompts the user if one is.
// If no TTY, it returns an error with the full remediation instructions.
// If user confirms, it applies the marker and returns success.
// If user declines, it returns without error (no changes made).
//
// hasStdin should be true when os.Stdin is a terminal (use !isatty.IsNotATerminal(int(os.Stdin.Fd())))
func AttemptRootMarkerMigration(entries []StemEntry, hasStdin bool) MigrationResult {
	// Check if migration is needed
	if !ChainHasNoDeclaredBoundary(entries) {
		return MigrationResult{Applied: false}
	}

	strayPath := entries[0].Path
	proposedDir := ProposeRootDirectory(entries)

	// No TTY: fail with error
	if !hasStdin {
		return MigrationResult{
			Applied: false,
			Error:   ComposeNoDeclaredBoundaryError(strayPath, proposedDir),
		}
	}

	// Has TTY: prompt user
	fmt.Fprintf(os.Stderr, "Your .stem chain includes %q which is outside this project.\n", strayPath)
	fmt.Fprintf(os.Stderr, "To establish a clear boundary, add 'root: true' to %s/.stem.\n\n", proposedDir)
	fmt.Fprintf(os.Stderr, "Apply this change now? (y/n) ")

	// Simple confirmation: read one character
	var response [1]byte
	n, err := os.Stdin.Read(response[:])
	if err != nil || n == 0 {
		return MigrationResult{Applied: false}
	}

	if response[0] != 'y' && response[0] != 'Y' {
		fmt.Fprintf(os.Stderr, "Skipped.\n")
		return MigrationResult{Applied: false}
	}

	fmt.Fprintf(os.Stderr, "Applying root marker to %s/.stem...\n", proposedDir)

	// Apply the marker
	stemPath := filepath.Join(proposedDir, stemFileName)
	if err := ApplyRootMarker(stemPath); err != nil {
		return MigrationResult{
			Applied: false,
			Error:   fmt.Sprintf("failed to apply root marker: %v", err),
		}
	}

	fmt.Fprintf(os.Stderr, "Root marker applied successfully.\n")
	return MigrationResult{Applied: true}
}
