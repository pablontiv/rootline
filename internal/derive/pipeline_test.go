package derive

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestDeriveAllWithResolver(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/a.md", Type: "markdown", Frontmatter: map[string]any{"titulo": "Alpha"}},
		{Path: "dir/b.md", Type: "markdown", Frontmatter: map[string]any{"titulo": "Beta"}},
	}

	stem := &rules.StemFile{
		Derive: map[string]any{
			"slug": `slugify(titulo)`,
		},
	}

	resolver := func(dir string) *rules.StemFile {
		return stem
	}

	DeriveAll(records, "/root", resolver)

	if records[0].Derived["slug"] != "alpha" {
		t.Errorf("record[0].Derived[slug] = %v, want %q", records[0].Derived["slug"], "alpha")
	}
	if records[1].Derived["slug"] != "beta" {
		t.Errorf("record[1].Derived[slug] = %v, want %q", records[1].Derived["slug"], "beta")
	}
}

func TestDeriveAllNilResolver(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{}},
	}
	// Should not panic.
	DeriveAll(records, "/root", nil)
	if records[0].Derived != nil {
		t.Error("expected nil Derived with nil resolver")
	}
}

func TestDeriveAllNoDerive(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"titulo": "Hello"}},
	}
	stem := &rules.StemFile{} // no derive
	resolver := func(dir string) *rules.StemFile { return stem }

	DeriveAll(records, "/root", resolver)
	if records[0].Derived != nil {
		t.Error("expected nil Derived when .stem has no derive")
	}
}

func TestDeriveAllWithChildren(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/parent.md", Frontmatter: map[string]any{"titulo": "Parent"}},
		{Path: "dir/child1.md", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "dir/child2.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	stem := &rules.StemFile{
		Derive: map[string]any{
			"sibling_count": "len(children)",
		},
	}
	resolver := func(dir string) *rules.StemFile { return stem }

	DeriveAll(records, "/root", resolver)

	// Each record should see 2 siblings (the other records in the same dir).
	if records[0].Derived["sibling_count"] != 2 {
		t.Errorf("parent sibling_count = %v, want 2", records[0].Derived["sibling_count"])
	}
}

func TestDeriveAllResolverReturnsNil(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{}},
	}
	resolver := func(dir string) *rules.StemFile { return nil }

	DeriveAll(records, "/root", resolver)
	if records[0].Derived != nil {
		t.Error("expected nil Derived when resolver returns nil")
	}
}
