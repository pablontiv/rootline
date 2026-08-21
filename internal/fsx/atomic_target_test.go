package fsx

import (
	"errors"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
)

func TestAtomicTargetFollowsInternalAliasOnce(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup requires Unix-compatible semantics")
	}
	root := t.TempDir()
	original := filepath.Join(root, "original")
	redirected := filepath.Join(root, "redirected")
	for _, dir := range []string{original, redirected} {
		if err := os.MkdirAll(dir, 0o755); err != nil {
			t.Fatal(err)
		}
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink("original", alias); err != nil {
		t.Fatal(err)
	}

	target, err := ResolveAtomicTarget(root, filepath.Join(alias, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = target.Close() }()
	wantDir, err := filepath.EvalSymlinks(original)
	if err != nil {
		t.Fatal(err)
	}
	if got, want := target.PhysicalPath(), filepath.Join(wantDir, ".stem"); got != want {
		t.Fatalf("PhysicalPath() = %q, want %q", got, want)
	}

	if err := os.Remove(alias); err != nil {
		t.Fatal(err)
	}
	if err := os.Symlink("redirected", alias); err != nil {
		t.Fatal(err)
	}
	if err := target.WriteFileAtomic([]byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if _, err := os.Stat(filepath.Join(redirected, ".stem")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("redirected target exists: %v", err)
	}
	if got, err := os.ReadFile(filepath.Join(original, ".stem")); err != nil || string(got) != "version: 2\n" {
		t.Fatalf("original target = %q, %v", got, err)
	}
}

func TestResolveAtomicTargetRejectsParentSubstitutedOutsideAllowedRootBeforeAcquisition(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup requires Unix-compatible semantics")
	}
	base := t.TempDir()
	root := filepath.Join(base, "root")
	inside := filepath.Join(root, "docs")
	outsideRoot := filepath.Join(base, "outside-root")
	outsideDocs := filepath.Join(outsideRoot, "docs")
	if err := os.MkdirAll(inside, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(outsideDocs, 0o755); err != nil {
		t.Fatal(err)
	}
	movedRoot := filepath.Join(base, "moved-root")
	wantParent, err := filepath.EvalSymlinks(inside)
	if err != nil {
		t.Fatal(err)
	}

	afterAtomicTargetParentResolved = func(_, physicalParent, _ string) error {
		if physicalParent != wantParent {
			return fmt.Errorf("physical parent = %q, want %q", physicalParent, wantParent)
		}
		if err := os.Rename(root, movedRoot); err != nil {
			return err
		}
		return os.Rename(outsideRoot, root)
	}
	defer func() { afterAtomicTargetParentResolved = nil }()

	target, err := ResolveAtomicTarget(root, filepath.Join(inside, ".stem"))
	if err == nil {
		defer func() { _ = target.Close() }()
		err = target.WriteFileAtomic([]byte("version: 2\n"), 0o644)
	}
	if err == nil {
		t.Fatal("substituted external parent was accepted")
	}
	if _, err := os.Stat(filepath.Join(inside, ".stem")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("outside .stem was created: %v", err)
	}
}

func TestResolveAtomicTargetRejectsExistingPhysicalEscape(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup requires Unix-compatible semantics")
	}
	root := t.TempDir()
	outside := filepath.Join(t.TempDir(), "outside")
	if err := os.MkdirAll(outside, 0o755); err != nil {
		t.Fatal(err)
	}
	alias := filepath.Join(root, "alias")
	if err := os.Symlink(outside, alias); err != nil {
		t.Fatal(err)
	}
	_, err := ResolveAtomicTarget(root, filepath.Join(alias, ".stem"))
	if err == nil {
		t.Fatal("ResolveAtomicTarget accepted escaping symlink parent")
	}
	if !strings.Contains(err.Error(), "escapes root") {
		t.Fatalf("ResolveAtomicTarget error = %v", err)
	}
	if _, err := os.Stat(filepath.Join(outside, ".stem")); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("outside target was created: %v", err)
	}
}

func TestAtomicTargetRejectsReplacedPhysicalParent(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "parent")
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := ResolveAtomicTarget(root, filepath.Join(parent, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = target.Close() }()
	moved := filepath.Join(root, "moved")
	if err := os.Rename(parent, moved); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := target.WriteFileAtomic([]byte("new"), 0o644); err == nil || !strings.Contains(err.Error(), "physical parent changed") {
		t.Fatalf("WriteFileAtomic error = %v", err)
	}
	for _, path := range []string{filepath.Join(parent, ".stem"), filepath.Join(moved, ".stem")} {
		if _, err := os.Stat(path); !errors.Is(err, fs.ErrNotExist) {
			t.Fatalf("unexpected write at %s: %v", path, err)
		}
	}
}

