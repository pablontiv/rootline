package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupStrayAncestor builds the scenario the migration exists for: a project
// with its own .stem, and a .stem in an ancestor directory outside the project.
// It returns the ancestor root, the project directory, and the target file.
func setupStrayAncestor(t *testing.T) (ancestor, project, target string) {
	t.Helper()

	ancestor = t.TempDir()
	if err := os.WriteFile(filepath.Join(ancestor, stemFileName), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	project = filepath.Join(ancestor, "code", "myproject")
	if err := os.MkdirAll(filepath.Join(project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, stemFileName), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", stemFileName), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	// A .git marker makes the project boundary discoverable as a HINT only.
	if err := os.Mkdir(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	target = filepath.Join(project, "docs")
	return ancestor, project, target
}

// TestProposeRootDirectory_StrayAncestorIsNeverProposed is the case the shipped
// test never built. With a stray ancestor .stem in the chain, the proposal must
// be the project directory — never the ancestor. Proposing the ancestor would
// declare the stray file as the governance root and lock it in permanently,
// which is the opposite of the intent.
func TestProposeRootDirectory_StrayAncestorIsNeverProposed(t *testing.T) {
	ancestor, project, target := setupStrayAncestor(t)

	entries, err := WalkUp(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(entries) < 3 {
		t.Fatalf("expected the chain to include the stray ancestor, got %d entries", len(entries))
	}
	if got := filepath.Dir(entries[0].Path); got != ancestor {
		t.Fatalf("precondition failed: expected topmost entry in %q, got %q", ancestor, got)
	}

	proposed := ProposeRootDirectory(entries, target)

	if proposed == ancestor {
		t.Fatalf("proposed the stray ancestor %q as the governance root", ancestor)
	}
	if proposed != project {
		t.Errorf("expected the project directory %q, got %q", project, proposed)
	}
}

// TestComposeNoDeclaredBoundaryError_NamesDistinctPaths guards against the
// message telling the user to mark the stray file itself. The stray path and
// the proposed directory must not be the same place.
func TestComposeNoDeclaredBoundaryError_NamesDistinctPaths(t *testing.T) {
	ancestor, project, target := setupStrayAncestor(t)

	entries, err := WalkUp(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proposed := ProposeRootDirectory(entries, target)
	msg := ComposeNoDeclaredBoundaryError(entries[0].Path, proposed)

	if !strings.Contains(msg, ancestor) {
		t.Errorf("message should name the stray .stem in %q, got:\n%s", ancestor, msg)
	}
	if !strings.Contains(msg, project) {
		t.Errorf("message should name the project %q as the fix location, got:\n%s", project, msg)
	}
	if !strings.Contains(msg, "root: true") {
		t.Errorf("message should contain the exact line to add, got:\n%s", msg)
	}
	if filepath.Dir(entries[0].Path) == proposed {
		t.Error("stray path and proposed directory resolve to the same place")
	}
}

// TestProposeRootDirectory_NoStrayProposesChainRoot keeps the original
// behaviour for the simple case: when the whole chain is inside the project,
// the topmost .stem is the right place for the marker.
func TestProposeRootDirectory_NoStrayProposesChainRoot(t *testing.T) {
	project := t.TempDir()
	if err := os.MkdirAll(filepath.Join(project, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, stemFileName), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, "docs", stemFileName), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.Mkdir(filepath.Join(project, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	target := filepath.Join(project, "docs")
	entries, err := WalkUp(target)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if got := ProposeRootDirectory(entries, target); got != project {
		t.Errorf("expected %q, got %q", project, got)
	}
}

// TestProposeRootDirectory_WithoutGitHintStaysConservative covers a project
// with no .git to hint the boundary. Proposing a directory outside the project
// is actively harmful, so with no evidence of where the project starts the
// proposal must not reach above the target.
func TestProposeRootDirectory_WithoutGitHintStaysConservative(t *testing.T) {
	ancestor := t.TempDir()
	if err := os.WriteFile(filepath.Join(ancestor, stemFileName), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	project := filepath.Join(ancestor, "code", "myproject")
	if err := os.MkdirAll(project, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(project, stemFileName), []byte("version: 2\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	entries, err := WalkUp(project)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	proposed := ProposeRootDirectory(entries, project)
	if proposed == ancestor {
		t.Fatalf("proposed the stray ancestor %q with no evidence it belongs to the project", ancestor)
	}
	if proposed != project {
		t.Errorf("expected %q, got %q", project, proposed)
	}
}
