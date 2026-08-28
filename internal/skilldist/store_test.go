package skilldist

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"
)

func TestStoreAppendsReceiptsWithoutRewritingHistory(t *testing.T) {
	store := NewStore(t.TempDir())
	first := Receipt{Version: 1, Kind: "rootline/skill-receipt", ID: "r1", Operation: OperationInstall, Complete: true}
	second := Receipt{Version: 1, Kind: "rootline/skill-receipt", ID: "r2", Operation: OperationUninstall, Complete: false}
	if err := store.Append(first); err != nil {
		t.Fatal(err)
	}
	before, err := os.ReadFile(store.receiptsPath())
	if err != nil {
		t.Fatal(err)
	}
	if err := store.Append(second); err != nil {
		t.Fatal(err)
	}
	after, err := os.ReadFile(store.receiptsPath())
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.HasPrefix(after, before) {
		t.Fatal("second append rewrote receipt history")
	}
	loaded, err := store.Load("r2")
	if err != nil || loaded.ID != "r2" {
		t.Fatalf("Load r2 = %#v, %v", loaded, err)
	}
}

func TestStoreBackupAndRestoreDirectoryExactly(t *testing.T) {
	stateRoot := t.TempDir()
	original := filepath.Join(t.TempDir(), "rootline")
	mustWriteSkillFile(t, original, "SKILL.md", "preimage")
	digest, err := DigestTree(original)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(stateRoot)
	if err := store.Reserve("r1"); err != nil {
		t.Fatal(err)
	}
	backup, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: original, Kind: KindDirectory, Digest: digest})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(original); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreBackup(backup, original); err != nil {
		t.Fatal(err)
	}
	restored, err := DigestTree(original)
	if err != nil || restored != digest {
		t.Fatalf("restored digest = %q, err=%v, want %q", restored, err, digest)
	}
}

func TestStoreAppendRejectsDuplicateReceiptID(t *testing.T) {
	store := NewStore(t.TempDir())
	receipt := Receipt{Version: 1, Kind: "rootline/skill-receipt", ID: "r1", Operation: OperationInstall}
	if err := store.Append(receipt); err != nil {
		t.Fatal(err)
	}
	if err := store.Append(receipt); err == nil {
		t.Fatal("duplicate Append succeeded")
	}
}

func TestStoreReserveRejectsDuplicateReceiptID(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Reserve("r1"); err != nil {
		t.Fatal(err)
	}
	if err := store.Reserve("r1"); err == nil {
		t.Fatal("duplicate Reserve succeeded")
	}
}

func TestStoreBackupsAreExclusivePerReceiptDestination(t *testing.T) {
	stateRoot := t.TempDir()
	claude := filepath.Join(t.TempDir(), "claude-rootline")
	agents := filepath.Join(t.TempDir(), "agents-rootline")
	mustWriteSkillFile(t, claude, "SKILL.md", "claude")
	mustWriteSkillFile(t, agents, "SKILL.md", "agents")
	claudeDigest, err := DigestTree(claude)
	if err != nil {
		t.Fatal(err)
	}
	agentsDigest, err := DigestTree(agents)
	if err != nil {
		t.Fatal(err)
	}

	store := NewStore(stateRoot)
	if err := store.Reserve("r1"); err != nil {
		t.Fatal(err)
	}
	claudeBackup, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: claude, Kind: KindDirectory, Digest: claudeDigest})
	if err != nil {
		t.Fatal(err)
	}
	agentsBackup, err := store.Backup("r1", DestinationState{ID: DestinationAgents, Path: agents, Kind: KindDirectory, Digest: agentsDigest})
	if err != nil {
		t.Fatal(err)
	}
	if claudeBackup.StoredPath == "" || agentsBackup.StoredPath == "" || claudeBackup.StoredPath == agentsBackup.StoredPath {
		t.Fatalf("destination backups not stored exclusively: claude=%q agents=%q", claudeBackup.StoredPath, agentsBackup.StoredPath)
	}
	if _, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: claude, Kind: KindDirectory, Digest: claudeDigest}); err == nil {
		t.Fatal("duplicate destination backup succeeded")
	}
}

func TestStoreBackupRequiresReservedReceipt(t *testing.T) {
	original := filepath.Join(t.TempDir(), "rootline")
	mustWriteSkillFile(t, original, "SKILL.md", "preimage")
	digest, err := DigestTree(original)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(t.TempDir())
	if _, err := store.Backup("missing", DestinationState{ID: DestinationClaude, Path: original, Kind: KindDirectory, Digest: digest}); err == nil {
		t.Fatal("Backup succeeded without Reserve")
	}
}

