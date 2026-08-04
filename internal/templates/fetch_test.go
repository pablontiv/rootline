package templates

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/gitenv"
)

// runGit is a helper to run git commands in tests.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec
	cmd.Dir = dir
	cmd.Env = append(gitenv.ClearedEnv(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

// runGitOutput runs git and returns the output as a string.
// Uses isolated environment to avoid confounding effects.
func runGitOutput(t *testing.T, dir string, args ...string) string {
	t.Helper()
	cmd := exec.Command("git", append([]string{"-C", dir}, args...)...) //nolint:gosec
	cmd.Env = gitenv.ClearedEnv()
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git -C %s %v failed: %v\n%s", dir, args, err, out)
	}
	return strings.TrimSpace(string(out))
}

// makeFixtureRepo creates a minimal git repo with the given .stem files.
// files maps relative path → content.
func makeFixtureRepo(t *testing.T, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	runGit(t, dir, "init")
	runGit(t, dir, "config", "user.email", "test@example.com")
	runGit(t, dir, "config", "user.name", "Test")

	for rel, content := range files {
		full := filepath.Join(dir, rel)
		if err := os.MkdirAll(filepath.Dir(full), 0755); err != nil {
			t.Fatalf("mkdir: %v", err)
		}
		if err := os.WriteFile(full, []byte(content), 0644); err != nil {
			t.Fatalf("write: %v", err)
		}
		runGit(t, dir, "add", rel)
	}
	runGit(t, dir, "commit", "-m", "init")
	return dir
}

// ── ParseRef tests ──────────────────────────────────────────────────────────

func TestParseRef_ShortForm(t *testing.T) {
	owner, repo, tag := ParseRef("owner/repo")
	if owner != "owner" || repo != "repo" || tag != "" {
		t.Fatalf("got owner=%q repo=%q tag=%q", owner, repo, tag)
	}
}

func TestParseRef_WithTag(t *testing.T) {
	owner, repo, tag := ParseRef("owner/repo@v1.2.3")
	if owner != "owner" || repo != "repo" || tag != "v1.2.3" {
		t.Fatalf("got owner=%q repo=%q tag=%q", owner, repo, tag)
	}
}

func TestParseRef_FullGitHub(t *testing.T) {
	owner, repo, tag := ParseRef("github.com/owner/repo")
	if owner != "owner" || repo != "repo" || tag != "" {
		t.Fatalf("got owner=%q repo=%q tag=%q", owner, repo, tag)
	}
}

func TestParseRef_FullGitHubWithTag(t *testing.T) {
	owner, repo, tag := ParseRef("github.com/owner/repo@v2.0.0")
	if owner != "owner" || repo != "repo" || tag != "v2.0.0" {
		t.Fatalf("got owner=%q repo=%q tag=%q", owner, repo, tag)
	}
}

func TestParseRef_Invalid(t *testing.T) {
	cases := []string{"", "noslash", "github.com/owner"}
	for _, c := range cases {
		owner, repo, tag := ParseRef(c)
		if owner != "" || repo != "" || tag != "" {
			t.Fatalf("ParseRef(%q): expected empty strings, got owner=%q repo=%q tag=%q", c, owner, repo, tag)
		}
	}
}

// ── FetchFromURL / FetchTemplate tests ─────────────────────────────────────

func TestFetchTemplate_LocalRepo(t *testing.T) {
	stemContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	repoDir := makeFixtureRepo(t, map[string]string{
		".stem": stemContent,
	})

	dest := t.TempDir()
	files, err := FetchFromURL("file://"+repoDir, "", dest, false, false)
	if err != nil {
		t.Fatalf("FetchFromURL: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	// Verify the file was actually written.
	destStem := filepath.Join(dest, ".stem")
	data, err := os.ReadFile(destStem)
	if err != nil {
		t.Fatalf("reading dest .stem: %v", err)
	}
	if string(data) != stemContent {
		t.Fatalf("content mismatch:\ngot:  %q\nwant: %q", string(data), stemContent)
	}
}

func TestFetchTemplate_SubdirStem(t *testing.T) {
	stemContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	repoDir := makeFixtureRepo(t, map[string]string{
		"sub/.stem": stemContent,
	})

	dest := t.TempDir()
	files, err := FetchFromURL("file://"+repoDir, "", dest, false, false)
	if err != nil {
		t.Fatalf("FetchFromURL: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	// Verify subdirectory structure preserved.
	destStem := filepath.Join(dest, "sub", ".stem")
	data, err := os.ReadFile(destStem)
	if err != nil {
		t.Fatalf("reading dest sub/.stem: %v", err)
	}
	if string(data) != stemContent {
		t.Fatalf("content mismatch:\ngot:  %q\nwant: %q", string(data), stemContent)
	}
}

func TestFetchTemplate_NoStemFiles(t *testing.T) {
	repoDir := makeFixtureRepo(t, map[string]string{
		"README.md": "# hello\n",
	})

	dest := t.TempDir()
	_, err := FetchFromURL("file://"+repoDir, "", dest, false, false)
	if err == nil {
		t.Fatal("expected error for repo with no .stem files")
	}
	if !strings.Contains(err.Error(), "no .stem files found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchTemplate_ExistingNoForce(t *testing.T) {
	stemContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	repoDir := makeFixtureRepo(t, map[string]string{
		".stem": stemContent,
	})

	dest := t.TempDir()
	// Pre-create the .stem in dest.
	if err := os.WriteFile(filepath.Join(dest, ".stem"), []byte("existing"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	_, err := FetchFromURL("file://"+repoDir, "", dest, false, false)
	if err == nil {
		t.Fatal("expected error when .stem exists and force=false")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchTemplate_ForceOverwrites(t *testing.T) {
	newContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	repoDir := makeFixtureRepo(t, map[string]string{
		".stem": newContent,
	})

	dest := t.TempDir()
	// Pre-create the .stem in dest.
	if err := os.WriteFile(filepath.Join(dest, ".stem"), []byte("old content"), 0644); err != nil {
		t.Fatalf("setup: %v", err)
	}

	files, err := FetchFromURL("file://"+repoDir, "", dest, true, false)
	if err != nil {
		t.Fatalf("FetchFromURL with force: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	data, err := os.ReadFile(filepath.Join(dest, ".stem"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if string(data) != newContent {
		t.Fatalf("expected new content, got: %q", string(data))
	}
}

func TestFetchTemplate_DryRun(t *testing.T) {
	stemContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	repoDir := makeFixtureRepo(t, map[string]string{
		".stem": stemContent,
	})

	dest := t.TempDir()
	files, err := FetchFromURL("file://"+repoDir, "", dest, false, true)
	if err != nil {
		t.Fatalf("FetchFromURL dryRun: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	// Verify nothing was actually written.
	destStem := filepath.Join(dest, ".stem")
	if _, err := os.Stat(destStem); !os.IsNotExist(err) {
		t.Fatal("dry run should not write files")
	}
}

func TestFetchTemplate_InvalidRef(t *testing.T) {
	dest := t.TempDir()
	_, err := FetchTemplate("invalid", dest, false, false)
	if err == nil {
		t.Fatal("expected error for invalid ref")
	}
	if !strings.Contains(err.Error(), "invalid template ref") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchTemplate_ValidRefNonexistentRepo(t *testing.T) {
	dest := t.TempDir()
	_, err := FetchTemplate("nonexistent-owner-xyz/nonexistent-repo-xyz", dest, false, false)
	if err == nil {
		t.Fatal("expected error for nonexistent repo")
	}
	// Should get a git clone error (not an invalid ref error).
	if strings.Contains(err.Error(), "invalid template ref") {
		t.Fatalf("expected clone error, got ref error: %v", err)
	}
}

func TestCopyFile(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")
	content := "hello world\n"
	if err := os.WriteFile(src, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	if err := copyFile(src, dst); err != nil {
		t.Fatalf("copyFile: %v", err)
	}
	data, err := os.ReadFile(dst)
	if err != nil {
		t.Fatalf("ReadFile dst: %v", err)
	}
	if string(data) != content {
		t.Errorf("got %q, want %q", string(data), content)
	}
}

func TestCopyFile_SrcNotExist(t *testing.T) {
	dir := t.TempDir()
	err := copyFile(filepath.Join(dir, "nope"), filepath.Join(dir, "dst"))
	if err == nil {
		t.Fatal("expected error for nonexistent src")
	}
}

func TestFetchTemplate_InvalidYAML(t *testing.T) {
	repoDir := makeFixtureRepo(t, map[string]string{
		".stem": "not: valid: yaml: [\n",
	})

	dest := t.TempDir()
	_, err := FetchFromURL("file://"+repoDir, "", dest, false, false)
	if err == nil {
		t.Fatal("expected error for invalid YAML .stem")
	}
	if !strings.Contains(err.Error(), "invalid .stem file") {
		t.Fatalf("unexpected error: %v", err)
	}
}

// ── Additional tests for uncovered branches ─────────────────────────────────

func TestFetchFromURL_WithTag(t *testing.T) {
	// Test cloning with a specific tag/branch.
	stemContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	repoDir := makeFixtureRepo(t, map[string]string{
		".stem": stemContent,
	})

	// Create a tag in the repo.
	runGit(t, repoDir, "tag", "v1.0.0")

	dest := t.TempDir()
	files, err := FetchFromURL("file://"+repoDir, "v1.0.0", dest, false, false)
	if err != nil {
		t.Fatalf("FetchFromURL with tag: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d", len(files))
	}

	// Verify the file was written.
	destStem := filepath.Join(dest, ".stem")
	data, err := os.ReadFile(destStem)
	if err != nil {
		t.Fatalf("reading dest .stem: %v", err)
	}
	if string(data) != stemContent {
		t.Fatalf("content mismatch")
	}
}

func TestFetchFromURL_BadTag(t *testing.T) {
	// Test cloning with a tag that doesn't exist.
	stemContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	repoDir := makeFixtureRepo(t, map[string]string{
		".stem": stemContent,
	})

	dest := t.TempDir()
	_, err := FetchFromURL("file://"+repoDir, "nonexistent-tag", dest, false, false)
	if err == nil {
		t.Fatal("expected error for nonexistent tag")
	}
	if !strings.Contains(err.Error(), "git clone failed") {
		t.Fatalf("expected git clone error, got: %v", err)
	}
}

func TestFetchFromURL_InvalidURL(t *testing.T) {
	// Test cloning from an invalid URL.
	dest := t.TempDir()
	_, err := FetchFromURL("file:///nonexistent/path/to/repo", "", dest, false, false)
	if err == nil {
		t.Fatal("expected error for invalid URL")
	}
	if !strings.Contains(err.Error(), "git clone failed") {
		t.Fatalf("expected git clone error, got: %v", err)
	}
}

func TestFetchFromURL_DestReadOnlyError(t *testing.T) {
	// Test when destination directory is read-only (copy fails).
	stemContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	repoDir := makeFixtureRepo(t, map[string]string{
		".stem": stemContent,
	})

	dest := t.TempDir()
	// Make dest read-only to trigger mkdir error.
	if err := os.Chmod(dest, 0555); err != nil { //nolint:gosec
		t.Fatalf("chmod: %v", err)
	}
	defer func() { _ = os.Chmod(dest, 0755) }() //nolint:gosec

	_, err := FetchFromURL("file://"+repoDir, "", dest, false, false)
	if err == nil {
		t.Fatal("expected error when dest is not writable")
	}
	// Should fail during mkdir or copy.
	if !strings.Contains(err.Error(), "creating directory") && !strings.Contains(err.Error(), "copying") {
		t.Fatalf("expected mkdir or copy error, got: %v", err)
	}
}

func TestFetchFromURL_MultipleStemFiles(t *testing.T) {
	// Test repo with multiple .stem files in different directories.
	stemContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	repoDir := makeFixtureRepo(t, map[string]string{
		".stem":          stemContent,
		"dir1/.stem":     stemContent,
		"dir2/.stem":     stemContent,
		"dir2/sub/.stem": stemContent,
	})

	dest := t.TempDir()
	files, err := FetchFromURL("file://"+repoDir, "", dest, false, false)
	if err != nil {
		t.Fatalf("FetchFromURL: %v", err)
	}
	if len(files) != 4 {
		t.Fatalf("expected 4 files, got %d: %v", len(files), files)
	}

	// Verify all files exist.
	expectedPaths := []string{".stem", "dir1/.stem", "dir2/.stem", "dir2/sub/.stem"}
	for _, p := range expectedPaths {
		fullPath := filepath.Join(dest, p)
		if _, err := os.Stat(fullPath); err != nil {
			t.Fatalf("expected file %s to exist: %v", p, err)
		}
	}
}

func TestFetchTemplate_WithGitHubURL(t *testing.T) {
	// Test that FetchTemplate resolves refs to GitHub URLs.
	// This will fail at clone time (no network), but we can verify the error
	// is a clone error, not a ref parsing error.
	dest := t.TempDir()
	_, err := FetchTemplate("owner/repo@v1.0.0", dest, false, false)
	if err == nil {
		t.Fatal("expected error")
	}
	// Should get a clone error (network/404), not a ref error.
	if strings.Contains(err.Error(), "invalid template ref") {
		t.Fatalf("expected clone error, not ref error: %v", err)
	}
}

func TestCopyFile_DestDirNotExist(t *testing.T) {
	// Test copyFile when dest dir doesn't exist (should fail, not auto-create).
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "nonexistent/dst.txt")
	content := "hello\n"
	if err := os.WriteFile(src, []byte(content), 0644); err != nil {
		t.Fatal(err)
	}
	// copyFile should fail because dir doesn't exist.
	err := copyFile(src, dst)
	if err == nil {
		t.Fatal("expected error when dest dir doesn't exist")
	}
}

func TestCopyFile_SrcOpenError(t *testing.T) {
	// Test copyFile when src cannot be read.
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	// Don't create src, so Open fails.
	err := copyFile(src, dst)
	if err == nil {
		t.Fatal("expected error when src doesn't exist")
	}
}

func TestCopyFile_DstCreateError(t *testing.T) {
	// Test copyFile when dest cannot be created (parent dir read-only).
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "ro_dir/dst.txt")

	if err := os.WriteFile(src, []byte("content"), 0644); err != nil {
		t.Fatal(err)
	}

	// Create read-only dir.
	roDir := filepath.Join(dir, "ro_dir")
	if err := os.Mkdir(roDir, 0555); err != nil { //nolint:gosec
		t.Fatal(err)
	}
	defer func() { _ = os.Chmod(roDir, 0755) }() //nolint:gosec

	err := copyFile(src, dst)
	if err == nil {
		t.Fatal("expected error when dest dir is read-only")
	}
}

func TestFetchFromURL_EmptyRepo(t *testing.T) {
	// Test repo with no commits (bare repo).
	repoDir := t.TempDir()
	runGit(t, repoDir, "init")
	// Don't add any files or commits.

	dest := t.TempDir()
	_, err := FetchFromURL("file://"+repoDir, "", dest, false, false)
	if err == nil {
		t.Fatal("expected error for empty repo with no .stem files")
	}
	if !strings.Contains(err.Error(), "no .stem files found") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestFetchTemplate_EmptyRef(t *testing.T) {
	// Test FetchTemplate with empty ref.
	dest := t.TempDir()
	_, err := FetchTemplate("", dest, false, false)
	if err == nil {
		t.Fatal("expected error for empty ref")
	}
	if !strings.Contains(err.Error(), "invalid template ref") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestCopyFile_IOCopyError(t *testing.T) {
	// Test copyFile when io.Copy fails (e.g., due to write error).
	// We simulate this by creating a source that's readable but then
	// making the destination unwritable after creation.
	dir := t.TempDir()
	src := filepath.Join(dir, "src.txt")
	dst := filepath.Join(dir, "dst.txt")

	srcContent := "test content for copy\n"
	if err := os.WriteFile(src, []byte(srcContent), 0644); err != nil {
		t.Fatal(err)
	}

	// Create dst as a directory instead of file, which will cause Create to fail.
	if err := os.Mkdir(dst, 0755); err != nil {
		t.Fatal(err)
	}

	err := copyFile(src, dst)
	if err == nil {
		t.Fatal("expected error when dst is a directory")
	}
}

// snapshotRepo captures the observable state of a repository: HEAD, commit count,
// tracked files and working-tree status. Comparing a snapshot taken before and after
// an operation proves whether that operation wrote into the repository.
func snapshotRepo(t *testing.T, dir string) string {
	t.Helper()
	return strings.Join([]string{
		"head=" + runGitOutput(t, dir, "rev-parse", "HEAD"),
		"commits=" + runGitOutput(t, dir, "rev-list", "--count", "HEAD"),
		"tracked=" + runGitOutput(t, dir, "ls-files"),
		"status=" + runGitOutput(t, dir, "status", "--porcelain"),
	}, "\n")
}

// TestFetchFromURL_HostileGitEnvironment proves the package ignores the caller's git
// environment. It points every repo-scoping git variable at a throwaway repository the
// test creates, runs the full fixture-build and clone path, and asserts both that the
// fetch succeeds and that the throwaway repository is untouched.
//
// Regression test for issue #81: an inherited GIT_DIR made the nested `git clone` fail
// and, worse, made the nested `git commit` land a real commit in the caller's repository.
func TestFetchFromURL_HostileGitEnvironment(t *testing.T) {
	// t.Setenv forbids t.Parallel, so this test runs sequentially.
	throwaway := t.TempDir()
	runGit(t, throwaway, "init")
	runGit(t, throwaway, "config", "user.email", "throwaway@example.com")
	runGit(t, throwaway, "config", "user.name", "Throwaway")
	if err := os.WriteFile(filepath.Join(throwaway, "file.txt"), []byte("initial\n"), 0644); err != nil {
		t.Fatalf("seeding throwaway repo: %v", err)
	}
	runGit(t, throwaway, "add", "file.txt")
	runGit(t, throwaway, "commit", "-m", "initial")

	before := snapshotRepo(t, throwaway)

	// Hostile environment: every repo-scoping variable points at the throwaway repo,
	// exactly as git exports them into hook processes.
	t.Setenv("GIT_DIR", filepath.Join(throwaway, ".git"))
	t.Setenv("GIT_WORK_TREE", throwaway)
	t.Setenv("GIT_INDEX_FILE", filepath.Join(throwaway, ".git", "index"))
	t.Setenv("GIT_OBJECT_DIRECTORY", filepath.Join(throwaway, ".git", "objects"))

	stemContent := "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  title:\n    type: string\n"
	fixture := makeFixtureRepo(t, map[string]string{".stem": stemContent})

	dest := t.TempDir()
	files, err := FetchFromURL("file://"+fixture, "", dest, false, false)
	if err != nil {
		t.Fatalf("FetchFromURL under a hostile git environment: %v", err)
	}
	if len(files) != 1 {
		t.Fatalf("expected 1 file, got %d: %v", len(files), files)
	}
	data, err := os.ReadFile(filepath.Join(dest, ".stem"))
	if err != nil {
		t.Fatalf("reading dest .stem: %v", err)
	}
	if string(data) != stemContent {
		t.Fatalf("content mismatch:\ngot:  %q\nwant: %q", string(data), stemContent)
	}

	if after := snapshotRepo(t, throwaway); after != before {
		t.Fatalf("the throwaway repository was written to.\nbefore:\n%s\n\nafter:\n%s", before, after)
	}
}
