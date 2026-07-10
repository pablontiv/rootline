package extract

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// ParseLinksAST extracts wiki-links by walking a goldmark AST.
// It skips FencedCodeBlock and CodeBlock nodes, making it more precise
// than the regex-based ParseLinks for code-heavy documents.
func ParseLinksAST(node ast.Node, source []byte) []Link {
	var links []Link
	walkBlocks(node, source, &links)
	return links
}

// walkBlocks iterates over block-level nodes, skipping code blocks,
// and extracts wiki-links from the raw text of each block's lines.
func walkBlocks(n ast.Node, source []byte, links *[]Link) {
	for child := n.FirstChild(); child != nil; child = child.NextSibling() {
		switch child.Kind() {
		case ast.KindFencedCodeBlock, ast.KindCodeBlock:
			continue
		}

		if child.HasChildren() && child.Type() == ast.TypeBlock {
			// For leaf blocks (paragraphs, headings, etc.) that have lines,
			// extract text from their raw lines.
			lines := child.Lines()
			if lines.Len() > 0 {
				extractLinksFromLines(child, source, links)
			}
			// Also recurse into nested blocks (e.g., blockquotes, list items).
			walkBlocks(child, source, links)
		}
	}
}

// extractLinksFromLines extracts wiki-links and markdown links from a block node's raw source lines,
// skipping inline code spans.
func extractLinksFromLines(block ast.Node, source []byte, links *[]Link) {
	lines := block.Lines()
	for i := 0; i < lines.Len(); i++ {
		seg := lines.At(i)
		lineText := string(seg.Value(source))
		lineNum := lineFromOffset(source, seg.Start)

		// Strip inline code spans before matching (same as ParseLinks).
		cleaned := removeInlineCode(lineText)

		for _, match := range wikilinkRe.FindAllStringSubmatch(cleaned, -1) {
			inner := match[1]
			link := Link{Line: lineNum, Style: StyleWikilink}
			if idx := strings.Index(inner, ":"); idx > 0 {
				link.Type = inner[:idx]
				link.Target = inner[idx+1:]
			} else {
				link.Type = "reference"
				link.Target = inner
			}
			*links = append(*links, link)
		}

		for _, match := range markdownLinkRe.FindAllStringSubmatch(cleaned, -1) {
			if match[1] == "!" {
				continue // image
			}
			link, ok := parseMarkdownDestination(match[2])
			if !ok {
				continue
			}
			link.Line = lineNum
			link.Source = "body"
			*links = append(*links, link)
		}
	}
}

// lineFromOffset converts a byte offset to a 1-based line number.
func lineFromOffset(source []byte, offset int) int {
	line := 1
	for i := 0; i < offset && i < len(source); i++ {
		if source[i] == '\n' {
			line++
		}
	}
	return line
}
