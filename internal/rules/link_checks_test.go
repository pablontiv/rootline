package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mdChecksSchema(c LinkChecks) LinkSchema {
	return LinkSchema{Styles: []string{extract.StyleMarkdown}, Checks: &c}
}

func mdLink(target string) extract.Link {
	return extract.Link{Target: target, Type: "reference", Style: extract.StyleMarkdown, Line: 1}
}

func TestCheckLinks_NilChecksIsNoop(t *testing.T) {
	schema := LinkSchema{Styles: []string{extract.StyleMarkdown}}
	errs := CheckLinks([]extract.Link{mdLink("missing.md")}, schema, "/nonexistent/src.md", "", nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors without checks, got %+v", errs)
	}
}

func TestCheckLinks_EncodingRejectsRawSpaces(t *testing.T) {
	schema := mdChecksSchema(LinkChecks{Encoding: true})
	errs := CheckLinks([]extract.Link{mdLink("my file.md")}, schema, "/tmp/src.md", "", nil)
	if len(errs) != 1 || errs[0].Rule != "link_encoding" {
		t.Fatalf("expected 1 link_encoding error, got %+v", errs)
	}
}

func TestCheckLinks_ResolveExisting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	errs := CheckLinks([]extract.Link{mdLink("guides/setup.md")}, schema, filepath.Join(dir, "src.md"), dir, nil)
	if len(errs) != 0 {
		t.Fatalf("expected resolve to pass, got %+v", errs)
	}
}

func TestCheckLinks_ResolveBrokenWithSuggestion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	errs := CheckLinks([]extract.Link{mdLink("guides/setpu.md")}, schema, filepath.Join(dir, "src.md"), dir, nil)
	if len(errs) != 1 || errs[0].Rule != "link_resolve" {
		t.Fatalf("expected 1 link_resolve error, got %+v", errs)
	}
	if errs[0].Suggestion != "setup.md" {
		t.Errorf("Suggestion = %q, want %q", errs[0].Suggestion, "setup.md")
	}
}

func TestCheckLinks_ResolveCaseMismatchIsBroken(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "Setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	// APFS would resolve this case-insensitively; ADO/git will not.
	errs := CheckLinks([]extract.Link{mdLink("guides/setup.md")}, schema, filepath.Join(dir, "src.md"), dir, nil)
	if len(errs) != 1 || errs[0].Rule != "link_resolve" {
		t.Fatalf("expected case mismatch to be broken, got %+v", errs)
	}
}

func TestCheckLinks_ResolveDirectoryTargetNeedsReadme(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "README.md"), "# Guides\n")
	writeFile(t, filepath.Join(dir, "empty", ".keep"), "")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	src := filepath.Join(dir, "src.md")
	if errs := CheckLinks([]extract.Link{mdLink("guides/")}, schema, src, dir, nil); len(errs) != 0 {
		t.Fatalf("dir with README should resolve, got %+v", errs)
	}
	if errs := CheckLinks([]extract.Link{mdLink("empty/")}, schema, src, dir, nil); len(errs) != 1 {
		t.Fatalf("dir without README should be broken, got %+v", errs)
	}
}

func TestCheckLinks_ResolveDecodesPercent20(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "my page.md"), "# P\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	errs := CheckLinks([]extract.Link{mdLink("my%20page.md")}, schema, filepath.Join(dir, "src.md"), dir, nil)
	if len(errs) != 0 {
		t.Fatalf("%%20 target should resolve, got %+v", errs)
	}
}

func TestCheckLinks_SkipsFilteredStyles(t *testing.T) {
	schema := mdChecksSchema(LinkChecks{Resolve: true, Encoding: true})
	links := []extract.Link{
		{Target: "no such file", Type: "reference", Style: extract.StyleWikilink, Line: 1}, // filtered style
	}
	if errs := CheckLinks(links, schema, "/tmp/src.md", "", nil); len(errs) != 0 {
		t.Fatalf("expected no errors, got %+v", errs)
	}
}

// Root-anchored targets are the idiomatic ADO code-wiki form. validate used to
// skip them outright, so a dangling /x.md passed while graph flagged it
// (issue #62 sub-defect 3). They are now resolved against the scan root.
func TestCheckLinks_RootAnchoredIsChecked(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "docs", "Page.md"), "# Page\n")
	writeFile(t, filepath.Join(root, "deep", "src.md"), "body")
	src := filepath.Join(root, "deep", "src.md")
	schema := mdChecksSchema(LinkChecks{Resolve: true})

	if errs := CheckLinks([]extract.Link{mdLink("/docs/Page.md")}, schema, src, root, nil); len(errs) != 0 {
		t.Errorf("existing root-anchored target should resolve, got %+v", errs)
	}
	errs := CheckLinks([]extract.Link{mdLink("/docs/Missing.md")}, schema, src, root, nil)
	if len(errs) != 1 || errs[0].Rule != "link_resolve" {
		t.Errorf("missing root-anchored target should be reported, got %+v", errs)
	}
}

