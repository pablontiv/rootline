# Init --template Implementation Plan

> **For agentic workers:** REQUIRED: Use superpowers:subagent-driven-development (if subagents available) or superpowers:executing-plans to implement this plan. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Add `--template <repo>` flag to `rootline init` that downloads `.stem` files from a remote GitHub repository.

**Architecture:** New `internal/templates/fetch.go` handles ref parsing, git clone with timeout, `.stem` discovery, YAML validation, and file copying. The `cmd/rootline/init.go` command gains a `--template` flag that branches to the fetch path instead of inference.

**Tech Stack:** `os/exec` for git clone, `context.WithTimeout` for 30s deadline, `gopkg.in/yaml.v3` for validation, `filepath.WalkDir` for .stem discovery.

**Spec:** `docs/superpowers/specs/2026-03-20-three-features-design.md` (Feature 2)

---

## Chunk 1: Template Fetching Engine

### Task 1: Implement ref parser

**Files:**
- Create: `internal/templates/fetch.go`
- Create: `internal/templates/fetch_test.go`

- [ ] **Step 1: Write failing tests for ParseRef**

```go
// internal/templates/fetch_test.go
package templates

import "testing"

func TestParseRef_ShortForm(t *testing.T) {
	owner, repo, tag := ParseRef("pablontiv/epic-tracking")
	if owner != "pablontiv" || repo != "epic-tracking" || tag != "" {
		t.Errorf("got owner=%q repo=%q tag=%q", owner, repo, tag)
	}
}

func TestParseRef_WithTag(t *testing.T) {
	owner, repo, tag := ParseRef("pablontiv/epic-tracking@v2")
	if owner != "pablontiv" || repo != "epic-tracking" || tag != "v2" {
		t.Errorf("got owner=%q repo=%q tag=%q", owner, repo, tag)
	}
}

func TestParseRef_FullGitHub(t *testing.T) {
	owner, repo, tag := ParseRef("github.com/org/repo")
	if owner != "org" || repo != "repo" || tag != "" {
		t.Errorf("got owner=%q repo=%q tag=%q", owner, repo, tag)
	}
}

func TestParseRef_FullGitHubWithTag(t *testing.T) {
	owner, repo, tag := ParseRef("github.com/org/repo@main")
	if owner != "org" || repo != "repo" || tag != "main" {
		t.Errorf("got owner=%q repo=%q tag=%q", owner, repo, tag)
	}
}

func TestParseRef_Invalid(t *testing.T) {
	owner, repo, _ := ParseRef("invalid")
	if owner != "" || repo != "" {
		t.Errorf("expected empty for invalid ref, got owner=%q repo=%q", owner, repo)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/templates/ -run TestParseRef -v`
Expected: FAIL with "no Go files"

- [ ] **Step 3: Implement ParseRef**

```go
// internal/templates/fetch.go
package templates

import (
	"strings"
)

// ParseRef parses a template reference into owner, repo, and optional tag.
// Accepted formats:
//   - "owner/repo"
//   - "owner/repo@tag"
//   - "github.com/owner/repo"
//   - "github.com/owner/repo@tag"
func ParseRef(ref string) (owner, repo, tag string) {
	// Strip github.com/ prefix if present.
	ref = strings.TrimPrefix(ref, "github.com/")

	// Split tag.
	if idx := strings.LastIndex(ref, "@"); idx != -1 {
		tag = ref[idx+1:]
		ref = ref[:idx]
	}

	parts := strings.SplitN(ref, "/", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return "", "", ""
	}

	return parts[0], parts[1], tag
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/templates/ -run TestParseRef -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/templates/fetch.go internal/templates/fetch_test.go
git commit -m "feat(templates): implement ref parser for owner/repo@tag format"
```

### Task 2: Implement FetchTemplate

**Files:**
- Modify: `internal/templates/fetch.go` (add FetchTemplate)
- Modify: `internal/templates/fetch_test.go` (add tests)

- [ ] **Step 1: Write failing test for FetchTemplate with local git repo fixture**

