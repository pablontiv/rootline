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
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)

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
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)
	os.WriteFile(filepath.Join(dir, "c.md"), []byte("---\nestado: review\n---\n# C\n"), 0644)
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# No frontmatter\n"), 0644)
	os.WriteFile(filepath.Join(dir, "notes.md"), []byte("# Also no frontmatter\n"), 0644)

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
	os.WriteFile(filepath.Join(dir, "a.md"), []byte("---\nestado: draft\n---\n# A\n"), 0644)
	os.WriteFile(filepath.Join(dir, "b.md"), []byte("---\nestado: done\n---\n# B\n"), 0644)

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
		os.WriteFile(filepath.Join(dir, fmt.Sprintf("f%d.md", i)),
			[]byte(fmt.Sprintf("---\nestado: s%d\n---\n# F%d\n", i, i)), 0644)
	}
	os.WriteFile(filepath.Join(dir, "readme.md"), []byte("# No frontmatter\n"), 0644)

	out, err := runCmd(t, "init", dir, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if strings.Contains(out, "Warning") {
		t.Errorf("expected no warning at 10%% ratio, got: %s", out)
	}
}
