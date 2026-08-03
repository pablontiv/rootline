package rules

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// downwardTree materialises a fixture from relative path -> content.
func downwardTree(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// relStemDirs reduces entries to their directories relative to root, so the
// assertions read as tree shapes rather than temp paths.
func relStemDirs(t *testing.T, root string, entries []StemEntry) []string {
	t.Helper()
	dirs := make([]string, 0, len(entries))
	for _, e := range entries {
		rel, err := filepath.Rel(root, filepath.Dir(e.Path))
		if err != nil {
			t.Fatal(err)
		}
		dirs = append(dirs, rel)
	}
	return dirs
}

func TestDownwardScan_CollectsEveryUnmarkedStemRootMostFirst(t *testing.T) {
	root := downwardTree(t, map[string]string{
		"wiki/.stem":          "version: 2\n",
		"wiki/entities/.stem": "version: 2\n",
		"zeta/.stem":          "version: 2\n",
	})

	entries, err := DownwardScan(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := relStemDirs(t, root, entries)
	want := []string{"wiki", filepath.Join("wiki", "entities"), "zeta"}
	if strings.Join(got, ",") != strings.Join(want, ",") {
		t.Errorf("dirs = %v, want %v (root-most first, then lexical)", got, want)
	}
}

// A tree with no .stem is ungoverned, not broken: the absence is the answer,
// so it must not surface as an error the caller has to special-case.
func TestDownwardScan_NoStemIsNotAnError(t *testing.T) {
	root := downwardTree(t, map[string]string{"a.md": "# a\n"})

	entries, err := DownwardScan(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %v, want none", relStemDirs(t, root, entries))
	}
}

// A .stem that declares root: true has already answered where its project
// starts. The scan must stop there rather than reopening the question, and must
// not report the stems it governs as undeclared.
func TestDownwardScan_StopsAtADeclaredBoundary(t *testing.T) {
	root := downwardTree(t, map[string]string{
		"wiki/.stem":          "version: 2\nroot: true\n",
		"wiki/entities/.stem": "version: 2\n",
		"zeta/.stem":          "version: 2\n",
	})

	entries, err := DownwardScan(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := relStemDirs(t, root, entries)
	if strings.Join(got, ",") != "zeta" {
		t.Errorf("dirs = %v, want [zeta] (the wiki subtree declares its own boundary)", got)
	}
}

func TestDownwardScan_ParseFailureStopsTheScan(t *testing.T) {
	root := downwardTree(t, map[string]string{
		"wiki/.stem":          "version: 2\nscope:\n  match: [[[unclosed\n",
		"wiki/entities/.stem": "version: 2\n",
	})

	entries, err := DownwardScan(root)
	if err == nil {
		t.Fatalf("an unparseable .stem must fail the scan, got entries %v", relStemDirs(t, root, entries))
	}
	if entries != nil {
		t.Errorf("entries = %v, want none on failure", relStemDirs(t, root, entries))
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected the parse failure, got: %v", err)
	}
}

func TestDownwardScan_RespectsStemignore(t *testing.T) {
	root := downwardTree(t, map[string]string{
		"docs/.stemignore":  "nested/.stem\n",
		"docs/.stem":        "version: 2\n",
		"docs/nested/.stem": "version: 2\n",
	})

	entries, err := DownwardScan(filepath.Join(root, "docs"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := relStemDirs(t, root, entries)
	if strings.Join(got, ",") != "docs" {
		t.Errorf("dirs = %v, want [docs] (docs/nested/.stem is ignored)", got)
	}
}

// .git holds no records and is never governed, so scanning into it would only
// find noise — the same exclusion index.Scan already makes.
func TestDownwardScan_SkipsGitDirectory(t *testing.T) {
	root := downwardTree(t, map[string]string{
		".git/modules/x/.stem": "version: 2\n",
		"wiki/.stem":           "version: 2\n",
	})

	entries, err := DownwardScan(root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := relStemDirs(t, root, entries)
	if strings.Join(got, ",") != "wiki" {
		t.Errorf("dirs = %v, want [wiki]", got)
	}
}

// The commands that run the scan are given whatever the user typed, which is
// often a file. Its directory is the tree in question.
func TestDownwardScan_FileTargetScansItsDirectory(t *testing.T) {
	root := downwardTree(t, map[string]string{
		"wiki/.stem": "version: 2\n",
		"wiki/a.md":  "# a\n",
	})

	entries, err := DownwardScan(filepath.Join(root, "wiki", "a.md"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	got := relStemDirs(t, root, entries)
	if strings.Join(got, ",") != "wiki" {
		t.Errorf("dirs = %v, want [wiki]", got)
	}
}

// A path that does not exist is the command's error to report, with the name
// the user typed. The scan has nothing to say about it.
func TestDownwardScan_MissingTargetIsNotTheScansError(t *testing.T) {
	root := downwardTree(t, map[string]string{"a.md": "# a\n"})

	entries, err := DownwardScan(filepath.Join(root, "nowhere"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) != 0 {
		t.Errorf("entries = %v, want none", relStemDirs(t, root, entries))
	}
}

func TestIsRealSchemaError(t *testing.T) {
	if IsRealSchemaError(nil) {
		t.Error("nil is not an error at all")
	}
	if IsRealSchemaError(ErrNoSchemaFound) {
		t.Error("a tree with no .stem is ungoverned, not broken")
	}
	if !IsRealSchemaError(errors.New("permission denied")) {
		t.Error("an IO failure is a real schema error")
	}
}