```go
func TestFetchTemplate_LocalRepo(t *testing.T) {
	// Create a fake "remote" repo with a .stem file.
	remote := t.TempDir()
	stemContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n"
	if err := os.WriteFile(filepath.Join(remote, ".stem"), []byte(stemContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Init git repo in remote.
	runGit(t, remote, "init")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "init")

	// Fetch into dest.
	dest := t.TempDir()
	files, err := FetchFromURL(remote, "", dest, false, false)
	if err != nil {
		t.Fatalf("FetchFromURL: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	data, err := os.ReadFile(filepath.Join(dest, ".stem"))
	if err != nil {
		t.Fatalf("reading copied .stem: %v", err)
	}
	if string(data) != stemContent {
		t.Errorf("content mismatch: got %q", string(data))
	}
}

func TestFetchTemplate_NoStemFiles(t *testing.T) {
	remote := t.TempDir()
	if err := os.WriteFile(filepath.Join(remote, "README.md"), []byte("# Hi\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, remote, "init")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "init")

	dest := t.TempDir()
	_, err := FetchFromURL(remote, "", dest, false, false)
	if err == nil {
		t.Fatal("expected error for repo with no .stem files")
	}
}

func TestFetchTemplate_ExistingNoForce(t *testing.T) {
	remote := t.TempDir()
	if err := os.WriteFile(filepath.Join(remote, ".stem"), []byte("version: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, remote, "init")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "init")

	dest := t.TempDir()
	// Pre-create .stem in dest.
	if err := os.WriteFile(filepath.Join(dest, ".stem"), []byte("existing\n"), 0644); err != nil {
		t.Fatal(err)
	}

	_, err := FetchFromURL(remote, "", dest, false, false)
	if err == nil {
		t.Fatal("expected error when .stem exists and force=false")
	}
}

func TestFetchTemplate_DryRun(t *testing.T) {
	remote := t.TempDir()
	if err := os.WriteFile(filepath.Join(remote, ".stem"), []byte("version: 2\n"), 0644); err != nil {
		t.Fatal(err)
	}
	runGit(t, remote, "init")
	runGit(t, remote, "add", ".")
	runGit(t, remote, "commit", "-m", "init")

	dest := t.TempDir()
	files, err := FetchFromURL(remote, "", dest, false, true)
	if err != nil {
		t.Fatalf("FetchFromURL dry-run: %v", err)
	}

	if len(files) != 1 {
		t.Fatalf("expected 1 file in dry-run, got %d", len(files))
	}

	// File should NOT exist in dest.
	if _, err := os.Stat(filepath.Join(dest, ".stem")); !os.IsNotExist(err) {
		t.Error("dry-run should not write files")
	}
}

func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...)
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}
```

Add imports at top of test file: `"os"`, `"os/exec"`, `"path/filepath"`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/templates/ -run "TestFetchTemplate" -v`
Expected: FAIL with "undefined: FetchFromURL"

- [ ] **Step 3: Implement FetchFromURL**

Add to `internal/templates/fetch.go`:

```go
import (
	"context"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
)

// FetchFromURL clones a git repo and copies .stem files to dest.
// If tag is non-empty, clones that specific branch/tag.
// Returns the list of relative paths of .stem files found.
func FetchFromURL(url, tag, dest string, force, dryRun bool) ([]string, error) {
	// Clone to temp dir with 30s timeout.
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	tmpDir, err := os.MkdirTemp("", "rootline-template-*")
	if err != nil {
		return nil, fmt.Errorf("creating temp dir: %w", err)
	}
	defer os.RemoveAll(tmpDir)

	args := []string{"clone", "--depth", "1"}
	if tag != "" {
		args = append(args, "--branch", tag)
	}
	args = append(args, url, tmpDir)

	cmd := exec.CommandContext(ctx, "git", args...)
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	if output, err := cmd.CombinedOutput(); err != nil {
		if ctx.Err() == context.DeadlineExceeded {
			return nil, fmt.Errorf("git clone timed out after 30s")
		}
		outStr := strings.TrimSpace(string(output))
		if strings.Contains(outStr, "Authentication") || strings.Contains(outStr, "could not read") {
			return nil, fmt.Errorf("authentication required for %s (private repositories are not supported)", url)
		}
		return nil, fmt.Errorf("git clone failed: %s", outStr)
	}

	// Find .stem files.
	var stemFiles []string
	err = filepath.WalkDir(tmpDir, func(path string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		if d.IsDir() {
			name := d.Name()
			if name == ".git" || (len(name) > 0 && name[0] == '.' && name != ".") {
				return filepath.SkipDir
			}
			return nil
		}
		if d.Name() == ".stem" {
			rel, _ := filepath.Rel(tmpDir, path)
			stemFiles = append(stemFiles, rel)
		}
		return nil
	})
	if err != nil {
		return nil, fmt.Errorf("walking cloned repo: %w", err)
	}

	if len(stemFiles) == 0 {
		return nil, fmt.Errorf("no .stem files found in %s", url)
	}

	// Validate YAML and copy.
	for _, rel := range stemFiles {
		srcPath := filepath.Join(tmpDir, rel)
		data, err := os.ReadFile(srcPath)
		if err != nil {
			return nil, fmt.Errorf("reading %s: %w", rel, err)
		}

		var parsed map[string]any
		if err := yaml.Unmarshal(data, &parsed); err != nil {
			return nil, fmt.Errorf("invalid .stem file %s: %w", rel, err)
		}

		if dryRun {
			continue
		}

		destPath := filepath.Join(dest, rel)
		if !force {
			if _, err := os.Stat(destPath); err == nil {
				return nil, fmt.Errorf("%s already exists (use --force to overwrite)", rel)
			}
		}

		// Ensure parent directory exists.
		if err := os.MkdirAll(filepath.Dir(destPath), 0755); err != nil {
			return nil, fmt.Errorf("creating directory for %s: %w", rel, err)
		}

		if err := os.WriteFile(destPath, data, 0644); err != nil {
			return nil, fmt.Errorf("writing %s: %w", rel, err)
		}
	}

	return stemFiles, nil
}

