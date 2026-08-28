package skilldist

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"syscall"
	"testing"
	"time"
)

func TestInstallRequiresExactPlanThenConvergesIdempotently(t *testing.T) {
	fixture := newServiceFixture(t)
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "old")

	planned := fixture.service.Install(context.Background(), fixture.repo, "")
	if planned.Attempted || planned.Complete || planned.Plan == nil || planned.Plan.Digest == "" {
		t.Fatalf("unexpected plan result: %#v", planned)
	}
	if _, err := os.Lstat(fixture.claudePath()); err != nil {
		t.Fatalf("plan mutated preimage: %v", err)
	}

	applied := fixture.service.Install(context.Background(), fixture.repo, planned.Plan.Digest)
	if applied.Failed() || !applied.Attempted || !applied.Complete || applied.Receipt == nil {
		t.Fatalf("apply result: %#v", applied)
	}
	assertSymlinkTo(t, fixture.claudePath(), fixture.skillPath())
	assertSymlinkTo(t, fixture.agentsPath(), fixture.skillPath())

	repeated := fixture.service.Install(context.Background(), fixture.repo, "")
	for _, action := range repeated.Plan.Actions {
		if action.Kind != ActionNoOp {
			t.Fatalf("repeat action = %#v, want no-op", action)
		}
	}
}

func TestInstallRejectsStalePreimageApprovalBeforeMutation(t *testing.T) {
	fixture := newServiceFixture(t)
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "first")
	planned := fixture.service.Install(context.Background(), fixture.repo, "")
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "changed")

	result := fixture.service.Install(context.Background(), fixture.repo, planned.Plan.Digest)
	if !result.Failed() || result.Attempted {
		t.Fatalf("stale approval result = %#v", result)
	}
	assertResultErrorCode(t, result, ErrPreimageDigestChanged)
	data, err := os.ReadFile(filepath.Join(fixture.claudePath(), "SKILL.md"))
	if err != nil || string(data) != "changed" {
		t.Fatalf("stale approval mutated preimage: data=%q err=%v", data, err)
	}
}

func TestInstallDetectsSourceContentDriftDuringPublication(t *testing.T) {
	fixture := newServiceFixture(t)
	planned := fixture.service.Install(context.Background(), fixture.repo, "")
	fixture.service.executor.beforeSymlink = func(DestinationState) error {
		mustWriteSkillFile(t, fixture.skillPath(), "SKILL.md", "drift")
		return nil
	}

	result := fixture.service.Install(context.Background(), fixture.repo, planned.Plan.Digest)
	if !result.Failed() || !result.Attempted || result.Complete {
		t.Fatalf("source drift result = %#v", result)
	}
	assertResultErrorCode(t, result, ErrSourceDigestChanged)
	assertPathAbsent(t, fixture.claudePath())
	assertPathAbsent(t, fixture.agentsPath())
}

func TestInstallVerifierRejectsLexicalTargetMismatch(t *testing.T) {
	fixture := newServiceFixture(t)
	alias := filepath.Join(t.TempDir(), "source-alias")
	if err := os.Symlink(fixture.skillPath(), alias); err != nil {
		t.Skipf("symlinks unavailable: %v", err)
	}
	planned := fixture.service.Install(context.Background(), fixture.repo, "")
	fixture.service.executor.symlink = func(_, newname string) error {
		return os.Symlink(alias, newname)
	}

	result := fixture.service.Install(context.Background(), fixture.repo, planned.Plan.Digest)
	if !result.Failed() || result.Complete {
		t.Fatalf("lexical mismatch result = %#v", result)
	}
	assertResultErrorCode(t, result, ErrVerificationFailed)
	assertPathAbsent(t, fixture.claudePath())
}

