package derive

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestDirectPipelinesRejectInvalidGovernanceWithoutPublishingDerivedState(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(`version: 2
root: true
schema:
  id:
    type: sequence
    match:
      "BAD*": {prefix: BAD, digits: 2.0}
derive:
  slug: 'slugify(title)'
aggregate:
  total: 'len(descendants)'
`), 0o644); err != nil {
		t.Fatal(err)
	}

	for name, run := range map[string]func(context.Context, []*extract.Record, string) error{
		"derive":    DeriveAllSimple,
		"enrich":    EnrichBuiltinsSimple,
		"aggregate": AggregateAllSimple,
	} {
		t.Run(name, func(t *testing.T) {
			record := &extract.Record{
				Path:        "BAD001.md",
				Frontmatter: map[string]any{"title": "Bad"},
				Derived:     map[string]any{"existing": "unchanged", "isIndex": true},
			}
			identity := mapIdentity(record.Derived)
			before := cloneAnyMap(record.Derived)

			err := run(context.Background(), []*extract.Record{record}, root)
			if err == nil || !strings.Contains(err.Error(), "BAD001.md") || !strings.Contains(err.Error(), "digits") {
				t.Fatalf("error = %v, want original BAD001 sequence resolution cause", err)
			}
			if mapIdentity(record.Derived) != identity {
				t.Fatal("Derived map identity changed after failed resolution")
			}
			if !sameAnyMap(record.Derived, before) {
				t.Fatalf("Derived = %#v, want %#v", record.Derived, before)
			}
		})
	}
}
