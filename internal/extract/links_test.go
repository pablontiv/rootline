package extract

import (
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

func TestParseLinks_SourceIsBody(t *testing.T) {
	links := ParseLinks("see [[T003]]")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Source != "body" {
		t.Errorf("expected Source='body', got %q", links[0].Source)
	}
}