func TestInstallRollbackRetainsVerifiedClaudeAndIncompleteReceiptWhenAgentsFails(t *testing.T) {
	fixture := newServiceFixture(t)
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "claude old")
	mustWriteSkillFile(t, fixture.agentsPath(), "SKILL.md", "agents old")
	planned := fixture.service.Install(context.Background(), fixture.repo, "")
	fixture.service.executor.beforeSymlink = func(state DestinationState) error {
		if state.ID == DestinationAgents {
			return operationError(ErrVerificationFailed, state.Path, string(state.ID), errors.New("agents publication failpoint"))
		}
		return nil
	}

	result := fixture.service.Install(context.Background(), fixture.repo, planned.Plan.Digest)
	if !result.Failed() || !result.Attempted || result.Complete || result.Receipt == nil {
		t.Fatalf("agents failpoint result = %#v", result)
	}
	assertSymlinkTo(t, fixture.claudePath(), fixture.skillPath())
	data, err := os.ReadFile(filepath.Join(fixture.agentsPath(), "SKILL.md"))
	if err != nil || string(data) != "agents old" {
		t.Fatalf("agents rollback data=%q err=%v", data, err)
	}
	latest, ok, err := fixture.service.store.Latest()
	if err != nil || !ok {
		t.Fatalf("Latest = %#v, %v, %v", latest, ok, err)
	}
	if latest.Complete || len(latest.Errors) == 0 {
		t.Fatalf("latest receipt = %#v, want incomplete with error", latest)
	}
}

func TestInstallPreservesRecreatedFinalPathAndIndependentBackupOnIncompleteRollback(t *testing.T) {
	fixture := newServiceFixture(t)
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "old backup")
	planned := fixture.service.Install(context.Background(), fixture.repo, "")
	fixture.service.executor.beforeSymlink = func(state DestinationState) error {
		if state.ID != DestinationClaude {
			return nil
		}
		mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "external")
		return nil
	}

	result := fixture.service.Install(context.Background(), fixture.repo, planned.Plan.Digest)
	if !result.Failed() || result.Complete || result.Receipt == nil {
		t.Fatalf("recreated final path result = %#v", result)
	}
	data, err := os.ReadFile(filepath.Join(fixture.claudePath(), "SKILL.md"))
	if err != nil || string(data) != "external" {
		t.Fatalf("external final path not preserved: data=%q err=%v", data, err)
	}
	if len(result.Receipt.Backups) != 1 {
		t.Fatalf("backups = %#v, want one independent backup", result.Receipt.Backups)
	}
	if err := fixture.service.store.VerifyBackup(result.Receipt.Backups[0]); err != nil {
		t.Fatalf("independent backup invalid: %v", err)
	}
	if len(result.Receipt.Actions) == 0 || result.Receipt.Actions[0].RolledBack {
		t.Fatalf("rollback unexpectedly complete: %#v", result.Receipt.Actions)
	}
}

func TestInstallSymlinkErrorsMapPermissionCodesAndVerificationFailuresRemainDistinct(t *testing.T) {
	tests := []struct {
		name    string
		install func(*testing.T, *serviceFixture)
		want    ErrorCode
	}{
		{
			name: "os.ErrPermission",
			install: func(t *testing.T, fixture *serviceFixture) {
				fixture.service.executor.symlink = func(_, _ string) error { return os.ErrPermission }
			},
			want: ErrSymlinkPermission,
		},
		{
			name: "EPERM",
			install: func(t *testing.T, fixture *serviceFixture) {
				fixture.service.executor.symlink = func(_, _ string) error { return syscall.EPERM }
			},
			want: ErrSymlinkPermission,
		},
		{
			name: "EACCES",
			install: func(t *testing.T, fixture *serviceFixture) {
				fixture.service.executor.symlink = func(_, _ string) error { return syscall.EACCES }
			},
			want: ErrSymlinkPermission,
		},
		{
			name: "verification failure",
			install: func(t *testing.T, fixture *serviceFixture) {
				wrong := filepath.Join(t.TempDir(), "wrong")
				mustWriteSkillFile(t, wrong, "SKILL.md", "wrong")
				fixture.service.executor.symlink = func(_, newname string) error { return os.Symlink(wrong, newname) }
			},
			want: ErrVerificationFailed,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newServiceFixture(t)
			planned := fixture.service.Install(context.Background(), fixture.repo, "")
			tt.install(t, fixture)
			result := fixture.service.Install(context.Background(), fixture.repo, planned.Plan.Digest)
			if !result.Failed() {
				t.Fatalf("result succeeded: %#v", result)
			}
			assertResultErrorCode(t, result, tt.want)
		})
	}
}