func TestStoreBackupAndRestorePreservesPermissionBits(t *testing.T) {
	original := filepath.Join(t.TempDir(), "rootline")
	nested := filepath.Join(original, "nested")
	secret := filepath.Join(nested, "secret.md")
	mustWriteSkillFile(t, original, "nested/secret.md", "secret")
	t.Cleanup(func() {
		chmodDirectoryForCleanup(nested)
		chmodDirectoryForCleanup(original)
	})
	// #nosec G302 -- restrictive directory mode is the behavior under test in a temp fixture.
	if err := os.Chmod(nested, 0o500); err != nil {
		t.Fatal(err)
	}
	if err := os.Chmod(secret, 0o600); err != nil {
		t.Fatal(err)
	}
	digest, err := DigestTree(original)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(t.TempDir())
	if err := store.Reserve("r1"); err != nil {
		t.Fatal(err)
	}
	backup, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: original, Kind: KindDirectory, Digest: digest})
	if err != nil {
		t.Fatal(err)
	}
	t.Cleanup(func() {
		chmodDirectoryForCleanup(filepath.Join(backup.StoredPath, "nested"))
		chmodDirectoryForCleanup(backup.StoredPath)
	})
	mustChmodDirectoryForCleanup(t, nested)
	if err := os.RemoveAll(original); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreBackup(backup, original); err != nil {
		t.Fatal(err)
	}
	dirInfo, err := os.Lstat(nested)
	if err != nil {
		t.Fatal(err)
	}
	fileInfo, err := os.Lstat(secret)
	if err != nil {
		t.Fatal(err)
	}
	if dirInfo.Mode().Perm() != 0o500 || fileInfo.Mode().Perm() != 0o600 {
		t.Fatalf("restored modes: dir=%#o file=%#o, want dir=0500 file=0600", dirInfo.Mode().Perm(), fileInfo.Mode().Perm())
	}
}

func TestStorePreservesInternalSymlinksLexically(t *testing.T) {
	original := filepath.Join(t.TempDir(), "rootline")
	mustWriteSkillFile(t, original, "docs", "target")
	link := filepath.Join(original, "nested", "link.md")
	if err := os.MkdirAll(filepath.Dir(link), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("../docs", link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	digest, err := DigestTree(original)
	if err != nil {
		t.Fatal(err)
	}
	store := NewStore(t.TempDir())
	if err := store.Reserve("r1"); err != nil {
		t.Fatal(err)
	}
	backup, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: original, Kind: KindDirectory, Digest: digest})
	if err != nil {
		t.Fatal(err)
	}
	if err := os.RemoveAll(original); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreBackup(backup, original); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(filepath.Join(original, "nested", "link.md"))
	if err != nil {
		t.Fatal(err)
	}
	if target != "../docs" {
		t.Fatalf("restored symlink target = %q, want %q", target, "../docs")
	}
}

func TestStoreBackupAndRestoreDestinationSymlinkExactly(t *testing.T) {
	parent := t.TempDir()
	outside := filepath.Join(parent, "target")
	mustWriteSkillFile(t, outside, "SKILL.md", "external")
	link := filepath.Join(parent, "rootline")
	if err := os.Symlink("target", link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	store := NewStore(t.TempDir())
	if err := store.Reserve("r1"); err != nil {
		t.Fatal(err)
	}
	backup, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: link, Kind: KindDivergentSymlink, LexicalTarget: "target"})
	if err != nil {
		t.Fatal(err)
	}
	if backup.LinkTarget != "target" {
		t.Fatalf("backup LinkTarget = %q, want target", backup.LinkTarget)
	}
	if backup.StoredPath != "" {
		t.Fatalf("symlink backup stored active tree at %q", backup.StoredPath)
	}
	if err := os.Remove(link); err != nil {
		t.Fatal(err)
	}
	if err := store.RestoreBackup(backup, link); err != nil {
		t.Fatal(err)
	}
	target, err := os.Readlink(link)
	if err != nil {
		t.Fatal(err)
	}
	if target != "target" {
		t.Fatalf("restored destination symlink = %q, want target", target)
	}
}

func TestStoreReturnsNoBackupForAbsentDestination(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Reserve("r1"); err != nil {
		t.Fatal(err)
	}
	backup, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: filepath.Join(t.TempDir(), "missing"), Kind: KindAbsent})
	if err != nil {
		t.Fatal(err)
	}
	if backup.StoredPath != "" || backup.LinkTarget != "" || backup.Kind != KindAbsent {
		t.Fatalf("absent backup = %#v, want metadata-only absent backup", backup)
	}
}

func TestStoreRejectsMalformedJSONL(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(store.receiptsPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(store.receiptsPath(), []byte("{bad json}\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Load("r1"); err == nil {
		t.Fatal("Load accepted malformed JSONL")
	}
	if _, _, err := store.Latest(); err == nil {
		t.Fatal("Latest accepted malformed JSONL")
	}
}

func TestStoreLatestEmptyDoesNotCreateStateDirectory(t *testing.T) {
	stateRoot := filepath.Join(t.TempDir(), "missing-state")
	store := NewStore(stateRoot)
	latest, ok, err := store.Latest()
	if err != nil {
		t.Fatal(err)
	}
	if ok || latest.ID != "" {
		t.Fatalf("Latest = %#v, %v, want empty false", latest, ok)
	}
	if _, err := os.Stat(filepath.Join(stateRoot, "rootline", "skill")); !os.IsNotExist(err) {
		t.Fatalf("Latest created state directory or unexpected stat error: %v", err)
	}
}

func mustChmodDirectoryForCleanup(t *testing.T, path string) {
	t.Helper()
	// #nosec G302 -- directory cleanup requires the owner execute bit; path is a temp test fixture.
	if err := os.Chmod(path, 0o700); err != nil {
		t.Fatal(err)
	}
}

func chmodDirectoryForCleanup(path string) {
	// #nosec G302 -- directory cleanup requires the owner execute bit; path is a temp test fixture.
	_ = os.Chmod(path, 0o700)
}
