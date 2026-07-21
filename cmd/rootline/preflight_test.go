package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// brokenStemTree builds a project whose .stem cannot be parsed.
func brokenStemTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".stem"),
		[]byte("version: 2\nroot: true\nscope:\n  match: [[[unclosed\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"),
		[]byte("---\ntitulo: x\n---\n# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

// TestPreflight_UnparseableStemFailsEveryGovernedCommand covers a gap the
// per-command work cannot close on its own.
//
// `query` only resolves a schema when --sort is used and passes no scope
// resolver, so without this check a corrupt .stem is invisible to it: the
// command returns records from a tree whose governance is broken and says
// nothing. The preflight is the only place that sees every governed command,
// so a hard parse or IO failure has to stop there.
func TestPreflight_UnparseableStemFailsEveryGovernedCommand(t *testing.T) {
	for _, cmd := range []string{"query", "tree", "describe", "graph", "stats"} {
		t.Run(cmd, func(t *testing.T) {
			root := brokenStemTree(t)

			resetFlags()
			out, err := runCmd(t, cmd, root)
			if err == nil {
				t.Fatalf("%s succeeded against an unparseable .stem\noutput: %s", cmd, out)
			}
			if !strings.Contains(err.Error(), "parsing") {
				t.Errorf("expected the parse failure to be reported, got: %v", err)
			}
		})
	}
}

// TestPreflight_ExemptionReachesSubcommands guards the migration path.
//
// The exemption list is keyed by command name, but cobra reports a subcommand's
// own name: `schema propose` answers "propose", not "schema". Checking only the
// leaf name silently un-exempts every subcommand of an exempt parent.
//
// The user this breaks is the one the whole migration exists for: an existing
// project that has a .stem but no root: true marker yet. They cannot reach
// `schema propose` — one of the tools that would help them — because the
// preflight stops it first.
func TestPreflight_ExemptionReachesSubcommands(t *testing.T) {
	markerlessProject := func(t *testing.T) string {
		t.Helper()
		root := t.TempDir()
		if err := os.WriteFile(filepath.Join(root, ".stem"),
			[]byte("version: 2\nscope:\n  match: \"*.md\"\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(filepath.Join(root, "a.md"),
			[]byte("---\ntitulo: x\n---\n# x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
		return root
	}

	t.Run("schema propose", func(t *testing.T) {
		root := markerlessProject(t)
		resetFlags()
		if _, err := runCmd(t, "schema", "propose", root); err != nil {
			t.Fatalf("schema propose must stay exempt as a subcommand of schema, got: %v", err)
		}
	})

	t.Run("hooks status", func(t *testing.T) {
		root := markerlessProject(t)
		mustChdir(t, root)
		resetFlags()
		// hooks reports its own failures; the preflight must not be the one
		// that stops it.
		if _, err := runCmd(t, "hooks", "status"); err != nil &&
			strings.Contains(err.Error(), "declared boundary") {
			t.Fatalf("hooks status was blocked by the boundary preflight: %v", err)
		}
	})
}

// TestPreflight_MissingSchemaIsLeftToTheCommand keeps the two conditions
// distinct. A tree with no .stem at all is not a broken project, and the
// bootstrap commands are allowed to operate on one, so the preflight must pass
// that case through rather than turning it into a hard failure of its own.
func TestPreflight_MissingSchemaIsLeftToTheCommand(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "a.md"),
		[]byte("---\ntitulo: x\n---\n# x\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	resetFlags()
	if _, err := runCmd(t, "schema", "propose", root); err != nil {
		t.Fatalf("schema propose must work without a .stem, got: %v", err)
	}
}