func TestStatusReportsReceiptDriftAfterCanonicalSkillUpdate(t *testing.T) {
	fixture := newServiceFixture(t)
	planned := fixture.service.Install(context.Background(), fixture.repo, "")
	applied := fixture.service.Install(context.Background(), fixture.repo, planned.Plan.Digest)
	if applied.Failed() {
		t.Fatalf("install: %#v", applied)
	}

	mustWriteSkillFile(t, fixture.skillPath(), "SKILL.md", "updated")
	commitSkillRepository(t, fixture.repo, "update canonical skill")

	status := fixture.service.Status(context.Background(), fixture.repo)
	if status.Failed() || !status.Complete || status.Attempted || !status.ReceiptDrift {
		t.Fatalf("status = %#v, want complete drift without mutating error", status)
	}
}

func TestUninstallRemovesOnlyIntactReceiptedSymlinks(t *testing.T) {
	fixture := newServiceFixture(t)
	plan := fixture.service.Install(context.Background(), fixture.repo, "")
	installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
	if installed.Failed() {
		t.Fatalf("install: %#v", installed)
	}

	uninstallPlan := fixture.service.Uninstall(context.Background(), "")
	if uninstallPlan.Attempted || uninstallPlan.Plan == nil {
		t.Fatalf("uninstall plan = %#v", uninstallPlan)
	}
	removed := fixture.service.Uninstall(context.Background(), uninstallPlan.Plan.Digest)
	if removed.Failed() || !removed.Complete {
		t.Fatalf("uninstall = %#v", removed)
	}
	for _, path := range []string{fixture.claudePath(), fixture.agentsPath()} {
		if _, err := os.Lstat(path); !os.IsNotExist(err) {
			t.Fatalf("%s still exists: %v", path, err)
		}
	}
}

func TestUninstallRefusesUnreceiptedOrRetargetedSymlink(t *testing.T) {
	t.Run("unreceipted", func(t *testing.T) {
		fixture := newServiceFixture(t)
		if err := os.MkdirAll(filepath.Dir(fixture.claudePath()), 0o700); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(fixture.skillPath(), fixture.claudePath()); err != nil {
			t.Skipf("symlinks unavailable: %v", err)
		}

		result := fixture.service.Uninstall(context.Background(), "")
		if !result.Failed() || result.Plan != nil || result.Attempted {
			t.Fatalf("unreceipted uninstall = %#v", result)
		}
		assertResultErrorCode(t, result, ErrRestoreConflict)
		assertSymlinkTo(t, fixture.claudePath(), fixture.skillPath())
	})

	t.Run("retargeted", func(t *testing.T) {
		fixture := newServiceFixture(t)
		plan := fixture.service.Install(context.Background(), fixture.repo, "")
		installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
		if installed.Failed() {
			t.Fatalf("install: %#v", installed)
		}
		wrong := filepath.Join(t.TempDir(), "wrong")
		mustWriteSkillFile(t, wrong, "SKILL.md", "wrong")
		if err := os.Remove(fixture.claudePath()); err != nil {
			t.Fatal(err)
		}
		if err := os.Symlink(wrong, fixture.claudePath()); err != nil {
			t.Fatal(err)
		}

		result := fixture.service.Uninstall(context.Background(), "")
		if !result.Failed() || result.Plan != nil || result.Attempted {
			t.Fatalf("retargeted uninstall = %#v", result)
		}
		assertResultErrorCode(t, result, ErrRestoreConflict)
		assertSymlinkTo(t, fixture.claudePath(), wrong)
	})
}

