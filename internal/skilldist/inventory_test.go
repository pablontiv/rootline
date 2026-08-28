package skilldist

import (
	"os"
	"path/filepath"
	"syscall"
	"testing"
)

func TestSupportedDestinationsReturnsClaudeAndAgentsInOrder(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	got := SupportedDestinations(home)
	want := []Destination{
		{ID: DestinationClaude, Path: filepath.Join(home, ".claude", "skills", "rootline")},
		{ID: DestinationAgents, Path: filepath.Join(home, ".agents", "skills", "rootline")},
	}
	if len(got) != len(want) {
		t.Fatalf("len(SupportedDestinations) = %d, want %d: %#v", len(got), len(want), got)
	}
	for i := range want {
		if got[i] != want[i] {
			t.Fatalf("SupportedDestinations()[%d] = %#v, want %#v", i, got[i], want[i])
		}
	}
}

func TestInventoryDestinationsClassifiesOnlyClaudeAndAgents(t *testing.T) {
	home := t.TempDir()
	sourceRoot := t.TempDir()
	mustWriteSkillFile(t, sourceRoot, "SKILL.md", "canonical")
	sourceDigest, err := DigestTree(sourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	source := Source{SkillPath: sourceRoot, Digest: sourceDigest}

	claude := filepath.Join(home, ".claude", "skills", "rootline")
	mustWriteSkillFile(t, claude, "SKILL.md", "copy")
	agents := filepath.Join(home, ".agents", "skills", "rootline")
	if err := os.MkdirAll(filepath.Dir(agents), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(sourceRoot, agents); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	legacy := filepath.Join(home, ".config", "opencode", "skills", "rootline", "sentinel")
	mustWriteSkillFile(t, filepath.Dir(legacy), filepath.Base(legacy), "untouched")

	states, err := InventoryDestinations(home, source)
	if err != nil {
		t.Fatalf("InventoryDestinations: %v", err)
	}
	if len(states) != 2 || states[0].ID != DestinationClaude || states[1].ID != DestinationAgents {
		t.Fatalf("states = %#v", states)
	}
	if states[0].Kind != KindDirectory || states[1].Kind != KindCorrectSymlink {
		t.Fatalf("unexpected classifications: %#v", states)
	}
	data, err := os.ReadFile(legacy)
	if err != nil || string(data) != "untouched" {
		t.Fatalf("legacy sentinel changed: data=%q err=%v", data, err)
	}
}

func TestInventoryDestinationsClassifiesDestinationKinds(t *testing.T) {
	t.Run("absent", func(t *testing.T) {
		home := t.TempDir()
		source := inventoryTestSource(t)
		states, err := InventoryDestinations(home, source)
		if err != nil {
			t.Fatalf("InventoryDestinations: %v", err)
		}
		for _, state := range states {
			if state.Kind != KindAbsent || state.Digest != "" || state.LexicalTarget != "" || state.CanonicalTarget != "" {
				t.Fatalf("state = %#v, want absent without evidence", state)
			}
		}
	})

	t.Run("divergent symlink", func(t *testing.T) {
		home := t.TempDir()
		source := inventoryTestSource(t)
		other := t.TempDir()
		mustWriteSkillFile(t, other, "SKILL.md", "other")
		agents := filepath.Join(home, ".agents", "skills", "rootline")
		if err := os.MkdirAll(filepath.Dir(agents), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(other, agents); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}
		states, err := InventoryDestinations(home, source)
		if err != nil {
			t.Fatalf("InventoryDestinations: %v", err)
		}
		if states[1].Kind != KindDivergentSymlink || states[1].LexicalTarget != other {
			t.Fatalf("agents state = %#v", states[1])
		}
	})

	t.Run("regular file", func(t *testing.T) {
		home := t.TempDir()
		source := inventoryTestSource(t)
		claude := filepath.Join(home, ".claude", "skills", "rootline")
		if err := os.MkdirAll(filepath.Dir(claude), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(claude, []byte("not a directory"), 0o644); err != nil {
			t.Fatal(err)
		}
		states, err := InventoryDestinations(home, source)
		if err != nil {
			t.Fatalf("InventoryDestinations: %v", err)
		}
		if states[0].Kind != KindUnsupported {
			t.Fatalf("claude state = %#v, want unsupported", states[0])
		}
	})

	t.Run("fifo when supported", func(t *testing.T) {
		home := t.TempDir()
		source := inventoryTestSource(t)
		claude := filepath.Join(home, ".claude", "skills", "rootline")
		if err := os.MkdirAll(filepath.Dir(claude), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := syscall.Mkfifo(claude, 0o644); err != nil {
			t.Skipf("fifo unavailable: %v", err)
		}
		states, err := InventoryDestinations(home, source)
		if err != nil {
			t.Fatalf("InventoryDestinations: %v", err)
		}
		if states[0].Kind != KindUnsupported {
			t.Fatalf("claude state = %#v, want unsupported", states[0])
		}
	})
}

func TestInventoryDestinationsRequiresLexicalAndCanonicalSymlinkAgreement(t *testing.T) {
	home := t.TempDir()
	source := inventoryTestSource(t)
	alias := filepath.Join(t.TempDir(), "source-alias")
	if err := os.Symlink(source.SkillPath, alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	agents := filepath.Join(home, ".agents", "skills", "rootline")
	if err := os.MkdirAll(filepath.Dir(agents), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink(alias, agents); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}

	states, err := InventoryDestinations(home, source)
	if err != nil {
		t.Fatalf("InventoryDestinations: %v", err)
	}
	if states[1].Kind != KindDivergentSymlink {
		t.Fatalf("agents state = %#v, want divergent symlink", states[1])
	}
	canonicalSource, err := filepath.EvalSymlinks(source.SkillPath)
	if err != nil {
		t.Fatal(err)
	}
	if states[1].CanonicalTarget != canonicalSource {
		t.Fatalf("CanonicalTarget = %q, want %q", states[1].CanonicalTarget, canonicalSource)
	}
}

func inventoryTestSource(t *testing.T) Source {
	t.Helper()
	sourceRoot := t.TempDir()
	mustWriteSkillFile(t, sourceRoot, "SKILL.md", "canonical")
	digest, err := DigestTree(sourceRoot)
	if err != nil {
		t.Fatal(err)
	}
	return Source{SkillPath: sourceRoot, Digest: digest}
}
