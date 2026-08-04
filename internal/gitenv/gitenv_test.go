package gitenv

import (
	"strings"
	"testing"
)

// TestClearedEnv_RemovesRepoScopingVars verifies all 11 repo-scoping variables are removed.
func TestClearedEnv_RemovesRepoScopingVars(t *testing.T) {
	// Set all 11 repo-scoping variables.
	repoScopingVars := map[string]string{
		"GIT_DIR":                          "/path/to/repo/.git",
		"GIT_WORK_TREE":                    "/path/to/repo",
		"GIT_INDEX_FILE":                   "/path/to/repo/.git/index",
		"GIT_OBJECT_DIRECTORY":             "/path/to/repo/.git/objects",
		"GIT_ALTERNATE_OBJECT_DIRECTORIES": "/path/to/alt/objects",
		"GIT_COMMON_DIR":                   "/path/to/common",
		"GIT_NAMESPACE":                    "namespace",
		"GIT_CEILING_DIRECTORIES":          "/path/to/ceiling",
		"GIT_DISCOVERY_ACROSS_FILESYSTEM":  "1",
		"GIT_PREFIX":                       "prefix",
		"GIT_INDEX_VERSION":                "4",
	}

	for name, value := range repoScopingVars {
		t.Setenv(name, value)
	}

	cleared := envMap(ClearedEnv())

	for name := range repoScopingVars {
		if value, found := cleared[name]; found {
			t.Errorf("ClearedEnv should remove %s but it survived with value %q", name, value)
		}
	}
}

// envMap parses an environment slice into name → value pairs, so assertions match
// variable names exactly instead of by substring.
func envMap(env []string) map[string]string {
	m := make(map[string]string, len(env))
	for _, entry := range env {
		if idx := strings.Index(entry, "="); idx > 0 {
			m[entry[:idx]] = entry[idx+1:]
		}
	}
	return m
}

// TestClearedEnv_PreservesOtherVars verifies non-denylisted variables remain.
func TestClearedEnv_PreservesOtherVars(t *testing.T) {
	preservedVars := map[string]string{ //nolint:gosec
		"PATH":                "/usr/bin:/bin",
		"HOME":                "/home/user",
		"GIT_TERMINAL_PROMPT": "0",
		"GIT_SSH_COMMAND":     "ssh -i /path/to/key",
		"GIT_ASKPASS":         "/usr/bin/ssh-askpass",
		"SSH_AUTH_SOCK":       "/tmp/ssh-sock",
		"http_proxy":          "http://proxy:8080",
		"https_proxy":         "https://proxy:8080",
		"GIT_CONFIG_GLOBAL":   "/home/user/.gitconfig",
	}

	for name, value := range preservedVars {
		t.Setenv(name, value)
	}

	clearedMap := envMap(ClearedEnv())

	for name, expectedValue := range preservedVars {
		value, found := clearedMap[name]
		if !found {
			t.Errorf("ClearedEnv should preserve %s but it was removed", name)
		} else if value != expectedValue {
			t.Errorf("ClearedEnv: %s has value %q but expected %q", name, value, expectedValue)
		}
	}
}

// TestClearedEnv_EmptyEnvironment verifies handling of normal environment.
func TestClearedEnv_EmptyEnvironment(t *testing.T) {
	// This test verifies that ClearedEnv doesn't crash and returns a valid slice.
	// In a normal test environment, os.Environ() will have entries.
	cleared := ClearedEnv()
	if cleared == nil {
		t.Error("ClearedEnv returned nil, expected a slice")
	}
	// We should have at least some environment variables after clearing.
	// The exact count depends on the test environment, so we just verify it's not nil.
}

// TestClearedEnv_MultipleGitVars verifies mix of repo-scoping and non-scoping Git variables.
func TestClearedEnv_MultipleGitVars(t *testing.T) {
	t.Setenv("GIT_DIR", "/hostile/.git")                   // Should be removed (repo-scoping)
	t.Setenv("GIT_SSH_COMMAND", "ssh -i /path/to/key")     // Should be preserved (auth)
	t.Setenv("GIT_CONFIG_GLOBAL", "/home/user/.gitconfig") // Should be preserved (non-scoping)
	t.Setenv("GIT_WORK_TREE", "/hostile")                  // Should be removed (repo-scoping)
	t.Setenv("PATH", "/usr/bin")                           // Should be preserved (system)
	t.Setenv("GIT_OBJECT_DIRECTORY", "/hostile/objects")   // Should be removed (repo-scoping)

	cleared := envMap(ClearedEnv())

	for _, name := range []string{"GIT_DIR", "GIT_WORK_TREE", "GIT_OBJECT_DIRECTORY"} {
		if _, found := cleared[name]; found {
			t.Errorf("%s should be removed", name)
		}
	}
	for _, name := range []string{"GIT_SSH_COMMAND", "GIT_CONFIG_GLOBAL", "PATH"} {
		if _, found := cleared[name]; !found {
			t.Errorf("%s should be preserved", name)
		}
	}
}

// TestClearedEnv_ReturnsCopy verifies that returned slice does not alias os.Environ().
func TestClearedEnv_ReturnsCopy(t *testing.T) {
	cleared1 := ClearedEnv()
	cleared2 := ClearedEnv()

	// Modifying cleared1 should not affect cleared2 (different allocations).
	if len(cleared1) > 0 {
		original := cleared1[0]
		cleared1[0] = "MODIFIED=true"
		if cleared2[0] == "MODIFIED=true" {
			t.Error("ClearedEnv should return a new slice copy, not aliased to os.Environ()")
		}
		cleared1[0] = original // Restore for cleanup.
	}
}