func TestUninstallDoesNotRestoreOldDirectoriesAutomatically(t *testing.T) {
	fixture := newServiceFixture(t)
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "claude original")
	mustWriteSkillFile(t, fixture.agentsPath(), "SKILL.md", "agents original")
	plan := fixture.service.Install(context.Background(), fixture.repo, "")
	installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
	if installed.Failed() {
		t.Fatalf("install: %#v", installed)
	}

	uninstallPlan := fixture.service.Uninstall(context.Background(), "")
	removed := fixture.service.Uninstall(context.Background(), uninstallPlan.Plan.Digest)
	if removed.Failed() || !removed.Complete {
		t.Fatalf("uninstall = %#v", removed)
	}
	assertPathAbsent(t, fixture.claudePath())
	assertPathAbsent(t, fixture.agentsPath())
}

func TestRestoreRecreatesRecordedDirectoryPreimage(t *testing.T) {
	fixture := newServiceFixture(t)
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "original")
	plan := fixture.service.Install(context.Background(), fixture.repo, "")
	installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
	if installed.Failed() {
		t.Fatalf("install: %#v", installed)
	}
	restorePlan := fixture.service.Restore(context.Background(), installed.Receipt.ID, "")
	if restorePlan.Failed() || restorePlan.Plan == nil || restorePlan.Attempted {
		t.Fatalf("restore plan = %#v", restorePlan)
	}
	restored := fixture.service.Restore(context.Background(), installed.Receipt.ID, restorePlan.Plan.Digest)
	if restored.Failed() || !restored.Complete {
		t.Fatalf("restore = %#v", restored)
	}
	data, err := os.ReadFile(filepath.Join(fixture.claudePath(), "SKILL.md"))
	if err != nil || string(data) != "original" {
		t.Fatalf("restored preimage data=%q err=%v", data, err)
	}
	assertPathAbsent(t, fixture.agentsPath())
	if restored.Receipt == nil || restored.Receipt.Operation != OperationRestore || restored.Receipt.ID == installed.Receipt.ID || len(restored.Receipt.Backups) != 2 {
		t.Fatalf("restore receipt = %#v", restored.Receipt)
	}
}

func TestRestoreRefusesMissingOrCorruptBackup(t *testing.T) {
	for _, tt := range []struct {
		name   string
		damage func(*testing.T, Backup)
	}{
		{
			name: "missing",
			damage: func(t *testing.T, backup Backup) {
				t.Helper()
				if err := os.RemoveAll(backup.StoredPath); err != nil {
					t.Fatal(err)
				}
			},
		},
		{
			name: "corrupt",
			damage: func(t *testing.T, backup Backup) {
				t.Helper()
				if err := os.WriteFile(filepath.Join(backup.StoredPath, "SKILL.md"), []byte("corrupt"), 0o600); err != nil {
					t.Fatal(err)
				}
			},
		},
	} {
		t.Run(tt.name, func(t *testing.T) {
			fixture := newServiceFixture(t)
			mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "original")
			plan := fixture.service.Install(context.Background(), fixture.repo, "")
			installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
			if installed.Failed() || len(installed.Receipt.Backups) == 0 {
				t.Fatalf("install = %#v", installed)
			}
			tt.damage(t, installed.Receipt.Backups[0])

			result := fixture.service.Restore(context.Background(), installed.Receipt.ID, "")
			if !result.Failed() || result.Plan != nil || result.Attempted {
				t.Fatalf("restore with %s backup = %#v", tt.name, result)
			}
			assertResultErrorCode(t, result, ErrVerificationFailed)
			assertSymlinkTo(t, fixture.claudePath(), fixture.skillPath())
		})
	}
}

func TestRestoreApprovalChangesWhenCurrentStateDrifts(t *testing.T) {
	fixture := newServiceFixture(t)
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "original")
	plan := fixture.service.Install(context.Background(), fixture.repo, "")
	installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
	if installed.Failed() {
		t.Fatalf("install: %#v", installed)
	}
	restorePlan := fixture.service.Restore(context.Background(), installed.Receipt.ID, "")
	if restorePlan.Failed() || restorePlan.Plan == nil {
		t.Fatalf("restore plan = %#v", restorePlan)
	}

	mustWriteSkillFile(t, fixture.skillPath(), "SKILL.md", "drift")
	result := fixture.service.Restore(context.Background(), installed.Receipt.ID, restorePlan.Plan.Digest)
	if !result.Failed() || result.Attempted {
		t.Fatalf("stale restore approval = %#v", result)
	}
	assertResultErrorCode(t, result, ErrPreimageDigestChanged)
	assertSymlinkTo(t, fixture.claudePath(), fixture.skillPath())
}

