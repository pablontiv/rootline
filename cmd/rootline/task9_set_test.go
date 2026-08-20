package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestTask9SetSourceOverridePreservesBodyAndMode(t *testing.T) {
	dir := setupTask9SourceProject(t, false)
	target := filepath.Join(dir, "body.md")
	if err := os.Chmod(target, 0o600); err != nil {
		t.Fatal(err)
	}
	before := string(mustReadFile(t, target))
	beforeBody := before[strings.Index(before, "---\n# Body")+4:]

	out, err := runCmd(t, "set", target, "notes=mode-safe override")
	if err != nil {
		t.Fatalf("set source-backed field failed: %v\n%s", err, out)
	}
	afterBytes := mustReadFile(t, target)
	after := string(afterBytes)
	afterBody := after[strings.Index(after, "---\n# Body")+4:]
	if afterBody != beforeBody {
		t.Fatalf("body bytes changed\nbefore:%q\nafter:%q", beforeBody, afterBody)
	}
	if strings.Count(after, "## Notes") != 1 || !strings.Contains(after, "notes: mode-safe override") {
		t.Fatalf("set should write one frontmatter override and no body section mutation:\n%s", after)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("file mode = %v, want 0600", got)
	}

	queryOut, err := runCmd(t, "query", dir, "--where", "notes == 'mode-safe override'", "--select", "path,notes")
	if err != nil || !strings.Contains(queryOut, `"path":"body.md"`) || !strings.Contains(queryOut, `"notes":"mode-safe override"`) {
		t.Fatalf("query follow-up = err %v output %s, want frontmatter override", err, queryOut)
	}
}

// runSet applies SetField proposals through fix.ApplyProposals and returns rollback
// evidence as an error string; literal fix.ApplyRepair rolled_back[] evidence is
// inapplicable to this API and belongs to repair apply coverage.
func TestTask9SetSourceOverrideFailedPostValidationRollsBackExactly(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte(`version: 2
root: true
scope:
  match: "*.md"
schema:
  status:
    type: enum
    values: [draft, ready]
  owner:
    type: string
  notes:
    type: string
    source: body.section["## Notes"]
validate:
  - rule: requires
    if: { status: ready }
    then: { fields: [owner] }
`), 0o644)
	target := filepath.Join(dir, "body.md")
	original := []byte("---\nstatus: draft\n---\n# Body\n\n## Notes\n\nallowed\n")
	mustWriteFile(t, target, original, 0o640)
	declareTestBoundary(t, dir)

	out, err := runCmd(t, "set", target, "status=ready")
	if err == nil {
		t.Fatalf("set unexpectedly succeeded after activating missing owner rule:\n%s", out)
	}
	if !strings.Contains(err.Error(), "post-mutation validation failed (rolled back)") || !strings.Contains(err.Error(), "owner") {
		t.Fatalf("set error = %v, want post-validation rollback for missing owner", err)
	}
	if strings.Contains(err.Error(), "not in allowed values") {
		t.Fatalf("set failed during enum prevalidation, not post-write validation: %v", err)
	}
	after := mustReadFile(t, target)
	if string(after) != string(original) {
		t.Fatalf("failed set did not roll back exact bytes\nwant:%q\ngot: %q", string(original), string(after))
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o640 {
		t.Fatalf("rolled-back mode = %v, want 0640", got)
	}
}
