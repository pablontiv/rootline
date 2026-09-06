package e2e

import (
	"bytes"
	"encoding/json"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	"github.com/pablontiv/rootline/internal/fix"
)

// A stored report must resume past satisfied paths, then become a read-only replay.
func TestRepairSetFieldResumeAndReplay(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "rootline")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "./cmd/rootline") //nolint:gosec // fixed compiler command; output path belongs to this test
	build.Dir = filepath.Join("..", "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build dev CLI: %v\n%s", err, output)
	}
	root := t.TempDir() // Deliberately no Git repository.
	body := "# Document\n\nKeep this text.\n\n---\n\nAnd this text.\n"
	files := map[string]string{
		".stem":         "version: 2\nroot: true\nscope:\n  match: '*.md'\nschema:\n  alpha:\n    type: string\n    required: true\n",
		"a.md":          "---\n# Preserve formatting of an already satisfied field.\nalpha:  'new'\n---\n" + body,
		"b.md":          "---\nalpha: old\n---\n" + body,
		"report.json":   `{"version":1,"kind":"rootline/proposals","proposals":[{"type":"set_field","field":"alpha","value":"new","paths":["a.md","b.md"]}]}`,
		"unrelated.txt": "Leave me alone.\n",
	}
	stamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	for name, content := range files {
		path := filepath.Join(root, name)
		if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
			t.Fatal(err)
		}
		if err := os.Chtimes(path, stamp, stamp); err != nil {
			t.Fatal(err)
		}
	}
	assertFile := func(name, want string, untouched bool) {
		t.Helper()
		path := filepath.Join(root, name)
		if got := string(mustReadE2EFile(t, path)); got != want {
			t.Fatalf("%s content: got %q, want %q", name, got, want)
		}
		info, err := os.Stat(path)
		if err != nil {
			t.Fatal(err)
		}
		if runtime.GOOS != "windows" && info.Mode().Perm() != 0o600 {
			t.Fatalf("%s mode changed to %o", name, info.Mode().Perm())
		}
		if untouched && !info.ModTime().Equal(stamp) {
			t.Fatalf("%s was unexpectedly written", name)
		}
	}
	run := func(dryRun bool, changed, skipped int) {
		t.Helper()
		args := []string{"repair", "apply", "--root", root, "--report", filepath.Join(root, "report.json"), "-o", "json"}
		if dryRun {
			args = append(args, "--dry-run")
		}
		cmd := exec.Command(bin, args...) //nolint:gosec // locally built CLI, isolated fixture paths
		cmd.Dir = root
		var stdout, stderr bytes.Buffer
		cmd.Stdout, cmd.Stderr = &stdout, &stderr
		if err := cmd.Run(); err != nil {
			t.Fatalf("repair apply: %v\nstdout: %s\nstderr: %s", err, &stdout, &stderr)
		}
		var result fix.RepairResult
		if err := json.Unmarshal(stdout.Bytes(), &result); err != nil {
			t.Fatalf("decode repair: %v\n%s", err, &stdout)
		}
		if stderr.Len() != 0 || result.Version != 1 || result.Kind != "rootline/repair" || !result.Complete || result.DryRun != dryRun ||
			len(result.Errors) != 0 || len(result.Rejected) != 0 || len(result.RolledBack) != 0 || len(result.Changed) != changed || len(result.Skipped) != skipped {
			t.Fatalf("unexpected repair result: %s\nstderr: %s", &stdout, &stderr)
		}
		if changed == 1 && !strings.Contains(result.Changed[0], "b.md") {
			t.Fatalf("pending b.md was not selected: %v", result.Changed)
		}
	}
	run(true, 1, 1)
	for name, content := range files {
		assertFile(name, content, true)
	}
	run(false, 1, 1)
	files["b.md"] = "---\nalpha: new\n---\n" + body
	for name, content := range files {
		assertFile(name, content, name != "b.md")
	}
	// Reset only the timestamp, so a redundant atomic rewrite is observable.
	if err := os.Chtimes(filepath.Join(root, "b.md"), stamp, stamp); err != nil {
		t.Fatal(err)
	}
	run(true, 0, 2)
	run(false, 0, 2)
	for name, content := range files {
		assertFile(name, content, true)
	}
	validate := exec.Command(bin, "validate", "--all", root, "-o", "json") //nolint:gosec // locally built CLI
	if output, err := validate.CombinedOutput(); err != nil {
		t.Fatalf("post-repair validation: %v\n%s", err, output)
	}
}
