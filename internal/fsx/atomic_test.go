package fsx

import (
	"errors"
	"io"
	"io/fs"
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

func TestStatInRootReportsExistingAndMissingTargets(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "docs", ".stem")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if _, err := StatInRoot(dir, target); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("StatInRoot missing error = %v, want fs.ErrNotExist", err)
	}
	if err := os.WriteFile(target, []byte("version: 2\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	info, err := StatInRoot(dir, target)
	if err != nil {
		t.Fatalf("StatInRoot existing: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
}

func TestWriteFileAtomicInRootCreatesAndPreservesExistingMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "docs", ".stem")
	if err := os.MkdirAll(filepath.Dir(target), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomicInRoot(dir, target, []byte("new\n"), 0o640); err != nil {
		t.Fatalf("WriteFileAtomicInRoot create: %v", err)
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
	if err := WriteFileAtomicInRoot(dir, target, []byte("replaced\n"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomicInRoot replace: %v", err)
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

func TestWriteFileAtomicInRootRejectsEscapingSymlinkParent(t *testing.T) {
	parent := t.TempDir()
	root := filepath.Join(parent, "root")
	outside := filepath.Join(parent, "outside")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	link := filepath.Join(root, "link")
	if err := os.Symlink(outside, link); err != nil {
		if errors.Is(err, os.ErrPermission) {
			t.Skipf("symlink unsupported: %v", err)
		}
		t.Fatal(err)
	}
	target := filepath.Join(link, ".stem")
	if err := WriteFileAtomicInRoot(root, target, []byte("escaped\n"), 0o644); err == nil {
		t.Fatal("WriteFileAtomicInRoot accepted escaping symlink parent")
	}
	if _, err := os.Stat(filepath.Join(outside, ".stem")); !os.IsNotExist(err) {
		t.Fatalf("outside target was created: %v", err)
	}
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

func TestWriteFileAtomicInRootCleansUpWhenRenameFails(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "target")
	if err := os.Mkdir(target, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(target, "keep"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := WriteFileAtomicInRoot(dir, target, []byte("new"), 0o644); err == nil {
		t.Fatal("WriteFileAtomicInRoot replaced non-empty directory")
	}
	assertNoStagingFiles(t, dir)
}

func TestRootedHelpersReportMissingRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	if _, err := StatInRoot(missing, filepath.Join(missing, ".stem")); err == nil {
		t.Fatal("StatInRoot accepted missing root")
	}
	if err := WriteFileAtomicInRoot(missing, filepath.Join(missing, ".stem"), []byte("new"), 0o644); err == nil {
		t.Fatal("WriteFileAtomicInRoot accepted missing root")
	}
}

func TestWriteFileAtomicInRootRejectsMissingParentAndLexicalEscape(t *testing.T) {
	dir := t.TempDir()
	if err := WriteFileAtomicInRoot(dir, filepath.Join(dir, "missing", ".stem"), []byte("new"), 0o644); err == nil {
		t.Fatal("WriteFileAtomicInRoot accepted missing parent")
	}
	if err := WriteFileAtomicInRoot(dir, filepath.Join(dir, "..", "escaped.stem"), []byte("new"), 0o644); err == nil {
		t.Fatal("WriteFileAtomicInRoot accepted lexical escape")
	}
}

func TestRootRelativePathAndRandomTempNameHelpers(t *testing.T) {
	dir := t.TempDir()
	rel, err := rootRelativePath(dir, filepath.Join(dir, "docs", ".stem"))
	if err != nil {
		t.Fatalf("rootRelativePath: %v", err)
	}
	if rel != filepath.Join("docs", ".stem") {
		t.Fatalf("rel = %q", rel)
	}
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
