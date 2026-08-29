package main

import (
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestPrePushDoesNotMutateHome(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("pre-push is a Bash hook")
	}
	repoRoot, err := filepath.Abs(filepath.Join("..", ".."))
	if err != nil {
		t.Fatal(err)
	}
	testHome := t.TempDir()
	sentinel := filepath.Join(testHome, ".claude", "skills", "rootline", "SKILL.md")
	if err := os.MkdirAll(filepath.Dir(sentinel), 0o700); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, sentinel, []byte("do not replace\n"), 0o600)

	cmd := exec.Command(filepath.Join(repoRoot, ".githooks", "pre-push"), "origin", "unused") //nolint:gosec // fixed repository hook under test
	cmd.Dir = repoRoot
	cmd.Stdin = strings.NewReader("")
	cmd.Env = append(os.Environ(), "HOME="+testHome)
	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("pre-push failed: %v\n%s", err, output)
	}
	data, err := os.ReadFile(sentinel)
	if err != nil || string(data) != "do not replace\n" {
		t.Fatalf("pre-push mutated home: data=%q err=%v", data, err)
	}
}

func TestHooksInstallAndStatus(t *testing.T) {
	// We're in a git repo, so this should work
	hookPath, err := preCommitPath()
	if err != nil {
		t.Skip("not in a git repo")
	}

	// Save and restore existing hook
	existing, hadHook := os.ReadFile(hookPath)
	defer func() {
		if hadHook == nil {
			_ = os.WriteFile(hookPath, existing, 0755) //nolint:gosec // test restores the repository hook path resolved by preCommitPath
		} else {
			_ = os.Remove(hookPath)
		}
	}()

	// Remove any existing hook
	_ = os.Remove(hookPath)

	// Test install
	out, err := runCmd(t, "hooks", "install")
	if err != nil {
		t.Fatalf("install error: %v", err)
	}
	if !strings.Contains(out, "Installed") {
		t.Errorf("expected 'Installed', got: %s", out)
	}

	// Verify file exists and is executable
	info, err := os.Stat(hookPath)
	if err != nil {
		t.Fatalf("hook file not found: %v", err)
	}
	if info.Mode()&0111 == 0 {
		t.Error("hook file should be executable")
	}

	// Verify marker
	content := mustReadFile(t, hookPath)
	if !strings.Contains(string(content), hookMarker) {
		t.Error("hook should contain rootline-managed marker")
	}

	// Test status
	out, _ = runCmd(t, "hooks", "status")
	if !strings.Contains(out, "installed") {
		t.Errorf("expected 'installed', got: %s", out)
	}

	// Test uninstall
	out, _ = runCmd(t, "hooks", "uninstall")
	if !strings.Contains(out, "Removed") {
		t.Errorf("expected 'Removed', got: %s", out)
	}

	// Test status after uninstall
	out, _ = runCmd(t, "hooks", "status")
	if !strings.Contains(out, "not installed") {
		t.Errorf("expected 'not installed', got: %s", out)
	}
}

func TestHooksInstallExistingNonRootline(t *testing.T) {
	hookPath, err := preCommitPath()
	if err != nil {
		t.Skip("not in a git repo")
	}

	existing, hadHook := os.ReadFile(hookPath)
	defer func() {
		if hadHook == nil {
			_ = os.WriteFile(hookPath, existing, 0755) //nolint:gosec // test restores the repository hook path resolved by preCommitPath
		} else {
			_ = os.Remove(hookPath)
		}
	}()

	// Write a non-rootline hook
	if err := os.MkdirAll(filepath.Dir(hookPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	mustWriteFile(t, hookPath, []byte("#!/bin/sh\necho custom\n"), 0755)

	_, err = runCmd(t, "hooks", "install")
	if err == nil {
		t.Fatal("expected error for existing non-rootline hook")
	}

	// With --force it should work
	out, err := runCmd(t, "hooks", "install", "--force")
	if err != nil {
		t.Fatalf("expected --force to succeed: %v", err)
	}
	if !strings.Contains(out, "Installed") {
		t.Errorf("expected 'Installed', got: %s", out)
	}

	// Clean up
	_ = os.Remove(hookPath)
}

func TestHooksUninstallNonRootline(t *testing.T) {
	hookPath, err := preCommitPath()
	if err != nil {
		t.Skip("not in a git repo")
	}

	existing, hadHook := os.ReadFile(hookPath)
	defer func() {
		if hadHook == nil {
			_ = os.WriteFile(hookPath, existing, 0755) //nolint:gosec // test restores the repository hook path resolved by preCommitPath
		} else {
			_ = os.Remove(hookPath)
		}
	}()

	// Write non-rootline hook
	if err := os.MkdirAll(filepath.Dir(hookPath), 0755); err != nil {
		t.Fatalf("MkdirAll: %v", err)
	}
	mustWriteFile(t, hookPath, []byte("#!/bin/sh\necho custom\n"), 0755)

	_, err = runCmd(t, "hooks", "uninstall")
	if err == nil {
		t.Fatal("expected error when uninstalling non-rootline hook")
	}

	// Clean up
	_ = os.Remove(hookPath)
}
