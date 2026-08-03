package rules

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"slices"
	"strings"

	"golang.org/x/term"
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

// DownwardScan collects every .stem at or below targetPath that has not
// declared, and is not covered by, a governance boundary.
//
// WalkUp answers "what governs this path", which is enough to resolve a schema
// but not to judge one: it looks only upward, so the same tree could pass from
// one directory and fail from another that happened to sit above the only
// .stem. Whether a project declares where it starts is a property of the tree,
// so the question has to be asked downward too.
//
// A .stem carrying root: true has already answered it. Its subtree is governed
// and is skipped whole — reporting the stems below a declared boundary as
// undeclared would be the same mistake in the other direction.
//
// Return semantics mirror the tree, not the walk:
//   - entries (non-empty): every ungoverned .stem, root-most first
//   - nil, nil: nothing left undeclared, including the tree with no .stem at
//     all, which is ungoverned rather than misgoverned
//   - error: a .stem that cannot be read or parsed, reported where it is found
//     instead of collected around
//
// The traversal shares index.Scan's universe — it honours .stemignore and never
// enters .git — because a file the project has excluded governs nothing and
// cannot be the reason a command is refused. filepath.WalkDir orders entries
// lexically and stats with Lstat, so the result is deterministic and symlinks
// are not followed out of the subtree.
func DownwardScan(targetPath string) ([]StemEntry, error) {
	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return nil, err
	}

	scanRoot := absTarget
	if info, statErr := os.Lstat(absTarget); statErr == nil && !info.IsDir() {
		scanRoot = filepath.Dir(absTarget)
	}

	var entries []StemEntry
	var ignores []stemIgnoreScope

	walkErr := filepath.WalkDir(scanRoot, func(dir string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			// An unreadable directory is not a governance verdict. A target
			// that does not exist belongs to the command that was given it.
			if d != nil && d.IsDir() {
				return fs.SkipDir
			}
			return nil //nolint:nilerr // absence and unreadability are the caller's to report
		}
		if !d.IsDir() {
			return nil
		}
		if d.Name() == ".git" && dir != scanRoot {
			return fs.SkipDir
		}

		if patterns, loadErr := parseStemIgnore(filepath.Join(dir, ".stemignore")); loadErr == nil {
			ignores = append(ignores, stemIgnoreScope{dir: dir, patterns: patterns})
		}

		stemPath := filepath.Join(dir, stemFileName)
		info, statErr := os.Lstat(stemPath)
		if statErr != nil || info.IsDir() {
			return nil
		}
		if stemIsIgnored(stemPath, ignores) {
			return nil
		}

		content, readErr := os.ReadFile(stemPath)
		if readErr != nil {
			return readErr
		}
		stem, parseErr := ParseStem(stemPath, content)
		if parseErr != nil {
			return parseErr
		}

		if stem.Root {
			return fs.SkipDir
		}
		entries = append(entries, StemEntry{Path: stemPath, Stem: stem})
		return nil
	})
	if walkErr != nil {
		return nil, walkErr
	}

	return entries, nil
}

// stemIgnoreScope holds the patterns of one .stemignore and the directory they
// apply to.
type stemIgnoreScope struct {
	dir      string
	patterns []string
}

// parseStemIgnore reads a .stemignore, dropping blanks and comments. A missing
// file is reported as an error so callers simply skip it.
func parseStemIgnore(path string) ([]string, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return nil, err
	}

	var patterns []string
	for _, line := range strings.Split(string(content), "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		patterns = append(patterns, line)
	}
	return patterns, nil
}

// stemIsIgnored matches absPath against every .stemignore that covers it, by
// base name and by path relative to the ignoring directory.
//
// This mirrors internal/index, which cannot be imported here: it already
// depends on this package, and inverting that would make the schema domain
// depend on the scanner that reads it.
func stemIsIgnored(absPath string, scopes []stemIgnoreScope) bool {
	name := filepath.Base(absPath)
	for _, scope := range scopes {
		// Compare with a separator so /sub never covers /sub-extra.
		if absPath != scope.dir && !strings.HasPrefix(absPath, scope.dir+string(filepath.Separator)) {
			continue
		}
		relFromIgnore, err := filepath.Rel(scope.dir, absPath)
		if err != nil {
			continue
		}
		for _, pattern := range scope.patterns {
			if matched, _ := filepath.Match(pattern, name); matched {
				return true
			}
			if matched, _ := filepath.Match(pattern, relFromIgnore); matched {
				return true
			}
		}
	}
	return false
}

