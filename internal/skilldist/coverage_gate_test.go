package skilldist

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServiceDefaultsAndErrorWrappers(t *testing.T) {
	home := filepath.Join(t.TempDir(), "home")
	state := filepath.Join(t.TempDir(), "xdg-state")
	t.Setenv("XDG_STATE_HOME", state)

	t.Setenv("HOME", home)
	defaultHomeService, err := New(Options{StateDir: state})
	if err != nil {
		t.Fatal(err)
	}
	if defaultHomeService.homeDir != home {
		t.Fatalf("default home service home = %q, want %q", defaultHomeService.homeDir, home)
	}
	service, err := New(Options{HomeDir: home})
	if err != nil {
		t.Fatal(err)
	}
	if service.homeDir != home || service.stateDir != state {
		t.Fatalf("service dirs = %q/%q, want %q/%q", service.homeDir, service.stateDir, home, state)
	}
	if id := service.newReceiptID(); len(id) != 32 {
		t.Fatalf("random receipt ID length = %d, want 32", len(id))
	}
	if got := defaultStateDir(home); got != state {
		t.Fatalf("defaultStateDir with XDG = %q, want %q", got, state)
	}
	t.Setenv("XDG_STATE_HOME", "")
	if got := defaultStateDir(home); got != filepath.Join(home, ".local", "state") {
		t.Fatalf("defaultStateDir fallback = %q", got)
	}

	if _, err := New(Options{HomeDir: ".", StateDir: state}); err == nil {
		t.Fatal("New accepted current directory as home")
	}
	if _, err := New(Options{HomeDir: string(filepath.Separator), StateDir: state}); err == nil {
		t.Fatal("New accepted filesystem root as home")
	}
	if _, err := New(Options{HomeDir: home, StateDir: string(filepath.Separator)}); err == nil {
		t.Fatal("New accepted filesystem root as state")
	}

	wrapped := errors.New("wrapped")
	if got := (&OperationError{Message: "explicit"}).Error(); got != "explicit" {
		t.Fatalf("OperationError explicit Error = %q", got)
	}
	if got := (&OperationError{Code: ErrBackupFailed, Err: wrapped}).Error(); !strings.Contains(got, string(ErrBackupFailed)) || !strings.Contains(got, "wrapped") {
		t.Fatalf("OperationError fallback Error = %q", got)
	}
	gitErr := &gitCommandError{args: []string{"status"}, err: wrapped}
	if !strings.Contains(gitErr.Error(), "git status failed") || !errors.Is(gitErr.Unwrap(), wrapped) {
		t.Fatalf("gitCommandError = %q unwrap=%v", gitErr.Error(), gitErr.Unwrap())
	}
	copyErr := copyDirectoryFailure{err: wrapped, createdDestination: true}
	if copyErr.Error() != "wrapped" || !errors.Is(copyErr.Unwrap(), wrapped) || !copyDirectoryCreatedDestination(copyErr) {
		t.Fatalf("copyDirectoryFailure did not expose expected error metadata")
	}
}

func TestReceiptDriftAndDestinationConvergenceBranches(t *testing.T) {
	source := Source{RepoRoot: "/repo", SkillPath: "/repo/.claude/skills/rootline", Commit: "abc", Digest: "sha256:source"}
	states := []DestinationState{
		{ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindCorrectSymlink, Digest: source.Digest, LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath},
		{ID: DestinationAgents, Path: "/home/.agents/skills/rootline", Kind: KindCorrectSymlink, Digest: source.Digest, LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath},
	}
	receipt := completeInstallReceiptForPlan("receipt-1", source)

	if !allDestinationsExact(states) {
		t.Fatal("exact destinations did not converge")
	}
	shortStates := states[:1]
	if allDestinationsExact(shortStates) {
		t.Fatal("short destination inventory converged")
	}
	notExact := append([]DestinationState(nil), states...)
	notExact[1].Kind = KindAbsent
	if allDestinationsExact(notExact) {
		t.Fatal("absent destination converged")
	}

	if receiptDrifted(Receipt{}, source, states) {
		// nil source is intentionally drift, assertion below keeps the branch visible.
	} else {
		t.Fatal("receipt with nil source did not drift")
	}
	changedSource := receipt
	changedSource.Source = copySourcePtr(Source{RepoRoot: source.RepoRoot, SkillPath: source.SkillPath, Commit: "def", Digest: source.Digest})
	if !receiptDrifted(changedSource, source, states) {
		t.Fatal("changed source commit did not drift")
	}
	missingAction := receipt
	missingAction.Actions = missingAction.Actions[:1]
	if !receiptDrifted(missingAction, source, states) {
		t.Fatal("missing destination action did not drift")
	}
	changedPath := receipt
	changedPath.Actions = append([]ActionResult(nil), receipt.Actions...)
	changedPath.Actions[0].After.Path = "/other"
	if !receiptDrifted(changedPath, source, states) {
		t.Fatal("changed recorded path did not drift")
	}
	if receiptDrifted(receipt, source, states) {
		t.Fatal("matching receipt drifted")
	}
}

