package extract

import (
	"strings"
	"testing"
)

func TestParseLinks_SimpleReference(t *testing.T) {
	links := ParseLinks("see [[T003]]")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Target != "T003" || links[0].Type != "reference" || links[0].Line != 1 {
		t.Errorf("got %+v", links[0])
	}
}

func TestParseLinks_TypedLink(t *testing.T) {
	links := ParseLinks("[[blocks:T003]]")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Target != "T003" || links[0].Type != "blocks" || links[0].Line != 1 {
		t.Errorf("got %+v", links[0])
	}
}

func TestParseLinks_ParentLink(t *testing.T) {
	links := ParseLinks("[[parent:../feature]]")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Target != "../feature" || links[0].Type != "parent" {
		t.Errorf("got %+v", links[0])
	}
}

func TestParseLinks_FencedCodeBlock(t *testing.T) {
	body := "before\n```\n[[ignored]]\n```\nafter"
	links := ParseLinks(body)
	if len(links) != 0 {
		t.Errorf("expected 0 links in fenced code block, got %d: %+v", len(links), links)
	}
}

func TestParseLinks_TildeFencedCodeBlock(t *testing.T) {
	body := "before\n~~~\n[[ignored]]\n~~~\nafter"
	links := ParseLinks(body)
	if len(links) != 0 {
		t.Errorf("expected 0 links in tilde fenced block, got %d: %+v", len(links), links)
	}
}

func TestParseLinks_InlineCode(t *testing.T) {
	links := ParseLinks("use `[[ignored]]` here")
	if len(links) != 0 {
		t.Errorf("expected 0 links in inline code, got %d: %+v", len(links), links)
	}
}

func TestContainsWikilink(t *testing.T) {
	if !ContainsWikilink("see [[T003]] for details") {
		t.Error("expected true for string with wikilink")
	}
	if ContainsWikilink("no links here") {
		t.Error("expected false for string without wikilink")
	}
	if ContainsWikilink("") {
		t.Error("expected false for empty string")
	}
}

func TestParseLinks_NoLinks(t *testing.T) {
	links := ParseLinks("no links here")
	if len(links) != 0 {
		t.Errorf("expected 0 links, got %d", len(links))
	}
}

func TestParseLinks_MultipleLinks(t *testing.T) {
	links := ParseLinks("[[a]] and [[b:c]]")
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].Target != "a" || links[0].Type != "reference" {
		t.Errorf("link 0: got %+v", links[0])
	}
	if links[1].Target != "c" || links[1].Type != "b" {
		t.Errorf("link 1: got %+v", links[1])
	}
}

func TestParseLinks_LineNumbers(t *testing.T) {
	body := "first line\n[[link1]]\nthird\n[[link2:x]]"
	links := ParseLinks(body)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].Line != 2 {
		t.Errorf("link 0 line: expected 2, got %d", links[0].Line)
	}
	if links[1].Line != 4 {
		t.Errorf("link 1 line: expected 4, got %d", links[1].Line)
	}
}

func TestParseLinks_MixedCodeAndLinks(t *testing.T) {
	body := "see [[real]] and `[[fake]]` and [[blocks:other]]"
	links := ParseLinks(body)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d: %+v", len(links), links)
	}
	if links[0].Target != "real" {
		t.Errorf("link 0: got %+v", links[0])
	}
	if links[1].Target != "other" || links[1].Type != "blocks" {
		t.Errorf("link 1: got %+v", links[1])
	}
}

func TestParseLinks_EmptyBody(t *testing.T) {
	links := ParseLinks("")
	if len(links) != 0 {
		t.Errorf("expected 0 links for empty body, got %d", len(links))
	}
}

func TestExtract_FrontmatterLinks(t *testing.T) {
	content := []byte(`---
spec: "[[specs/my-design]]"
backlog_ids:
  - "[[B039-velero]]"
  - "[[B040-probe]]"
---
# Title

Body with [[body-link]].
`)
	ext := &MarkdownExtractor{}
	rec, err := ext.Extract("test.md", content)
	if err != nil {
		t.Fatalf("Extract error: %v", err)
	}
	if len(rec.Links) != 4 {
		t.Fatalf("expected 4 links, got %d: %+v", len(rec.Links), rec.Links)
	}

	fmLinks := 0
	bodyLinks := 0
	for _, l := range rec.Links {
		if strings.HasPrefix(l.Source, "frontmatter:") {
			fmLinks++
		} else if l.Source == "body" {
			bodyLinks++
		}
	}
	if fmLinks != 3 {
		t.Errorf("expected 3 frontmatter links, got %d", fmLinks)
	}
	if bodyLinks != 1 {
		t.Errorf("expected 1 body link, got %d", bodyLinks)
	}
}

func TestParseFrontmatterLinks_StringValue(t *testing.T) {
	fm := map[string]any{
		"spec": "[[specs/2026-03-17-backup-design]]",
	}
	links := ParseFrontmatterLinks(fm)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Target != "specs/2026-03-17-backup-design" {
		t.Errorf("expected target 'specs/2026-03-17-backup-design', got %q", links[0].Target)
	}
	if links[0].Type != "reference" {
		t.Errorf("expected type 'reference', got %q", links[0].Type)
	}
	if links[0].Source != "frontmatter:spec" {
		t.Errorf("expected source 'frontmatter:spec', got %q", links[0].Source)
	}
}

func TestParseFrontmatterLinks_SliceValue(t *testing.T) {
	fm := map[string]any{
		"backlog_ids": []any{
			"[[B039-velero-backup]]",
			"[[B040-probe-failure]]",
		},
	}
	links := ParseFrontmatterLinks(fm)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	if links[0].Target != "B039-velero-backup" {
		t.Errorf("expected target 'B039-velero-backup', got %q", links[0].Target)
	}
	if links[0].Source != "frontmatter:backlog_ids" {
		t.Errorf("expected source 'frontmatter:backlog_ids', got %q", links[0].Source)
	}
	if links[1].Target != "B040-probe-failure" {
		t.Errorf("expected target 'B040-probe-failure', got %q", links[1].Target)
	}
}

func TestParseFrontmatterLinks_TypedLink(t *testing.T) {
	fm := map[string]any{
		"depends": "[[blocks:other-feature]]",
	}
	links := ParseFrontmatterLinks(fm)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Type != "blocks" {
		t.Errorf("expected type 'blocks', got %q", links[0].Type)
	}
	if links[0].Target != "other-feature" {
		t.Errorf("expected target 'other-feature', got %q", links[0].Target)
	}
}

func TestParseFrontmatterLinks_NoLinks(t *testing.T) {
	fm := map[string]any{
		"tipo":   "plan",
		"estado": "pendiente",
	}
	links := ParseFrontmatterLinks(fm)
	if len(links) != 0 {
		t.Fatalf("expected 0 links, got %d", len(links))
	}
}

func TestParseFrontmatterLinks_NilMap(t *testing.T) {
	links := ParseFrontmatterLinks(nil)
	if len(links) != 0 {
		t.Fatalf("expected 0 links, got %d", len(links))
	}
}

func TestParseLinks_SourceIsBody(t *testing.T) {
	links := ParseLinks("see [[T003]]")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Source != "body" {
		t.Errorf("expected Source='body', got %q", links[0].Source)
	}
}
