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
	errs := CheckLinks([]extract.Link{mdLink("missing.md")}, schema, "/nonexistent/src.md", nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors without checks, got %+v", errs)
	}
}

func TestCheckLinks_EncodingRejectsRawSpaces(t *testing.T) {
	schema := mdChecksSchema(LinkChecks{Encoding: true})
	errs := CheckLinks([]extract.Link{mdLink("my file.md")}, schema, "/tmp/src.md", nil)
	if len(errs) != 1 || errs[0].Rule != "link_encoding" {
		t.Fatalf("expected 1 link_encoding error, got %+v", errs)
	}
}

func TestCheckLinks_ResolveExisting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	errs := CheckLinks([]extract.Link{mdLink("guides/setup.md")}, schema, filepath.Join(dir, "src.md"), nil)
	if len(errs) != 0 {
		t.Fatalf("expected resolve to pass, got %+v", errs)
	}
}

func TestCheckLinks_ResolveBrokenWithSuggestion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	errs := CheckLinks([]extract.Link{mdLink("guides/setpu.md")}, schema, filepath.Join(dir, "src.md"), nil)
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
	errs := CheckLinks([]extract.Link{mdLink("guides/setup.md")}, schema, filepath.Join(dir, "src.md"), nil)
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
	if errs := CheckLinks([]extract.Link{mdLink("guides/")}, schema, src, nil); len(errs) != 0 {
		t.Fatalf("dir with README should resolve, got %+v", errs)
	}
	if errs := CheckLinks([]extract.Link{mdLink("empty/")}, schema, src, nil); len(errs) != 1 {
		t.Fatalf("dir without README should be broken, got %+v", errs)
	}
}

func TestCheckLinks_ResolveDecodesPercent20(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "my page.md"), "# P\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	errs := CheckLinks([]extract.Link{mdLink("my%20page.md")}, schema, filepath.Join(dir, "src.md"), nil)
	if len(errs) != 0 {
		t.Fatalf("%%20 target should resolve, got %+v", errs)
	}
}

func TestCheckLinks_SkipsAbsoluteAndFilteredStyles(t *testing.T) {
	schema := mdChecksSchema(LinkChecks{Resolve: true, Encoding: true})
	links := []extract.Link{
		mdLink("/Root/Page.md"), // absolute: skipped
		{Target: "no such file", Type: "reference", Style: extract.StyleWikilink, Line: 1}, // filtered style
	}
	if errs := CheckLinks(links, schema, "/tmp/src.md", nil); len(errs) != 0 {
		t.Fatalf("expected no errors, got %+v", errs)
	}
}
