package fix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

// These tests cover issue #60 defect 2: repair apply documented a rollback it
// never performed, and gated post-validation on the whole run's error count so
// one unrelated failure switched the validation net off for every other file.

const rollbackStem = `version: 2
root: true
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Pending, Done]
`

// newRollbackFixture builds a governed directory holding the named documents.
func newRollbackFixture(t *testing.T, docs map[string]string) string {
	t.Helper()
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(rollbackStem), 0644); err != nil {
		t.Fatalf("writing .stem: %v", err)
	}
	for name, body := range docs {
		if err := os.WriteFile(filepath.Join(dir, name), []byte(body), 0644); err != nil {
			t.Fatalf("writing %s: %v", name, err)
		}
	}
	return dir
}

func TestApplyRepair_RollsBackOnPostValidationFailure(t *testing.T) {
	original := "---\nestado: Pending\n---\n\n# Doc\n\nbody\n"
	dir := newRollbackFixture(t, map[string]string{"a.md": original})

	proposals := []proposal.Proposal{{
		Type:        proposal.CorrectValue,
		Field:       "estado",
		Description: "x",
		Paths:       []string{"a.md"},
		From:        "Pending",
		To:          "TOTALLY_INVALID",
	}}

	result, err := ApplyRepair(proposals, false, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatalf("reading a.md: %v", err)
	}
	if string(got) != original {
		t.Errorf("expected original bytes restored after failed post-validation.\nwant:\n%s\ngot:\n%s", original, got)
	}
	if len(result.RolledBack) != 1 {
		t.Fatalf("expected 1 rolled_back entry, got %d (%+v)", len(result.RolledBack), result.RolledBack)
	}
	if result.RolledBack[0].Path != "a.md" {
		t.Errorf("expected rolled_back path a.md, got %s", result.RolledBack[0].Path)
	}
	if len(result.RolledBack[0].Errors) == 0 {
		t.Error("expected the triggering validation errors to be reported")
	}
}

func TestApplyRepair_RollbackValidationErrorsAreDeterministic(t *testing.T) {
	dir := t.TempDir()
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(`version: 2
root: true
scope:
  match: "*.md"
schema:
  status:
    type: enum
    required: true
    values: [Pending, Completed]
  priority:
    type: enum
    required: true
    values: [Low, High]
`), 0o644); err != nil {
		t.Fatal(err)
	}

	const original = "---\nstatus: Pending\npriority: Low\n---\n\n# Doc\n"
	path := filepath.Join(dir, "a.md")
	if err := os.WriteFile(path, []byte(original), 0o600); err != nil {
		t.Fatal(err)
	}

	proposals := []proposal.Proposal{
		{Type: proposal.CorrectValue, Field: "status", Paths: []string{"a.md"}, From: "Pending", To: "Broken"},
		{Type: proposal.CorrectValue, Field: "priority", Paths: []string{"a.md"}, From: "Low", To: "Urgent"},
	}
	want := []string{
		"priority: value Urgent is not in allowed values: [Low, High]",
		"status: value Broken is not in allowed values: [Pending, Completed]",
	}

	for i := 0; i < 64; i++ {
		result, err := ApplyRepair(proposals, false, dir, false)
		if err != nil {
			t.Fatalf("ApplyRepair run %d failed: %v", i, err)
		}
		if len(result.RolledBack) != 1 {
			t.Fatalf("run %d rolled_back = %#v, want one entry", i, result.RolledBack)
		}
		got := result.RolledBack[0].Errors
		if len(got) != len(want) || got[0] != want[0] || got[1] != want[1] {
			t.Fatalf("run %d errors = %#v, want %#v", i, got, want)
		}
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != original {
		t.Fatalf("bytes after repeated rollback = %q, want %q", got, original)
	}
	info, err := os.Stat(path)
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode after repeated rollback = %o, want 600", info.Mode().Perm())
	}
}

