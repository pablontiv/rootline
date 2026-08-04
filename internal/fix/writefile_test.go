package fix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// tempFilesIn reports any leftover staging files in dir. WriteFileAtomic must
// never leave one behind, on success or on failure: a stray .rootline-*.tmp in
// a governed directory is a file the scanner would later try to make sense of.
func tempFilesIn(t *testing.T, dir string) []string {
	t.Helper()

	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatalf("reading %s: %v", dir, err)
	}
	var leftovers []string
	for _, e := range entries {
		if strings.HasPrefix(e.Name(), tempFilePrefix) {
			leftovers = append(leftovers, e.Name())
		}
	}
	return leftovers
}

func TestWriteFileAtomicReplacesContent(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(target, []byte("old"), 0o644); err != nil {
		t.Fatalf("seeding target: %v", err)
	}

	if err := WriteFileAtomic(target, []byte("new"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(got) != "new" {
		t.Errorf("content = %q, want %q", got, "new")
	}
	if leftovers := tempFilesIn(t, dir); leftovers != nil {
		t.Errorf("staging files left behind: %v", leftovers)
	}
}

func TestWriteFileAtomicCreatesMissingFile(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "new.md")

	if err := WriteFileAtomic(target, []byte("body"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(got) != "body" {
		t.Errorf("content = %q, want %q", got, "body")
	}
}

// TestWriteFileAtomicAppliesRequestedMode pins the mode on the target rather
// than whatever os.CreateTemp chose for the staging file, which is 0600.
func TestWriteFileAtomicAppliesRequestedMode(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "doc.md")

	if err := WriteFileAtomic(target, []byte("body"), 0o644); err != nil {
		t.Fatalf("WriteFileAtomic: %v", err)
	}

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat: %v", err)
	}
	if got := info.Mode().Perm(); got != 0o644 {
		t.Errorf("mode = %o, want %o — the staging file's 0600 leaked through", got, 0o644)
	}
}

// TestWriteFileAtomicLeavesTargetIntactOnFailure is the guarantee the helper
// exists for. A bare os.WriteFile truncates first and writes second, so a write
// that dies partway leaves a half-written document. Staging into a sibling file
// and renaming means the target is either its old self or its new self.
func TestWriteFileAtomicLeavesTargetIntactOnFailure(t *testing.T) {
	dir := t.TempDir()
	target := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(target, []byte("original"), 0o644); err != nil {
		t.Fatalf("seeding target: %v", err)
	}

	// A read-only directory refuses the staging file, so the write fails before
	// the target is touched at all.
	if err := os.Chmod(dir, 0o555); err != nil { //nolint:gosec // a directory needs its execute bits; G302 is about file modes
		t.Fatalf("making the directory read-only: %v", err)
	}
	t.Cleanup(func() { _ = os.Chmod(dir, 0o755) }) //nolint:gosec // restoring the temp directory so t.TempDir can clean it up

	if err := WriteFileAtomic(target, []byte("replacement"), 0o644); err == nil {
		t.Fatal("WriteFileAtomic reported success writing into a read-only directory")
	}

	got, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading target: %v", err)
	}
	if string(got) != "original" {
		t.Errorf("content = %q, want the original bytes untouched", got)
	}

	if err := os.Chmod(dir, 0o755); err != nil { //nolint:gosec // restoring the temp directory to read it back
		t.Fatalf("restoring the directory: %v", err)
	}
	if leftovers := tempFilesIn(t, dir); leftovers != nil {
		t.Errorf("a failed write left staging files behind: %v", leftovers)
	}
}

func TestWriteFileAtomicRejectsUnwritableTarget(t *testing.T) {
	dir := t.TempDir()
	// The parent exists but the named subdirectory does not, so staging fails.
	target := filepath.Join(dir, "missing", "doc.md")

	if err := WriteFileAtomic(target, []byte("body"), 0o644); err == nil {
		t.Fatal("WriteFileAtomic reported success for a target whose directory does not exist")
	}
}
