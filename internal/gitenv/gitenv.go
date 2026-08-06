package gitenv

import (
	"os"
	"strings"
)

// ClearedEnv returns os.Environ() with repo-scoping git environment variables removed.
// Preserves all other environment variables, including authentication config
// (GIT_SSH_COMMAND, GIT_ASKPASS, SSH_AUTH_SOCK), PATH, HOME, and locale settings.
//
// Why: issue #81 — an inherited GIT_DIR, GIT_WORK_TREE, or similar repo-scoping variable
// in the caller's environment causes nested git invocations (in templates fetching or
// migration operations) to inadvertently write to the caller's repository instead of
// their intended targets. ClearedEnv() creates a clean environment for git commands
// that respects the explicit repository scope (passed via -C, etc.) without inheriting
// caller's repo context.
func ClearedEnv() []string {
	removed := make(map[string]bool, len(scopingVars))
	for _, name := range scopingVars {
		removed[name] = true
	}

	var cleared []string
	for _, env := range os.Environ() {
		if idx := strings.Index(env, "="); idx > 0 {
			name := env[:idx]
			if !removed[name] {
				cleared = append(cleared, env)
			}
		}
	}
	return cleared
}

// scopingVars is the denylist of repo-scoping variables ClearedEnv removes.
var scopingVars = []string{
	"GIT_DIR",
	"GIT_WORK_TREE",
	"GIT_INDEX_FILE",
	"GIT_OBJECT_DIRECTORY",
	"GIT_ALTERNATE_OBJECT_DIRECTORIES",
	"GIT_COMMON_DIR",
	"GIT_NAMESPACE",
	"GIT_CEILING_DIRECTORIES",
	"GIT_DISCOVERY_ACROSS_FILESYSTEM",
	"GIT_PREFIX",
	"GIT_INDEX_VERSION",
}

// ScopingVars returns the names of the repo-scoping git variables ClearedEnv removes.
//
// ClearedEnv covers the common case — a git subprocess that must target its own
// directory. A caller that instead needs its OWN process to stop inheriting the
// caller's repository (a test fixture, or any code re-entered from a git hook, which
// git always invokes with GIT_DIR and GIT_INDEX_FILE exported) has to unset the names
// itself; this is that list, so the two never drift apart.
func ScopingVars() []string {
	names := make([]string, len(scopingVars))
	copy(names, scopingVars)
	return names
}
