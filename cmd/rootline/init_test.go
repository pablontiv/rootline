package main

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"
)

func TestInitDryRun(t *testing.T) {
	dir := setupTestDir(t) // from commands_test.go
	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "version:") {
		t.Errorf("expected version: in output, got: %s", out)
	}
	if !strings.Contains(out, "schema:") {
		t.Errorf("expected schema: in output, got: %s", out)
	}
	if !strings.Contains(out, "estado:") {
		t.Errorf("expected estado field inferred, got: %s", out)
	}
}

func TestInitWritesFile(t *testing.T) {
	dir := t.TempDir()
	// Create markdown files (no .stem)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)

	out, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Created") {
		t.Errorf("expected 'Created' message, got: %s", out)
	}

	// Verify .stem was written
	stemPath := filepath.Join(dir, ".stem")
	content, err := os.ReadFile(stemPath)
	if err != nil {
		t.Fatalf("expected .stem file to exist: %v", err)
	}
	if !strings.Contains(string(content), "version:") {
		t.Errorf("expected valid .stem content, got: %s", string(content))
	}
}

func TestInitExistingStem(t *testing.T) {
	dir := setupTestDir(t) // has .stem already
	_, err := runCmd(t, "init", dir)
	if err == nil {
		t.Fatal("expected error for existing .stem")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestInitForce(t *testing.T) {
	dir := setupTestDir(t) // has .stem
	out, err := runCmd(t, "init", dir, "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Created") {
		t.Errorf("expected 'Created' message, got: %s", out)
	}
}

func TestInitNoFiles(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmd(t, "init", dir)
	if err == nil {
		t.Fatal("expected error for empty directory")
	}
	if !strings.Contains(err.Error(), "no markdown files") {
		t.Errorf("expected 'no markdown files' error, got: %v", err)
	}
}

func TestInitMixedContentWarning(t *testing.T) {
	dir := t.TempDir()
	// Create 3 files with frontmatter and 2 without (40% without > 20% threshold)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "c.md"), []byte("---\nestado: review\n---\n# C\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "readme.md"), []byte("# No frontmatter\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "notes.md"), []byte("# Also no frontmatter\n"), 0644)

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Warning") {
		t.Errorf("expected Warning for mixed content, got: %s", out)
	}
	if !strings.Contains(out, "2 of 5") {
		t.Errorf("expected '2 of 5' in warning, got: %s", out)
	}
	// Schema should still be generated
	if !strings.Contains(out, "version:") {
		t.Errorf("expected schema generated despite warning, got: %s", out)
	}
}

func TestInitCleanContentNoWarning(t *testing.T) {
	dir := t.TempDir()
	// All files have frontmatter
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "Warning") {
		t.Errorf("expected no warning when all files have frontmatter, got: %s", out)
	}
}

func TestInitMixedBelowThresholdNoWarning(t *testing.T) {
	dir := t.TempDir()
	// 1 of 10 files without frontmatter = 10% < 20% threshold
	for i := 0; i < 9; i++ {
		mustWriteFile(t, filepath.Join(dir, fmt.Sprintf("f%d.md", i)),
			[]byte(fmt.Sprintf("---\nestado: s%d\n---\n# F%d\n", i, i)), 0644)
	}
	mustWriteFile(t, filepath.Join(dir, "readme.md"), []byte("# No frontmatter\n"), 0644)

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "Warning") {
		t.Errorf("expected no warning at 10%% ratio, got: %s", out)
	}
}

