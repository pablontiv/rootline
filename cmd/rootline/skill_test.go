package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/pablontiv/rootline/internal/gitenv"
	"github.com/pablontiv/rootline/internal/skilldist"
)

type skillCommandFixture struct {
	repo  string
	home  string
	state string
}

func newSkillCommandFixture(t *testing.T) *skillCommandFixture {
	t.Helper()
	fixture := &skillCommandFixture{
		repo:  initSkillCommandRepository(t),
		home:  filepath.Join(t.TempDir(), "home"),
		state: filepath.Join(t.TempDir(), "state"),
	}
	counter := 0
	previousFactory := skillServiceFactory
	skillServiceFactory = func() (*skilldist.Service, error) {
		return skilldist.New(skilldist.Options{
			HomeDir:  fixture.home,
			StateDir: fixture.state,
			Now: func() time.Time {
				return time.Date(2026, 8, 28, 12, 0, 0, 0, time.UTC)
			},
			NewReceiptID: func() string {
				counter++
				return fmt.Sprintf("receipt-%d", counter)
			},
		})
	}
	t.Cleanup(func() { skillServiceFactory = previousFactory })
	return fixture
}

func TestSkillInstallPlanApplyAndFieldExtraction(t *testing.T) {
	fixture := newSkillCommandFixture(t)
	out, err := runCmd(t, "skill", "install", "--source", fixture.repo, "--field", "plan_digest")
	if err != nil {
		t.Fatalf("plan: %v\n%s", err, out)
	}
	var digest string
	if err := json.Unmarshal([]byte(out), &digest); err != nil || digest == "" {
		t.Fatalf("plan_digest output=%q err=%v", out, err)
	}
	out, err = runCmd(t, "skill", "install", "--source", fixture.repo, "--approve", digest)
	if err != nil {
		t.Fatalf("apply: %v\n%s", err, out)
	}
	var result SkillEnvelope
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatal(err)
	}
	if result.Kind != "rootline/skill-install" || !result.Complete || result.Receipt == nil {
		t.Fatalf("result = %#v", result)
	}

	out, err = runCmd(t, "skill", "status", "--source", fixture.repo, "--field", "receipt.id")
	if err != nil {
		t.Fatalf("status receipt field: %v\n%s", err, out)
	}
	var receiptID string
	if err := json.Unmarshal([]byte(out), &receiptID); err != nil || receiptID == "" {
		t.Fatalf("receipt.id output=%q err=%v", out, err)
	}
}

func TestSkillInstallPlanOnlyExitsZeroDespiteIncomplete(t *testing.T) {
	fixture := newSkillCommandFixture(t)
	out, err := runCmd(t, "skill", "install", "--source", fixture.repo)
	if err != nil {
		t.Fatalf("plan-only should exit zero: %v\n%s", err, out)
	}
	result := decodeSkillEnvelope(t, out)
	if result.Complete || result.PlanDigest == "" || len(result.Errors) != 0 {
		t.Fatalf("plan-only result = %#v", result)
	}
}

func TestSkillCommandsRejectNonJSONOutput(t *testing.T) {
	fixture := newSkillCommandFixture(t)
	_, err := runCmd(t, "skill", "status", "--source", fixture.repo, "-o", "table")
	if err == nil || !strings.Contains(err.Error(), "does not support output format") {
		t.Fatalf("error = %v", err)
	}
}

func TestSkillCommandIsBoundaryPreflightExempt(t *testing.T) {
	fixture := newSkillCommandFixture(t)
	unbounded := t.TempDir()
	mustWriteFile(t, filepath.Join(unbounded, ".stem"), []byte("version: 2\n"), 0o644)
	mustChdir(t, unbounded)

	out, err := runCmd(t, "skill", "status", "--source", fixture.repo)
	if err != nil || strings.Contains(out, "declared governance boundary") {
		t.Fatalf("status was governance-gated: out=%s err=%v", out, err)
	}
}

func TestSkillStaleApprovalEmitsEnvelopeBeforeNonZeroExit(t *testing.T) {
	fixture := newSkillCommandFixture(t)
	mustWriteSkillCommandFile(t, filepath.Join(fixture.home, ".claude", "skills", "rootline"), "SKILL.md", "first")
	planOut, err := runCmd(t, "skill", "install", "--source", fixture.repo)
	if err != nil {
		t.Fatalf("plan: %v\n%s", err, planOut)
	}
	plan := decodeSkillEnvelope(t, planOut)

	mustWriteSkillCommandFile(t, filepath.Join(fixture.home, ".claude", "skills", "rootline"), "SKILL.md", "changed")
	out, err := runCmd(t, "skill", "install", "--source", fixture.repo, "--approve", string(plan.PlanDigest))
	if err == nil {
		t.Fatalf("expected stale approval error, output: %s", out)
	}
	result := decodeSkillEnvelope(t, out)
	if result.Kind != "rootline/skill-install" || len(result.Errors) == 0 || result.Errors[0].Code != skilldist.ErrPreimageDigestChanged {
		t.Fatalf("stale approval result = %#v", result)
	}
}