// FetchTemplate is the high-level function called by the CLI.
// It parses the ref, resolves the URL, and delegates to FetchFromURL.
func FetchTemplate(ref, dest string, force, dryRun bool) ([]string, error) {
	owner, repo, tag := ParseRef(ref)
	if owner == "" || repo == "" {
		return nil, fmt.Errorf("invalid template reference %q (expected owner/repo or github.com/owner/repo)", ref)
	}

	// Check git is available.
	if _, err := exec.LookPath("git"); err != nil {
		return nil, fmt.Errorf("git is required for --template")
	}

	url := fmt.Sprintf("https://github.com/%s/%s.git", owner, repo)
	return FetchFromURL(url, tag, dest, force, dryRun)
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/templates/ -run "TestFetchTemplate" -v`
Expected: PASS

- [ ] **Step 5: Commit**

```bash
git add internal/templates/fetch.go internal/templates/fetch_test.go
git commit -m "feat(templates): implement FetchFromURL with git clone, YAML validation, and copy"
```

## Chunk 2: CLI Integration

### Task 3: Wire --template flag into init command

**Files:**
- Modify: `cmd/rootline/init.go:17-20` (add initTemplate var)
- Modify: `cmd/rootline/init.go:30-34` (register flag)
- Modify: `cmd/rootline/init.go:36-68` (branch to template mode)

- [ ] **Step 1: Write failing test**

Add to `cmd/rootline/init_test.go` (create if needed):

```go
func TestInit_TemplateInvalidRef(t *testing.T) {
	dest := t.TempDir()
	cmd := newRootCmd()
	cmd.SetArgs([]string{"init", dest, "--template", "invalid"})
	err := cmd.Execute()
	if err == nil {
		t.Fatal("expected error for invalid template ref")
	}
}
```

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./cmd/rootline/ -run TestInit_TemplateInvalidRef -v`
Expected: FAIL (unknown flag --template)

- [ ] **Step 3: Add --template flag to init.go**

Add variable (line ~17):

```go
var (
	initDryRun  bool
	initForce   bool
	initTemplate string
)
```

Register flag in `init()`:

```go
initCmd.Flags().StringVar(&initTemplate, "template", "", "fetch .stem from remote repo (owner/repo[@tag])")
```

Add template branch at the start of `runInit`, before scanning (after absTarget resolution, line ~47):

```go
// Template mode: download .stem from remote repo.
if initTemplate != "" {
	files, err := templates.FetchTemplate(initTemplate, absTarget, initForce, initDryRun)
	if err != nil {
		return err
	}
	if initDryRun {
		for _, f := range files {
			fmt.Fprintf(cmd.OutOrStdout(), "Would copy: %s\n", f)
		}
		return nil
	}
	fmt.Fprintf(cmd.OutOrStdout(), "Installed %d .stem file(s) from %s\n", len(files), initTemplate)
	return nil
}
```

Add import: `"github.com/pablontiv/rootline/internal/templates"`

- [ ] **Step 4: Run test to verify it passes**

Run: `go test ./cmd/rootline/ -run TestInit_TemplateInvalidRef -v`
Expected: PASS

- [ ] **Step 5: Run full test suite**

Run: `go test ./internal/templates/ ./cmd/rootline/ -v -count=1`
Expected: All PASS

- [ ] **Step 6: Build check**

Run: `go build ./...`
Expected: Clean build

- [ ] **Step 7: Commit**

```bash
git add cmd/rootline/init.go
git commit -m "feat(init): add --template flag for remote .stem fetching"
```

### Task 4: Verification

- [ ] **Step 1: Run all tests**

Run: `go test ./... -race`
Expected: All PASS

- [ ] **Step 2: Check coverage**

Run: `go test ./internal/templates/ -coverprofile=c.out && go tool cover -func=c.out | tail -5`
Expected: Good coverage for new code

- [ ] **Step 3: Build check**

Run: `go build ./...`
Expected: Clean build