func TestInitAutoHierarchy(t *testing.T) {
	dir := t.TempDir()

	// Create a 2-level hierarchy: E##-*/F##-*
	for _, epic := range []string{"E01-infra", "E02-platform"} {
		epicDir := filepath.Join(dir, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\nestado: Pending\n---\n# "+epic+"\n"), 0644)

		for _, feat := range []string{"F01-net", "F02-store"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\nestado: Pending\ntipo: feature\n---\n# "+feat+"\n"), 0644)
		}
	}

	out, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should create single .stem with levels.
	if !strings.Contains(out, "levels") {
		t.Errorf("expected hierarchical output message, got: %s", out)
	}

	// Root .stem should exist with match-based schema (v2 format).
	rootStem := filepath.Join(dir, ".stem")
	content, err := os.ReadFile(rootStem)
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "version: 2") {
		t.Errorf("expected version: 2 in root .stem, got: %s", s)
	}
	if !strings.Contains(s, "schema:") {
		t.Errorf("expected schema: section in root .stem, got: %s", s)
	}
	if !strings.Contains(s, "match:") {
		t.Errorf("expected match: annotations in root .stem, got: %s", s)
	}
	if !strings.Contains(s, "estado:") {
		t.Errorf("expected estado field in root .stem, got: %s", s)
	}

	// No child .stem files should be created.
	childStem := filepath.Join(dir, "E01-infra", ".stem")
	if _, err := os.Stat(childStem); err == nil {
		t.Error("expected no child .stem in E01-infra/ (match-based schema is in root .stem)")
	}
}

