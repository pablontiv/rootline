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
	// Denylist of repo-scoping variables to remove.
	removed := map[string]bool{
		"GIT_DIR":                          true,
		"GIT_WORK_TREE":                    true,
		"GIT_INDEX_FILE":                   true,
		"GIT_OBJECT_DIRECTORY":             true,
		"GIT_ALTERNATE_OBJECT_DIRECTORIES": true,
		"GIT_COMMON_DIR":                   true,
		"GIT_NAMESPACE":                    true,
		"GIT_CEILING_DIRECTORIES":          true,
		"GIT_DISCOVERY_ACROSS_FILESYSTEM":  true,
		"GIT_PREFIX":                       true,
		"GIT_INDEX_VERSION":                true,
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
