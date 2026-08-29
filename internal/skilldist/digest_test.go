package skilldist

import (
	"os"
	"path/filepath"
	"testing"
)

func TestDigestTreeIsDeterministicAndContentSensitive(t *testing.T) {
	root := t.TempDir()
	mustWriteSkillFile(t, root, "z.md", "z")
	mustWriteSkillFile(t, root, "nested/a.md", "a")

	first, err := DigestTree(root)
	if err != nil {
		t.Fatalf("DigestTree: %v", err)
	}
	second, err := DigestTree(root)
	if err != nil {
		t.Fatalf("DigestTree second call: %v", err)
	}
	if first != second {
		t.Fatalf("digest changed without content change: %q != %q", first, second)
	}

	if err := os.WriteFile(filepath.Join(root, "nested", "a.md"), []byte("changed"), 0o644); err != nil {
		t.Fatal(err)
	}
	changed, err := DigestTree(root)
	if err != nil {
		t.Fatalf("DigestTree changed tree: %v", err)
	}
	if changed == first {
		t.Fatal("content change did not change digest")
	}
}

func TestDigestTreeHashesSymlinkTargetWithoutFollowingIt(t *testing.T) {
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside.md")
	if err := os.WriteFile(outside, []byte("outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "linked.md")
	if err := os.Symlink(outside, link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	first, err := DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(outside, []byte("changed outside"), 0o644); err != nil {
		t.Fatal(err)
	}
	second, err := DigestTree(root)
	if err != nil {
		t.Fatal(err)
	}
	if first != second {
		t.Fatal("DigestTree followed a symlink instead of hashing its lexical target")
	}
}
