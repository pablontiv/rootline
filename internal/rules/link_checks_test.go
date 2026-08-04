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

// A schema with no checks block still resolves: broken-target detection is
// always on, matching graph. Only anchors and encoding need declaring.
func TestCheckLinks_NilChecksStillResolves(t *testing.T) {
	schema := LinkSchema{Styles: []string{extract.StyleMarkdown}}
	errs := CheckLinks([]extract.Link{mdLink("missing.md")}, schema, "/nonexistent/src.md", "", nil)
	if len(errs) != 1 || errs[0].Rule != "link_resolve" {
		t.Fatalf("expected link_resolve without a checks block, got %+v", errs)
	}
}

func TestCheckLinks_EncodingRejectsRawSpaces(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "my file.md"), "# F\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	// The target exists, so only the encoding rule may fire.
	schema := mdChecksSchema(LinkChecks{Encoding: true})
	errs := CheckLinks([]extract.Link{mdLink("my file.md")}, schema, filepath.Join(dir, "src.md"), dir, nil)
	if len(errs) != 1 || errs[0].Rule != "link_encoding" {
		t.Fatalf("expected 1 link_encoding error, got %+v", errs)
	}
}

func TestCheckLinks_ResolveExisting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: boolPtr(true)})
	errs := CheckLinks([]extract.Link{mdLink("guides/setup.md")}, schema, filepath.Join(dir, "src.md"), dir, nil)
	if len(errs) != 0 {
		t.Fatalf("expected resolve to pass, got %+v", errs)
	}
}

func TestCheckLinks_ResolveBrokenWithSuggestion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: boolPtr(true)})
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
	schema := mdChecksSchema(LinkChecks{Resolve: boolPtr(true)})
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
	schema := mdChecksSchema(LinkChecks{Resolve: boolPtr(true)})
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
	schema := mdChecksSchema(LinkChecks{Resolve: boolPtr(true)})
	errs := CheckLinks([]extract.Link{mdLink("my%20page.md")}, schema, filepath.Join(dir, "src.md"), dir, nil)
	if len(errs) != 0 {
		t.Fatalf("%%20 target should resolve, got %+v", errs)
	}
}

func TestCheckLinks_SkipsFilteredStyles(t *testing.T) {
	schema := mdChecksSchema(LinkChecks{Resolve: boolPtr(true), Encoding: true})
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
	schema := mdChecksSchema(LinkChecks{Resolve: boolPtr(true)})

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
	schema := wikiChecksSchema(LinkChecks{Resolve: boolPtr(true)})

	if errs := CheckLinks([]extract.Link{wikiLink("b")}, schema, filepath.Join(dir, "t.md"), dir, nil); len(errs) != 0 {
		t.Fatalf("[[b]] with b.md present must resolve, got %+v", errs)
	}
}

func TestCheckLinks_WikilinkPathQualifiedResolves(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "sub", "README.md"), "# Sub\n")
	writeFile(t, filepath.Join(dir, "t.md"), "body")
	schema := wikiChecksSchema(LinkChecks{Resolve: boolPtr(true)})

	if errs := CheckLinks([]extract.Link{wikiLink("sub/README")}, schema, filepath.Join(dir, "t.md"), dir, nil); len(errs) != 0 {
		t.Fatalf("[[sub/README]] must resolve, got %+v", errs)
	}
}

func TestCheckLinks_WikilinkGenuinelyMissingIsBroken(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "t.md"), "body")
	schema := wikiChecksSchema(LinkChecks{Resolve: boolPtr(true)})

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
	schema := mdChecksSchema(LinkChecks{Resolve: boolPtr(true), Anchors: true})
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
	schema := mdChecksSchema(LinkChecks{Resolve: boolPtr(true), Anchors: true})
	link := extract.Link{Target: "master.md", Anchor: "intro", Type: "reference", Style: extract.StyleMarkdown, Line: 1}
	if errs := CheckLinks([]extract.Link{link}, schema, filepath.Join(dir, "src.md"), dir, nil); len(errs) != 0 {
		t.Fatalf("nil cache should parse on the fly: %+v", errs)
	}
}

