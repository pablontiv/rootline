package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestFilterLinksByStyles_DefaultDropsMarkdown(t *testing.T) {
	root := t.TempDir() // no .stem at all
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := &extract.Record{Path: "a.md", Links: []extract.Link{
		{Target: "[[w]]", Style: extract.StyleWikilink},
		{Target: "b.md", Style: extract.StyleMarkdown},
		{Target: "legacy"}, // style-less → wikilink
	}}
	if err := FilterLinksByStyles([]*extract.Record{rec}, root); err != nil {
		t.Fatalf("FilterLinksByStyles: %v", err)
	}
	if len(rec.Links) != 2 {
		t.Fatalf("Links = %+v, want wikilink + legacy only", rec.Links)
	}
}

func TestFilterLinksByStyles_StemDeclaresMarkdown(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	stem := "version: 2\nlinks:\n  styles: [markdown]\n"
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := &extract.Record{Path: "a.md", Links: []extract.Link{
		{Target: "[[w]]", Style: extract.StyleWikilink},
		{Target: "b.md", Style: extract.StyleMarkdown},
	}}
	if err := FilterLinksByStyles([]*extract.Record{rec}, root); err != nil {
		t.Fatalf("FilterLinksByStyles: %v", err)
	}
	if len(rec.Links) != 1 || rec.Links[0].Style != extract.StyleMarkdown {
		t.Fatalf("Links = %+v, want markdown only", rec.Links)
	}
}

func TestFilterLinksByStylesPropagatesInvalidRecordSchema(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(`version: 2
root: true
schema:
  id:
    type: sequence
    match:
      "bad*": {prefix: BAD, digits: 2.0}
links:
  styles: [markdown]
`), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := &extract.Record{Path: "bad.md", Links: []extract.Link{{Target: "x", Style: extract.StyleMarkdown}}}

	err := FilterLinksByStyles([]*extract.Record{rec}, root)
	if err == nil || !strings.Contains(err.Error(), "bad.md") || !strings.Contains(err.Error(), "digits") {
		t.Fatalf("FilterLinksByStyles error = %v, want schema digits cause", err)
	}
}