func TestInitAutoHierarchyDryRun(t *testing.T) {
	dir := t.TempDir()

	for _, epic := range []string{"E01-a", "E02-b"} {
		epicDir := filepath.Join(dir, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\nestado: Pending\n---\n# Epic\n"), 0644)

		for _, feat := range []string{"F01-x", "F02-y"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\nestado: Pending\n---\n# Feature\n"), 0644)
		}
	}

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should show single .stem with match-based schema (v2 format).
	if !strings.Contains(out, "# --- ") {
		t.Errorf("expected file separator in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "version: 2") {
		t.Errorf("expected version: 2 in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "schema:") {
		t.Errorf("expected schema: section in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "match:") {
		t.Errorf("expected match: annotations in dry-run output, got: %s", out)
	}

	// No files should be written.
	if _, err := os.Stat(filepath.Join(dir, ".stem")); err == nil {
		t.Error("expected no .stem file written in dry-run mode")
	}
}

func TestInitAutoHierarchy_GeneratesAggregate(t *testing.T) {
	dir := t.TempDir()

	// Create a 2-level hierarchy with enum field "estado" that has overlapping values
	// at both levels (required for fieldsCompatible to classify as root field).
	estados := []string{"Pending", "In Progress", "Completed"}
	for i, epic := range []string{"E01-infra", "E02-platform"} {
		epicDir := filepath.Join(dir, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\nestado: "+estados[i]+"\n---\n# "+epic+"\n"), 0644)

		for j, feat := range []string{"F01-net", "F02-store"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\nestado: "+estados[j]+"\n---\n# "+feat+"\n"), 0644)
		}
	}

	out, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should contain the note about auto-generated aggregate.
	if !strings.Contains(out, "Note: auto-generated aggregate for 'estado'") {
		t.Errorf("expected aggregate note in output, got: %s", out)
	}

	// Root .stem should have aggregate section.
	rootStem := filepath.Join(dir, ".stem")
	content, err := os.ReadFile(rootStem)
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	s := string(content)
	if !strings.Contains(s, "aggregate:") {
		t.Errorf("expected aggregate: section in root .stem, got:\n%s", s)
	}
	if !strings.Contains(s, "estado:") {
		t.Errorf("expected estado field in aggregate section, got:\n%s", s)
	}
	// Should use descendants-based expressions.
	if !strings.Contains(s, "descendants") {
		t.Errorf("expected descendants-based expression, got:\n%s", s)
	}
}

func TestInitAutoHierarchy_NoAggregateForNonEnum(t *testing.T) {
	dir := t.TempDir()

	// Create hierarchy where the shared field is string, not enum.
	for _, epic := range []string{"E01-a", "E02-b"} {
		epicDir := filepath.Join(dir, epic)
		_ = os.MkdirAll(epicDir, 0755)
		mustWriteFile(t, filepath.Join(epicDir, "README.md"),
			[]byte("---\ntitle: something\n---\n# "+epic+"\n"), 0644)

		for _, feat := range []string{"F01-x", "F02-y"} {
			featDir := filepath.Join(epicDir, feat)
			_ = os.MkdirAll(featDir, 0755)
			mustWriteFile(t, filepath.Join(featDir, "README.md"),
				[]byte("---\ntitle: other\n---\n# "+feat+"\n"), 0644)
		}
	}

	out, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should NOT contain aggregate note.
	if strings.Contains(out, "Note: auto-generated aggregate") {
		t.Errorf("expected no aggregate note for non-enum fields, got: %s", out)
	}

	// Root .stem should NOT have aggregate section.
	content, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	if strings.Contains(string(content), "aggregate:") {
		t.Errorf("expected no aggregate section for non-enum fields, got:\n%s", string(content))
	}
}

func TestInitTemplateInvalidRef(t *testing.T) {
	dir := t.TempDir()
	_, err := runCmd(t, "init", dir, "--template", "invalid")
	if err == nil {
		t.Fatal("expected error for invalid template ref")
	}
	if !strings.Contains(err.Error(), "invalid template ref") {
		t.Fatalf("unexpected error: %v", err)
	}
}

func TestInitTemplateDryRun(t *testing.T) {
	// Create a local fixture repo.
	remote := t.TempDir()
	mustWriteFile(t, filepath.Join(remote, ".stem"), []byte("version: 2\nscope:\n  match: \"*.md\"\n"), 0644)
	mustRunGit(t, remote, "init")
	mustRunGit(t, remote, "add", ".")
	mustRunGit(t, remote, "commit", "-m", "init")

	dir := t.TempDir()
	// Use FetchFromURL directly via the --template flag isn't possible with local paths,
	// but we can test the invalid ref error path from cmd level.
	_, err := runCmd(t, "init", dir, "--template", "nonexistent/repo-that-does-not-exist-12345")
	if err == nil {
		t.Fatal("expected error for nonexistent remote repo")
	}
}

func mustRunGit(t *testing.T, dir string, args ...string) {
	t.Helper()
	cmd := exec.Command("git", args...) //nolint:gosec
	cmd.Dir = dir
	cmd.Env = append(os.Environ(), "GIT_TERMINAL_PROMPT=0")
	out, err := cmd.CombinedOutput()
	if err != nil {
		t.Fatalf("git %v failed: %v\n%s", args, err, out)
	}
}

func TestInitStructuralInference(t *testing.T) {
	dir := t.TempDir()
	// Create 4 subdirectories all with README.md to trigger structural inference.
	for _, name := range []string{"A", "B", "C", "D"} {
		subdir := filepath.Join(dir, name)
		_ = os.MkdirAll(subdir, 0755)
		mustWriteFile(t, filepath.Join(subdir, "README.md"),
			[]byte("---\nestado: draft\n---\n# "+name+"\n"), 0644)
	}

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "structural:") {
		t.Errorf("expected structural: section in output, got: %s", out)
	}
	if !strings.Contains(out, "require_index:") {
		t.Errorf("expected require_index in output, got: %s", out)
	}
}

func TestInitFlatFallback(t *testing.T) {
	dir := t.TempDir()

	// Create flat directory without naming patterns.
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)

	out, err := runCmd(t, "init", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Should fall back to flat mode.
	if !strings.Contains(out, "Created") {
		t.Errorf("expected 'Created' message, got: %s", out)
	}
	if strings.Contains(out, "levels detected") {
		t.Errorf("expected flat mode (no levels), got: %s", out)
	}

	// Single .stem file should exist.
	content, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatalf("expected .stem file: %v", err)
	}
	if !strings.Contains(string(content), "version:") {
		t.Errorf("expected version:, got: %s", string(content))
	}
}
