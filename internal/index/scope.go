package index

import (
	"path/filepath"

	"github.com/pablontiv/rootline/internal/rules"
)

// ScopeResolver returns the effective StemFile for a given directory, or an error.
// A hard error (parse/IO failure) aborts the scan.
// ErrNoSchemaFound indicates no schema applies to this directory and that directory should be skipped.
// Used by the scanner to filter files by scope before extraction.
type ScopeResolver func(dir string) (*rules.StemFile, error)

// MatchesScope checks whether a file path matches the scope.match
// pattern from an effective StemFile. If the stem is nil or has no
// scope.match defined, all files match (no scope = match everything).
// Uses filepath.Match for glob evaluation.
func MatchesScope(filePath string, effectiveStem *rules.StemFile) bool {
	if effectiveStem == nil || effectiveStem.Scope.Match == "" {
		return true
	}
	name := filepath.Base(filePath)
	matched, err := filepath.Match(effectiveStem.Scope.Match, name)
	if err != nil {
		// Invalid pattern — fail open (don't silently exclude files).
		return true
	}
	return matched
}

// IsIgnored reports whether absPath is excluded by a .stemignore anywhere
// between root and the file's own directory.
//
// Scan applies .stemignore while walking, but a command handed an explicit
// file never walks. Without this, `validate <file>` checked a file that
// `validate --all` skipped, so the pre-commit hook and CI enforced different
// rules on the same file.
func IsIgnored(root, absPath string) bool {
	var stack []ignoreEntry
	dir := filepath.Dir(absPath)
	for {
		if patterns, err := parseStemignore(filepath.Join(dir, ".stemignore")); err == nil && len(patterns) > 0 {
			stack = append(stack, ignoreEntry{dir: dir, patterns: patterns})
		}
		if dir == root || dir == filepath.Dir(dir) {
			break
		}
		dir = filepath.Dir(dir)
	}
	return isIgnored(absPath, stack)
}
