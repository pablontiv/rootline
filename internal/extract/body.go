package extract

import (
	"bytes"
	"sort"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
	"github.com/yuin/goldmark/text"
)

// HeadingKey identifies one markdown heading.
type HeadingKey struct {
	Level int    `json:"level"`
	Text  string `json:"text"`
}

// SectionPath identifies a section from its root heading to its own heading.
type SectionPath []HeadingKey

// SectionSelector identifies a section by a contiguous suffix of its path.
type SectionSelector []HeadingKey

// Section represents a heading-delimited section in a markdown body.
type Section struct {
	Heading   string      `json:"heading"`
	Level     int         `json:"level"`
	Path      SectionPath `json:"path,omitempty"`
	Content   string      `json:"content"`
	StartLine int         `json:"start_line"`
}

// ExtractSections splits a markdown body into sections delimited by headings.
// It walks the AST to identify headings, so headings inside code blocks are ignored.
// If the document has no headings, a single section with the entire body is returned.
func ExtractSections(node ast.Node, source []byte) []Section {
	type headingInfo struct {
		text        string
		level       int
		startLine   int
		startOffset int
		endOffset   int
	}

	lineIndex := newSourceLineIndex(source)
	var headings []headingInfo

	// Collect headings from the AST (top-level block children only).
	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() != ast.KindHeading {
			continue
		}
		h := child.(*ast.Heading)
		lines := h.Lines()
		position := h.Pos()
		if position < 0 && lines.Len() > 0 {
			position = lines.At(0).Start
		}
		startLine := lineIndex.lineForOffset(position)
		startOffset := lineIndex.offset(startLine)
		lineEnd := lineIndex.offset(startLine + 1)
		endOffset := lineEnd
		headingText := ""

		if _, text, ok := parseATXHeading(string(source[startOffset:lineEnd])); ok {
			headingText = text
		} else if lines.Len() > 0 {
			var text strings.Builder
			for i := 0; i < lines.Len(); i++ {
				segment := lines.At(i)
				text.Write(segment.Value(source))
			}
			headingText = strings.TrimSpace(text.String())

			lastSeg := lines.At(lines.Len() - 1)
			endOffset = lastSeg.Stop
			lastLine := lineIndex.lineForOffset(lastSeg.Start)
			underlineStart := lineIndex.offset(lastLine + 1)
			underlineEnd := lineIndex.offset(lastLine + 2)
			if _, ok := parseSetextUnderline(string(source[underlineStart:underlineEnd])); ok {
				endOffset = underlineEnd
			}
		}

		headings = append(headings, headingInfo{
			text:        headingText,
			level:       h.Level,
			startLine:   startLine,
			startOffset: startOffset,
			endOffset:   endOffset,
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
			contentEnd = headings[i+1].startOffset
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

	return sectionsWithPaths(sections)
}

// CodeBlock represents a fenced code block in a markdown body.
type CodeBlock struct {
	Language  string `json:"language"`
	Content   string `json:"content"`
	StartLine int    `json:"start_line"`
}

// ExtractCodeBlocks extracts fenced code blocks from a markdown AST.
// Inline code spans are ignored.
func ExtractCodeBlocks(node ast.Node, source []byte) []CodeBlock {
	var blocks []CodeBlock

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() != ast.KindFencedCodeBlock {
			continue
		}
		fcb := child.(*ast.FencedCodeBlock)

		language := ""
		if info := fcb.Info; info != nil {
			seg := info.Segment
			language = strings.TrimSpace(string(seg.Value(source)))
		}

		var content strings.Builder
		lines := fcb.Lines()
		for i := 0; i < lines.Len(); i++ {
			seg := lines.At(i)
			content.Write(seg.Value(source))
		}

		startLine := 0
		if lines.Len() > 0 {
			startLine = lineFromOffset(source, lines.At(0).Start)
		}

		blocks = append(blocks, CodeBlock{
			Language:  language,
			Content:   content.String(),
			StartLine: startLine,
		})
	}

	return blocks
}

// Table represents a markdown table.
type Table struct {
	Headers []string   `json:"headers"`
	Rows    [][]string `json:"rows"`
}

// ExtractTables extracts tables from a markdown AST.
// Requires the document to be parsed with the goldmark table extension.
func ExtractTables(node ast.Node, source []byte) []Table {
	var tables []Table

	for child := node.FirstChild(); child != nil; child = child.NextSibling() {
		if child.Kind() != east.KindTable {
			continue
		}

		var headers []string
		var rows [][]string

		for row := child.FirstChild(); row != nil; row = row.NextSibling() {
			var cells []string
			for cell := row.FirstChild(); cell != nil; cell = cell.NextSibling() {
				var text strings.Builder
				for c := cell.FirstChild(); c != nil; c = c.NextSibling() {
					if c.Kind() == ast.KindText {
						seg := c.(*ast.Text).Segment
						text.Write(seg.Value(source))
					}
				}
				cells = append(cells, strings.TrimSpace(text.String()))
			}

			if row.Kind() == east.KindTableHeader {
				headers = cells
			} else {
				rows = append(rows, cells)
			}
		}

		tables = append(tables, Table{
			Headers: headers,
			Rows:    rows,
		})
	}

	return tables
}

// ExtractSectionsFromText splits markdown text into heading-delimited sections.
func ExtractSectionsFromText(body string) []Section {
	source := []byte(body)
	node := goldmark.DefaultParser().Parse(text.NewReader(source))
	return ExtractSections(node, source)
}

func sectionsWithPaths(sections []Section) []Section {
	path := make(SectionPath, 0, 6)
	for i := range sections {
		if sections[i].Level <= 0 {
			sections[i].Path = nil
			continue
		}
		for len(path) > 0 && path[len(path)-1].Level >= sections[i].Level {
			path = path[:len(path)-1]
		}
		path = append(path, HeadingKey{Level: sections[i].Level, Text: sections[i].Heading})
		sections[i].Path = append(SectionPath(nil), path...)
	}
	return sections
}

func parseATXHeading(line string) (int, string, bool) {
	line = strings.TrimRight(line, "\r")
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if indent > 3 {
		return 0, "", false
	}
	rest := line[indent:]
	level := len(rest) - len(strings.TrimLeft(rest, "#"))
	if level == 0 || level > 6 || (level < len(rest) && rest[level] != ' ' && rest[level] != '\t') {
		return 0, "", false
	}
	text := strings.TrimSpace(rest[level:])
	if i := len(text) - 1; i >= 0 && text[i] == '#' {
		for i >= 0 && text[i] == '#' {
			i--
		}
		if i < 0 {
			text = ""
		} else if text[i] == ' ' || text[i] == '\t' {
			text = strings.TrimSpace(text[:i])
		}
	}
	return level, text, true
}

func parseSetextUnderline(line string) (int, bool) {
	line = strings.TrimRight(line, "\r\n")
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if indent > 3 {
		return 0, false
	}
	rest := strings.TrimSpace(line[indent:])
	if rest == "" || (rest[0] != '=' && rest[0] != '-') {
		return 0, false
	}
	for i := range rest {
		if rest[i] != rest[0] {
			return 0, false
		}
	}
	if rest[0] == '=' {
		return 1, true
	}
	return 2, true
}

type sourceLineIndex struct {
	starts       []int
	sourceLength int
}

func newSourceLineIndex(source []byte) sourceLineIndex {
	starts := make([]int, 1, bytes.Count(source, []byte{'\n'})+1)
	for offset, value := range source {
		if value == '\n' {
			starts = append(starts, offset+1)
		}
	}
	return sourceLineIndex{starts: starts, sourceLength: len(source)}
}

func (index sourceLineIndex) lineForOffset(offset int) int {
	if offset < 0 {
		offset = 0
	} else if offset > index.sourceLength {
		offset = index.sourceLength
	}
	return sort.Search(len(index.starts), func(i int) bool {
		return index.starts[i] > offset
	})
}

func (index sourceLineIndex) offset(line int) int {
	if line <= 1 {
		return 0
	}
	if line > len(index.starts) {
		return index.sourceLength
	}
	return index.starts[line-1]
}

// ExtractBodyH1 returns the text of the first H1 heading in the body,
// stripping the "# " prefix. Returns empty string if no H1 is found.
func ExtractBodyH1(body string) string {
	lines := strings.Split(body, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimSpace(strings.TrimPrefix(line, "# "))
		}
	}
	return ""
}

// ExtractBodySection extracts the content under a specific heading in the body.
// heading should be the full heading line (e.g., "## Heading").
// Returns empty string if the section is not found.
func ExtractBodySection(body string, heading string) string {
	lines := strings.Split(body, "\n")
	var result []string
	found := false

	for _, line := range lines {
		trimmed := strings.TrimSpace(line)
		if trimmed == heading {
			found = true
			continue
		}

		if found {
			// Stop at the next heading (any level)
			if strings.HasPrefix(trimmed, "#") && trimmed != "" {
				break
			}
			// Skip the heading line itself, collect content
			if trimmed != "" {
				result = append(result, line)
			}
		}
	}

	if len(result) > 0 {
		return strings.TrimSpace(strings.Join(result, "\n"))
	}
	return ""
}
