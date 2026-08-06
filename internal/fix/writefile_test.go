package fix

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"sync/atomic"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
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

// atomicityStem governs one enum field, which is enough to drive a repair that
// rewrites a document end to end.
const atomicityStem = `version: 2
root: true
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`

// tornObservations rewrites a governed document `rounds` times while a reader
// polls it, and reports how many of those reads landed on a document that was
// neither its old self nor its new self.
//
// rewrite performs one rewrite and is what varies between callers: a bare
// truncating write (which must tear) or a real repair run (which must not).
// Comparing the two through the same harness is what makes a clean result mean
// something — zero tearing only proves atomicity if the harness could have seen
// tearing at all.
func tornObservations(t *testing.T, rounds int, rewrite func(dir, target string, round int)) int64 {
	t.Helper()

	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(atomicityStem), 0o644); err != nil {
		t.Fatalf("seeding .stem: %v", err)
	}

	// A body large enough that a truncate-then-write leaves a window a reader
	// can land in. Smaller and the defect is still there but unobservable.
	body := strings.Repeat("filler line to widen the write window\n", 8000)
	target := filepath.Join(dir, "task.md")
	if err := os.WriteFile(target, []byte(documentWithEstado("Pending", body)), 0o644); err != nil {
		t.Fatalf("seeding target: %v", err)
	}

	var stop atomic.Bool
	var torn atomic.Int64
	readerDone := make(chan struct{})

	go func() {
		defer close(readerDone)
		for !stop.Load() {
			content, err := os.ReadFile(target)
			if err != nil {
				// Only content the reader actually got is evidence.
				continue
			}
			// Every observation must be a whole document: both frontmatter
			// delimiters present and the body intact to its last byte.
			s := string(content)
			if !strings.HasPrefix(s, "---\n") || !strings.Contains(s, "\n---\n") ||
				!bytes.HasSuffix(content, []byte(body)) {
				torn.Add(1)
			}
		}
	}()

	for i := range rounds {
		rewrite(dir, target, i)
	}

	stop.Store(true)
	<-readerDone
	return torn.Load()
}

// documentWithEstado renders the fixture document for a given field value.
func documentWithEstado(estado, body string) string {
	return "---\nestado: " + estado + "\n---\n\n# Task\n\n" + body
}

// estadoForRound alternates the field value so every round is a real rewrite.
func estadoForRound(i int) (from, to string) {
	if i%2 == 1 {
		return "Completed", "Pending"
	}
	return "Pending", "Completed"
}

// TestApplyRepairIsObservablyAtomic is the end-to-end statement of the property
// the helper exists for, and the test that was RED on the code this replaced.
//
// The unit tests above prove WriteFileAtomic stages and renames. This one proves
// the repair path actually goes through it, by asserting what a user cares
// about: someone reading a governed document while a repair runs must see either
// the old document or the new one, never a fragment of either.
//
// The control run is not decoration. A bare os.WriteFile truncates before it
// writes, so the first state it exposes is a zero-length file; if the harness
// cannot catch even that, it has no power on this machine and a clean result
// from the repair path would mean nothing. So the control decides whether the
// assertion runs at all, and the test declares itself inconclusive rather than
// passing on no evidence.
func TestApplyRepairIsObservablyAtomic(t *testing.T) {
	// The control is cheap — bare writes — so it runs many more rounds and gives
	// a firm answer about whether this machine can observe tearing at all. The
	// experiment drives the full repair path, which is far slower per round.
	const controlRounds, repairRounds = 60, 8

	// Control: the shape this fix replaced. Truncate, then write.
	bare := tornObservations(t, controlRounds, func(_, target string, i int) {
		_, to := estadoForRound(i)
		content, err := os.ReadFile(target)
		if err != nil {
			t.Fatalf("reading the control target: %v", err)
		}
		body := string(content)
		body = body[strings.Index(body[4:], "---\n")+4:]
		if err := os.WriteFile(target, []byte(documentWithEstado(to, body)), 0o644); err != nil { //nolint:gosec // the control writer: target is this test's own t.TempDir fixture
			t.Fatalf("control write: %v", err)
		}
	})
	if bare == 0 {
		t.Skip("inconclusive: the harness observed no tearing even from a truncating write")
	}
	t.Logf("control: a truncating write exposed %d half-written states", bare)

	// The real path: the same document, rewritten by repair apply.
	torn := tornObservations(t, repairRounds, func(dir, _ string, i int) {
		from, to := estadoForRound(i)
		props := []proposal.Proposal{{
			Type:        proposal.CorrectValue,
			Field:       "estado",
			From:        from,
			To:          to,
			Paths:       []string{"task.md"},
			Description: "correct estado",
		}}
		if _, err := ApplyRepair(props, false, dir, false); err != nil {
			t.Fatalf("ApplyRepair: %v", err)
		}
	})
	if torn != 0 {
		t.Errorf("a concurrent reader observed %d half-written states during repair apply; "+
			"the repair path is not writing atomically", torn)
	}
}
