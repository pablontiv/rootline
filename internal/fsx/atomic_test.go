package fsx

import (
	"errors"
	"io"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestWriteFileAtomicLeavesOriginalIntactAfterPartialWrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".stem")
	if err := os.WriteFile(target, []byte("original\n"), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	wantErr := errors.New("injected short write")
	err := writeFileAtomic(target, 0o644, func(dst io.Writer) error {
		if _, err := io.WriteString(dst, "partial"); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("writeFileAtomic error = %v, want %v", err, wantErr)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "original\n" {
		t.Fatalf("target content = %q, want original content", got)
	}
	assertNoStagingFiles(t, dir)
}

func TestWriteFileAtomicReplacesContentAndPreservesExistingMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".stem")
	if err := os.WriteFile(target, []byte("old\n"), 0o600); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	if err := WriteFileAtomic(target, []byte("new\n"), 0o640); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "new\n" {
		t.Fatalf("target content = %q, want %q", got, "new\n")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("target mode = %o, want existing mode 600", got)
	}
	assertNoStagingFiles(t, dir)
}

func TestWriteFileAtomicCreatesFileWithRequestedMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".stem")
	if err := WriteFileAtomic(target, []byte("new\n"), 0o640); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("target mode = %o, want requested mode 640", got)
	}
}

func TestWriteFileAtomicCleansUpWhenRenameFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatalf("create target directory: %v", err)
	}
	if err := os.WriteFile(filepath.Join(target, "keep"), []byte("x"), 0o644); err != nil {
		t.Fatalf("seed target directory: %v", err)
	}
	if err := WriteFileAtomic(target, []byte("new"), 0o644); err == nil {
		t.Fatal("WriteFileAtomic succeeded replacing a non-empty directory")
	}
	assertNoStagingFiles(t, dir)
}

func TestWriteFileAtomicRejectsMissingParent(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFileAtomic(filepath.Join(dir, "missing", ".stem"), []byte("new"), 0o644); err == nil {
		t.Fatal("WriteFileAtomic succeeded with a missing parent directory")
	}
}

func TestCopyFileAtomicStreamsContentAndPreservesExistingMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.stem")
	target := filepath.Join(dir, ".stem")
	content := strings.Repeat("schema:\n", 4096)
	if err := os.WriteFile(src, []byte(content), 0o640); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := os.WriteFile(target, []byte("old\n"), 0o600); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	if err := CopyFileAtomic(src, target); err != nil {
		t.Fatalf("CopyFileAtomic: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != content {
		t.Fatal("copied content differs from source")
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("target mode = %o, want existing mode 600", got)
	}
	assertNoStagingFiles(t, dir)
}

func TestCopyFileAtomicCreatesFileWithSourceMode(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source.stem")
	target := filepath.Join(dir, ".stem")
	if err := os.WriteFile(src, []byte("schema:\n"), 0o640); err != nil {
		t.Fatalf("write source: %v", err)
	}
	if err := CopyFileAtomic(src, target); err != nil {
		t.Fatalf("CopyFileAtomic: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat target: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("target mode = %o, want source mode 640", got)
	}
}

func TestCopyFileAtomicLeavesDestinationUntouchedWhenSourceCannotOpen(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".stem")
	if err := os.WriteFile(target, []byte("original\n"), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}

	if err := CopyFileAtomic(filepath.Join(dir, "missing.stem"), target); err == nil {
		t.Fatal("CopyFileAtomic succeeded with a missing source")
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("read target: %v", err)
	}
	if string(got) != "original\n" {
		t.Fatalf("target content = %q, want original content", got)
	}
	assertNoStagingFiles(t, dir)
}

func assertNoStagingFiles(t *testing.T, dir string) {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("read directory: %v", err)
	}
	for _, entry := range entries {
		if strings.HasPrefix(entry.Name(), tempFilePrefix) {
			t.Errorf("staging file left behind: %s", entry.Name())
		}
	}
}
