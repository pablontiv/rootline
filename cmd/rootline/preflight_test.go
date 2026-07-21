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
