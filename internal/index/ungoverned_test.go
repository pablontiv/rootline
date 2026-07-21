package index

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// resolverForTree returns a ScopeResolver backed by the real walk-up, so these
// tests exercise the same path the commands do.
func resolverForTree() ScopeResolver {
	return func(dir string) (*rules.StemFile, error) {
		entries, err := rules.WalkUp(dir)
		if err != nil {
			return nil, err
		}
		return rules.MergeStemFiles(entries), nil
	}
}

func ungovernedTree(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	for _, name := range []string{"a.md", "b.md"} {
		if err := os.WriteFile(filepath.Join(root, name), []byte("---\ntitle: x\n---\n# x\n"), 0o644); err != nil {
			t.Fatal(err)
		}
	}
	return root
}

// TestScan_UngovernedIsAnErrorByDefault pins the governed contract: a tree with
// no .stem anywhere must not scan successfully, because a command reporting
// success over zero records is the false green this change exists to remove.
func TestScan_UngovernedIsAnErrorByDefault(t *testing.T) {
	root := ungovernedTree(t)

	records, err := Scan(context.Background(), root, extract.NewRegistry(), WithScopeResolver(resolverForTree()))
	if err == nil {
		t.Fatalf("expected ErrNoSchemaFound, got %d records", len(records))
	}
	if !errors.Is(err, rules.ErrNoSchemaFound) {
		t.Fatalf("expected ErrNoSchemaFound, got: %v", err)
	}
}

// TestScan_AllowUngovernedScansSchemalessTrees covers the bootstrap commands.
//
// `schema propose` and `analyze` derive a schema FROM documents that do not
// have one yet. Requiring a resolved schema in order to read those documents
// makes them useless for their only purpose: you cannot demand a schema as the
// precondition for creating a schema.
func TestScan_AllowUngovernedScansSchemalessTrees(t *testing.T) {
	root := ungovernedTree(t)

	records, err := Scan(context.Background(), root, extract.NewRegistry(),
		WithScopeResolver(resolverForTree()), AllowUngoverned())
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(records) != 2 {
		t.Fatalf("got %d records, want 2 — ungoverned files must still be readable", len(records))
	}
}

// TestScan_AllowUngovernedStillAbortsOnHardErrors guards the escape hatch: it
// tolerates a missing schema, never a broken one.
func TestScan_AllowUngovernedStillAbortsOnHardErrors(t *testing.T) {
	root := ungovernedTree(t)
	boom := errors.New("unreadable .stem")

	_, err := Scan(context.Background(), root, extract.NewRegistry(),
		WithScopeResolver(func(string) (*rules.StemFile, error) { return nil, boom }),
		AllowUngoverned())
	if !errors.Is(err, boom) {
		t.Fatalf("expected the hard error to propagate, got: %v", err)
	}
}