// IsRealSchemaError reports whether err is a genuine IO or parse failure rather
// than the absence of a schema.
//
// WalkUp reports ErrNoSchemaFound and a real failure through the same channel,
// but they are not the same answer: a tree with no .stem is ungoverned, while
// one whose .stem cannot be read or parsed is broken. Commands that can still
// do useful work without a schema use this to tell the two apart instead of
// treating every non-nil error as fatal.
func IsRealSchemaError(err error) bool {
	return err != nil && !errors.Is(err, ErrNoSchemaFound)
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

// ProposeRootDirectory returns the directory whose .stem should carry the
// root: true marker for the given target.
//
// The topmost entry in the chain is NOT automatically the answer. When the walk
// collected a .stem from an ancestor outside the project, proposing that
// directory would declare the stray file as the governance root and lock it in
// permanently — the opposite of what the migration is for.
//
// The project boundary is inferred from a .git directory at or above the
// target. Git is consulted as a HINT ONLY: its absence must never break the
// proposal, and it never bounds discovery itself. With a hint, the proposal is
// the outermost .stem at or below that boundary. Without one, there is no
// evidence that any ancestor belongs to the project, so the proposal stays
// conservative and does not reach above the target's own nearest .stem.
//
// Returns "" when the chain is empty.
func ProposeRootDirectory(entries []StemEntry, targetPath string) string {
	if len(entries) == 0 {
		return ""
	}

	boundary := projectBoundaryHint(targetPath)

	// With a hint, take the outermost .stem that lies at or below it.
	if boundary != "" {
		for _, entry := range entries {
			if dir := filepath.Dir(entry.Path); isAtOrBelow(dir, boundary) {
				return dir
			}
		}
	}

	// No hint, or nothing in the chain sits inside it: fall back to the entry
	// closest to the target. Proposing too narrow a root is recoverable;
	// proposing a directory outside the project is not.
	return filepath.Dir(entries[len(entries)-1].Path)
}

// projectBoundaryHint returns the nearest directory at or above targetPath that
// contains a .git entry, or "" when there is none. This is advisory only — it
// suggests where a project starts, and never bounds schema discovery.
func projectBoundaryHint(targetPath string) string {
	dir, err := filepath.Abs(targetPath)
	if err != nil {
		return ""
	}
	if info, statErr := os.Stat(dir); statErr == nil && !info.IsDir() {
		dir = filepath.Dir(dir)
	}

	for {
		if _, statErr := os.Stat(filepath.Join(dir, ".git")); statErr == nil {
			return dir
		}
		parent := filepath.Dir(dir)
		if parent == dir {
			return ""
		}
		dir = parent
	}
}

// isAtOrBelow reports whether dir is root itself or nested inside it. It
// compares path components, so /home/user2 is not treated as inside /home/user.
func isAtOrBelow(dir, root string) bool {
	rel, err := filepath.Rel(root, dir)
	if err != nil {
		return false
	}
	return rel == "." || (rel != ".." && !strings.HasPrefix(rel, ".."+string(filepath.Separator)))
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
// A stray path may be empty, meaning the walk found no .stem outside the
// project — the boundary is simply undeclared.
func ComposeNoDeclaredBoundaryError(strayStemPath, proposedDir string) string {
	var b strings.Builder

	b.WriteString("Schema discovery reached the filesystem root without finding a declared boundary.\n\n")
	if strayStemPath != "" {
		fmt.Fprintf(&b, "It collected %s, which is outside this project.\n", strayStemPath)
		b.WriteString("That file is now governing records it has nothing to do with.\n\n")
	} else {
		b.WriteString("No .stem in this project declares where the project starts.\n\n")
	}

	fmt.Fprintf(&b, "Fix: add this line to %s\n\n", filepath.Join(proposedDir, stemFileName))
	b.WriteString("  root: true\n\n")
	b.WriteString("Discovery then stops there and never reads .stem files above it.\n")
	b.WriteString("Run rootline in a terminal to be prompted and have this applied for you.\n")

	return b.String()
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
// IsTerminal reports whether stdin is attached to a terminal. Callers must pass
// its result rather than prompting unconditionally: a prompt in CI, a git hook,
// or an agent hangs the pipeline until timeout, which is worse than failing.
//
// This performs a real terminal check (an ioctl) rather than inspecting the
// file mode. Testing os.ModeCharDevice is wrong: /dev/null is itself a
// character device, so `rootline ... < /dev/null` would be misread as
// interactive, prompt into a pipeline, read EOF, and exit successfully —
// silently skipping the failure that non-interactive callers must get.
func IsTerminal() bool {
	return term.IsTerminal(int(os.Stdin.Fd()))
}

// hasStdin must be true only when os.Stdin is a terminal; use isTerminal().
func AttemptRootMarkerMigration(entries []StemEntry, targetPath string, hasStdin bool) MigrationResult {
	// Check if migration is needed
	if !ChainHasNoDeclaredBoundary(entries) {
		return MigrationResult{Applied: false}
	}

	proposedDir := ProposeRootDirectory(entries, targetPath)
	strayPath := entries[0].Path

	// When the chain is entirely inside the proposed root there is no stray
	// ancestor — the project simply has not declared its boundary yet.
	if filepath.Dir(strayPath) == proposedDir {
		strayPath = ""
	}

	// No TTY: fail with error
	if !hasStdin {
		return MigrationResult{
			Applied: false,
			Error:   ComposeNoDeclaredBoundaryError(strayPath, proposedDir),
		}
	}

	// Has TTY: prompt user
	if strayPath != "" {
		fmt.Fprintf(os.Stderr, "Your .stem chain includes %q, which is outside this project.\n", strayPath)
	} else {
		fmt.Fprintf(os.Stderr, "No .stem in this project declares a governance boundary.\n")
	}
	fmt.Fprintf(os.Stderr, "To establish one, add 'root: true' to %s/.stem.\n\n", proposedDir)
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
