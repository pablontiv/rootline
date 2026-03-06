package derive

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestEnrichBuiltins_IndexDetection(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{}},
		{Path: "dir/T001.md", Type: "markdown", Frontmatter: map[string]any{}},
		{Path: "dir/T002.md", Type: "markdown", Frontmatter: map[string]any{}},
	}

	resolver := func(dir string) *rules.StemFile { return &rules.StemFile{} }
	EnrichBuiltins(context.Background(), records, "/root", resolver)

	if records[0].Derived["isIndex"] != true {
		t.Errorf("README.isIndex = %v, want true", records[0].Derived["isIndex"])
	}
	if records[1].Derived["isIndex"] != false {
		t.Errorf("T001.isIndex = %v, want false", records[1].Derived["isIndex"])
	}
	if records[2].Derived["isIndex"] != false {
		t.Errorf("T002.isIndex = %v, want false", records[2].Derived["isIndex"])
	}
}

func TestEnrichBuiltins_CustomRequireIndex(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/index.md", Type: "markdown", Frontmatter: map[string]any{}},
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{}},
	}

	stem := &rules.StemFile{
		Structural: rules.StructuralRules{
			Subdirs: rules.SubdirRules{RequireIndex: "index.md"},
		},
	}
	resolver := func(dir string) *rules.StemFile { return stem }
	EnrichBuiltins(context.Background(), records, "/root", resolver)

	if records[0].Derived["isIndex"] != true {
		t.Errorf("index.md.isIndex = %v, want true", records[0].Derived["isIndex"])
	}
	if records[1].Derived["isIndex"] != false {
		t.Errorf("README.md.isIndex = %v, want false (custom require_index)", records[1].Derived["isIndex"])
	}
}

func TestEnrichBuiltins_NilResolver(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{}},
	}

	EnrichBuiltins(context.Background(), records, "/root", nil)

	if records[0].Derived != nil {
		t.Errorf("Derived = %v, want nil (nil resolver)", records[0].Derived)
	}
}

func TestEnrichBuiltins_SurvivesDeriveAll(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{"titulo": "Test"}},
		{Path: "dir/T001.md", Type: "markdown", Frontmatter: map[string]any{"titulo": "Task"}},
	}

	stem := &rules.StemFile{
		Derive: map[string]any{
			"slug": "lower(titulo)",
		},
	}
	resolver := func(dir string) *rules.StemFile { return stem }

	// Pipeline: DeriveAll → EnrichBuiltins (real order)
	DeriveAll(context.Background(), records, "/root", resolver)
	EnrichBuiltins(context.Background(), records, "/root", resolver)

	// isIndex should be set after enrichment.
	if records[0].Derived["isIndex"] != true {
		t.Errorf("README.isIndex = %v, want true", records[0].Derived["isIndex"])
	}
	// Derive results should still be present.
	if records[0].Derived["slug"] != "test" {
		t.Errorf("README.slug = %v, want 'test'", records[0].Derived["slug"])
	}
}
