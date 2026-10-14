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

func isolateHooksGitEnvironment(t *testing.T) {
	t.Helper()
	isolateAmbientGitScope(t)
	t.Setenv("GIT_CONFIG_GLOBAL", os.DevNull)
	t.Setenv("GIT_CONFIG_NOSYSTEM", "1")
	t.Setenv("GIT_CONFIG_COUNT", "0")
	t.Setenv("GIT_CONFIG_PARAMETERS", "")
	t.Setenv("GIT_TEMPLATE_DIR", t.TempDir())
}

func makeHooksRepo(t *testing.T) string {
	t.Helper()
	isolateHooksGitEnvironment(t)
	dir := t.TempDir()
	runFixtureGit(t, dir, "init", "--quiet")
	// Git returns a physical absolute path, including on systems where the
	// temporary directory is reached through a symlink.
	dir, err := filepath.EvalSymlinks(dir)
	if err != nil {
		t.Fatal(err)
	}
	return dir
}

func TestHooksInstallAndStatus(t *testing.T) {
	for _, hooksPath := range []string{"unset", "relative", "absolute"} {
		for _, fromSubdirectory := range []bool{false, true} {
			location := "root"
			if fromSubdirectory {
				location = "subdirectory"
			}
			t.Run(hooksPath+"/"+location, func(t *testing.T) {
				repo := makeHooksRepo(t)
				hooksDir := filepath.Join(repo, ".git", "hooks")
				unusedHook := ""
				if hooksPath != "unset" {
					unusedHook = filepath.Join(hooksDir, "pre-commit")
					if err := os.MkdirAll(hooksDir, 0o755); err != nil {
						t.Fatal(err)
					}
					mustWriteFile(t, unusedHook, []byte(hookScript), 0o755)
					configuredPath := "managed hooks"
					if hooksPath == "absolute" {
						configuredPath = filepath.Join(repo, "absolute hooks")
						hooksDir = configuredPath
					} else {
						hooksDir = filepath.Join(repo, configuredPath)
					}
					runFixtureGit(t, repo, "config", "--local", "core.hooksPath", configuredPath)
				}

				cwd := repo
				if fromSubdirectory {
					cwd = filepath.Join(repo, "nested", "directory")
					if err := os.MkdirAll(cwd, 0o755); err != nil {
						t.Fatal(err)
					}
				}
				t.Chdir(cwd)
				gotDir, err := gitHooksDir()
				if err != nil {
					t.Fatal(err)
				}
				if gotDir != hooksDir || !filepath.IsAbs(gotDir) {
					t.Fatalf("gitHooksDir() = %q, want absolute %q", gotDir, hooksDir)
				}
				hookPath := filepath.Join(hooksDir, "pre-commit")

				out, err := runCmd(t, "hooks", "status")
				if err != nil || out != "not installed\n" {
					t.Fatalf("initial status = %q, %v; want not installed", out, err)
				}
				out, err = runCmd(t, "hooks", "install")
				if err != nil || out != "Installed pre-commit hook\n" {
					t.Fatalf("install = %q, %v", out, err)
				}
				info, err := os.Stat(hookPath)
				if err != nil {
					t.Fatalf("hook file not found: %v", err)
				}
				if info.Mode()&0o111 == 0 {
					t.Error("hook file should be executable")
				}
				if content := mustReadFile(t, hookPath); string(content) != hookScript {
					t.Fatalf("installed hook = %q, want %q", content, hookScript)
				}
				out, err = runCmd(t, "hooks", "status")
				if err != nil || out != "installed\n" {
					t.Fatalf("installed status = %q, %v; want installed", out, err)
				}
				out, err = runCmd(t, "hooks", "uninstall")
				if err != nil || out != "Removed pre-commit hook\n" {
					t.Fatalf("uninstall = %q, %v", out, err)
				}
				if _, err := os.Stat(hookPath); !os.IsNotExist(err) {
					t.Fatalf("hook should be removed, stat error = %v", err)
				}
				out, err = runCmd(t, "hooks", "status")
				if err != nil || out != "not installed\n" {
					t.Fatalf("status after uninstall = %q, %v; want not installed", out, err)
				}
				out, err = runCmd(t, "hooks", "uninstall")
				if err != nil || out != "No pre-commit hook installed\n" {
					t.Fatalf("uninstall missing hook = %q, %v", out, err)
				}
				if unusedHook != "" {
					if content := mustReadFile(t, unusedHook); string(content) != hookScript {
						t.Fatalf("unused .git/hooks hook changed: %q", content)
					}
				}
			})
		}
	}
}

func TestHooksInstallExistingNonRootline(t *testing.T) {
	repo := makeHooksRepo(t)
	t.Chdir(repo)
	hookPath, err := preCommitPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatal(err)
	}
	customHook := "#!/bin/sh\necho custom\n"
	mustWriteFile(t, hookPath, []byte(customHook), 0o755)

	if _, err := runCmd(t, "hooks", "install"); err == nil {
		t.Fatal("expected error for existing non-rootline hook")
	}
	if content := mustReadFile(t, hookPath); string(content) != customHook {
		t.Fatalf("install overwrote non-rootline hook: %q", content)
	}
	out, err := runCmd(t, "hooks", "status")
	if err != nil || out != "not installed (non-rootline hook exists)\n" {
		t.Fatalf("non-rootline status = %q, %v", out, err)
	}
	out, err = runCmd(t, "hooks", "install", "--force")
	if err != nil || out != "Installed pre-commit hook\n" {
		t.Fatalf("forced install = %q, %v", out, err)
	}
	if content := mustReadFile(t, hookPath); string(content) != hookScript {
		t.Fatalf("forced install hook = %q, want %q", content, hookScript)
	}
}

func TestHooksUninstallNonRootline(t *testing.T) {
	repo := makeHooksRepo(t)
	t.Chdir(repo)
	hookPath, err := preCommitPath()
	if err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(filepath.Dir(hookPath), 0o755); err != nil {
		t.Fatal(err)
	}
	customHook := "#!/bin/sh\necho custom\n"
	mustWriteFile(t, hookPath, []byte(customHook), 0o755)

	if _, err := runCmd(t, "hooks", "uninstall"); err == nil {
		t.Fatal("expected error when uninstalling non-rootline hook")
	}
	if content := mustReadFile(t, hookPath); string(content) != customHook {
		t.Fatalf("uninstall changed non-rootline hook: %q", content)
	}
}

func TestHooksNotRepository(t *testing.T) {
	isolateHooksGitEnvironment(t)
	dir := t.TempDir()
	t.Chdir(dir)
	t.Setenv("GIT_CEILING_DIRECTORIES", filepath.Dir(dir))
	for _, command := range []string{"install", "status", "uninstall"} {
		t.Run(command, func(t *testing.T) {
			if _, err := runCmd(t, "hooks", command); err == nil || err.Error() != "not a git repository" {
				t.Fatalf("%s error = %v, want not a git repository", command, err)
			}
		})
	}
}
