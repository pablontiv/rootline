package main

import (
	"path/filepath"
	"slices"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

// A misspelled --where field is the highest-frequency user error on this CLI,
// and it used to be indistinguishable from "no records match": zero results,
// empty stderr, exit 0. Every command that accepts --where must say so.
func TestWhere_UnknownFieldWarnsOnEveryCommand(t *testing.T) {
	docs := setupFormatProject(t)

	cases := []struct {
		name string
		args []string
	}{
		{"query", []string{"query", docs, "--count", "--where", "estadoo == 'Pending'"}},
		{"stats", []string{"stats", docs, "--where", "estadoo == 'Pending'"}},
		{"tree", []string{"tree", docs, "--where", "estadoo == 'Pending'"}},
		{"validate", []string{"validate", "--all", docs, "--where", "estadoo == 'Pending'"}},
		{"graph", []string{"graph", docs, "--check", "--where", "estadoo == 'Pending'"}},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			out, _ := runCmd(t, tc.args...)
			if !strings.Contains(out, `unknown field "estadoo"`) {
				t.Errorf("expected an unknown-field warning, got: %s", out)
			}
			if !strings.Contains(out, `did you mean "estado"`) {
				t.Errorf("expected a fuzzy suggestion, got: %s", out)
			}
		})
	}
}

// The warning must not become an error: a field absent from every record is a
// legal filter that yields zero matches, and pipelines depend on that.
func TestWhere_UnknownFieldDoesNotChangeExitCode(t *testing.T) {
	docs := setupFormatProject(t)

	if _, err := runCmd(t, "query", docs, "--count", "--where", "estadoo == 'Pending'"); err != nil {
		t.Fatalf("unknown --where field must warn, not fail: %v", err)
	}
}

// No false positives: a schema field, a derived/builtin field, and a
// frontmatter key that no .stem declares must all pass silently.
func TestWhere_KnownFieldsStaySilent(t *testing.T) {
	docs := setupFormatProject(t)

	for _, where := range []string{
		"estado == 'Pending'",
		"path != ''",
		"body != ''",
	} {
		out, err := runCmd(t, "query", docs, "--count", "--where", where)
		if err != nil {
			t.Fatalf("--where %q: unexpected error: %v", where, err)
		}
		if strings.Contains(out, "unknown field") {
			t.Errorf("--where %q: unexpected warning: %s", where, out)
		}
	}
}

// A bad sort *direction* already exits 1. A bad sort *field* silently produced
// scan order, so the command validated half its input.
func TestSort_UnknownFieldErrors(t *testing.T) {
	docs := setupFormatProject(t)

	_, err := runCmd(t, "query", docs, "--select", "path,estado", "--sort", "nosuch:asc")
	if err == nil {
		t.Fatal("expected an error for an unknown sort field, got none")
	}
	if !strings.Contains(err.Error(), "nosuch") {
		t.Errorf("error = %v, want it to name the offending field", err)
	}
}

func TestSort_UnknownFieldSuggests(t *testing.T) {
	docs := setupFormatProject(t)

	_, err := runCmd(t, "query", docs, "--select", "path,estado", "--sort", "estadoo:asc")
	if err == nil {
		t.Fatal("expected an error, got none")
	}
	if !strings.Contains(err.Error(), `did you mean "estado"`) {
		t.Errorf("error = %v, want a fuzzy suggestion", err)
	}
}

// Sorting by a schema field, a builtin, and a field only some records carry
// must keep working.
func TestSort_KnownFieldsAccepted(t *testing.T) {
	docs := setupFormatProject(t)

	for _, key := range []string{"estado:asc", "path:desc", "estado:asc,path:asc"} {
		if _, err := runCmd(t, "query", docs, "--select", "path,estado", "--sort", key); err != nil {
			t.Errorf("--sort %q: unexpected error: %v", key, err)
		}
	}
}

// An empty result set must not make every field look unknown: the legal field
// names come from the scanned corpus, before --where narrows it.
func TestSort_ValidatesAgainstUnfilteredCorpus(t *testing.T) {
	docs := setupFormatProject(t)

	_, err := runCmd(t, "query", docs, "--select", "path,estado",
		"--where", "estado == 'NeverMatches'", "--sort", "estado:asc")
	if err != nil {
		t.Fatalf("a valid sort field must survive a filter that matches nothing: %v", err)
	}
}

// A corpus with no records and no schema has nothing to validate against;
// inventing verdicts there would be noise, not diagnosis.
func TestFieldCheck_EmptyCorpusIsSilent(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\n"), 0o644)

	out, err := runCmd(t, "query", root, "--count", "--where", "whatever == 'x'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "unknown field") {
		t.Errorf("empty corpus should not warn, got: %s", out)
	}
}

func TestKnownWhereFields_UnionsRecordsSchemaAndBuiltins(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte(
		"version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [Pending]\n"+
			"derive:\n  slug: slugify(path)\naggregate:\n  total: count(children)\n"), 0o644)

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"owner": "pablo"}},
		{Path: "b.md", Derived: map[string]any{"computed": 1}},
	}

	got := knownWhereFields(records, root)

	for _, want := range []string{"path", "body", "type", "sections", "estado", "slug", "total", "owner", "computed"} {
		if !slices.Contains(got, want) {
			t.Errorf("knownWhereFields is missing %q; got %v", want, got)
		}
	}
	if !slices.IsSorted(got) {
		t.Errorf("knownWhereFields must be deterministic; got %v", got)
	}
}

func TestKnownWhereFields_NilWhenNothingIsDeclaredOrObserved(t *testing.T) {
	root := t.TempDir()
	mustWriteFile(t, filepath.Join(root, ".stem"), []byte("version: 2\nroot: true\n"), 0o644)

	if got := knownWhereFields(nil, root); got != nil {
		t.Errorf("expected nil for a corpus with no records and no schema, got %v", got)
	}
}
