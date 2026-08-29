package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"github.com/pablontiv/rootline/internal/skilldist"
)

func TestSkillLifecycleEndToEnd(t *testing.T) {
	fixture := newSkillCommandFixture(t)
	t.Setenv("HOME", fixture.home)
	t.Setenv("XDG_STATE_HOME", fixture.state)

	oldClaude := filepath.Join(fixture.home, ".claude", "skills", "rootline", "SKILL.md")
	mustMkdirAll(t, filepath.Dir(oldClaude))
	mustWriteFile(t, oldClaude, []byte("old claude\n"), 0o600)
	legacy := filepath.Join(fixture.home, ".config", "opencode", "skills", "rootline", "sentinel")
	mustMkdirAll(t, filepath.Dir(legacy))
	mustWriteFile(t, legacy, []byte("legacy untouched\n"), 0o600)

	installDigest := runSkillField(t, "plan_digest", "skill", "install", "--source", fixture.repo)
	installOut, installErr := runCmd(t, "skill", "install", "--source", fixture.repo, "--approve", installDigest)
	if installErr != nil {
		if runtime.GOOS != "windows" || !skillEnvelopeHasCode(t, installOut, skilldist.ErrSymlinkPermission) {
			t.Fatalf("install: %v\n%s", installErr, installOut)
		}
		assertNoActiveCopiedInstallation(t, fixture)
		legacyData, err := os.ReadFile(legacy)
		if err != nil || string(legacyData) != "legacy untouched\n" {
			t.Fatalf("obsolete-path sentinel changed: data=%q err=%v", legacyData, err)
		}
		return
	}
	receiptID := decodeSkillReceiptID(t, installOut)

	statusOut := runSkillSuccess(t, "skill", "status", "--source", fixture.repo)
	assertSkillStatusConverged(t, statusOut)

	uninstallDigest := runSkillField(t, "plan_digest", "skill", "uninstall")
	runSkillSuccess(t, "skill", "uninstall", "--approve", uninstallDigest)

	restoreDigest := runSkillField(t, "plan_digest", "skill", "restore", "--receipt", receiptID)
	runSkillSuccess(t, "skill", "restore", "--receipt", receiptID, "--approve", restoreDigest)

	data, err := os.ReadFile(oldClaude)
	if err != nil || string(data) != "old claude\n" {
		t.Fatalf("Claude preimage not restored: data=%q err=%v", data, err)
	}
	legacyData, err := os.ReadFile(legacy)
	if err != nil || string(legacyData) != "legacy untouched\n" {
		t.Fatalf("obsolete-path sentinel changed: data=%q err=%v", legacyData, err)
	}
}

func runSkillField(t *testing.T, field string, args ...string) string {
	t.Helper()
	out, err := runCmd(t, append(args, "--field", field)...)
	if err != nil {
		t.Fatalf("%v --field %s: %v\n%s", args, field, err, out)
	}
	var value string
	if err := json.Unmarshal([]byte(out), &value); err != nil || value == "" {
		t.Fatalf("%v --field %s output=%q err=%v", args, field, out, err)
	}
	return value
}

func runSkillSuccess(t *testing.T, args ...string) string {
	t.Helper()
	out, err := runCmd(t, args...)
	if err != nil {
		t.Fatalf("%v: %v\n%s", args, err, out)
	}
	return out
}

func decodeSkillReceiptID(t *testing.T, out string) string {
	t.Helper()
	result := decodeSkillEnvelope(t, out)
	if result.Receipt == nil || result.Receipt.ID == "" {
		t.Fatalf("receipt missing from install output: %#v", result)
	}
	return result.Receipt.ID
}

func skillEnvelopeHasCode(t *testing.T, out string, code skilldist.ErrorCode) bool {
	t.Helper()
	result := decodeSkillEnvelope(t, out)
	for _, opErr := range result.Errors {
		if opErr.Code == code {
			return true
		}
	}
	return false
}

func assertSkillStatusConverged(t *testing.T, out string) {
	t.Helper()
	result := decodeSkillEnvelope(t, out)
	if result.Kind != "rootline/skill-status" || !result.Complete || result.Receipt == nil || result.ReceiptDrift || len(result.Errors) != 0 {
		t.Fatalf("status = %#v, want converged with current receipt", result)
	}
	for _, destination := range result.Destinations {
		if destination.Kind != skilldist.KindCorrectSymlink {
			t.Fatalf("destination = %#v, want correct symlink", destination)
		}
	}
}

func assertNoActiveCopiedInstallation(t *testing.T, fixture *skillCommandFixture) {
	t.Helper()
	for _, path := range []string{
		filepath.Join(fixture.home, ".claude", "skills", "rootline"),
		filepath.Join(fixture.home, ".agents", "skills", "rootline"),
	} {
		info, err := os.Lstat(path)
		if os.IsNotExist(err) {
			continue
		}
		if err != nil {
			t.Fatalf("lstat %s: %v", path, err)
		}
		if info.Mode()&os.ModeSymlink != 0 {
			continue
		}
		if filepath.Clean(path) == filepath.Join(fixture.home, ".claude", "skills", "rootline") && info.IsDir() {
			data, readErr := os.ReadFile(filepath.Join(path, "SKILL.md"))
			if readErr == nil && string(data) == "old claude\n" {
				continue
			}
		}
		t.Fatalf("active non-symlink installation remains at %s: mode=%s", path, info.Mode())
	}
}

func mustMkdirAll(t *testing.T, path string) {
	t.Helper()
	if err := os.MkdirAll(path, 0o755); err != nil {
		t.Fatal(err)
	}
}
