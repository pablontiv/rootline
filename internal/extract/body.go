package extract

import (
	"strings"

	"github.com/yuin/goldmark/ast"
)

// Section represents a heading-delimited section in a markdown body.
type Section struct {
	Heading   string `json:"heading"`
	Level     int    `json:"level"`
	Content   string `json:"content"`
	StartLine int    `json:"start_line"`
}

// ExtractSections splits a markdown body into sections delimited by headings.
// It walks the AST to identify headings, so headings inside code blocks are ignored.
// If the document has no headings, a single section with the entire body is returned.
func ExtractSections(node ast.Node, source []byte) []Section {
	type headingInfo struct {
		text      string
		level     int
		startLine int
		endOffset int // byte offset after the heading line
	}

	var headings []headingInfo

	// Collect headings from the AST (top-level block children only).
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() != ast.KindHeading {
			continue
		}
		h := child.(*ast.Heading)
		// Extract heading text from child text nodes.
		var text strings.Builder
		for c := h.FirstChild(); c != nil; c = c.NextSibling() {
			if c.Kind() == ast.KindText {
				seg := c.(*ast.Text).Segment
				text.Write(seg.Value(source))
			}
		}

		lines := h.Lines()
		startLine := 0
		endOffset := 0
		if lines.Len() > 0 {
			seg := lines.At(0)
			startLine = lineFromOffset(source, seg.Start)
			lastSeg := lines.At(lines.Len() - 1)
			endOffset = lastSeg.Stop
		}

		headings = append(headings, headingInfo{
			text:      text.String(),
			level:     h.Level,
			startLine: startLine,
			endOffset: endOffset,
		})
	}

	// No headings → single section with the entire body.
	if len(headings) == 0 {
		return []Section{{
			Heading:   "",
			Level:     0,
			Content:   string(source),
			StartLine: 1,
		}}
	}

	var sections []Section
	for i, h := range headings {
		var contentEnd int
		if i+1 < len(headings) {
			// Content ends where the next heading's line starts.
			contentEnd = lineOffset(source, headings[i+1].startLine)
		} else {
			contentEnd = len(source)
		}

		content := ""
		if h.endOffset < contentEnd {
			content = strings.TrimSpace(string(source[h.endOffset:contentEnd]))
		}

		sections = append(sections, Section{
			Heading:   h.text,
			Level:     h.level,
			Content:   content,
			StartLine: h.startLine,
		})
	}

	return sections
}

// lineOffset returns the byte offset of the start of a 1-based line number.
func lineOffset(source []byte, line int) int {
	current := 1
	for i := 0; i < len(source); i++ {
		if current == line {
			return i
		}
		if source[i] == '\n' {
			current++
		}
	}
	return len(source)
}
