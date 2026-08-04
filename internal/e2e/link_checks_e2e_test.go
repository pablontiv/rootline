package e2e

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestE2E_MarkdownLinkChecks(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
links:
  styles: [markdown]
  checks:
    resolve: true
    anchors: true
    encoding: true
`,
		"master.md":       "# Master\n\n## Glosario de siglas\n\ntext\n",
		"guides/setup.md": "# Setup\n\nVer [master](../master.md#glosario-de-siglas).\n",
		"broken.md":       "[roto](guides/setpu.md)\n[anchor malo](master.md#no-existe)\n[espacio](my file.md)\n",
	})

	ctx := context.Background()
	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, root, reg)
	if err != nil {
		t.Fatal(err)
	}

	cache := rules.NewHeadingCache()
	errsByRule := map[string]int{}
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		effective, err := rules.ResolveForRecord(filepath.Dir(absPath), rec.Path)
		if err != nil {
			t.Fatalf("resolve %s: %v", rec.Path, err)
		}
		all := rules.Validate(ctx, rec, effective)
		all = append(all, rules.CheckLinks(rec.Links, effective.Links, absPath, root, cache)...)
		for _, e := range all {
			errsByRule[e.Rule]++
		}
	}

	if errsByRule["link_resolve"] != 2 { // setpu.md + "my file.md" (also fails resolve)
		t.Errorf("link_resolve = %d, want 2 (map: %v)", errsByRule["link_resolve"], errsByRule)
	}
	if errsByRule["link_anchor"] != 1 {
		t.Errorf("link_anchor = %d, want 1 (map: %v)", errsByRule["link_anchor"], errsByRule)
	}
	if errsByRule["link_encoding"] != 1 {
		t.Errorf("link_encoding = %d, want 1 (map: %v)", errsByRule["link_encoding"], errsByRule)
	}
}
