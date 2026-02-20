package main

import (
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
