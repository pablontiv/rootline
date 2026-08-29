package skilldist

import (
	"bytes"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
)

func TestStoreAppendsReceiptsWithoutRewritingHistory(t *testing.T) {
	store := NewStore(t.TempDir())
	first := semanticallyValidInstallReceipt()
	second := semanticallyValidInstallReceipt()
	second.ID = "r2"
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

func TestStoreRestorePublicationConflictPreservesExternalDestinationAndBackup(t *testing.T) {
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
	store.publishCandidate = func(candidate, destination string) error {
		mustWriteSkillFile(t, destination, "SKILL.md", "external")
		return atomicPublishNoReplace(candidate, destination)
	}

	err = store.RestoreBackup(backup, original)
	assertOperationErrorCode(t, err, ErrRestoreConflict)
	data, readErr := os.ReadFile(filepath.Join(original, "SKILL.md"))
	if readErr != nil || string(data) != "external" {
		t.Fatalf("external destination data = %q, err=%v", data, readErr)
	}
	if err := store.VerifyBackup(backup); err != nil {
		t.Fatalf("stored backup was not preserved: %v", err)
	}
	assertNoRootlineStageSiblings(t, original)
}

func TestStoreAppendRejectsDuplicateReceiptID(t *testing.T) {
	store := NewStore(t.TempDir())
	receipt := semanticallyValidInstallReceipt()
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

func TestStoreDuplicateDirectoryBackupPreservesExistingBackup(t *testing.T) {
	stateRoot := t.TempDir()
	original := filepath.Join(t.TempDir(), "rootline")
	mustWriteSkillFile(t, original, "SKILL.md", "preimage")
	mustWriteSkillFile(t, original, "nested/original.md", "original")
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
	if err := store.VerifyBackup(backup); err != nil {
		t.Fatal(err)
	}

	if _, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: original, Kind: KindDirectory, Digest: digest}); err == nil {
		t.Fatal("duplicate destination backup succeeded")
	}
	if err := store.VerifyBackup(backup); err != nil {
		t.Fatalf("first backup was not preserved after duplicate backup attempt: %v", err)
	}
	preservedDigest, err := DigestTree(backup.StoredPath)
	if err != nil {
		t.Fatal(err)
	}
	if preservedDigest != digest {
		t.Fatalf("preserved backup digest = %q, want %q", preservedDigest, digest)
	}
	content, err := os.ReadFile(filepath.Join(backup.StoredPath, "nested", "original.md"))
	if err != nil {
		t.Fatal(err)
	}
	if string(content) != "original" {
		t.Fatalf("preserved backup content = %q, want original", content)
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

func TestStoreRejectsSemanticallyMalformedReceipts(t *testing.T) {
	valid := semanticallyValidInstallReceipt()
	tests := []struct {
		name   string
		mutate func(*Receipt)
		want   string
	}{
		{name: "wrong version", mutate: func(r *Receipt) { r.Version = 2 }, want: "version"},
		{name: "wrong kind", mutate: func(r *Receipt) { r.Kind = "rootline/other" }, want: "kind"},
		{name: "unknown operation", mutate: func(r *Receipt) { r.Operation = Operation("repair") }, want: "operation"},
		{name: "duplicate destination", mutate: func(r *Receipt) {
			r.Actions[1].Destination = r.Actions[0].Destination
			r.Actions[1].Before.ID = r.Actions[0].Before.ID
			r.Actions[1].After.ID = r.Actions[0].After.ID
		}, want: "duplicate destination"},
		{name: "unsupported destination", mutate: func(r *Receipt) {
			r.Actions[0].Destination = DestinationID("opencode")
			r.Actions[0].Before.ID = DestinationID("opencode")
			r.Actions[0].After.ID = DestinationID("opencode")
		}, want: "unsupported destination"},
		{name: "missing backup for replaced preimage", mutate: func(r *Receipt) { r.Backups = nil }, want: "missing backup"},
		{name: "install create has non-absent before", mutate: func(r *Receipt) { r.Actions[1].Before.Kind = KindDirectory }, want: "create_symlink"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			store := NewStore(t.TempDir())
			receipt := valid
			receipt.Actions = append([]ActionResult(nil), valid.Actions...)
			receipt.Backups = append([]Backup(nil), valid.Backups...)
			tt.mutate(&receipt)
			writeReceiptJSONL(t, store, receipt)
			_, err := store.Load(receipt.ID)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("Load error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func TestStoreAcceptsInstallNoOpReceiptWithoutBackup(t *testing.T) {
	store := NewStore(t.TempDir())
	receipt := semanticallyValidInstallReceipt()
	receipt.Actions = []ActionResult{
		{
			Destination: DestinationClaude,
			Action:      ActionNoOp,
			Before:      DestinationState{ID: DestinationClaude, Path: "/home/agent/.claude/skills/rootline", Kind: KindCorrectSymlink, LexicalTarget: "/repo/.claude/skills/rootline", CanonicalTarget: "/repo/.claude/skills/rootline", Digest: "sha256:source"},
			After:       DestinationState{ID: DestinationClaude, Path: "/home/agent/.claude/skills/rootline", Kind: KindCorrectSymlink, LexicalTarget: "/repo/.claude/skills/rootline", CanonicalTarget: "/repo/.claude/skills/rootline", Digest: "sha256:source"},
			Complete:    true,
		},
	}
	receipt.Backups = nil
	writeReceiptJSONL(t, store, receipt)
	loaded, err := store.Load(receipt.ID)
	if err != nil {
		t.Fatalf("Load valid no-op receipt: %v", err)
	}
	if len(loaded.Backups) != 0 || loaded.Actions[0].Action != ActionNoOp {
		t.Fatalf("loaded no-op receipt = %#v", loaded)
	}
}

func TestReceiptSemanticValidationBranches(t *testing.T) {
	valid := semanticallyValidInstallReceipt()
	incomplete := cloneReceipt(valid)
	incomplete.Complete = false
	incomplete.Actions[0].Complete = false
	incomplete.Actions[0].After = DestinationState{}
	incomplete.Actions[0].Error = operationError(ErrBackupFailed, incomplete.Actions[0].Before.Path, string(incomplete.Actions[0].Destination), os.ErrPermission)
	incomplete.Backups = nil
	if err := validateReceiptSemantics(incomplete); err != nil {
		t.Fatalf("incomplete failed receipt should remain loadable: %v", err)
	}
	tests := []struct {
		name   string
		mutate func(*Receipt)
		want   string
	}{
		{name: "missing plan digest", mutate: func(r *Receipt) { r.PlanDigest = "" }, want: "plan digest"},
		{name: "missing source", mutate: func(r *Receipt) { r.Source = nil }, want: "source evidence"},
		{name: "incomplete source", mutate: func(r *Receipt) { r.Source.SkillPath = "" }, want: "source path"},
		{name: "backup missing original", mutate: func(r *Receipt) { r.Backups[0].OriginalPath = "" }, want: "original path"},
		{name: "directory backup missing stored path", mutate: func(r *Receipt) { r.Backups[0].StoredPath = "" }, want: "stored path"},
		{name: "symlink backup missing target", mutate: func(r *Receipt) {
			r.Backups[0] = Backup{Destination: DestinationClaude, OriginalPath: "/x", Kind: KindCorrectSymlink}
		}, want: "link target"},
		{name: "unsupported backup kind", mutate: func(r *Receipt) { r.Backups[0].Kind = KindUnsupported }, want: "unsupported kind"},
		{name: "unsupported action", mutate: func(r *Receipt) { r.Actions[0].Action = ActionKind("copy_tree") }, want: "unsupported action"},
		{name: "before destination mismatch", mutate: func(r *Receipt) { r.Actions[0].Before.ID = DestinationAgents }, want: "before evidence"},
		{name: "after destination mismatch", mutate: func(r *Receipt) { r.Actions[0].After.ID = DestinationAgents }, want: "after evidence"},
		{name: "missing before path", mutate: func(r *Receipt) { r.Actions[0].Before.Path = "" }, want: "before evidence"},
		{name: "missing complete after path", mutate: func(r *Receipt) { r.Actions[0].After.Path = "" }, want: "after evidence"},
		{name: "incomplete action missing error", mutate: func(r *Receipt) {
			r.Complete = false
			r.Actions[0].Complete = false
			r.Actions[0].After = DestinationState{}
			r.Actions[0].Error = nil
		}, want: "no error evidence"},
		{name: "install no-op invalid before", mutate: func(r *Receipt) {
			r.Actions = []ActionResult{{Destination: DestinationClaude, Action: ActionNoOp, Before: DestinationState{ID: DestinationClaude, Path: "/x", Kind: KindDirectory}, After: DestinationState{ID: DestinationClaude, Path: "/x", Kind: KindDirectory}, Complete: true}}
			r.Backups = []Backup{{Destination: DestinationClaude, OriginalPath: "/x", StoredPath: "/b", Kind: KindDirectory}}
		}, want: "no_op"},
		{name: "install replace invalid before", mutate: func(r *Receipt) {
			r.Actions[0].Before.Kind = KindCorrectSymlink
			r.Actions[0].Action = ActionReplaceWithSymlink
			r.Backups[0] = Backup{Destination: DestinationClaude, OriginalPath: "/x", Kind: KindCorrectSymlink, LinkTarget: "/repo/.claude/skills/rootline"}
		}, want: "replace_with_symlink"},
		{name: "install after not symlink", mutate: func(r *Receipt) { r.Actions[1].After.Kind = KindDirectory }, want: "must finish"},
		{name: "uninstall invalid action", mutate: func(r *Receipt) { makeUninstallReceipt(r); r.Actions[0].Action = ActionCreateSymlink }, want: "not valid for uninstall"},
		{name: "uninstall invalid before", mutate: func(r *Receipt) { makeUninstallReceipt(r); r.Actions[0].Before.Kind = KindDirectory }, want: "requires correct symlink"},
		{name: "uninstall invalid after", mutate: func(r *Receipt) { makeUninstallReceipt(r); r.Actions[0].After.Kind = KindCorrectSymlink }, want: "must finish absent"},
		{name: "restore invalid action", mutate: func(r *Receipt) { r.Operation = OperationRestore; r.Actions[0].Action = ActionCreateSymlink }, want: "not valid for restore"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			receipt := cloneReceipt(valid)
			tt.mutate(&receipt)
			err := validateReceiptSemantics(receipt)
			if err == nil || !strings.Contains(err.Error(), tt.want) {
				t.Fatalf("validateReceiptSemantics error = %v, want containing %q", err, tt.want)
			}
		})
	}
}

func writeReceiptJSONL(t *testing.T, store *Store, receipt Receipt) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(store.receiptsPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	data, err := json.Marshal(receipt)
	if err != nil {
		t.Fatal(err)
	}
	data = append(data, '\n')
	if err := os.WriteFile(store.receiptsPath(), data, 0o600); err != nil {
		t.Fatal(err)
	}
}

func cloneReceipt(receipt Receipt) Receipt {
	clone := receipt
	if receipt.Source != nil {
		source := *receipt.Source
		clone.Source = &source
	}
	clone.Actions = append([]ActionResult(nil), receipt.Actions...)
	clone.Backups = append([]Backup(nil), receipt.Backups...)
	clone.Errors = append([]OperationError(nil), receipt.Errors...)
	return clone
}

func makeUninstallReceipt(receipt *Receipt) {
	receipt.Operation = OperationUninstall
	receipt.Backups = nil
	receipt.Actions = []ActionResult{{
		Destination: DestinationClaude,
		Action:      ActionRemoveManagedSymlink,
		Before:      DestinationState{ID: DestinationClaude, Path: "/home/agent/.claude/skills/rootline", Kind: KindCorrectSymlink, LexicalTarget: "/repo/.claude/skills/rootline", CanonicalTarget: "/repo/.claude/skills/rootline", Digest: "sha256:source"},
		After:       DestinationState{ID: DestinationClaude, Path: "/home/agent/.claude/skills/rootline", Kind: KindAbsent},
		Complete:    true,
	}}
}

func semanticallyValidInstallReceipt() Receipt {
	return Receipt{
		Version:    1,
		Kind:       receiptKind,
		ID:         "r1",
		Timestamp:  time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC),
		Operation:  OperationInstall,
		Complete:   true,
		Source:     &Source{RepoRoot: "/repo", SkillPath: "/repo/.claude/skills/rootline", Commit: "abc123", Digest: "sha256:source"},
		PlanDigest: "sha256:plan",
		Actions: []ActionResult{
			{
				Destination: DestinationClaude,
				Action:      ActionReplaceWithSymlink,
				Before:      DestinationState{ID: DestinationClaude, Path: "/home/agent/.claude/skills/rootline", Kind: KindDirectory, Digest: "sha256:old"},
				After:       DestinationState{ID: DestinationClaude, Path: "/home/agent/.claude/skills/rootline", Kind: KindCorrectSymlink, LexicalTarget: "/repo/.claude/skills/rootline", CanonicalTarget: "/repo/.claude/skills/rootline", Digest: "sha256:source"},
				Complete:    true,
			},
			{
				Destination: DestinationAgents,
				Action:      ActionCreateSymlink,
				Before:      DestinationState{ID: DestinationAgents, Path: "/home/agent/.agents/skills/rootline", Kind: KindAbsent},
				After:       DestinationState{ID: DestinationAgents, Path: "/home/agent/.agents/skills/rootline", Kind: KindCorrectSymlink, LexicalTarget: "/repo/.claude/skills/rootline", CanonicalTarget: "/repo/.claude/skills/rootline", Digest: "sha256:source"},
				Complete:    true,
			},
		},
		Backups: []Backup{{Destination: DestinationClaude, OriginalPath: "/home/agent/.claude/skills/rootline", StoredPath: "/state/backups/r1/claude", Kind: KindDirectory, Digest: "sha256:old"}},
		Errors:  []OperationError{},
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