func TestRestoreAfterUninstallRecreatesRecordedPreimage(t *testing.T) {
	fixture := newServiceFixture(t)
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "old claude\n")
	installPlan := fixture.service.Install(context.Background(), fixture.repo, "")
	installed := fixture.service.Install(context.Background(), fixture.repo, installPlan.Plan.Digest)
	if installed.Failed() || installed.Receipt == nil {
		t.Fatalf("install = %#v", installed)
	}
	uninstallPlan := fixture.service.Uninstall(context.Background(), "")
	uninstalled := fixture.service.Uninstall(context.Background(), uninstallPlan.Plan.Digest)
	if uninstalled.Failed() || !uninstalled.Complete {
		t.Fatalf("uninstall = %#v", uninstalled)
	}

	restorePlan := fixture.service.Restore(context.Background(), installed.Receipt.ID, "")
	if restorePlan.Failed() || restorePlan.Plan == nil || restorePlan.Attempted {
		t.Fatalf("restore plan after uninstall = %#v", restorePlan)
	}
	restored := fixture.service.Restore(context.Background(), installed.Receipt.ID, restorePlan.Plan.Digest)
	if restored.Failed() || !restored.Complete {
		t.Fatalf("restore after uninstall = %#v", restored)
	}
	data, err := os.ReadFile(filepath.Join(fixture.claudePath(), "SKILL.md"))
	if err != nil || string(data) != "old claude\n" {
		t.Fatalf("restored data=%q err=%v", data, err)
	}
	assertPathAbsent(t, fixture.agentsPath())
}

