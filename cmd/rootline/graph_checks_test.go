package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

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
