package index

import (
	"path/filepath"

	"github.com/pablontiv/rootline/internal/rules"
)

// ScopeResolver returns the effective StemFile for a given directory.
// Used by the scanner to filter files by scope before extraction.
type ScopeResolver func(dir string) *rules.StemFile

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
