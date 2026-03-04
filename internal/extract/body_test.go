package extract

import (
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func parseSections(body string) []Section {
	source := []byte(body)
	reader := text.NewReader(source)
	node := goldmark.DefaultParser().Parse(reader)
	return ExtractSections(node, source)
}

func TestExtractSections_MultipleHeadings(t *testing.T) {
	body := "## A\n\nContent A\n\n### B\n\nContent B\n\n## C\n\nContent C\n"
	sections := parseSections(body)

	if len(sections) != 3 {
		t.Fatalf("expected 3 sections, got %d: %+v", len(sections), sections)
	}

	if sections[0].Heading != "A" || sections[0].Level != 2 {
		t.Errorf("section 0: got %+v", sections[0])
	}
	if sections[0].Content != "Content A" {
		t.Errorf("section 0 content: got %q", sections[0].Content)
	}

	if sections[1].Heading != "B" || sections[1].Level != 3 {
		t.Errorf("section 1: got %+v", sections[1])
	}
	if sections[1].Content != "Content B" {
		t.Errorf("section 1 content: got %q", sections[1].Content)
	}

	if sections[2].Heading != "C" || sections[2].Level != 2 {
		t.Errorf("section 2: got %+v", sections[2])
	}
	if sections[2].Content != "Content C" {
		t.Errorf("section 2 content: got %q", sections[2].Content)
	}
}

func TestExtractSections_HeadingInCodeBlock(t *testing.T) {
	body := "## Real\n\nSome text\n\n```\n## Fake\n```\n\n## Also Real\n\nMore text\n"
	sections := parseSections(body)

	if len(sections) != 2 {
		t.Fatalf("expected 2 sections (code block heading excluded), got %d: %+v", len(sections), sections)
	}
	if sections[0].Heading != "Real" {
		t.Errorf("section 0 heading: got %q", sections[0].Heading)
	}
	if sections[1].Heading != "Also Real" {
		t.Errorf("section 1 heading: got %q", sections[1].Heading)
	}
}

func TestExtractSections_NoHeadings(t *testing.T) {
	body := "Just a paragraph.\n\nAnother paragraph.\n"
	sections := parseSections(body)

	if len(sections) != 1 {
		t.Fatalf("expected 1 section, got %d", len(sections))
	}
	if sections[0].Heading != "" {
		t.Errorf("heading should be empty, got %q", sections[0].Heading)
	}
	if sections[0].Level != 0 {
		t.Errorf("level should be 0, got %d", sections[0].Level)
	}
	if sections[0].StartLine != 1 {
		t.Errorf("start_line should be 1, got %d", sections[0].StartLine)
	}
}

func TestExtractSections_EmptyBody(t *testing.T) {
	sections := parseSections("")
	if len(sections) != 1 {
		t.Fatalf("expected 1 section for empty body, got %d", len(sections))
	}
}
