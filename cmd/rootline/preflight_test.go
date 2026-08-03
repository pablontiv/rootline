package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// unmarkedStemTree builds a project that has a .stem but never declares where
// the project starts — the exact shape the boundary preflight rejects.
func unmarkedStemTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	wiki := filepath.Join(root, "wiki")
	if err := os.MkdirAll(wiki, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wiki, ".stem"),
		[]byte("version: 2\nscope:\n  match: \"*.md\"\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(wiki, "a.md"),
		[]byte("---\nestado: Pending\n---\n# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

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

// TestPreflight_AnalyzeExemptionWorks covers the second exemption category.
//
// `analyze` infers a schema from the documents themselves and resolves no
// .stem at all, so gating it on a declared boundary blocks the one command
// that could tell an ungoverned tree what its schema should be. It is the
// entry point of the `analyze --incremental` -> `schema apply` loop, and
// `schema propose` — its sibling on the same loop — is already exempt.
func TestPreflight_AnalyzeExemptionWorks(t *testing.T) {
	root := unmarkedStemTree(t)

	out, err := runCmd(t, "analyze", filepath.Join(root, "wiki"))
	if err != nil {
		t.Fatalf("analyze must stay exempt on an unmarked .stem tree, got: %v", err)
	}

	var report map[string]any
	if jsonErr := json.Unmarshal([]byte(out), &report); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", jsonErr, out)
	}
	if report["kind"] != "rootline/analyze" {
		t.Errorf("kind = %v, want rootline/analyze", report["kind"])
	}
}

// TestPreflight_QueryStatsNotExempt pins the boundary of the exemption.
//
// `query` and `stats` skip schema resolution voluntarily, not because they are
// schema-independent: an unmarked chain may have collected .stem files from
// outside the project, and both would happily report records governed by them.
// Widening the exemption to cover them is the accident this test prevents.
func TestPreflight_QueryStatsNotExempt(t *testing.T) {
	for _, cmd := range []string{"query", "stats"} {
		t.Run(cmd, func(t *testing.T) {
			root := unmarkedStemTree(t)

			out, err := runCmd(t, cmd, filepath.Join(root, "wiki"))
			if err == nil {
				t.Fatalf("%s must stay governed by the preflight\noutput: %s", cmd, out)
			}
			if !strings.Contains(err.Error(), "declared boundary") {
				t.Errorf("expected the boundary error, got: %v", err)
			}
		})
	}
}