func TestStoreValidationAndBackupFailureBranches(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := store.Append(Receipt{}); err == nil {
		t.Fatal("Append accepted receipt with empty ID")
	}
	if err := store.Reserve("../bad"); err == nil {
		t.Fatal("Reserve accepted unsafe receipt ID")
	}
	if _, err := store.Backup("../bad", DestinationState{ID: DestinationClaude, Kind: KindAbsent}); err == nil {
		t.Fatal("Backup accepted unsafe receipt ID")
	}
	if err := store.Reserve("r1"); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Backup("r1", DestinationState{ID: DestinationID("bad/id"), Kind: KindAbsent}); err == nil {
		t.Fatal("Backup accepted unsafe destination ID")
	}
	unsupported := filepath.Join(t.TempDir(), "unsupported")
	if _, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: unsupported, Kind: KindUnsupported}); err == nil {
		t.Fatal("Backup accepted unsupported destination kind")
	}
	existing := filepath.Join(t.TempDir(), "existing")
	if err := os.WriteFile(existing, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.Backup("r1", DestinationState{ID: DestinationClaude, Path: existing, Kind: KindAbsent}); err == nil {
		t.Fatal("Backup accepted existing path for absent preimage")
	}
	if err := store.RestoreBackup(Backup{Destination: DestinationClaude, Kind: KindAbsent}, existing); err == nil {
		t.Fatal("RestoreBackup overwrote existing destination")
	}
	if err := store.RestoreBackup(Backup{Destination: DestinationClaude, Kind: KindUnsupported}, filepath.Join(t.TempDir(), "dest")); err == nil {
		t.Fatal("RestoreBackup accepted unsupported backup kind")
	}

	if err := store.VerifyBackup(Backup{Destination: DestinationClaude, Kind: KindDirectory}); err == nil {
		t.Fatal("VerifyBackup accepted directory without stored path")
	}
	dir := filepath.Join(t.TempDir(), "dir")
	mustWriteSkillFile(t, dir, "SKILL.md", "actual")
	if err := store.VerifyBackup(Backup{Destination: DestinationClaude, OriginalPath: dir, StoredPath: dir, Kind: KindDirectory, Digest: "sha256:wrong"}); err == nil {
		t.Fatal("VerifyBackup accepted digest mismatch")
	}
	if err := store.VerifyBackup(Backup{Destination: DestinationClaude, Kind: KindCorrectSymlink}); err == nil {
		t.Fatal("VerifyBackup accepted symlink without link target")
	}
	if err := store.VerifyBackup(Backup{Destination: DestinationClaude, Kind: KindCorrectSymlink, LinkTarget: "target", StoredPath: dir}); err == nil {
		t.Fatal("VerifyBackup accepted symlink with stored path")
	}
	if err := store.VerifyBackup(Backup{Destination: DestinationClaude, Kind: KindUnsupported}); err == nil {
		t.Fatal("VerifyBackup accepted unsupported backup kind")
	}

	filePath := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(filePath, []byte("file"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := store.backupDirectory("r1", DestinationState{ID: DestinationClaude, Path: filePath, Kind: KindDirectory}, Backup{Destination: DestinationClaude, OriginalPath: filePath, Kind: KindDirectory}); err == nil {
		t.Fatal("backupDirectory accepted regular file")
	}
	wrongDigest := DestinationState{ID: DestinationClaude, Path: dir, Kind: KindDirectory, Digest: "sha256:wrong"}
	if _, err := store.backupDirectory("r1", wrongDigest, Backup{Destination: DestinationClaude, OriginalPath: dir, Kind: KindDirectory}); err == nil {
		t.Fatal("backupDirectory accepted changed digest")
	}

	parent := t.TempDir()
	target := filepath.Join(parent, "target")
	mustWriteSkillFile(t, target, "SKILL.md", "target")
	link := filepath.Join(parent, "link")
	if err := os.Symlink("target", link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := store.backupSymlink("r1", DestinationState{ID: DestinationAgents, Path: link, Kind: KindDivergentSymlink, LexicalTarget: "wrong"}, Backup{Destination: DestinationAgents, OriginalPath: link, Kind: KindDivergentSymlink}); err == nil {
		t.Fatal("backupSymlink accepted changed lexical target")
	}
}

func TestRemoveAndRestoreVerificationFailureBranches(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := removeReceiptedSymlink(DestinationState{ID: DestinationClaude, Path: missing, Kind: KindCorrectSymlink}); err == nil {
		t.Fatal("removeReceiptedSymlink accepted missing path")
	}
	filePath := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(filePath, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if _, err := removeReceiptedSymlink(DestinationState{ID: DestinationClaude, Path: filePath, Kind: KindCorrectSymlink}); err == nil {
		t.Fatal("removeReceiptedSymlink accepted non-symlink path")
	}
	parent := t.TempDir()
	target := filepath.Join(parent, "target")
	mustWriteSkillFile(t, target, "SKILL.md", "target")
	link := filepath.Join(parent, "link")
	if err := os.Symlink("target", link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	if _, err := removeReceiptedSymlink(DestinationState{ID: DestinationClaude, Path: link, Kind: KindCorrectSymlink, LexicalTarget: "wrong"}); err == nil {
		t.Fatal("removeReceiptedSymlink accepted changed lexical target")
	}

	if _, err := verifyRestoredPreimage(DestinationState{ID: DestinationClaude, Kind: KindDirectory}, Backup{Kind: KindDirectory}, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("verifyRestoredPreimage accepted missing directory")
	}
	if _, err := verifyRestoredPreimage(DestinationState{ID: DestinationClaude, Kind: KindDirectory}, Backup{Kind: KindDirectory}, filePath); err == nil {
		t.Fatal("verifyRestoredPreimage accepted file as directory")
	}
	dir := filepath.Join(t.TempDir(), "dir")
	mustWriteSkillFile(t, dir, "SKILL.md", "data")
	if _, err := verifyRestoredPreimage(DestinationState{ID: DestinationClaude, Kind: KindDirectory, Digest: "sha256:wrong"}, Backup{Kind: KindDirectory}, dir); err == nil {
		t.Fatal("verifyRestoredPreimage accepted directory digest mismatch")
	}
	if _, err := verifyRestoredPreimage(DestinationState{ID: DestinationClaude, Kind: KindDivergentSymlink, LexicalTarget: "target"}, Backup{Kind: KindDivergentSymlink, LinkTarget: "target"}, filepath.Join(t.TempDir(), "missing")); err == nil {
		t.Fatal("verifyRestoredPreimage accepted missing symlink")
	}
	if _, err := verifyRestoredPreimage(DestinationState{ID: DestinationClaude, Kind: KindDivergentSymlink, LexicalTarget: "target"}, Backup{Kind: KindDivergentSymlink, LinkTarget: "target"}, filePath); err == nil {
		t.Fatal("verifyRestoredPreimage accepted file as symlink")
	}
	link = filepath.Join(parent, "link-target-mismatch")
	if err := os.Symlink("target", link); err != nil {
		t.Fatal(err)
	}
	if _, err := verifyRestoredPreimage(DestinationState{ID: DestinationClaude, Kind: KindDivergentSymlink, LexicalTarget: "wrong"}, Backup{Kind: KindDivergentSymlink, LinkTarget: "target"}, link); err == nil {
		t.Fatal("verifyRestoredPreimage accepted symlink target mismatch")
	}
}

func TestRestoreDestinationDirectFailureBranches(t *testing.T) {
	service, err := New(Options{HomeDir: filepath.Join(t.TempDir(), "home"), StateDir: filepath.Join(t.TempDir(), "state")})
	if err != nil {
		t.Fatal(err)
	}
	missingCurrent := DestinationState{ID: DestinationClaude, Path: filepath.Join(t.TempDir(), "missing"), Kind: KindCorrectSymlink}
	if _, _, err := service.restoreDestination(Action{Kind: ActionRestorePreimage, Destination: missingCurrent}, DestinationState{}, Backup{}, Backup{}); err == nil {
		t.Fatal("restoreDestination accepted missing managed link")
	}

	dest := filepath.Join(t.TempDir(), "dest")
	invalidBackup := Backup{Destination: DestinationClaude, Kind: KindDirectory, StoredPath: filepath.Join(t.TempDir(), "missing"), Digest: "sha256:missing"}
	if _, _, err := service.restoreDestination(Action{Kind: ActionRestorePreimage, Destination: DestinationState{ID: DestinationClaude, Path: dest, Kind: KindAbsent}}, DestinationState{ID: DestinationClaude, Path: dest, Kind: KindDirectory}, invalidBackup, Backup{Destination: DestinationClaude, Kind: KindAbsent}); err == nil {
		t.Fatal("restoreDestination accepted invalid preimage backup")
	}

	stored := filepath.Join(t.TempDir(), "stored")
	mustWriteSkillFile(t, stored, "SKILL.md", "stored")
	digest, err := DigestTree(stored)
	if err != nil {
		t.Fatal(err)
	}
	badPreimage := DestinationState{ID: DestinationClaude, Path: dest, Kind: KindDirectory, Digest: "sha256:wrong"}
	validBackup := Backup{Destination: DestinationClaude, OriginalPath: dest, StoredPath: stored, Kind: KindDirectory, Digest: digest}
	if _, _, err := service.restoreDestination(Action{Kind: ActionRestorePreimage, Destination: DestinationState{ID: DestinationClaude, Path: dest, Kind: KindAbsent}}, badPreimage, validBackup, Backup{Destination: DestinationClaude, Kind: KindAbsent}); err == nil {
		t.Fatal("restoreDestination accepted verification mismatch")
	}
}

func TestRestoreHelperBranches(t *testing.T) {
	service, err := New(Options{HomeDir: filepath.Join(t.TempDir(), "home"), StateDir: filepath.Join(t.TempDir(), "state")})
	if err != nil {
		t.Fatal(err)
	}
	missing := filepath.Join(t.TempDir(), "missing")
	if !service.rollbackCurrentDestination(Backup{Destination: DestinationClaude, Kind: KindAbsent}, missing) {
		t.Fatal("rollback of absent current destination failed")
	}
	existing := filepath.Join(t.TempDir(), "existing")
	if err := os.WriteFile(existing, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if service.rollbackCurrentDestination(Backup{Destination: DestinationClaude, Kind: KindAbsent}, existing) {
		t.Fatal("rollback reported success while destination existed")
	}
	if service.rollbackCurrentDestination(Backup{Destination: DestinationClaude, Kind: KindDirectory, StoredPath: filepath.Join(t.TempDir(), "missing")}, missing) {
		t.Fatal("rollback reported success for invalid backup")
	}

	current := DestinationState{ID: DestinationClaude, Path: missing, Kind: KindCorrectSymlink}
	if state := inventoryStateBestEffort(current, DestinationState{}, Backup{}); state.Kind != KindAbsent {
		t.Fatalf("best-effort absent state = %#v", state)
	}
	preimageDir := filepath.Join(t.TempDir(), "preimage")
	mustWriteSkillFile(t, preimageDir, "SKILL.md", "preimage")
	digest, err := DigestTree(preimageDir)
	if err != nil {
		t.Fatal(err)
	}
	backup := Backup{Destination: DestinationClaude, OriginalPath: preimageDir, StoredPath: preimageDir, Kind: KindDirectory, Digest: digest}
	if state := inventoryStateBestEffort(DestinationState{ID: DestinationClaude, Path: preimageDir, Kind: KindCorrectSymlink}, DestinationState{ID: DestinationClaude, Kind: KindDirectory, Digest: digest}, backup); state.Kind != KindDirectory || state.Digest != digest {
		t.Fatalf("best-effort restored state = %#v", state)
	}
	fallbackCurrent := DestinationState{ID: DestinationClaude, Path: existing, Kind: KindCorrectSymlink}
	if state := inventoryStateBestEffort(fallbackCurrent, DestinationState{ID: DestinationClaude, Kind: KindDirectory}, Backup{Kind: KindDirectory}); state != fallbackCurrent {
		t.Fatalf("best-effort fallback = %#v, want %#v", state, fallbackCurrent)
	}

	parent := t.TempDir()
	target := filepath.Join(parent, "target")
	mustWriteSkillFile(t, target, "SKILL.md", "target")
	link := filepath.Join(parent, "link")
	if err := os.Symlink("target", link); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	canonicalTarget, err := filepath.EvalSymlinks(target)
	if err != nil {
		t.Fatal(err)
	}
	state, err := verifyRestoredPreimage(
		DestinationState{ID: DestinationAgents, Kind: KindDivergentSymlink, LexicalTarget: "target", CanonicalTarget: canonicalTarget},
		Backup{Destination: DestinationAgents, Kind: KindDivergentSymlink, LinkTarget: "target"},
		link,
	)
	if err != nil || state.Kind != KindDivergentSymlink || state.LexicalTarget != "target" {
		t.Fatalf("verifyRestoredPreimage symlink = %#v err=%v", state, err)
	}
	if _, err := verifyRestoredPreimage(DestinationState{ID: DestinationClaude, Kind: KindUnsupported}, Backup{Kind: KindUnsupported}, filepath.Join(t.TempDir(), "dest")); err == nil {
		t.Fatal("verifyRestoredPreimage accepted unsupported preimage")
	}
}

func TestServiceNoOpAndStaleApprovalBranches(t *testing.T) {
	fixture := newServiceFixture(t)
	installPlan := fixture.service.Install(context.Background(), fixture.repo, "")
	installed := fixture.service.Install(context.Background(), fixture.repo, installPlan.Plan.Digest)
	if installed.Failed() {
		t.Fatalf("install = %#v", installed)
	}

	noOpPlan := fixture.service.Install(context.Background(), fixture.repo, "")
	noOpInstall := fixture.service.Install(context.Background(), fixture.repo, noOpPlan.Plan.Digest)
	if noOpInstall.Failed() || !noOpInstall.Complete || noOpInstall.Receipt == nil {
		t.Fatalf("no-op install = %#v", noOpInstall)
	}
	for _, action := range noOpInstall.Receipt.Actions {
		if action.Action != ActionNoOp || !action.Complete {
			t.Fatalf("no-op receipt action = %#v", action)
		}
	}
	if stale := fixture.service.Install(context.Background(), fixture.repo, Digest("sha256:stale")); !stale.Failed() || stale.Attempted {
		t.Fatalf("stale install approval = %#v", stale)
	}
	if stale := fixture.service.Uninstall(context.Background(), Digest("sha256:stale")); !stale.Failed() || stale.Attempted {
		t.Fatalf("stale uninstall approval = %#v", stale)
	}
	restorePlan := fixture.service.Restore(context.Background(), installed.Receipt.ID, "")
	if restorePlan.Failed() {
		t.Fatalf("restore plan = %#v", restorePlan)
	}
	if stale := fixture.service.Restore(context.Background(), installed.Receipt.ID, Digest("sha256:stale")); !stale.Failed() || stale.Attempted {
		t.Fatalf("stale restore approval = %#v", stale)
	}
}

func TestServiceFailureBranchesReportEnvelopes(t *testing.T) {
	t.Run("install reserve failure", func(t *testing.T) {
		fixture := newServiceFixture(t)
		plan := fixture.service.Install(context.Background(), fixture.repo, "")
		if err := os.MkdirAll(fixture.service.store.receiptBackupDir("receipt-1"), 0o700); err != nil {
			t.Fatal(err)
		}
		result := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
		if !result.Failed() || result.Receipt == nil || result.Complete {
			t.Fatalf("install reserve failure = %#v", result)
		}
		assertResultErrorCode(t, result, ErrBackupFailed)
	})

	t.Run("uninstall reserve failure", func(t *testing.T) {
		fixture := newServiceFixture(t)
		plan := fixture.service.Install(context.Background(), fixture.repo, "")
		installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
		if installed.Failed() {
			t.Fatalf("install = %#v", installed)
		}
		uninstallPlan := fixture.service.Uninstall(context.Background(), "")
		if err := os.MkdirAll(fixture.service.store.receiptBackupDir("receipt-2"), 0o700); err != nil {
			t.Fatal(err)
		}
		result := fixture.service.Uninstall(context.Background(), uninstallPlan.Plan.Digest)
		if !result.Failed() || result.Receipt == nil || result.Complete {
			t.Fatalf("uninstall reserve failure = %#v", result)
		}
		assertResultErrorCode(t, result, ErrBackupFailed)
	})

	t.Run("restore reserve failure", func(t *testing.T) {
		fixture := newServiceFixture(t)
		mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "old")
		plan := fixture.service.Install(context.Background(), fixture.repo, "")
		installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
		if installed.Failed() {
			t.Fatalf("install = %#v", installed)
		}
		restorePlan := fixture.service.Restore(context.Background(), installed.Receipt.ID, "")
		if err := os.MkdirAll(fixture.service.store.receiptBackupDir("receipt-2"), 0o700); err != nil {
			t.Fatal(err)
		}
		result := fixture.service.Restore(context.Background(), installed.Receipt.ID, restorePlan.Plan.Digest)
		if !result.Failed() || result.Receipt == nil || result.Complete {
			t.Fatalf("restore reserve failure = %#v", result)
		}
		assertResultErrorCode(t, result, ErrBackupFailed)
	})

	t.Run("planning failures", func(t *testing.T) {
		service, err := New(Options{HomeDir: filepath.Join(t.TempDir(), "home"), StateDir: filepath.Join(t.TempDir(), "state")})
		if err != nil {
			t.Fatal(err)
		}
		if result := service.Install(context.Background(), filepath.Join(t.TempDir(), "missing"), ""); !result.Failed() || result.Attempted {
			t.Fatalf("missing source install = %#v", result)
		}
		if result := service.Status(context.Background(), filepath.Join(t.TempDir(), "missing")); !result.Failed() || result.Attempted {
			t.Fatalf("missing source status = %#v", result)
		}
		if result := service.Uninstall(context.Background(), ""); !result.Failed() || result.Attempted {
			t.Fatalf("uninstall without receipt = %#v", result)
		}
		if result := service.Restore(context.Background(), "missing", ""); !result.Failed() || result.Attempted {
			t.Fatalf("restore missing receipt = %#v", result)
		}
	})

	t.Run("status and restore surface store scan failures", func(t *testing.T) {
		fixture := newServiceFixture(t)
		if err := os.MkdirAll(filepath.Dir(fixture.service.store.receiptsPath()), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fixture.service.store.receiptsPath(), []byte("{bad json}\n"), 0o600); err != nil {
			t.Fatal(err)
		}
		if result := fixture.service.Status(context.Background(), fixture.repo); !result.Failed() || result.Attempted {
			t.Fatalf("status with corrupt store = %#v", result)
		}
		if result := fixture.service.Restore(context.Background(), "receipt-1", ""); !result.Failed() || result.Attempted {
			t.Fatalf("restore with corrupt store = %#v", result)
		}
	})

	t.Run("unsupported destination prevents install planning", func(t *testing.T) {
		fixture := newServiceFixture(t)
		if err := os.MkdirAll(filepath.Dir(fixture.claudePath()), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(fixture.claudePath(), []byte("not a directory"), 0o600); err != nil {
			t.Fatal(err)
		}
		result := fixture.service.Install(context.Background(), fixture.repo, "")
		if !result.Failed() || result.Plan != nil || result.Attempted {
			t.Fatalf("unsupported destination install = %#v", result)
		}
		assertResultErrorCode(t, result, ErrUnsupportedFileType)
	})

	t.Run("uninstall detects missing source tree", func(t *testing.T) {
		fixture := newServiceFixture(t)
		plan := fixture.service.Install(context.Background(), fixture.repo, "")
		installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
		if installed.Failed() {
			t.Fatalf("install = %#v", installed)
		}
		if err := os.RemoveAll(fixture.skillPath()); err != nil {
			t.Fatal(err)
		}
		result := fixture.service.Uninstall(context.Background(), "")
		if !result.Failed() || result.Plan != nil || result.Attempted {
			t.Fatalf("uninstall missing source = %#v", result)
		}
		assertResultErrorCode(t, result, ErrVerificationFailed)
	})
}

func TestStoreReceiptParsingAndCopyBranches(t *testing.T) {
	store := NewStore(t.TempDir())
	if err := os.MkdirAll(filepath.Dir(store.receiptsPath()), 0o700); err != nil {
		t.Fatal(err)
	}
	for _, tt := range []struct {
		name string
		data []byte
	}{
		{name: "empty line", data: []byte("\n")},
		{name: "missing id", data: []byte(`{"version":1}` + "\n")},
		{name: "duplicate id", data: []byte(`{"id":"r1"}` + "\n" + `{"id":"r1"}` + "\n")},
	} {
		t.Run(tt.name, func(t *testing.T) {
			if err := os.WriteFile(store.receiptsPath(), tt.data, 0o600); err != nil {
				t.Fatal(err)
			}
			if _, err := store.scanReceipts(); err == nil {
				t.Fatalf("scanReceipts accepted %s", tt.name)
			}
		})
	}

	if err := copyDirectory("", filepath.Join(t.TempDir(), "dest")); err == nil {
		t.Fatal("copyDirectory accepted empty source")
	}
	fileSource := filepath.Join(t.TempDir(), "file")
	if err := os.WriteFile(fileSource, []byte("data"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := copyDirectory(fileSource, filepath.Join(t.TempDir(), "dest")); err == nil {
		t.Fatal("copyDirectory accepted regular file source")
	}
	missingSource := filepath.Join(t.TempDir(), "missing")
	if err := copyDirectory(missingSource, filepath.Join(t.TempDir(), "dest")); err == nil {
		t.Fatal("copyDirectory accepted missing source")
	}
}

func TestSourceResolutionFailureBranches(t *testing.T) {
	if _, err := ResolveSource(context.Background(), ""); err == nil {
		t.Fatal("ResolveSource accepted empty path")
	}
	nonRepo := t.TempDir()
	mustWriteSkillFile(t, filepath.Join(nonRepo, ".claude", "skills", "rootline"), "SKILL.md", "---\nname: rootline\n---\n")
	if _, err := ResolveSource(context.Background(), nonRepo); err == nil {
		t.Fatal("ResolveSource accepted non-git directory")
	}
	repo := initSkillRepository(t)
	if err := os.RemoveAll(filepath.Join(repo, ".claude", "skills", "rootline", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveSource(context.Background(), repo); err == nil {
		t.Fatal("ResolveSource accepted missing canonical skill file")
	}
	repo = initSkillRepository(t)
	if err := os.Remove(filepath.Join(repo, ".claude", "skills", "rootline", "SKILL.md")); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(repo, ".claude", "skills", "rootline", "SKILL.md"), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := ResolveSource(context.Background(), repo); err == nil {
		t.Fatal("ResolveSource accepted directory canonical skill file")
	}
}

func TestPublishSymlinkAndSourceGuardBranches(t *testing.T) {
	fixture := newServiceFixture(t)
	source, err := ResolveSource(context.Background(), fixture.repo)
	if err != nil {
		t.Fatal(err)
	}
	backup := Backup{Destination: DestinationClaude, Kind: KindAbsent}

	driftPath := filepath.Join(fixture.home, "drift", "rootline")
	executor := &installExecutor{symlink: func(oldname, newname string) error {
		if err := os.Symlink(oldname, newname); err != nil {
			return err
		}
		mustWriteSkillFile(t, fixture.skillPath(), "SKILL.md", "drift")
		return nil
	}}
	outcome, err := executor.publishSymlink(DestinationState{ID: DestinationClaude, Path: driftPath, Kind: KindAbsent}, source, backup)
	if err == nil || !outcome.rolledBack {
		t.Fatalf("publish source drift after symlink = outcome %#v err=%v", outcome, err)
	}
	assertPathAbsent(t, driftPath)

	missingSource := source
	missingSource.SkillPath = filepath.Join(t.TempDir(), "missing")
	if _, err := (&installExecutor{symlink: os.Symlink}).publishSymlink(DestinationState{ID: DestinationAgents, Path: filepath.Join(fixture.home, "missing-source"), Kind: KindAbsent}, missingSource, backup); err == nil {
		t.Fatal("publishSymlink accepted missing source")
	}
	if _, err := (&installExecutor{symlink: os.Symlink}).publishSymlink(DestinationState{ID: DestinationAgents, Path: filepath.Join(fixture.home, "noop"), Kind: KindAbsent}, source, backup); err == nil {
		t.Fatal("publishSymlink accepted drifted source digest")
	}

	assertOperationErrorCode(t, mapSymlinkError(errors.New("plain"), DestinationState{ID: DestinationClaude, Path: driftPath}), ErrVerificationFailed)
	assertOperationErrorCode(t, mapSymlinkError(os.ErrExist, DestinationState{ID: DestinationClaude, Path: driftPath}), ErrRestoreConflict)
	if _, err := uniqueSiblingPath(filepath.Join(t.TempDir(), "rootline")); err != nil {
		t.Fatal(err)
	}
	if got := filepathClean(""); got != "" {
		t.Fatalf("filepathClean empty = %q", got)
	}
	var result Result
	addResultError(&result, nil)
	if len(result.Errors) != 0 {
		t.Fatalf("nil addResultError mutated result: %#v", result)
	}
	if got := coerceOperationError(ErrBackupFailed, "", "", nil); got != nil {
		t.Fatalf("coerceOperationError nil = %#v", got)
	}
}

func TestSourcePrimaryCheckoutGuardBranches(t *testing.T) {
	missingGit := t.TempDir()
	if err := requirePrimaryCheckout(missingGit); err == nil {
		t.Fatal("requirePrimaryCheckout accepted missing .git")
	}
	linked := t.TempDir()
	if err := os.WriteFile(filepath.Join(linked, ".git"), []byte("gitdir: elsewhere"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := requirePrimaryCheckout(linked); err == nil {
		t.Fatal("requirePrimaryCheckout accepted linked-worktree .git file")
	}
}

func TestActionKindRejectsUnknownEntryKind(t *testing.T) {
	if _, err := installActionKind(DestinationState{ID: DestinationClaude, Path: "/tmp/rootline", Kind: EntryKind("mystery")}); err == nil {
		t.Fatal("installActionKind accepted unknown entry kind")
	}
	if _, err := restoreActionKind(DestinationState{ID: DestinationClaude, Path: "/tmp/rootline", Kind: EntryKind("mystery")}); err == nil {
		t.Fatal("restoreActionKind accepted unknown entry kind")
	}
}

func TestReceiptAndPlanValidationBranches(t *testing.T) {
	source := Source{RepoRoot: "/repo", SkillPath: "/repo/.claude/skills/rootline", Commit: "abc", Digest: "sha256:source"}
	for _, receipt := range []Receipt{
		{Complete: true, Source: copySourcePtr(source)},
		{ID: "r1", Complete: false, Source: copySourcePtr(source)},
		{ID: "r1", Complete: true},
	} {
		if _, err := receiptPlanSource(receipt); err == nil {
			t.Fatalf("receiptPlanSource accepted %#v", receipt)
		}
	}

	current := DestinationState{ID: DestinationClaude, Path: "/home/.claude/skills/rootline", Kind: KindCorrectSymlink, Digest: source.Digest, LexicalTarget: source.SkillPath, CanonicalTarget: source.SkillPath}
	receipt := completeInstallReceiptForPlan("r1", source)
	receipt.Actions[0].Before = DestinationState{ID: DestinationClaude, Path: current.Path, Kind: KindUnsupported}
	if _, err := BuildRestorePlan(receipt, []DestinationState{current}); err == nil {
		t.Fatal("BuildRestorePlan accepted unsupported preimage")
	}
	receipt = completeInstallReceiptForPlan("r1", source)
	absent := DestinationState{ID: DestinationClaude, Path: current.Path, Kind: KindAbsent}
	if _, err := BuildRestorePlan(receipt, []DestinationState{absent}); err != nil {
		t.Fatalf("BuildRestorePlan rejected safe absent current state: %v", err)
	}
	receipt.Actions[0].After.Kind = KindAbsent
	if _, err := BuildRestorePlan(receipt, []DestinationState{absent}); err == nil {
		t.Fatal("BuildRestorePlan accepted absent state without recorded managed link")
	}
}
