package e2e

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

// TestResolveForRecordBackwardCompat tests that a .stem without levels
// resolves correctly using standard walk-up merge.
func TestResolveForRecordBackwardCompat(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 1
scope:
  match: "*.md"
schema:
  titulo:
    type: string
    required: true
  estado:
    type: string
    enum: [Pending, Active, Done]
`,
		"docs/readme.md": "---\ntitulo: My Doc\nestado: Active\n---\n",
		"docs/notes.md":  "---\ntitulo: Notes\nestado: Pending\n---\n",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	resolver := buildScopeResolver()

	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("Scan error: %v", err)
	}

	if len(records) != 2 {
		t.Fatalf("expected 2 records, got %d", len(records))
	}

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			t.Fatalf("ResolveForRecord(%s): %v", rec.Path, resolveErr)
		}

		errs := rules.Validate(ctx, rec, effective)
		if len(errs) != 0 {
			t.Errorf("record %s should pass validation, got: %v", rec.Path, errs)
		}
	}
}