func TestAtomicTargetIdentityOperations(t *testing.T) {
	root := t.TempDir()
	docs := filepath.Join(root, "docs")
	other := filepath.Join(root, "other")
	if err := os.MkdirAll(docs, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.MkdirAll(other, 0o755); err != nil {
		t.Fatal(err)
	}
	first, err := ResolveAtomicTarget(root, filepath.Join(docs, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = first.Close() }()
	second, err := ResolveAtomicTarget(root, filepath.Join(docs, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = second.Close() }()
	otherParent, err := ResolveAtomicTarget(root, filepath.Join(other, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = otherParent.Close() }()
	otherName, err := ResolveAtomicTarget(root, filepath.Join(docs, "schema.stem"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = otherName.Close() }()

	if !first.SameTarget(second) {
		t.Fatal("same parent/name target was not matched by identity")
	}
	if first.SameTarget(otherParent) {
		t.Fatal("different parent matched as same target")
	}
	if first.SameTarget(otherName) {
		t.Fatal("different basename matched as same target")
	}
	if first.SameTarget(nil) {
		t.Fatal("nil target matched")
	}
	if got := first.TargetName(); got != ".stem" {
		t.Fatalf("TargetName() = %q, want .stem", got)
	}
	matches, err := first.ParentMatchesDir(docs)
	if err != nil || !matches {
		t.Fatalf("ParentMatchesDir(docs) = %v, %v; want true, nil", matches, err)
	}
	matches, err = first.ParentMatchesDir(other)
	if err != nil || matches {
		t.Fatalf("ParentMatchesDir(other) = %v, %v; want false, nil", matches, err)
	}
	if _, err := first.ParentMatchesDir(filepath.Join(root, "missing")); err == nil {
		t.Fatal("ParentMatchesDir accepted missing directory")
	}
	matches, err = first.MatchesExistingTargetPath(filepath.Join(docs, ".stem"))
	if err != nil || !matches {
		t.Fatalf("MatchesExistingTargetPath(docs/.stem) = %v, %v; want true, nil", matches, err)
	}
	matches, err = first.MatchesExistingTargetPath(filepath.Join(docs, "schema.stem"))
	if err != nil || matches {
		t.Fatalf("MatchesExistingTargetPath(schema.stem) = %v, %v; want false, nil", matches, err)
	}
	if ok, err := (*AtomicTarget)(nil).ParentMatchesDir(docs); err != nil || ok {
		t.Fatalf("nil ParentMatchesDir = %v, %v; want false, nil", ok, err)
	}
	if ok, err := (*AtomicTarget)(nil).MatchesExistingTargetPath(filepath.Join(docs, ".stem")); err != nil || ok {
		t.Fatalf("nil MatchesExistingTargetPath = %v, %v; want false, nil", ok, err)
	}
}

func TestAtomicTargetStatCreateCandidateAndModes(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "docs")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := ResolveAtomicTarget(root, filepath.Join(parent, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = target.Close() }()
	if _, err := target.Stat(); !errors.Is(err, fs.ErrNotExist) {
		t.Fatalf("Stat() error = %v", err)
	}
	if err := os.WriteFile(filepath.Join(parent, ".stem"), []byte("old\n"), 0o600); err != nil {
		t.Fatal(err)
	}
	if err := target.WriteFileAtomic([]byte("new\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	info, err := os.Stat(filepath.Join(parent, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("mode = %o, want 600", got)
	}
}

func TestAtomicTargetCloseTwice(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "docs")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := ResolveAtomicTarget(root, filepath.Join(parent, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	if got := target.LogicalPath(); got != filepath.Clean(filepath.Join(parent, ".stem")) {
		t.Fatalf("LogicalPath() = %q", got)
	}
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}
}

func TestAtomicTargetStatAfterCloseReportsError(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "docs")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := ResolveAtomicTarget(root, filepath.Join(parent, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	if err := target.Close(); err != nil {
		t.Fatal(err)
	}
	if _, err := target.Stat(); err == nil {
		t.Fatal("Stat() succeeded after Close")
	}
}

func TestAtomicTargetWriteFileAtomicReportsMissingPhysicalParent(t *testing.T) {
	root := t.TempDir()
	parent := filepath.Join(root, "docs")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := ResolveAtomicTarget(root, filepath.Join(parent, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = target.Close() }()
	if err := os.RemoveAll(parent); err != nil {
		t.Fatal(err)
	}
	if err := target.WriteFileAtomic([]byte("x"), 0o644); err == nil || !strings.Contains(err.Error(), "stating physical parent") {
		t.Fatalf("WriteFileAtomic error = %v", err)
	}
}

func TestResolveAtomicTargetRejectsMissingAllowedRoot(t *testing.T) {
	missing := filepath.Join(t.TempDir(), "missing")
	_, err := ResolveAtomicTarget(missing, filepath.Join(missing, ".stem"))
	if err == nil {
		t.Fatal("ResolveAtomicTarget accepted missing allowed root")
	}
	if !strings.Contains(err.Error(), "opening root "+missing) {
		t.Fatalf("ResolveAtomicTarget error = %v", err)
	}
}

func TestResolveAtomicTargetHandlesSymlinkAllowedRoot(t *testing.T) {
	if runtime.GOOS == "windows" {
		t.Skip("symlink setup requires Unix-compatible semantics")
	}
	root := t.TempDir()
	link := filepath.Join(t.TempDir(), "root-link")
	if err := os.Symlink(root, link); err != nil {
		t.Fatal(err)
	}
	parent := filepath.Join(root, "docs")
	if err := os.MkdirAll(parent, 0o755); err != nil {
		t.Fatal(err)
	}
	target, err := ResolveAtomicTarget(link, filepath.Join(parent, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	defer func() { _ = target.Close() }()
	wantRoot, err := filepath.EvalSymlinks(root)
	if err != nil {
		t.Fatal(err)
	}
	if got := target.PhysicalPath(); got != filepath.Join(wantRoot, "docs", ".stem") {
		t.Fatalf("PhysicalPath() = %q", got)
	}
}

func TestResolveAtomicTargetRejectsMissingParent(t *testing.T) {
	root := t.TempDir()
	logicalTarget := filepath.Join(root, "missing", ".stem")
	_, err := ResolveAtomicTarget(root, logicalTarget)
	if err == nil {
		t.Fatal("ResolveAtomicTarget accepted missing parent")
	}
	if !strings.Contains(err.Error(), "stat "+logicalTarget) {
		t.Fatalf("ResolveAtomicTarget error = %v", err)
	}
}
