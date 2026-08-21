package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewForceFinalFileAliasValidatesViaSelectedRealPath(t *testing.T) {
	dir := newSectionSourceProject(t, `links:
  checks:
    anchors: true
schema:
  anchor:
    type: string
    required: true
    source: 'body.section["## New Anchor"]'
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    default: '[[real#new-anchor]]'
`)
	realTarget := filepath.Join(dir, "real.md")
	aliasTarget := filepath.Join(dir, "alias.md")
	old := "---\n---\n# Real\n\n## Old Anchor\n"
	mustWriteFile(t, realTarget, []byte(old), 0o644)
	if err := os.Symlink(realTarget, aliasTarget); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	if out, err := runCmd(t, "new", aliasTarget, "--force"); err != nil {
		t.Fatalf("force via alias should accept anchor introduced only by prospective bytes selected as real path: %v\noutput: %s", err, out)
	}
	got := string(mustReadFile(t, realTarget))
	if !strings.Contains(got, "## New Anchor") || strings.Contains(got, "## Old Anchor") {
		t.Fatalf("force through alias should replace real target with prospective bytes, got:\n%s", got)
	}
}

func TestNewForceFinalFileAliasRejectsRemovedAnchorViaSelectedRealPath(t *testing.T) {
	dir := newSectionSourceProject(t, `links:
  checks:
    anchors: true
schema:
  notes:
    type: string
    required: true
    source: 'body.section["## Notes"]'
    default: '[[real#old-anchor]]'
  replacement:
    type: string
    required: true
    source: 'body.section["## Replacement"]'
`)
	realTarget := filepath.Join(dir, "real.md")
	aliasTarget := filepath.Join(dir, "alias.md")
	old := "---\n---\n# Real\n\n## Old Anchor\n"
	mustWriteFile(t, realTarget, []byte(old), 0o644)
	if err := os.Symlink(realTarget, aliasTarget); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	out, err := runCmd(t, "new", aliasTarget, "--force")
	if err == nil {
		t.Fatalf("force via alias should reject anchor absent from prospective bytes, output: %s", out)
	}
	if got := string(mustReadFile(t, realTarget)); got != old {
		t.Fatalf("failed force through alias must leave real target unchanged\ngot: %q\nwant:%q", got, old)
	}
}

func TestNewForceRejectsCaseFoldExistingFinalNameCollision(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  status:
    type: string
    required: true
    default: Open
`)
	existing := filepath.Join(dir, "Target.md")
	requested := filepath.Join(dir, "target.md")
	old := "# Existing\n"
	mustWriteFile(t, existing, []byte(old), 0o644)
	if _, err := os.Stat(requested); os.IsNotExist(err) {
		t.Skip("filesystem is case-sensitive; collision path does not resolve")
	} else if err != nil {
		t.Skipf("cannot probe case behavior: %v", err)
	}

	out, err := runCmd(t, "new", requested, "--force")
	if err == nil {
		t.Fatalf("forced differently-cased existing final name should fail before write, output: %s", out)
	}
	if got := string(mustReadFile(t, existing)); got != old {
		t.Fatalf("case-collision rejection must leave existing bytes unchanged\ngot: %q\nwant:%q", got, old)
	}
}

func TestNewForceAllowsDistinctCaseFinalNamesOnCaseSensitiveFilesystem(t *testing.T) {
	dir := newSectionSourceProject(t, `schema:
  status:
    type: string
    required: true
    default: Open
`)
	existing := filepath.Join(dir, "Target.md")
	requested := filepath.Join(dir, "target.md")
	old := "# Existing\n"
	mustWriteFile(t, existing, []byte(old), 0o644)
	if _, err := os.Stat(requested); err == nil {
		t.Skip("filesystem is case-insensitive; distinct-case creation cannot be represented")
	} else if !os.IsNotExist(err) {
		t.Skipf("cannot probe case behavior: %v", err)
	}

	if out, err := runCmd(t, "new", requested, "--force"); err != nil {
		t.Fatalf("case-sensitive filesystem should allow distinct final names: %v\noutput: %s", err, out)
	}
	if got := string(mustReadFile(t, existing)); got != old {
		t.Fatalf("creating distinct case path must not alter existing file\ngot: %q\nwant:%q", got, old)
	}
	if _, err := os.Stat(requested); err != nil {
		t.Fatalf("requested distinct-case target should be created: %v", err)
	}
}
