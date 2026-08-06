package fix

import (
	"context"
	"io/fs"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/proposal"
)

// modeStem governs one enum field, which is enough to drive a real rewrite of a
// document through both write paths in this package.
const modeStem = `version: 2
root: true
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Completed]
`

// seedGovernedDocument writes a .stem and one document at the requested mode and
// returns the directory and the document's absolute path.
func seedGovernedDocument(t *testing.T, perm fs.FileMode) (dir, target string) {
	t.Helper()

	dir = t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(modeStem), 0o644); err != nil {
		t.Fatalf("seeding .stem: %v", err)
	}

	target = filepath.Join(dir, "task.md")
	content := "---\nestado: Pending\n---\n\n# Task\n\nrestricted body\n"
	if err := os.WriteFile(target, []byte(content), perm); err != nil { //nolint:gosec // a deliberately restrictive fixture mode is the subject of the test
		t.Fatalf("seeding target: %v", err)
	}
	// os.WriteFile applies the process umask, so pin the mode explicitly.
	if err := os.Chmod(target, perm); err != nil {
		t.Fatalf("pinning target mode: %v", err)
	}
	return dir, target
}

// assertMode reads the target's mode back and fails if it is not exactly want.
func assertMode(t *testing.T, target string, want fs.FileMode) {
	t.Helper()

	info, err := os.Stat(target)
	if err != nil {
		t.Fatalf("stat %s: %v", target, err)
	}
	if got := info.Mode().Perm(); got != want {
		t.Errorf("mode = %#o, want %#o — the rewrite changed who can read the document", got, want)
	}
}

// assertRewritten fails unless the document actually changed, so a mode assertion
// can never pass because nothing was written at all.
func assertRewritten(t *testing.T, target string) {
	t.Helper()

	content, err := os.ReadFile(target)
	if err != nil {
		t.Fatalf("reading %s: %v", target, err)
	}
	if !strings.Contains(string(content), "estado: Completed") {
		t.Fatalf("document was not rewritten; content = %q", content)
	}
}

// correctEstado is the single proposal both write paths are driven with.
func correctEstado() proposal.Proposal {
	return proposal.Proposal{
		Type:        proposal.CorrectValue,
		Field:       "estado",
		From:        "Pending",
		To:          "Completed",
		Paths:       []string{"task.md"},
		Description: "correct estado",
	}
}

// TestApplyRepairPreservesRestrictiveFileMode is the regression statement for the
// permission-widening defect. Rewriting a record's frontmatter is a content
// operation: it carries no instruction to change who may read the file. A 0600
// document that `repair apply` rewrites must still be a 0600 document afterwards.
func TestApplyRepairPreservesRestrictiveFileMode(t *testing.T) {
	dir, target := seedGovernedDocument(t, 0o600)

	if _, err := ApplyRepair([]proposal.Proposal{correctEstado()}, false, dir, false); err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	assertRewritten(t, target)
	assertMode(t, target, 0o600)
}

// TestApplyRepairPreservesReadOnlyFileMode pins the stricter case. A 0400 document
// is both narrower and non-writable, so widening it to 0644 also grants write
// access nobody asked for.
func TestApplyRepairPreservesReadOnlyFileMode(t *testing.T) {
	dir, target := seedGovernedDocument(t, 0o400)

	if _, err := ApplyRepair([]proposal.Proposal{correctEstado()}, false, dir, false); err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	assertRewritten(t, target)
	assertMode(t, target, 0o400)
}

// TestApplyProposalsPreservesRestrictiveFileMode covers the second write path in
// this package. `fix --all` reaches documents through ApplyProposals rather than
// ApplyRepair, so a fix confined to the repair path would leave this one widening.
func TestApplyProposalsPreservesRestrictiveFileMode(t *testing.T) {
	dir, target := seedGovernedDocument(t, 0o600)

	report := &proposal.Report{Proposals: []proposal.Proposal{correctEstado()}}
	records := []*extract.Record{{
		Path:        "task.md",
		Frontmatter: map[string]any{"estado": "Pending"},
	}}
	if _, err := ApplyProposals(context.Background(), report, dir, records, false); err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}

	assertRewritten(t, target)
	assertMode(t, target, 0o600)
}
