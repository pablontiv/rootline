package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestNewCreatesFile(t *testing.T) {
	dir := setupTestDir(t) // has .stem with estado (enum) and tipo (string)
	target := filepath.Join(dir, "newdoc.md")

	out, err := runCmd(t, "new", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Created") {
		t.Errorf("expected 'Created' message, got: %s", out)
	}

	content := mustReadFile(t, target)
	s := string(content)
	if !strings.Contains(s, "---") {
		t.Error("expected frontmatter delimiters")
	}
	if !strings.Contains(s, "estado:") {
		t.Error("expected estado field in frontmatter")
	}
}

func TestNewRequiredFields(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "test-doc.md")

	_, err := runCmd(t, "new", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := mustReadFile(t, target)
	s := string(content)
	// estado is required+enum, should appear with first value
	if !strings.Contains(s, "estado: Completado") && !strings.Contains(s, "estado: Pending") {
		t.Errorf("expected estado with enum value, got: %s", s)
	}
}

func TestNewEnumComments(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "test-doc.md")

	_, err := runCmd(t, "new", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := mustReadFile(t, target)
	s := string(content)
	if !strings.Contains(s, "# [") {
		t.Errorf("expected enum value comment, got: %s", s)
	}
}

func TestNewExistingFile(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "doc1.md") // already exists

	_, err := runCmd(t, "new", target)
	if err == nil {
		t.Fatal("expected error for existing file")
	}
	if !strings.Contains(err.Error(), "already exists") {
		t.Errorf("expected 'already exists' error, got: %v", err)
	}
}

func TestNewForce(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "doc1.md")

	out, err := runCmd(t, "new", target, "--force")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !strings.Contains(out, "Created") {
		t.Errorf("expected 'Created' message, got: %s", out)
	}
}

func TestNewNoStem(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "doc.md")

	_, err := runCmd(t, "new", target)
	if err == nil {
		t.Fatal("expected error for missing .stem")
	}
}

func TestNewDryRun(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "preview.md")

	out, err := runCmd(t, "new", target, "--dry-run")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// Should print content to stdout
	if !strings.Contains(out, "---") {
		t.Error("expected frontmatter in dry-run output")
	}
	if !strings.Contains(out, "estado:") {
		t.Error("expected estado field in dry-run output")
	}
	// File should NOT exist on disk
	if _, err := os.Stat(target); err == nil {
		t.Error("dry-run should not create file on disk")
	}
}

func TestNewTitleFromFilename(t *testing.T) {
	dir := setupTestDir(t)
	target := filepath.Join(dir, "my-test-doc.md")

	_, err := runCmd(t, "new", target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	content := mustReadFile(t, target)
	s := string(content)
	if !strings.Contains(s, "# My Test Doc") {
		t.Errorf("expected title from filename, got: %s", s)
	}
}