func TestSkillRestoreRequiresReceipt(t *testing.T) {
	out, err := runCmd(t, "skill", "restore")
	if err == nil {
		t.Fatalf("restore without receipt succeeded: %s", out)
	}
	result := decodeSkillEnvelope(t, out)
	if result.Kind != "rootline/skill-restore" || len(result.Errors) == 0 || result.Errors[0].Code != skilldist.ErrRestoreConflict {
		t.Fatalf("restore without receipt envelope = %#v", result)
	}
	assertRawSkillEnvelopeHasExplicitNullReceipt(t, out)
}

func TestSkillSetupFailureEmitsEnvelopeBeforeNonZeroExit(t *testing.T) {
	fixture := newSkillCommandFixture(t)
	previousFactory := skillServiceFactory
	skillServiceFactory = func() (*skilldist.Service, error) {
		return nil, fmt.Errorf("injected setup failure")
	}
	t.Cleanup(func() { skillServiceFactory = previousFactory })

	out, err := runCmd(t, "skill", "status", "--source", fixture.repo)
	if err == nil {
		t.Fatalf("setup failure succeeded: %s", out)
	}
	result := decodeSkillEnvelope(t, out)
	if result.Kind != "rootline/skill-status" || len(result.Errors) == 0 || !strings.Contains(result.Errors[0].Message, "injected setup failure") {
		t.Fatalf("setup failure envelope = %#v", result)
	}
	assertRawSkillEnvelopeHasExplicitNullReceipt(t, out)
}

func TestSkillPlanEnvelopeIncludesExplicitNullReceipt(t *testing.T) {
	fixture := newSkillCommandFixture(t)
	out, err := runCmd(t, "skill", "install", "--source", fixture.repo)
	if err != nil {
		t.Fatalf("install plan: %v\n%s", err, out)
	}
	assertRawSkillEnvelopeHasExplicitNullReceipt(t, out)
}

func TestDefaultSkillStateRootUsesInjectedWindowsUserConfigDir(t *testing.T) {
	got, err := defaultSkillStateRootForOS("windows", `C:\\Users\\agent`, func(string) string { return "" }, func() (string, error) {
		return `C:\\Users\\agent\\AppData\\Roaming`, nil
	})
	if err != nil {
		t.Fatal(err)
	}
	if got != filepath.Clean(`C:\\Users\\agent\\AppData\\Roaming`) {
		t.Fatalf("windows state root = %q", got)
	}
}

func TestSkillEnvelopeKindsAndReceiptField(t *testing.T) {
	fixture := newSkillCommandFixture(t)

	installPlanOut, err := runCmd(t, "skill", "install", "--source", fixture.repo)
	if err != nil {
		t.Fatalf("install plan: %v\n%s", err, installPlanOut)
	}
	installPlan := decodeSkillEnvelope(t, installPlanOut)
	assertSkillKind(t, installPlan, "rootline/skill-install")

	installOut, err := runCmd(t, "skill", "install", "--source", fixture.repo, "--approve", string(installPlan.PlanDigest))
	if err != nil {
		t.Fatalf("install apply: %v\n%s", err, installOut)
	}
	installed := decodeSkillEnvelope(t, installOut)
	assertSkillKind(t, installed, "rootline/skill-install")
	if installed.Receipt == nil || installed.Receipt.ID == "" {
		t.Fatalf("installed receipt = %#v", installed.Receipt)
	}

	statusOut, err := runCmd(t, "skill", "status", "--source", fixture.repo)
	if err != nil {
		t.Fatalf("status: %v\n%s", err, statusOut)
	}
	assertSkillKind(t, decodeSkillEnvelope(t, statusOut), "rootline/skill-status")

	restorePlanOut, err := runCmd(t, "skill", "restore", "--receipt", installed.Receipt.ID)
	if err != nil {
		t.Fatalf("restore plan: %v\n%s", err, restorePlanOut)
	}
	restorePlan := decodeSkillEnvelope(t, restorePlanOut)
	assertSkillKind(t, restorePlan, "rootline/skill-restore")

	fieldOut, err := runCmd(t, "skill", "status", "--source", fixture.repo, "--field", "receipt.id")
	if err != nil {
		t.Fatalf("status receipt field: %v\n%s", err, fieldOut)
	}
	var statusReceiptID string
	if err := json.Unmarshal([]byte(fieldOut), &statusReceiptID); err != nil || statusReceiptID == "" {
		t.Fatalf("status receipt.id output=%q err=%v", fieldOut, err)
	}

	uninstallPlanOut, err := runCmd(t, "skill", "uninstall")
	if err != nil {
		t.Fatalf("uninstall plan: %v\n%s", err, uninstallPlanOut)
	}
	uninstallPlan := decodeSkillEnvelope(t, uninstallPlanOut)
	assertSkillKind(t, uninstallPlan, "rootline/skill-uninstall")

	uninstallOut, err := runCmd(t, "skill", "uninstall", "--approve", string(uninstallPlan.PlanDigest))
	if err != nil {
		t.Fatalf("uninstall apply: %v\n%s", err, uninstallOut)
	}
	assertSkillKind(t, decodeSkillEnvelope(t, uninstallOut), "rootline/skill-uninstall")
}

