package main

import (
	"bytes"
	"context"
	"errors"
	"fmt"
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

func TestSetPostValidationRollbackIsAtomic(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nschema:\n  status:\n    type: enum\n    values: [draft, ready]\n"), 0o644)
	target := filepath.Join(dir, "record.md")
	original := []byte("---\r\n# Keep original formatting\r\nstatus: draft\r\n---\r\n# Original\r\n")
	mutated := []byte("---\nstatus: invalid\n---\n# Mutated\n")
	mustWriteFile(t, target, mutated, 0o600)
	// A hard link retains the pre-restore inode, exposing any truncate/write.
	oldInode := filepath.Join(dir, "before-restore")
	if err := os.Link(target, oldInode); err != nil {
		t.Fatal(err)
	}

	err := postValidateOrRollback(context.Background(), target, dir, original)
	if err == nil || !strings.HasPrefix(err.Error(), "post-mutation validation failed (rolled back): status: ") {
		t.Fatalf("post-validation error = %v, want existing successful rollback wording", err)
	}
	if got := mustReadFile(t, target); !bytes.Equal(got, original) {
		t.Fatalf("restored bytes = %q, want %q", got, original)
	}
	if got := mustReadFile(t, oldInode); !bytes.Equal(got, mutated) {
		t.Fatalf("old inode bytes = %q, want unchanged mutated bytes %q", got, mutated)
	}
	info, err := os.Stat(target)
	if err != nil {
		t.Fatal(err)
	}
	if got := info.Mode().Perm(); got != 0o600 {
		t.Fatalf("restored mode = %v, want 0600", got)
	}
}

func TestSetPostValidationRestoreFailureDoesNotClaimRollback(t *testing.T) {
	for _, stage := range []string{"extraction", "resolution", "validation"} {
		t.Run(stage, func(t *testing.T) {
			t.Parallel()
			dir := t.TempDir()
			mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nschema:\n  status:\n    type: enum\n    values: [draft, ready]\n"), 0o644)
			target := filepath.Join(dir, "record.md")
			original := []byte("---\nstatus: draft\n---\n# Original\n")
			mutated := []byte("---\nstatus: invalid\n---\n# Mutated\n")
			mustWriteFile(t, target, mutated, 0o640)
			if stage == "resolution" {
				// Schema resolution fails independently of document validation.
				mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nroot: true\nschema: [broken\n"), 0o644)
			}
			restoreErr := errors.New("injected atomic restore failure")
			calls := 0
			restore := func(path string, content []byte, mode os.FileMode) error {
				calls++
				if path != target || !bytes.Equal(content, original) || mode != 0o644 {
					t.Fatalf("restore arguments = (%q, %q, %v), want original target and bytes", path, content, mode)
				}
				return restoreErr
			}
			var err error
			var wantStage, wantCause string
			switch stage {
			case "extraction":
				// The current extractor reports parse findings on the record,
				// so exercise the error composer with an extraction error directly.
				cause := errors.New("injected extraction failure")
				wantStage = fmt.Sprintf("re-extracting %s after mutation", target)
				wantCause = cause.Error()
				err = restoreSetOriginal(target, original, wantStage, cause, restore)
				if !errors.Is(err, cause) {
					t.Fatalf("restore failure lost extraction cause: %v", err)
				}
			case "resolution":
				wantStage = fmt.Sprintf("resolving .stem for %s after mutation", target)
				wantCause = "yaml"
				err = postValidateOrRollbackWithRestore(context.Background(), target, dir, original, restore)
			case "validation":
				wantStage = "post-mutation validation failed"
				wantCause = "status"
				err = postValidateOrRollbackWithRestore(context.Background(), target, dir, original, restore)
			}
			if err == nil {
				t.Fatal("restore failure unexpectedly succeeded")
			}
			if calls != 1 {
				t.Fatalf("restore calls = %d, want 1", calls)
			}
			if strings.Contains(err.Error(), "rolled back") {
				t.Fatalf("failed restore falsely claims rollback: %v", err)
			}
			for _, want := range []string{wantStage, wantCause, target, "restoring original content failed", restoreErr.Error()} {
				if !strings.Contains(err.Error(), want) {
					t.Fatalf("restore failure = %v, want %q", err, want)
				}
			}
			if !errors.Is(err, restoreErr) {
				t.Fatalf("restore failure lost restore error: %v", err)
			}
			if got := mustReadFile(t, target); !bytes.Equal(got, mutated) {
				t.Fatalf("failed restore bytes = %q, want mutated bytes %q", got, mutated)
			}
			info, statErr := os.Stat(target)
			if statErr != nil {
				t.Fatal(statErr)
			}
			if got := info.Mode().Perm(); got != 0o640 {
				t.Fatalf("failed restore mode = %v, want unchanged 0640", got)
			}
		})
	}
}
