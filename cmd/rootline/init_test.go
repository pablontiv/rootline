package main

import (
	"fmt"
	"os"
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
	if !strings.Contains(out, "version: 1") {
		t.Errorf("expected version: 1 in output, got: %s", out)
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
	if !strings.Contains(string(content), "version: 1") {
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
	if !strings.Contains(out, "version: 1") {
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

	// Should create multiple .stem files.
	if !strings.Contains(out, "levels detected") {
		t.Errorf("expected hierarchical output message, got: %s", out)
	}

	// Root .stem should exist.
	rootStem := filepath.Join(dir, ".stem")
	content, err := os.ReadFile(rootStem)
	if err != nil {
		t.Fatalf("expected root .stem: %v", err)
	}
	if !strings.Contains(string(content), "version: 1") {
		t.Errorf("expected version: 1 in root .stem")
	}
	if !strings.Contains(string(content), "prefix: E") {
		t.Errorf("expected prefix: E in root .stem, got: %s", string(content))
	}

	// Child .stem in E01-infra/ should exist with F prefix.
	childStem := filepath.Join(dir, "E01-infra", ".stem")
	childContent, err := os.ReadFile(childStem)
	if err != nil {
		t.Fatalf("expected child .stem in E01-infra/: %v", err)
	}
	if !strings.Contains(string(childContent), "prefix: F") {
		t.Errorf("expected prefix: F in child .stem, got: %s", string(childContent))
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

	// Should show separators for each .stem file.
	if !strings.Contains(out, "# --- ") {
		t.Errorf("expected file separators in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "prefix: E") {
		t.Errorf("expected prefix: E in dry-run output, got: %s", out)
	}
	if !strings.Contains(out, "prefix: F") {
		t.Errorf("expected prefix: F in dry-run output, got: %s", out)
	}

	// No files should be written.
	if _, err := os.Stat(filepath.Join(dir, ".stem")); err == nil {
		t.Error("expected no .stem file written in dry-run mode")
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
	if !strings.Contains(string(content), "version: 1") {
		t.Errorf("expected version: 1, got: %s", string(content))
	}
}