func TestCheckLinks_AnchorCacheReuse(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "target.md"), "## Section1\n\n## Section2\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: boolPtr(true), Anchors: true})
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

// With basename fallback on, validate cannot decide a bare cross-directory
// target: CheckLinks sees one record and has no index to match against, while
// graph does. Reporting it broken would be wrong (it may well resolve) and
// skipping it silently would recreate the disagreement issue #62 is about, so
// validate says explicitly that it could not verify this one.
func TestCheckLinks_BasenameFallbackTargetIsUnverifiable(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "t.md"), "body")
	schema := wikiChecksSchema(LinkChecks{Resolve: boolPtr(true)})
	schema.BasenameFallback = true

	errs := CheckLinks([]extract.Link{wikiLink("far")}, schema, filepath.Join(dir, "t.md"), dir, nil)
	if len(errs) != 1 {
		t.Fatalf("expected 1 diagnostic, got %+v", errs)
	}
	if errs[0].Rule != "link_unverifiable" {
		t.Errorf("Rule = %q, want link_unverifiable", errs[0].Rule)
	}
	if errs[0].Severity != "warn" {
		t.Errorf("Severity = %q, want warn — the link may well resolve", errs[0].Severity)
	}
	// Severity is not cosmetic: only "warn" is routed to Warnings, so any
	// other value would make the record invalid.
	if r := NewValidationResult("t.md", errs); !r.Valid || len(r.Warnings) != 1 {
		t.Errorf("unverifiable link must not invalidate the record: valid=%v errors=%d warnings=%d",
			r.Valid, len(r.Errors), len(r.Warnings))
	}
}

// Without the knob, the same target is an ordinary broken link.
func TestCheckLinks_WithoutFallbackBareTargetIsBroken(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "t.md"), "body")
	errs := CheckLinks([]extract.Link{wikiLink("far")}, wikiChecksSchema(LinkChecks{Resolve: boolPtr(true)}),
		filepath.Join(dir, "t.md"), dir, nil)
	if len(errs) != 1 || errs[0].Rule != "link_resolve" || errs[0].Severity != "error" {
		t.Fatalf("expected a link_resolve error, got %+v", errs)
	}
}

// Broken-target detection is always on. graph --check has always failed on a
// broken link with no opt-in; validate stayed silent unless links.checks
// declared resolve, so the two commands disagreed on the one property both
// claim to check (issue #62 sub-defect 2). validate now matches graph.
func TestCheckLinks_ResolveIsOnByDefault(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := LinkSchema{Styles: []string{extract.StyleMarkdown}} // no checks block at all

	errs := CheckLinks([]extract.Link{mdLink("missing.md")}, schema, filepath.Join(dir, "src.md"), dir, nil)
	if len(errs) != 1 || errs[0].Rule != "link_resolve" {
		t.Fatalf("expected link_resolve without a checks block, got %+v", errs)
	}
}

// A repository that genuinely wants dangling links opts out explicitly.
func TestCheckLinks_ResolveCanBeDisabled(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	off := false
	schema := LinkSchema{Styles: []string{extract.StyleMarkdown}, Checks: &LinkChecks{Resolve: &off}}

	if errs := CheckLinks([]extract.Link{mdLink("missing.md")}, schema, filepath.Join(dir, "src.md"), dir, nil); len(errs) != 0 {
		t.Fatalf("resolve: false must silence the check, got %+v", errs)
	}
}

// Anchors and encoding stay opt-in; only resolve flipped.
func TestCheckLinks_AnchorsAndEncodingStayOptIn(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "my file.md"), "# F\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := LinkSchema{Styles: []string{extract.StyleMarkdown}}

	errs := CheckLinks([]extract.Link{mdLink("my file.md")}, schema, filepath.Join(dir, "src.md"), dir, nil)
	for _, e := range errs {
		if e.Rule == "link_encoding" {
			t.Errorf("encoding must stay opt-in, got %+v", errs)
		}
	}
}

// boolPtr is a helper for the tri-state links.checks.resolve field.
func boolPtr(b bool) *bool { return &b }
