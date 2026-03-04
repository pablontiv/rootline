package extract

import (
	"testing"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/extension"
	"github.com/yuin/goldmark/text"
)

func parseSections(body string) []Section {
	source := []byte(body)
	reader := text.NewReader(source)
	node := goldmark.DefaultParser().Parse(reader)
	return ExtractSections(node, source)
}

// parseWithTableExt parses markdown with the table extension enabled.
func parseWithTableExt(body string) ([]CodeBlock, []Table) {
	source := []byte(body)
	md := goldmark.New(goldmark.WithExtensions(extension.Table))
	reader := text.NewReader(source)
	node := md.Parser().Parse(reader)
	return ExtractCodeBlocks(node, source), ExtractTables(node, source)
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

func TestExtractCodeBlocks_Basic(t *testing.T) {
	body := "Some text\n\n```go\nfunc main() {}\n```\n\n```yaml\nkey: value\n```\n"
	blocks, _ := parseWithTableExt(body)

	if len(blocks) != 2 {
		t.Fatalf("expected 2 code blocks, got %d", len(blocks))
	}
	if blocks[0].Language != "go" {
		t.Errorf("block 0 language: got %q", blocks[0].Language)
	}
	if blocks[0].Content != "func main() {}\n" {
		t.Errorf("block 0 content: got %q", blocks[0].Content)
	}
	if blocks[1].Language != "yaml" {
		t.Errorf("block 1 language: got %q", blocks[1].Language)
	}
	if blocks[1].Content != "key: value\n" {
		t.Errorf("block 1 content: got %q", blocks[1].Content)
	}
}

func TestExtractCodeBlocks_NoLanguage(t *testing.T) {
	body := "Text\n\n```\nplain content\n```\n"
	blocks, _ := parseWithTableExt(body)

	if len(blocks) != 1 {
		t.Fatalf("expected 1 code block, got %d", len(blocks))
	}
	if blocks[0].Language != "" {
		t.Errorf("expected empty language, got %q", blocks[0].Language)
	}
}

func TestExtractCodeBlocks_IgnoresInlineCode(t *testing.T) {
	body := "Some `inline code` here.\n"
	blocks, _ := parseWithTableExt(body)

	if len(blocks) != 0 {
		t.Fatalf("expected 0 code blocks (inline code ignored), got %d", len(blocks))
	}
}

func TestExtractTables_Basic(t *testing.T) {
	body := "| H1 | H2 |\n|---|---|\n| a | b |\n| c | d |\n"
	_, tables := parseWithTableExt(body)

	if len(tables) != 1 {
		t.Fatalf("expected 1 table, got %d", len(tables))
	}
	if len(tables[0].Headers) != 2 || tables[0].Headers[0] != "H1" || tables[0].Headers[1] != "H2" {
		t.Errorf("headers: got %v", tables[0].Headers)
	}
	if len(tables[0].Rows) != 2 {
		t.Fatalf("expected 2 rows, got %d", len(tables[0].Rows))
	}
	if tables[0].Rows[0][0] != "a" || tables[0].Rows[0][1] != "b" {
		t.Errorf("row 0: got %v", tables[0].Rows[0])
	}
	if tables[0].Rows[1][0] != "c" || tables[0].Rows[1][1] != "d" {
		t.Errorf("row 1: got %v", tables[0].Rows[1])
	}
}

func TestExtractTables_Mixed(t *testing.T) {
	body := "Some text\n\n```go\ncode\n```\n\n| Col |\n|---|\n| val |\n"
	blocks, tables := parseWithTableExt(body)

	if len(blocks) != 1 {
		t.Errorf("expected 1 code block, got %d", len(blocks))
	}
	if len(tables) != 1 {
		t.Errorf("expected 1 table, got %d", len(tables))
	}
}