func TestRestorePreservesUnobservedDestinationState(t *testing.T) {
	fixture := newServiceFixture(t)
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "original")
	plan := fixture.service.Install(context.Background(), fixture.repo, "")
	installed := fixture.service.Install(context.Background(), fixture.repo, plan.Plan.Digest)
	if installed.Failed() {
		t.Fatalf("install: %#v", installed)
	}
	restorePlan := fixture.service.Restore(context.Background(), installed.Receipt.ID, "")
	if restorePlan.Failed() || restorePlan.Plan == nil {
		t.Fatalf("restore plan = %#v", restorePlan)
	}
	if err := os.Remove(fixture.claudePath()); err != nil {
		t.Fatal(err)
	}
	mustWriteSkillFile(t, fixture.claudePath(), "SKILL.md", "external")

	result := fixture.service.Restore(context.Background(), installed.Receipt.ID, restorePlan.Plan.Digest)
	if !result.Failed() || result.Attempted {
		t.Fatalf("restore over unobserved state = %#v", result)
	}
	assertResultErrorCode(t, result, ErrRestoreConflict)
	data, err := os.ReadFile(filepath.Join(fixture.claudePath(), "SKILL.md"))
	if err != nil || string(data) != "external" {
		t.Fatalf("unobserved state was not preserved: data=%q err=%v", data, err)
	}
}

type serviceFixture struct {
	repo    string
	home    string
	state   string
	service *Service
}

func newServiceFixture(t *testing.T) *serviceFixture {
	t.Helper()
	repo := initSkillRepository(t)
	home := filepath.Join(t.TempDir(), "home")
	state := filepath.Join(t.TempDir(), "state")
	counter := 0
	service, err := New(Options{
		HomeDir:  home,
		StateDir: state,
		Now: func() time.Time {
			return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
		},
		NewReceiptID: func() string {
			counter++
			return fmt.Sprintf("receipt-%d", counter)
		},
	})
	if err != nil {
		t.Fatal(err)
	}
	return &serviceFixture{repo: repo, home: home, state: state, service: service}
}

func (f *serviceFixture) skillPath() string {
	return filepath.Join(f.repo, ".claude", "skills", "rootline")
}

func (f *serviceFixture) claudePath() string {
	return filepath.Join(f.home, ".claude", "skills", "rootline")
}

func (f *serviceFixture) agentsPath() string {
	return filepath.Join(f.home, ".agents", "skills", "rootline")
}

func assertSymlinkTo(t *testing.T, path, target string) {
	t.Helper()
	info, err := os.Lstat(path)
	if err != nil {
		t.Fatalf("Lstat(%q): %v", path, err)
	}
	if info.Mode()&os.ModeSymlink == 0 {
		t.Fatalf("%q mode = %v, want symlink", path, info.Mode())
	}
	got, err := os.Readlink(path)
	if err != nil {
		t.Fatalf("Readlink(%q): %v", path, err)
	}
	if filepath.Clean(got) != filepath.Clean(target) {
		t.Fatalf("Readlink(%q) = %q, want %q", path, got, target)
	}
}

func assertPathAbsent(t *testing.T, path string) {
	t.Helper()
	if _, err := os.Lstat(path); !os.IsNotExist(err) {
		t.Fatalf("Lstat(%q) err = %v, want not exist", path, err)
	}
}

func assertResultErrorCode(t *testing.T, result Result, code ErrorCode) {
	t.Helper()
	for _, opErr := range result.Errors {
		if opErr.Code == code {
			return
		}
	}
	t.Fatalf("result errors = %#v, want code %s", result.Errors, code)
}
