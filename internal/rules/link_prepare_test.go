package rules

import (
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func recWithLinks(path string, links ...extract.Link) *extract.Record {
	return &extract.Record{Path: path, Type: "markdown", Links: links}
}

// PrepareLinks resolves every governed link through the canonical resolver and
// rewrites it to the root-relative form the graph uses as a node key, so graph
// and validate agree on which links are broken (issue #62).
func TestPrepareLinks_ResolvesWikilinkToNodeKey(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "b.md"), "# B\n")
	writeFile(t, filepath.Join(root, "t.md"), "body")
	rec := recWithLinks("t.md", extract.Link{Target: "b", Type: "reference", Style: extract.StyleWikilink, Line: 1})

	PrepareLinks([]*extract.Record{rec}, root)

	if got := rec.Links[0].Target; got != "b.md" {
		t.Errorf("Target = %q, want %q", got, "b.md")
	}
	if got := rec.Links[0].Resolution; got != extract.LinkResolved {
		t.Errorf("Resolution = %q, want %q", got, extract.LinkResolved)
	}
}

// A path-qualified extension-less wikilink resolves too — graph used to report
// this broken while suggesting the correct file one line below (sub-defect 10).
func TestPrepareLinks_ResolvesPathQualifiedWikilink(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "sub", "README.md"), "# Sub\n")
	writeFile(t, filepath.Join(root, "t.md"), "body")
	rec := recWithLinks("t.md", extract.Link{Target: "sub/README", Type: "reference", Style: extract.StyleWikilink, Line: 1})

	PrepareLinks([]*extract.Record{rec}, root)

	if got := rec.Links[0].Target; got != filepath.Join("sub", "README.md") {
		t.Errorf("Target = %q, want sub/README.md", got)
	}
	if rec.Links[0].Resolution != extract.LinkResolved {
		t.Errorf("Resolution = %q, want resolved", rec.Links[0].Resolution)
	}
}

// Basename fallback still applies: a bare target matches a uniquely-named
// record anywhere in the tree, and an ambiguous one resolves to nothing rather
// than guessing. Gating this behind a schema knob is a separate change.
func TestPrepareLinks_BasenameFallback(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "t.md"), "body")
	link := func(target string) extract.Link {
		return extract.Link{Target: target, Type: "reference", Style: extract.StyleWikilink, Line: 1}
	}
	unique := []*extract.Record{recWithLinks("t.md", link("far")), recWithLinks(filepath.Join("deep", "far.md"))}
	PrepareLinks(unique, root)
	if unique[0].Links[0].Resolution != extract.LinkResolved || unique[0].Links[0].Target != filepath.Join("deep", "far.md") {
		t.Errorf("unique basename should resolve, got %+v", unique[0].Links[0])
	}

	ambiguous := []*extract.Record{recWithLinks("t.md", link("dup")),
		recWithLinks(filepath.Join("a", "dup.md")), recWithLinks(filepath.Join("b", "dup.md"))}
	PrepareLinks(ambiguous, root)
	if ambiguous[0].Links[0].Resolution != extract.LinkUnresolved {
		t.Errorf("ambiguous basename must stay unresolved, got %q", ambiguous[0].Links[0].Resolution)
	}
}

func TestPrepareLinks_MarksMissingTargetUnresolved(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "t.md"), "body")
	rec := recWithLinks("t.md", extract.Link{Target: "nope", Type: "reference", Style: extract.StyleWikilink, Line: 1})

	PrepareLinks([]*extract.Record{rec}, root)

	if rec.Links[0].Resolution != extract.LinkUnresolved {
		t.Errorf("Resolution = %q, want unresolved", rec.Links[0].Resolution)
	}
}

// A target that exists but is not a governed record still RESOLVES: the schema
// declares what is governed, not what exists. graph must not call it broken.
func TestPrepareLinks_ExistingButUngovernedTargetResolves(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "ungoverned.md"), "# U\n")
	writeFile(t, filepath.Join(root, "t.md"), "body")
	rec := recWithLinks("t.md", extract.Link{Target: "ungoverned.md", Type: "reference", Style: extract.StyleMarkdown, Line: 1})

	PrepareLinks([]*extract.Record{rec}, root)

	if rec.Links[0].Resolution != extract.LinkResolved {
		t.Errorf("Resolution = %q, want resolved", rec.Links[0].Resolution)
	}
}

func TestPrepareLinks_RootAnchoredTarget(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "Page.md"), "# P\n")
	writeFile(t, filepath.Join(root, "deep", "t.md"), "body")
	rec := recWithLinks(filepath.Join("deep", "t.md"),
		extract.Link{Target: "/docs/Page.md", Type: "reference", Style: extract.StyleMarkdown, Line: 1})

	PrepareLinks([]*extract.Record{rec}, root)

	if got := rec.Links[0].Target; got != filepath.Join("docs", "Page.md") {
		t.Errorf("Target = %q, want docs/Page.md", got)
	}
}

// A target that resolves but cannot be expressed relative to root gives the
// graph no key to match, so it must not be recorded as resolved.
func TestPrepareLinks_UnrelatableTargetStaysUnresolved(t *testing.T) {
	root := t.TempDir()
	other := t.TempDir()
	writeFile(t, filepath.Join(other, "elsewhere.md"), "# E\n")
	writeFile(t, filepath.Join(root, "t.md"), "body")
	rec := recWithLinks("t.md", extract.Link{
		Target: filepath.Join(other, "elsewhere.md"), Type: "reference",
		Style: extract.StyleMarkdown, Line: 1,
	})

	PrepareLinks([]*extract.Record{rec}, root)

	if rec.Links[0].Resolution == extract.LinkResolved && rec.Links[0].Target == filepath.Join(other, "elsewhere.md") {
		t.Errorf("target outside root marked resolved without a usable key: %+v", rec.Links[0])
	}
}