// Wikilinks with checks had zero coverage: every case above pins StyleMarkdown.
// This is the combination issue #62's headline reproduction exercises.
func wikiChecksSchema(c LinkChecks) LinkSchema {
	return LinkSchema{Styles: []string{extract.StyleWikilink}, Checks: &c}
}

func wikiLink(target string) extract.Link {
	return extract.Link{Target: target, Type: "reference", Style: extract.StyleWikilink, Line: 1}
}

func TestCheckLinks_WikilinkResolvesWithoutExtension(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "b.md"), "# B\n")
	writeFile(t, filepath.Join(dir, "t.md"), "body")
	schema := wikiChecksSchema(LinkChecks{Resolve: true})

	if errs := CheckLinks([]extract.Link{wikiLink("b")}, schema, filepath.Join(dir, "t.md"), dir, nil); len(errs) != 0 {
		t.Fatalf("[[b]] with b.md present must resolve, got %+v", errs)
	}
}

func TestCheckLinks_WikilinkPathQualifiedResolves(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "sub", "README.md"), "# Sub\n")
	writeFile(t, filepath.Join(dir, "t.md"), "body")
	schema := wikiChecksSchema(LinkChecks{Resolve: true})

	if errs := CheckLinks([]extract.Link{wikiLink("sub/README")}, schema, filepath.Join(dir, "t.md"), dir, nil); len(errs) != 0 {
		t.Fatalf("[[sub/README]] must resolve, got %+v", errs)
	}
}

func TestCheckLinks_WikilinkGenuinelyMissingIsBroken(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "t.md"), "body")
	schema := wikiChecksSchema(LinkChecks{Resolve: true})

	errs := CheckLinks([]extract.Link{wikiLink("nope")}, schema, filepath.Join(dir, "t.md"), dir, nil)
	if len(errs) != 1 || errs[0].Rule != "link_resolve" {
		t.Fatalf("missing wikilink target must be reported, got %+v", errs)
	}
}

func TestSlugifyHeading(t *testing.T) {
	cases := map[string]string{
		"6. Zonas inciertas (black boxes)": "6-zonas-inciertas-black-boxes",
		"Glosario de siglas":               "glosario-de-siglas",
		"  Setup & Config  ":               "setup--config",
		"Ya_valido":                        "ya_valido",
	}
	for in, want := range cases {
		if got := slugifyHeading(in); got != want {
			t.Errorf("slugifyHeading(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckLinks_AnchorValidAndInvalid(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "master.md"), "# Title\n\n## 6. Zonas inciertas (black boxes)\n\ntext\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true, Anchors: true})
	src := filepath.Join(dir, "src.md")
	cache := NewHeadingCache()

	good := extract.Link{Target: "master.md", Anchor: "6-zonas-inciertas-black-boxes", Type: "reference", Style: extract.StyleMarkdown, Line: 1}
	if errs := CheckLinks([]extract.Link{good}, schema, src, dir, cache); len(errs) != 0 {
		t.Fatalf("valid anchor rejected: %+v", errs)
	}

	bad := good
	bad.Anchor = "7-no-existe"
	errs := CheckLinks([]extract.Link{bad}, schema, src, dir, cache)
	if len(errs) != 1 || errs[0].Rule != "link_anchor" {
		t.Fatalf("expected 1 link_anchor error, got %+v", errs)
	}
}

func TestCheckLinks_AnchorNilCacheStillWorks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "master.md"), "## Intro\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true, Anchors: true})
	link := extract.Link{Target: "master.md", Anchor: "intro", Type: "reference", Style: extract.StyleMarkdown, Line: 1}
	if errs := CheckLinks([]extract.Link{link}, schema, filepath.Join(dir, "src.md"), dir, nil); len(errs) != 0 {
		t.Fatalf("nil cache should parse on the fly: %+v", errs)
	}
}

func TestCheckLinks_AnchorCacheReuse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "target.md"), "## Section1\n\n## Section2\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true, Anchors: true})
	cache := NewHeadingCache()
	src := filepath.Join(dir, "src.md")

	links := []extract.Link{
		{Target: "target.md", Anchor: "section1", Type: "reference", Style: extract.StyleMarkdown, Line: 1},
		{Target: "target.md", Anchor: "section2", Type: "reference", Style: extract.StyleMarkdown, Line: 2},
	}

	if errs := CheckLinks(links, schema, src, dir, cache); len(errs) != 0 {
		t.Fatalf("both anchors should pass; got %+v", errs)
	}
	if len(cache.slugs) != 1 {
		t.Errorf("cache.slugs = %d entries, want 1 (shared target)", len(cache.slugs))
	}
}
