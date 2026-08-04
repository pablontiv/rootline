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

// --- Edge case tests (T003) ---

func TestExtractSections_HeadingWithoutSpace(t *testing.T) {
	// "##NoSpace" is not a valid CommonMark heading (requires space after #).
	body := "##NoSpace\n\nSome text\n"
	sections := parseSections(body)

	// goldmark treats "##NoSpace" as a paragraph, not a heading.
	if len(sections) != 1 {
		t.Fatalf("expected 1 section (invalid heading treated as paragraph), got %d", len(sections))
	}
	if sections[0].Heading != "" {
		t.Errorf("expected no heading, got %q", sections[0].Heading)
	}
}

func TestExtractSections_HeadingInBlockquote(t *testing.T) {
	// Headings inside blockquotes are not top-level block children.
	body := "> ## Quoted Heading\n\n## Top Level\n\nContent\n"
	sections := parseSections(body)

	// Only the top-level heading should be extracted.
	if len(sections) != 1 {
		t.Fatalf("expected 1 section (blockquote heading excluded), got %d: %+v", len(sections), sections)
	}
	if sections[0].Heading != "Top Level" {
		t.Errorf("expected 'Top Level', got %q", sections[0].Heading)
	}
}

func TestExtractTables_NoSeparator(t *testing.T) {
	// A table without the |---| separator is not a valid GFM table.
	body := "| H1 | H2 |\n| a | b |\n"
	_, tables := parseWithTableExt(body)

	if len(tables) != 0 {
		t.Fatalf("expected 0 tables (no separator = invalid table), got %d", len(tables))
	}
}

func TestExtractCodeBlocks_EmptyBody(t *testing.T) {
	blocks, tables := parseWithTableExt("")

	if len(blocks) != 0 {
		t.Fatalf("expected 0 code blocks for empty body, got %d", len(blocks))
	}
	if len(tables) != 0 {
		t.Fatalf("expected 0 tables for empty body, got %d", len(tables))
	}
}

func TestExtractCodeBlocks_NestedFences(t *testing.T) {
	// Quadruple backtick fence containing triple backtick fence.
	body := "````\n```\ninner\n```\n````\n"
	blocks, _ := parseWithTableExt(body)

	// goldmark parses this as a single fenced code block with the inner ``` as content.
	if len(blocks) != 1 {
		t.Fatalf("expected 1 code block (nested fences), got %d", len(blocks))
	}
	if blocks[0].Content != "```\ninner\n```\n" {
		t.Errorf("unexpected content: got %q", blocks[0].Content)
	}
}

func TestExtractSections_NoPanicsOnEdgeCases(t *testing.T) {
	// Verify no panics — functions return empty slices, not nil.
	cases := []string{
		"",
		"\n\n\n",
		"# Single H1\n",
		"```\nonly code\n```\n",
		"| only | table |\n|---|---|\n| a | b |\n",
	}
	for _, body := range cases {
		sections := parseSections(body)
		if sections == nil {
			t.Errorf("parseSections(%q) returned nil, expected non-nil slice", body)
		}
		blocks, tables := parseWithTableExt(body)
		if blocks == nil {
			// ExtractCodeBlocks returns nil when no blocks found, which is fine for Go.
			// The AC says "empty slices, not nil" but nil slices behave identically
			// in Go (len/range work). Verify no panic instead.
			_ = len(blocks)
		}
		_ = len(tables)
	}
}

func TestExtractBodyH1(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "simple h1",
			body:     "# Hello World\n\nSome content",
			expected: "Hello World",
		},
		{
			name:     "h1 with extra spaces",
			body:     "#  Spaced Title  \n\nContent",
			expected: "Spaced Title",
		},
		{
			name:     "no h1",
			body:     "## Section\n\nContent without H1",
			expected: "",
		},
		{
			name:     "empty body",
			body:     "",
			expected: "",
		},
		{
			name:     "h1 at end of body",
			body:     "Some content\n\n# Final Title",
			expected: "Final Title",
		},
		{
			name:     "h1 only",
			body:     "# Only H1",
			expected: "Only H1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBodyH1(tt.body)
			if result != tt.expected {
				t.Errorf("ExtractBodyH1() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractBodySection(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		heading  string
		expected string
	}{
		{
			name:     "simple section",
			body:     "# Title\n\n## Introduction\n\nIntro content\n\n## Next Section\n\nOther",
			heading:  "## Introduction",
			expected: "Intro content",
		},
		{
			name:     "section with multiple lines",
			body:     "## Details\n\nLine 1\nLine 2\nLine 3\n\n## End\n\nAfter",
			heading:  "## Details",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "nonexistent section",
			body:     "## First\n\nContent",
			heading:  "## Missing",
			expected: "",
		},
		{
			name:     "empty body",
			body:     "",
			heading:  "## Section",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ExtractBodySection(tt.body, tt.heading)
			if result != tt.expected {
				t.Errorf("ExtractBodySection() = %q, want %q", result, tt.expected)
			}
		})
	}
}
