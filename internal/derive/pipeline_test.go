package derive

import (
	"context"
	"os"
	"path/filepath"
	"strings"
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

	resolver := func(dir, recordPath string) (*rules.StemFile, error) {
		return stem, nil
	}

	DeriveAll(context.Background(), records, "/root", resolver)

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
	DeriveAll(context.Background(), records, "/root", nil)
	if records[0].Derived != nil {
		t.Error("expected nil Derived with nil resolver")
	}
}

func TestDeriveAllNoDerive(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"titulo": "Hello"}},
	}
	stem := &rules.StemFile{} // no derive
	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return stem, nil }

	DeriveAll(context.Background(), records, "/root", resolver)
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
	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return stem, nil }

	DeriveAll(context.Background(), records, "/root", resolver)

	// Each record should see 2 siblings (the other records in the same dir).
	if records[0].Derived["sibling_count"] != 2 {
		t.Errorf("parent sibling_count = %v, want 2", records[0].Derived["sibling_count"])
	}
}

func TestDeriveAllResolverReturnsNil(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{}},
	}
	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return nil, nil }

	DeriveAll(context.Background(), records, "/root", resolver)
	if records[0].Derived != nil {
		t.Error("expected nil Derived when resolver returns nil")
	}
}

func TestHasDeriveFields(t *testing.T) {
	if HasDeriveFields(nil) {
		t.Error("HasDeriveFields(nil) = true, want false")
	}
	records := []*extract.Record{{Path: "a.md"}}
	if !HasDeriveFields(records) {
		t.Error("HasDeriveFields(non-empty) = false, want true")
	}
}

func TestDeriveAllSimple(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"titulo": "Hello"}},
	}
	// DeriveAllSimple uses DefaultResolver which walks the real FS.
	// With no .stem file present, it should not panic and leave Derived nil.
	if err := DeriveAllSimple(context.Background(), records, t.TempDir()); err != nil {
		t.Fatalf("DeriveAllSimple: %v", err)
	}
}

func TestDefaultResolver(t *testing.T) {
	dir := t.TempDir()
	resolver := DefaultResolver()
	// No .stem in temp dir maps explicitly to nil, nil bootstrap behavior.
	stem, err := resolver(dir, "")
	if err != nil {
		t.Fatalf("DefaultResolver no schema error = %v, want nil", err)
	}
	if stem != nil {
		t.Fatalf("DefaultResolver no schema stem = %#v, want nil", stem)
	}
}

func TestDeriveAllSimpleInvalidLaterRecordReturnsResolutionErrorWithoutMutatingAnyRecord(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(`version: 2
root: true
scope:
  match: "*.md"
schema:
  id:
    type: sequence
    required: false
    match:
      "BAD*": {prefix: BAD, digits: 2.0}
derive:
  slug: 'slugify(titulo)'
`), 0o644); err != nil {
		t.Fatal(err)
	}
	records := []*extract.Record{
		{Path: "GOOD.md", Frontmatter: map[string]any{"titulo": "Good"}, Derived: map[string]any{"existing": "kept"}},
		{Path: "BAD001.md", Frontmatter: map[string]any{"titulo": "Bad"}, Derived: map[string]any{"existing": "also kept"}},
	}
	beforeIdentity := []uintptr{mapIdentity(records[0].Derived), mapIdentity(records[1].Derived)}
	beforeContent := []map[string]any{cloneAnyMap(records[0].Derived), cloneAnyMap(records[1].Derived)}

	err := DeriveAllSimple(context.Background(), records, root)
	if err == nil || !strings.Contains(err.Error(), "BAD001.md") || !strings.Contains(err.Error(), "digits") {
		t.Fatalf("DeriveAllSimple error = %v, want original sequence digits resolution cause", err)
	}
	for i, rec := range records {
		if mapIdentity(rec.Derived) != beforeIdentity[i] {
			t.Fatalf("record %d Derived identity changed", i)
		}
		if got, want := rec.Derived, beforeContent[i]; !sameAnyMap(got, want) {
			t.Fatalf("record %d Derived = %#v, want %#v", i, got, want)
		}
	}
}

func sameAnyMap(got, want map[string]any) bool {
	if len(got) != len(want) {
		return false
	}
	for k, wantValue := range want {
		if got[k] != wantValue {
			return false
		}
	}
	return true
}
