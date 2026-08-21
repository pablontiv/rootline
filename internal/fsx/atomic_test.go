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

func TestWriteFileAtomicRejectsFileAsParentDuringStat(t *testing.T) {
	dir := t.TempDir()
	blocker := filepath.Join(dir, "blocker")
	if err := os.WriteFile(blocker, []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomic(filepath.Join(blocker, ".stem"), []byte("new"), 0o644); err == nil {
		t.Fatal("WriteFileAtomic accepted file as parent")
	}
}

func TestWriteFileAtomicRejectsMissingParent(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFileAtomic(filepath.Join(dir, "missing", ".stem"), []byte("new"), 0o644); err == nil {
		t.Fatal("WriteFileAtomic succeeded with a missing parent directory")
	}
}

func TestWriteFileAtomicRootCreatesAndPreservesExistingMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "docs", ".stem")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := writeFileAtomicRoot(root, filepath.Join("docs", ".stem"), 0o640, func(dst io.Writer) error {
		_, err := io.WriteString(dst, "new\n")
		return err
	}); err != nil {
		t.Fatalf("writeFileAtomicRoot create: %v", err)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("created mode = %o, want 640", got)
	}
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}
	if err := writeFileAtomicRoot(root, filepath.Join("docs", ".stem"), 0o644, func(dst io.Writer) error {
		_, err := io.WriteString(dst, "replaced\n")
		return err
	}); err != nil {
		t.Fatalf("writeFileAtomicRoot replace: %v", err)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "replaced\n" {
		t.Fatalf("content = %q", got)
	}
	info, err = os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("replaced mode = %o, want existing 600", got)
	}
	assertNoStagingFiles(t, filepath.Dir(target))
}

func TestWriteFileAtomicRootRejectsEscapingSymlinkParent(t *testing.T) {
	parent := t.TempDir()
	rootDir := filepath.Join(parent, "root")
	outside := filepath.Join(parent, "outside")
	if err := os.MkdirAll(rootDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(rootDir, "link")
	if err := os.Symlink(outside, link); err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("symlink unsupported: %v", err)
		}
		t.Fatal(err)
	}
	root, err := os.OpenRoot(rootDir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := writeFileAtomicRoot(root, filepath.Join("link", ".stem"), 0o644, func(dst io.Writer) error {
		_, err := io.WriteString(dst, "escaped\n")
		return err
	}); err == nil {
		t.Fatal("writeFileAtomicRoot accepted escaping symlink parent")
	}
	if _, err := os.Stat(filepath.Join(outside, ".stem")); !os.IsNotExist(err) {
		t.Fatalf("outside target was created: %v", err)
	}
}

func TestWriteFileAtomicRootCleansUpWhenRenameFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "keep"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	if err := writeFileAtomicRoot(root, "target", 0o644, func(dst io.Writer) error {
		_, err := io.WriteString(dst, "new")
		return err
	}); err == nil {
		t.Fatal("writeFileAtomicRoot replaced non-empty directory")
	}
	assertNoStagingFiles(t, dir)
}

func TestWriteFileAtomicRootLeavesOriginalAfterPartialWrite(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, ".stem")
	if err := os.WriteFile(target, []byte("original\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	wantErr := errors.New("rooted short write")
	err = writeFileAtomicRoot(root, ".stem", 0o644, func(dst io.Writer) error {
		if _, err := io.WriteString(dst, "partial"); err != nil {
			return err
		}
		return wantErr
	})
	if !errors.Is(err, wantErr) {
		t.Fatalf("writeFileAtomicRoot error = %v, want %v", err, wantErr)
	}
	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != "original\n" {
		t.Fatalf("content changed to %q", got)
	}
	assertNoStagingFiles(t, dir)
}

func TestRandomTempNameHelper(t *testing.T) {
	name, err := randomRootTempName()
	if err != nil {
		t.Fatalf("randomRootTempName: %v", err)
	}
	if !strings.HasPrefix(name, tempFilePrefix) || !strings.HasSuffix(name, ".tmp") {
		t.Fatalf("temp name = %q", name)
	}
}

func TestCreateRootTempFailsOnClosedRoot(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	if err := root.Close(); err != nil {
		t.Fatal(err)
	}
	if _, _, err := createRootTemp(root); err == nil {
		t.Fatal("createRootTemp accepted closed root")
	}
}

func TestCreateRootTempCreatesUniqueRootedFile(t *testing.T) {
	dir := t.TempDir()
	root, err := os.OpenRoot(dir)
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = root.Close() }()
	file, name, err := createRootTemp(root)
	if err != nil {
		t.Fatalf("createRootTemp: %v", err)
	}
	if _, err := file.WriteString("staged"); err != nil {
		t.Fatal(err)
	}
	if err := file.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(dir, name)); err != nil {
		t.Fatalf("stat temp: %v", err)
	}
	if err := root.Remove(name); err != nil {
		t.Fatalf("remove temp: %v", err)
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

func TestCopyFileAtomicLeavesDestinationUntouchedWhenSourceIsDirectory(t *testing.T) {
	dir := t.TempDir()
	src := filepath.Join(dir, "source-dir")
	if err := os.Mkdir(src, 0o755); err != nil {
		t.Fatal(err)
	}
	target := filepath.Join(dir, ".stem")
	if err := os.WriteFile(target, []byte("original\n"), 0o644); err != nil {
		t.Fatalf("seed target: %v", err)
	}
	if err := CopyFileAtomic(src, target); err == nil {
		t.Fatal("CopyFileAtomic accepted directory source")
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
