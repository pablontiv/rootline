package main

import (
	"os"
	"path/filepath"
	"reflect"
	"testing"
)

func TestValidateCorruptGovernanceUsesCommandScopeIgnoreBoundary(t *testing.T) {
	t.Run("all", func(t *testing.T) {
		scanRoot := setupAncestorIgnoredValidateScope(t, "version: 2\nroot: true\nscope:\n  match: [\n")
		mustChdir(t, scanRoot)

		stdout, err := executeValidate(t, "--all", ".", "-o", "json")
		if err != ErrValidationFailed {
			t.Fatalf("all err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
		}
		env := decodeEnvelope(t, stdout)
		assertSkippedResultPaths(t, env, []string{"ignored.md", "keep.md"})
		assertNoticeCodes(t, env, []string{"schema_resolution_failed", "schema_resolution_failed"})
		assertSkippedSchemaResolutionSummary(t, env, 2)
	})

	t.Run("explicit and multi", func(t *testing.T) {
		scanRoot := setupAncestorIgnoredValidateScope(t, "version: 2\nroot: true\nscope:\n  match: [\n")
		mustChdir(t, scanRoot)

		stdout, err := executeValidate(t, "ignored.md", "-o", "json")
		if err != ErrValidationFailed {
			t.Fatalf("explicit err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
		}
		env := decodeEnvelope(t, stdout)
		assertSkippedResultPaths(t, env, []string{"ignored.md"})
		assertNoticeCodes(t, env, []string{"schema_resolution_failed"})
		assertSkippedSchemaResolutionSummary(t, env, 1)

		stdout, err = executeValidate(t, "ignored.md", "keep.md", "-o", "json")
		if err != ErrValidationFailed {
			t.Fatalf("multi err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
		}
		env = decodeEnvelope(t, stdout)
		if got := envelopePaths(t, env); !reflect.DeepEqual(got, []string{"ignored.md", "keep.md"}) {
			t.Fatalf("multi paths = %v", got)
		}
		assertSkippedResultPaths(t, env, []string{"ignored.md", "keep.md"})
		assertNoticeCodes(t, env, []string{"schema_resolution_failed", "schema_resolution_failed"})
		assertSkippedSchemaResolutionSummary(t, env, 2)
	})

	t.Run("staged", func(t *testing.T) {
		scanRoot := setupStagedAncestorIgnoredValidateScope(t, "version: 2\nroot: true\nscope:\n  match: [\n")
		mustChdir(t, scanRoot)

		stdout, err := executeValidate(t, "--staged", "-o", "json")
		if err != ErrValidationFailed {
			t.Fatalf("staged err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
		}
		env := decodeEnvelope(t, stdout)
		assertSkippedResultPaths(t, env, []string{"ignored.md", "keep.md"})
		assertNoticeCodes(t, env, []string{"schema_resolution_failed", "schema_resolution_failed"})
		assertSkippedSchemaResolutionSummary(t, env, 2)
	})
}

func TestExcludedFromGovernanceDoesNotReadOutsideCommandScope(t *testing.T) {
	outer := t.TempDir()
	commandScope := filepath.Join(outer, "scan")
	foreignScope := filepath.Join(outer, "foreign")
	if err := os.MkdirAll(commandScope, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(foreignScope, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(outer, ".stemignore"), []byte("ignored.md\n"), 0o644)
	mustWriteFile(t, filepath.Join(foreignScope, ".stem"), []byte("version: 2\nroot: true\nscope:\n  match: [\n"), 0o644)
	path := filepath.Join(foreignScope, "ignored.md")
	mustWriteFile(t, path, []byte("---\ntitle: Ignored\n---\n# Ignored\n"), 0o644)

	if reason := excludedFromGovernance(path, commandScope); reason != "" {
		t.Fatalf("reason = %q, want no ignore authority outside the command scope", reason)
	}
}

func TestValidateMissingGovernanceDoesNotReadAncestorIgnore(t *testing.T) {
	for _, mode := range []struct {
		name string
		args []string
	}{
		{name: "explicit", args: []string{"ignored.md", "-o", "json"}},
		{name: "multi", args: []string{"ignored.md", "keep.md", "-o", "json"}},
		{name: "staged", args: []string{"--staged", "-o", "json"}},
	} {
		t.Run(mode.name, func(t *testing.T) {
			var scanRoot string
			if mode.name == "staged" {
				scanRoot = setupStagedAncestorIgnoredValidateScope(t, "")
			} else {
				scanRoot = setupAncestorIgnoredValidateScope(t, "")
			}
			mustChdir(t, scanRoot)

			stdout, err := executeValidate(t, mode.args...)
			if err != ErrValidationFailed {
				t.Fatalf("err = %v, want ErrValidationFailed\nstdout=%s", err, stdout)
			}
			env := decodeEnvelope(t, stdout)
			want := []string{"ignored.md"}
			if mode.name != "explicit" {
				want = []string{"ignored.md", "keep.md"}
			}
			assertSkippedResultPaths(t, env, want)
			assertNoticeCodes(t, env, makeSchemaResolutionCodes(len(want)))
			assertSkippedSchemaResolutionSummary(t, env, len(want))
		})
	}
}

func setupAncestorIgnoredValidateScope(t *testing.T, stem string) string {
	t.Helper()
	outer := t.TempDir()
	scanRoot := filepath.Join(outer, "scan")
	if err := os.MkdirAll(scanRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(outer, ".stemignore"), []byte("ignored.md\n"), 0o644)
	if stem != "" {
		mustWriteFile(t, filepath.Join(scanRoot, ".stem"), []byte(stem), 0o644)
	}
	mustWriteFile(t, filepath.Join(scanRoot, "ignored.md"), []byte("---\ntitle: Ignored\n---\n# Ignored\n"), 0o644)
	mustWriteFile(t, filepath.Join(scanRoot, "keep.md"), []byte("---\ntitle: Keep\n---\n# Keep\n"), 0o644)
	return scanRoot
}

func setupStagedAncestorIgnoredValidateScope(t *testing.T, stem string) string {
	t.Helper()
	isolateAmbientGitScope(t)
	outer := t.TempDir()
	scanRoot := filepath.Join(outer, "scan")
	if err := os.MkdirAll(scanRoot, 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(outer, ".stemignore"), []byte("ignored.md\n"), 0o644)
	if stem != "" {
		mustWriteFile(t, filepath.Join(scanRoot, ".stem"), []byte(stem), 0o644)
	}
	mustWriteFile(t, filepath.Join(scanRoot, "ignored.md"), []byte("---\ntitle: Ignored\n---\n# Ignored\n"), 0o644)
	mustWriteFile(t, filepath.Join(scanRoot, "keep.md"), []byte("---\ntitle: Keep\n---\n# Keep\n"), 0o644)
	runFixtureGit(t, scanRoot, "init", "--quiet")
	runFixtureGit(t, scanRoot, "add", "--", "ignored.md", "keep.md")
	return scanRoot
}

func assertSkippedSchemaResolutionSummary(t *testing.T, env map[string]any, count int) {
	t.Helper()
	want := float64(count)
	assertSummaryCounts(t, env, map[string]float64{"total": want, "valid": 0, "invalid": want, "errors_count": want})
}

func makeSchemaResolutionCodes(n int) []string {
	codes := make([]string, n)
	for i := range codes {
		codes[i] = "schema_resolution_failed"
	}
	return codes
}
