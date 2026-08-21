package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewForceHardLinkAliasValidatesViaSelectedRealPath(t *testing.T) {
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
	createHardLinkOrSkip(t, realTarget, aliasTarget)

	if out, err := runCmd(t, "new", aliasTarget, "--force"); err != nil {
		t.Fatalf("force via hard-link alias should accept anchor introduced only by prospective bytes: %v\noutput: %s", err, out)
	}
	assertSameFile(t, realTarget, aliasTarget)
	gotReal := string(mustReadFile(t, realTarget))
	gotAlias := string(mustReadFile(t, aliasTarget))
	if gotReal != gotAlias || !strings.Contains(gotReal, "## New Anchor") || strings.Contains(gotReal, "## Old Anchor") {
		t.Fatalf("force through hard-link alias should replace shared inode bytes\nreal:\n%s\nalias:\n%s", gotReal, gotAlias)
	}
}

func TestNewForceHardLinkAliasRejectsRemovedAnchorAndPreservesNames(t *testing.T) {
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
	createHardLinkOrSkip(t, realTarget, aliasTarget)

	out, err := runCmd(t, "new", aliasTarget, "--force")
	if err == nil {
		t.Fatalf("force via hard-link alias should reject anchor absent from prospective bytes, output: %s", out)
	}
	assertSameFile(t, realTarget, aliasTarget)
	if gotReal, gotAlias := string(mustReadFile(t, realTarget)), string(mustReadFile(t, aliasTarget)); gotReal != old || gotAlias != old {
		t.Fatalf("failed force through hard-link alias must preserve both names\nreal: %q\nalias:%q\nwant: %q", gotReal, gotAlias, old)
	}
}

func createHardLinkOrSkip(t *testing.T, oldname, newname string) {
	t.Helper()
	if err := os.Link(oldname, newname); err != nil {
		t.Skipf("hard links unavailable: %v", err)
	}
	assertSameFile(t, oldname, newname)
}

func assertSameFile(t *testing.T, first, second string) {
	t.Helper()
	firstInfo, err := os.Stat(first)
	if err != nil {
		t.Fatalf("stat %s: %v", first, err)
	}
	secondInfo, err := os.Stat(second)
	if err != nil {
		t.Fatalf("stat %s: %v", second, err)
	}
	if !os.SameFile(firstInfo, secondInfo) {
		t.Fatalf("%s and %s are not the same file", first, second)
	}
}
