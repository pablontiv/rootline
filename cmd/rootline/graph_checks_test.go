package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func writeGraphCheckFixtureFile(t *testing.T, root, rel, content string) {
	t.Helper()
	p := filepath.Join(root, rel)
	if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// graph --check used to report only cycles and broken links, so a green run
// proved nothing about anchors or encoding even when the schema asked for
// those checks — validate was the only command that could see them
// (issue #62 sub-defect 4).
func TestGraphCheckReportsAnchorAndEncoding(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".stem", "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nlinks:\n  styles: [markdown]\n  checks:\n    resolve: true\n    anchors: true\n    encoding: true\n")
	write("target.md", "---\ntitulo: T\n---\n\n# Heading One\n")
	write("has space.md", "---\ntitulo: S\n---\n\n# S\n")
	write("bad-anchor.md", "---\ntitulo: A\n---\n\nSee [x](target.md#no-such-heading).\n")
	write("bad-encoding.md", "---\ntitulo: E\n---\n\nSee [x](has space.md).\n")

	out, err := runCmd(t, "graph", dir, "--check")
	if err == nil {
		t.Errorf("expected a non-zero result when anchor/encoding checks fail, got nil")
	}
	if !strings.Contains(out, "no-such-heading") {
		t.Errorf("anchor violation not reported:\n%s", out)
	}
	if !strings.Contains(out, "has space.md") || !strings.Contains(out, "encoding") {
		t.Errorf("encoding violation not reported:\n%s", out)
	}
}

// A repository whose schema declares no checks keeps a clean report: the
// checks are opt-in and graph must not invent them.
func TestGraphCheckWithoutChecksStaysQuiet(t *testing.T) {
	dir := t.TempDir()
	write := func(rel, content string) {
		t.Helper()
		p := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	write(".stem", "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nlinks:\n  styles: [markdown]\n")
	write("target.md", "---\ntitulo: T\n---\n\n# Heading One\n")
	write("a.md", "---\ntitulo: A\n---\n\nSee [x](target.md#no-such-heading).\n")

	out, err := runCmd(t, "graph", dir, "--check")
	if err != nil {
		t.Errorf("expected clean run without declared checks, got %v:\n%s", err, out)
	}
}

func TestGraphCheck_BasenameFallbackUniqueLinkDoesNotReportUnverifiable(t *testing.T) {
	dir := t.TempDir()
	writeGraphCheckFixtureFile(t, dir, ".stem", "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nlinks:\n  styles: [wikilink]\n  basename_fallback: true\n  checks:\n    resolve: true\n")
	writeGraphCheckFixtureFile(t, dir, "source.md", "---\ntitle: Source\n---\n\n# Source\n\n[[target]]\n")
	writeGraphCheckFixtureFile(t, dir, "docs/target.md", "---\ntitle: Target\n---\n\n# Target\n")

	out, err := runCmd(t, "graph", "--check", dir)
	if err != nil {
		t.Fatalf("expected clean check for uniquely resolvable basename target, got: %v\noutput: %s", err, out)
	}
	if strings.Contains(out, "Link check failures:") {
		t.Fatalf("unexpected duplicate link-check failures for basename fallback success:\n%s", out)
	}
	if strings.Contains(out, "link_unverifiable") {
		t.Fatalf("unexpected link_unverifiable for basename fallback success:\n%s", out)
	}
	if !strings.Contains(out, "No cycles or broken links found.") {
		t.Fatalf("expected clean summary, got:\n%s", out)
	}
}

func TestGraphCheck_BasenameFallbackBrokenLinksAreNotDuplicated(t *testing.T) {
	cases := []struct {
		name  string
		files map[string]string
	}{
		{
			name: "missing target",
			files: map[string]string{
				"source.md": "---\ntitle: Source\n---\n\n# Source\n\n[[missing-target]]\n",
			},
		},
		{
			name: "ambiguous basename",
			files: map[string]string{
				"source.md":        "---\ntitle: Source\n---\n\n# Source\n\n[[target]]\n",
				"first/target.md":  "---\ntitle: First\n---\n\n# First\n",
				"second/target.md": "---\ntitle: Second\n---\n\n# Second\n",
			},
		},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := t.TempDir()
			writeGraphCheckFixtureFile(t, dir, ".stem", "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nlinks:\n  styles: [wikilink]\n  basename_fallback: true\n  checks:\n    resolve: true\n")
			for rel, content := range tc.files {
				writeGraphCheckFixtureFile(t, dir, rel, content)
			}

			out, err := runCmd(t, "graph", "--check", dir)
			if err != ErrValidationFailed {
				t.Fatalf("expected ErrValidationFailed for unresolved target, got %v\noutput: %s", err, out)
			}
			if !strings.Contains(out, "Broken links: 1") {
				t.Fatalf("expected exactly one broken link report, got:\n%s", out)
			}
			if strings.Contains(out, "Link check failures:") {
				t.Fatalf("broken link must not be duplicated as a link-check failure:\n%s", out)
			}
			if strings.Contains(out, "link_unverifiable") {
				t.Fatalf("broken link must not be duplicated as link_unverifiable:\n%s", out)
			}
		})
	}
}
