package templates

import (
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

// runGit is a helper to run git commands in tests.
func runGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
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
