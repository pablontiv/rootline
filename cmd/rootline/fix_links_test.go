package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func setupFixLinkFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	files := map[string]string{
		".stem":    "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  titulo:\n    type: string\nlinks:\n  styles: [markdown]\n",
		"guide.md": "---\ntitulo: G\n---\n\n# Guide\n",
		"a.md":     "---\ntitulo: A\n---\n\nSee [g](guied.md).\n",
	}
	for rel, content := range files {
		p := filepath.Join(dir, rel)
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return dir
}

// fix --all reported a clean run on a record validate fails with link_resolve,
// even though that error already carries a fuzzy suggestion. A clean fix
// followed by a failing validate reads as a tool bug (issue #62 sub-defect 12).
func TestFixAllSurfacesLinkFindings(t *testing.T) {
	dir := setupFixLinkFixture(t)
	out, err := runCmd(t, "fix", "--all", dir)
	if err != nil {
		t.Fatalf("fix --all failed: %v\n%s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\n%s", err, out)
	}
	findings, ok := result["link_findings"].([]any)
	if !ok || len(findings) == 0 {
		t.Fatalf("expected link_findings to be reported, got: %s", out)
	}
	if !strings.Contains(out, "guied.md") {
		t.Errorf("finding should name the unresolved target:\n%s", out)
	}
	if !strings.Contains(out, "guide.md") {
		t.Errorf("finding should carry the fuzzy suggestion validate already computed:\n%s", out)
	}
}

// fix must not rewrite link bodies on a fuzzy guess: that is a destructive
// edit outside its data-repair contract. The finding is reported, the document
// is left alone.
func TestFixAllDoesNotRewriteLinkBodies(t *testing.T) {
	dir := setupFixLinkFixture(t)
	before, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if _, err := runCmd(t, "fix", "--all", dir); err != nil {
		t.Fatalf("fix --all failed: %v", err)
	}
	after, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Errorf("fix rewrote a link body:\nbefore: %q\nafter:  %q", before, after)
	}
}

// A repository with no link problems reports no findings.
func TestFixAllNoLinkFindingsWhenClean(t *testing.T) {
	dir := t.TempDir()
	for rel, content := range map[string]string{
		".stem":    "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  titulo:\n    type: string\nlinks:\n  styles: [markdown]\n",
		"guide.md": "---\ntitulo: G\n---\n\n# Guide\n",
		"a.md":     "---\ntitulo: A\n---\n\nSee [g](guide.md).\n",
	} {
		if err := os.WriteFile(filepath.Join(dir, rel), []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	out, err := runCmd(t, "fix", "--all", dir)
	if err != nil {
		t.Fatalf("fix --all failed: %v\n%s", err, out)
	}
	if strings.Contains(out, "link_findings") {
		t.Errorf("clean repository must report no link findings:\n%s", out)
	}
}

// Dry-run is the preview of what fix will do, so it must report the same link
// findings the applied run reports. Collecting them and then dropping them on
// the dry-run branch would hide the very defect the field exists to expose.
func TestFixAllDryRunSurfacesLinkFindings(t *testing.T) {
	dir := setupFixLinkFixture(t)
	out, err := runCmd(t, "fix", "--all", dir, "--dry-run")
	if err != nil {
		t.Fatalf("fix --all --dry-run failed: %v\n%s", err, out)
	}
	if !strings.Contains(out, "link_findings") || !strings.Contains(out, "guied.md") {
		t.Errorf("dry-run dropped link findings:\n%s", out)
	}
}