func TestApplyRepair_RolledBackFileIsNotReportedAsChanged(t *testing.T) {
	dir := newRollbackFixture(t, map[string]string{
		"a.md": "---\nestado: Pending\n---\n\n# A\n",
	})

	proposals := []proposal.Proposal{{
		Type:  proposal.CorrectValue,
		Field: "estado",
		Paths: []string{"a.md"},
		From:  "Pending",
		To:    "NOPE",
	}}

	result, err := ApplyRepair(proposals, false, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	for _, c := range result.Changed {
		if strings.Contains(c, "a.md") {
			t.Errorf("a rolled-back file must not be reported as changed, got: %v", result.Changed)
		}
	}
}

func TestApplyRepair_UnrelatedErrorDoesNotDisablePostValidation(t *testing.T) {
	original := "---\nestado: Pending\n---\n\n# A\n"
	dir := newRollbackFixture(t, map[string]string{"a.md": original})

	// The first proposal names a path that does not exist, which lands a read
	// error in the run. The second targets a real file with an invalid value.
	proposals := []proposal.Proposal{
		{
			Type:  proposal.CorrectValue,
			Field: "estado",
			Paths: []string{"missing.md"},
			From:  "Pending",
			To:    "Done",
		},
		{
			Type:  proposal.CorrectValue,
			Field: "estado",
			Paths: []string{"a.md"},
			From:  "Pending",
			To:    "BOGUS",
		},
	}

	result, err := ApplyRepair(proposals, false, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatalf("reading a.md: %v", err)
	}
	if string(got) != original {
		t.Errorf("an unrelated read error must not disable post-validation for other files.\nwant:\n%s\ngot:\n%s", original, got)
	}
	if len(result.RolledBack) != 1 {
		t.Errorf("expected a.md to be rolled back despite the unrelated error, got %+v", result.RolledBack)
	}
}

func TestApplyRepair_ValidWriteSurvivesAlongsideRollback(t *testing.T) {
	dir := newRollbackFixture(t, map[string]string{
		"good.md": "---\nestado: Pending\n---\n\n# Good\n",
		"bad.md":  "---\nestado: Pending\n---\n\n# Bad\n",
	})

	proposals := []proposal.Proposal{
		{Type: proposal.CorrectValue, Field: "estado", Paths: []string{"good.md"}, From: "Pending", To: "Done"},
		{Type: proposal.CorrectValue, Field: "estado", Paths: []string{"bad.md"}, From: "Pending", To: "INVALID"},
	}

	result, err := ApplyRepair(proposals, false, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	good, err := os.ReadFile(filepath.Join(dir, "good.md"))
	if err != nil {
		t.Fatalf("reading good.md: %v", err)
	}
	if !strings.Contains(string(good), "estado: Done") {
		t.Errorf("expected the valid write to persist, got:\n%s", good)
	}

	bad, err := os.ReadFile(filepath.Join(dir, "bad.md"))
	if err != nil {
		t.Fatalf("reading bad.md: %v", err)
	}
	if !strings.Contains(string(bad), "estado: Pending") {
		t.Errorf("expected the invalid write to be rolled back, got:\n%s", bad)
	}

	if len(result.RolledBack) != 1 || result.RolledBack[0].Path != "bad.md" {
		t.Errorf("expected only bad.md rolled back, got %+v", result.RolledBack)
	}
}

func TestApplyRepair_CorrectLinkIsPostValidatedAndRolledBack(t *testing.T) {
	// correct_link writes through its own read/replace path rather than through
	// RewriteFrontmatter, so it is the handler most easily left outside the
	// post-validation net. Its write must be caught like any other.
	original := "---\nestado: Pending\n---\n\n# A\n\nsee [[target]]\n"
	dir := newRollbackFixture(t, map[string]string{"a.md": original})

	proposals := []proposal.Proposal{
		// Rewrites the frontmatter value to something the schema rejects.
		{Type: proposal.CorrectValue, Field: "estado", Paths: []string{"a.md"}, From: "Pending", To: "NOT_A_VALUE"},
		// Touches the same file through the link path.
		{Type: proposal.CorrectLink, Field: "links", Paths: []string{"a.md"}, From: "target", To: "renamed"},
	}

	result, err := ApplyRepair(proposals, false, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatalf("reading a.md: %v", err)
	}
	if string(got) != original {
		t.Errorf("expected the whole file restored after failed post-validation.\nwant:\n%s\ngot:\n%s", original, got)
	}
	if len(result.RolledBack) != 1 {
		t.Fatalf("expected a.md rolled back, got %+v", result.RolledBack)
	}
	// Both handlers touched a.md, so both of its Changed entries must be withdrawn.
	for _, c := range result.Changed {
		if strings.Contains(c, "a.md") {
			t.Errorf("a rolled-back file must leave no Changed entry, got: %v", result.Changed)
		}
	}
}

func TestApplyRepair_DryRunNeverRollsBack(t *testing.T) {
	original := "---\nestado: Pending\n---\n\n# A\n"
	dir := newRollbackFixture(t, map[string]string{"a.md": original})

	proposals := []proposal.Proposal{{
		Type:  proposal.CorrectValue,
		Field: "estado",
		Paths: []string{"a.md"},
		From:  "Pending",
		To:    "INVALID",
	}}

	result, err := ApplyRepair(proposals, true, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatalf("reading a.md: %v", err)
	}
	if string(got) != original {
		t.Errorf("dry-run must not touch the file at all, got:\n%s", got)
	}
	if len(result.RolledBack) != 0 {
		t.Errorf("dry-run writes nothing, so nothing can roll back, got %+v", result.RolledBack)
	}
}
