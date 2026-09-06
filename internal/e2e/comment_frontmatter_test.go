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
)

// Adding the first field must not discard metadata comments that have no key.
func TestCommentOnlyFrontmatterCLI(t *testing.T) {
	bin := filepath.Join(t.TempDir(), "rootline")
	if runtime.GOOS == "windows" {
		bin += ".exe"
	}
	build := exec.Command("go", "build", "-o", bin, "./cmd/rootline") //nolint:gosec // builds this checkout into a test-owned directory
	build.Dir = filepath.Join("..", "..")
	if output, err := build.CombinedOutput(); err != nil {
		t.Fatalf("build dev CLI: %v\n%s", err, output)
	}
	for _, flow := range []string{"fix", "fix-all", "repair", "set"} {
		t.Run(flow, func(t *testing.T) {
			root := t.TempDir() // Product operation must not require Git.
			body := "## QA body\n\nKeep café and <literal> unchanged.\n\n---\n\nTail\n"
			original := "---\n# Keep this metadata comment.\n\n  # Keep this second note.\n---\n" + body
			files := map[string]string{
				".stem":         "version: 2\nroot: true\nscope:\n  match: '*.md'\nschema:\n  status:\n    type: string\n    required: true\n    default: ready\n",
				"probe.md":      original,
				"unrelated.txt": "Leave this file alone.\n",
			}
			stamp := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
			write := func(name, content string) {
				t.Helper()
				path := filepath.Join(root, name)
				if err := os.WriteFile(path, []byte(content), 0o600); err != nil {
					t.Fatal(err)
				}
				if err := os.Chtimes(path, stamp, stamp); err != nil {
					t.Fatal(err)
				}
			}
			for name, content := range files {
				write(name, content)
			}
			run := func(args ...string) string {
				t.Helper()
				cmd := exec.Command(bin, args...) //nolint:gosec // locally compiled CLI and test-owned fixture
				cmd.Dir = root
				var stdout, stderr bytes.Buffer
				cmd.Stdout, cmd.Stderr = &stdout, &stderr
				if err := cmd.Run(); err != nil || stderr.Len() != 0 {
					t.Fatalf("%v: %v\nstdout: %s\nstderr: %s", args, err, &stdout, &stderr)
				}
				return stdout.String()
			}
			assertFiles := func() {
				t.Helper()
				for name, want := range files {
					path := filepath.Join(root, name)
					if got := string(mustReadE2EFile(t, path)); got != want {
						t.Fatalf("%s changed unexpectedly: got %q, want %q", name, got, want)
					}
					info, err := os.Stat(path)
					if err != nil {
						t.Fatal(err)
					}
					if !info.ModTime().Equal(stamp) || (runtime.GOOS != "windows" && info.Mode().Perm() != 0o600) {
						t.Fatalf("%s was rewritten or its permissions changed", name)
					}
				}
			}
			args := []string{"fix", "probe.md"}
			switch flow {
			case "fix-all":
				args = []string{"fix", "--all", ".", "-o", "json"}
			case "repair":
				report := run("fix", "--all", ".", "--dry-run", "-o", "json")
				assertFiles()
				files["report.json"] = report
				write("report.json", report)
				args = []string{"repair", "apply", "--root", ".", "--report", "report.json", "-o", "json"}
			case "set":
				args = []string{"set", "probe.md", "status=ready"}
			}
			run(append(append([]string{}, args...), "--dry-run")...)
			assertFiles()
			output := run(args...)
			if flow == "fix-all" || flow == "repair" {
				var envelope struct {
					Version int `json:"version"`
				}
				if err := json.Unmarshal([]byte(output), &envelope); err != nil || envelope.Version != 1 {
					t.Fatalf("invalid mutation envelope: %v\n%s", err, output)
				}
			}
			result := string(mustReadE2EFile(t, filepath.Join(root, "probe.md")))
			parts := strings.SplitN(result, "---\n", 3)
			if len(parts) != 3 || parts[0] != "" || parts[2] != body {
				t.Fatalf("Markdown body changed: %q", result)
			}
			first, second := "# Keep this metadata comment.", "# Keep this second note."
			if strings.Count(parts[1], first) != 1 || strings.Count(parts[1], second) != 1 || strings.Index(parts[1], first) >= strings.Index(parts[1], second) {
				t.Fatalf("metadata comments lost, duplicated or reordered: %q", result)
			}
			if !strings.Contains(parts[1], "status: ready\n") {
				t.Fatalf("requested/default field missing: %q", result)
			}
			files["probe.md"] = result
			// A no-op rewrite must be observable even on coarse timestamp filesystems.
			if err := os.Chtimes(filepath.Join(root, "probe.md"), stamp, stamp); err != nil {
				t.Fatal(err)
			}
			run("validate", "--all", ".", "-o", "json")
			assertFiles()
			// set performs explicit writes even for equal assignments; the
			// read-only replay contract here belongs to fix and repair.
			if flow == "set" {
				return
			}
			run(args...)
			assertFiles()
		})
	}
}
