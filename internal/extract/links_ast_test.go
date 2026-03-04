package extract

import (
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func parseAST(body string) []Link {
	source := []byte(body)
	reader := text.NewReader(source)
	parser := goldmark.DefaultParser()
	node := parser.Parse(reader)
	return ParseLinksAST(node, source)
}

func TestParseLinksAST_Equivalence(t *testing.T) {
	cases := []struct {
		name string
		body string
	}{
		{"simple reference", "see [[T003]]"},
		{"typed link", "[[blocks:T003]]"},
		{"parent link", "[[parent:../feature]]"},
		{"no links", "no links here"},
		{"multiple links", "[[a]] and [[b:c]]"},
		{"line numbers", "first line\n[[link1]]\nthird\n[[link2:x]]"},
		{"mixed code and links", "see [[real]] and `[[fake]]` and [[blocks:other]]"},
		{"empty body", ""},
		{"fenced code block", "before\n```\n[[ignored]]\n```\nafter"},
		{"tilde fenced code block", "before\n~~~\n[[ignored]]\n~~~\nafter"},
		{"inline code", "use `[[ignored]]` here"},
	}

	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			regex := ParseLinks(tc.body)
			astLinks := parseAST(tc.body)

			if len(regex) != len(astLinks) {
				t.Fatalf("count mismatch: ParseLinks=%d, ParseLinksAST=%d\nParseLinks: %+v\nParseLinksAST: %+v",
					len(regex), len(astLinks), regex, astLinks)
			}

			for i := range regex {
				if regex[i].Target != astLinks[i].Target {
					t.Errorf("[%d] target mismatch: ParseLinks=%q, ParseLinksAST=%q", i, regex[i].Target, astLinks[i].Target)
				}
				if regex[i].Type != astLinks[i].Type {
					t.Errorf("[%d] type mismatch: ParseLinks=%q, ParseLinksAST=%q", i, regex[i].Type, astLinks[i].Type)
				}
				if regex[i].Line != astLinks[i].Line {
					t.Errorf("[%d] line mismatch: ParseLinks=%d, ParseLinksAST=%d", i, regex[i].Line, astLinks[i].Line)
				}
			}
		})
	}
}

func TestParseLinksAST_FencedCodeBlockExclusion(t *testing.T) {
	body := "real [[link1]]\n```go\n[[inside-code]]\n```\nreal [[link2]]"
	links := parseAST(body)

	if len(links) != 2 {
		t.Fatalf("expected 2 links (code block excluded), got %d: %+v", len(links), links)
	}
	if links[0].Target != "link1" {
		t.Errorf("link 0: expected target 'link1', got %q", links[0].Target)
	}
	if links[1].Target != "link2" {
		t.Errorf("link 1: expected target 'link2', got %q", links[1].Target)
	}
}