func TestSkillSourceDefaultsToCurrentGitRepository(t *testing.T) {
	fixture := newSkillCommandFixture(t)
	mustChdir(t, fixture.repo)

	out, err := runCmd(t, "skill", "status")
	if err != nil {
		t.Fatalf("status: %v\n%s", err, out)
	}
	result := decodeSkillEnvelope(t, out)
	if result.Source == nil || !sameSkillCommandPath(t, result.Source.RepoRoot, fixture.repo) {
		t.Fatalf("source = %#v, want repo %q", result.Source, fixture.repo)
	}
}

func TestSkillCommandExitMatchesServiceFailure(t *testing.T) {
	fixture := newSkillCommandFixture(t)
	mustWriteSkillCommandFile(t, filepath.Join(fixture.home, ".claude", "skills", "rootline"), "SKILL.md", "first")
	planOut, err := runCmd(t, "skill", "install", "--source", fixture.repo)
	if err != nil {
		t.Fatalf("plan: %v\n%s", err, planOut)
	}
	plan := decodeSkillEnvelope(t, planOut)
	if plan.Complete || len(plan.Errors) != 0 {
		t.Fatalf("plan should be incomplete but not failed: %#v", plan)
	}

	successOut, successErr := runCmd(t, "skill", "install", "--source", fixture.repo, "--approve", string(plan.PlanDigest))
	mustWriteSkillCommandFile(t, filepath.Join(fixture.home, ".claude", "skills", "rootline"), "SKILL.md", "changed")
	staleOut, staleErr := runCmd(t, "skill", "install", "--source", fixture.repo, "--approve", string(plan.PlanDigest))

	cases := []struct {
		name string
		out  string
		err  error
	}{
		{name: "plan", out: planOut, err: nil},
		{name: "success", out: successOut, err: successErr},
		{name: "stale", out: staleOut, err: staleErr},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			result := decodeSkillEnvelope(t, tc.out)
			failed := len(result.Errors) > 0
			if failed != (tc.err != nil) {
				t.Fatalf("failed=%v err=%v result=%#v", failed, tc.err, result)
			}
		})
	}
}

func decodeSkillEnvelope(t *testing.T, out string) SkillEnvelope {
	t.Helper()
	var result SkillEnvelope
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid skill envelope: %v\n%s", err, out)
	}
	return result
}

func assertRawSkillEnvelopeHasExplicitNullReceipt(t *testing.T, out string) {
	t.Helper()
	var raw map[string]json.RawMessage
	if err := json.Unmarshal([]byte(out), &raw); err != nil {
		t.Fatalf("invalid skill envelope: %v\n%s", err, out)
	}
	receipt, ok := raw["receipt"]
	if !ok {
		t.Fatalf("receipt field omitted from raw envelope: %s", out)
	}
	if string(receipt) != "null" {
		t.Fatalf("receipt field = %s, want null", receipt)
	}
}

func assertSkillKind(t *testing.T, result SkillEnvelope, want string) {
	t.Helper()
	if result.Version != 1 || result.Kind != want {
		t.Fatalf("version/kind = %d/%q, want 1/%q: %#v", result.Version, result.Kind, want, result)
	}
}

func sameSkillCommandPath(t *testing.T, left, right string) bool {
	t.Helper()
	leftInfo, leftErr := os.Stat(left)
	rightInfo, rightErr := os.Stat(right)
	return leftErr == nil && rightErr == nil && os.SameFile(leftInfo, rightInfo)
}

func initSkillCommandRepository(t *testing.T) string {
	t.Helper()
	repo := t.TempDir()
	runSkillCommandGit(t, repo, "init")
	runSkillCommandGit(t, repo, "config", "user.name", "Rootline Test")
	runSkillCommandGit(t, repo, "config", "user.email", "rootline-test@example.invalid")
	mustWriteSkillCommandFile(t, filepath.Join(repo, ".claude", "skills", "rootline"), "SKILL.md", "---\nname: rootline\n---\n")
	commitSkillCommandRepository(t, repo, "add canonical skill")
	return repo
}

func commitSkillCommandRepository(t *testing.T, repo, message string) {
	t.Helper()
	runSkillCommandGit(t, repo, "add", "--all")
	runSkillCommandGit(t, repo, "commit", "-m", message)
}

func mustWriteSkillCommandFile(t *testing.T, root, name, content string) {
	t.Helper()
	path := filepath.Join(root, name)
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func runSkillCommandGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	// #nosec G204 -- test fixture runs fixed git executable with temp repo paths and test-controlled args.
	cmd := exec.CommandContext(context.Background(), "git", append([]string{"-C", dir}, args...)...)
	cmd.Env = gitenv.ClearedEnv()
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, output)
	}
}
