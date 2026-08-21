package main

import (
	"os"
	"path/filepath"
	"testing"
)

func TestNewRejectsSymlinkedParentEscapeWithoutWritingOutside(t *testing.T) {
	root := newSectionSourceProject(t, `schema:
  status:
    type: string
    required: true
    default: Open
`)
	outside := t.TempDir()
	alias := filepath.Join(root, "escape")
	if err := os.Symlink(outside, alias); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	outsideTarget := filepath.Join(outside, "T001-task.md")
	aliasTarget := filepath.Join(alias, "T001-task.md")
	out, err := runCmd(t, "new", aliasTarget)
	if err == nil {
		t.Fatalf("expected symlinked parent escape to fail, output: %s", out)
	}
	if _, statErr := os.Stat(outsideTarget); !os.IsNotExist(statErr) {
		t.Fatalf("failed escape must not write outside root, stat err=%v", statErr)
	}
}
