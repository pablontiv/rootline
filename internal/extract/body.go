package extract

import (
	"strings"

	"github.com/yuin/goldmark/ast"
	east "github.com/yuin/goldmark/extension/ast"
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
		lines := h.Lines()
		startLine := 0
		endOffset := 0
		headingText := ""
		if lines.Len() > 0 {
			firstSeg := lines.At(0)
			startLine = lineFromOffset(source, firstSeg.Start)
			lastSeg := lines.At(lines.Len() - 1)
			endOffset = lastSeg.Stop

			lineStart := lineOffset(source, startLine)
			lineEnd := lineOffset(source, startLine+1)
			if _, text, ok := parseATXHeading(string(source[lineStart:lineEnd])); ok {
				headingText = text
			} else {
				var text strings.Builder
				for i := 0; i < lines.Len(); i++ {
					segment := lines.At(i)
					text.Write(segment.Value(source))
				}
				headingText = strings.TrimSpace(text.String())

				lastLine := lineFromOffset(source, lastSeg.Start)
				underlineStart := lineOffset(source, lastLine+1)
				underlineEnd := lineOffset(source, lastLine+2)
				if _, ok := parseSetextUnderline(string(source[underlineStart:underlineEnd])); ok {
					endOffset = underlineEnd
				}
			}
		}

		headings = append(headings, headingInfo{
			text:      headingText,
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
	type heading struct {
		text        string
		level       int
		line, start int
		end         int
	}
	var headings []heading
	candidateLine, candidateStart, candidateOK := 0, 0, false
	var fence *fenceContext
	for start, lineNo := 0, 1; start <= len(source); lineNo++ {
		end := start
		for end < len(source) && source[end] != '\n' {
			end++
		}
		lineEnd := end
		if end < len(source) {
			lineEnd++
		}
		line := string(source[start:end])

		if fence != nil {
			content, belongs := fence.content(line)
			if belongs {
				if char, length, trailing, ok := parseBareFenceLine(content); ok && char == fence.char && length >= fence.length && strings.TrimSpace(trailing) == "" {
					fence = nil
				}
				candidateOK = false
				if end >= len(source) {
					break
				}
				start = lineEnd
				continue
			}
			fence = nil
		}

		if opened, ok := parseFenceOpening(line); ok {
			fence = &opened
			candidateOK = false
		} else if level, text, ok := parseATXHeading(line); ok {
			headings = append(headings, heading{text: text, level: level, line: lineNo, start: start, end: lineEnd})
			candidateOK = false
		} else if level, ok := parseSetextUnderline(line); ok && candidateOK {
			headingText := strings.TrimSpace(string(source[candidateStart:start]))
			headings = append(headings, heading{text: headingText, level: level, line: candidateLine, start: candidateStart, end: lineEnd})
			candidateOK = false
		} else if _, ok := parseSetextUnderline(line); ok {
			candidateOK = false
		} else if strings.TrimSpace(line) == "" {
			candidateOK = false
		} else if !candidateOK {
			candidateLine, candidateStart, candidateOK = lineNo, start, true
		}
		if end >= len(source) {
			break
		}
		start = lineEnd
	}
	if len(headings) == 0 {
		return []Section{{Heading: "", Level: 0, Content: body, StartLine: 1}}
	}
	sections := make([]Section, 0, len(headings))
	for i, h := range headings {
		contentEnd := len(source)
		if i+1 < len(headings) {
			contentEnd = headings[i+1].start
		}
		content := ""
		if h.end < contentEnd {
			content = strings.TrimSpace(string(source[h.end:contentEnd]))
		}
		sections = append(sections, Section{Heading: h.text, Level: h.level, Content: content, StartLine: h.line})
	}
	return sectionsWithPaths(sections)
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
	if i := len(text) - 1; i > 0 && text[i] == '#' {
		for i >= 0 && text[i] == '#' {
			i--
		}
		if i >= 0 && (text[i] == ' ' || text[i] == '\t') {
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

type fenceContext struct {
	char       byte
	length     int
	quoteDepth int
	listIndent int
}

func parseFenceOpening(line string) (fenceContext, bool) {
	line = strings.TrimRight(line, "\r")
	if char, length, _, ok := parseBareFenceLine(line); ok {
		return fenceContext{char: char, length: length}, true
	}

	content, quoteDepth, listIndent, ok := fenceContainerContent(line)
	if !ok {
		return fenceContext{}, false
	}
	char, length, _, ok := parseBareFenceLine(content)
	if !ok {
		return fenceContext{}, false
	}
	return fenceContext{char: char, length: length, quoteDepth: quoteDepth, listIndent: listIndent}, true
}

func parseBareFenceLine(line string) (byte, int, string, bool) {
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if indent > 3 || indent >= len(line) || (line[indent] != '`' && line[indent] != '~') {
		return 0, 0, "", false
	}
	char, count := line[indent], 0
	i := indent
	for ; i < len(line) && line[i] == char; i++ {
		count++
	}
	trailing := line[i:]
	if count < 3 || (char == '`' && strings.Contains(trailing, "`")) {
		return 0, 0, "", false
	}
	return char, count, trailing, true
}

func fenceContainerContent(line string) (content string, quoteDepth, listIndent int, ok bool) {
	indent := len(line) - len(strings.TrimLeft(line, " "))
	if indent > 3 {
		return "", 0, 0, false
	}
	rest := line[indent:]
	for len(rest) > 0 && rest[0] == '>' {
		quoteDepth++
		rest = rest[1:]
		if len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
			rest = rest[1:]
		}
	}

	listStart := len(rest) - len(strings.TrimLeft(rest, " "))
	listRest := rest[listStart:]
	markerEnd := 0
	if len(listRest) > 1 && (listRest[0] == '-' || listRest[0] == '+' || listRest[0] == '*') {
		markerEnd = 1
	} else {
		for markerEnd < len(listRest) && markerEnd < 9 && listRest[markerEnd] >= '0' && listRest[markerEnd] <= '9' {
			markerEnd++
		}
		if markerEnd == 0 || markerEnd >= len(listRest) || (listRest[markerEnd] != '.' && listRest[markerEnd] != ')') {
			markerEnd = 0
		} else {
			markerEnd++
		}
	}
	if markerEnd == 0 || markerEnd >= len(listRest) || (listRest[markerEnd] != ' ' && listRest[markerEnd] != '\t') {
		if quoteDepth > 0 {
			return rest, quoteDepth, 0, true
		}
		return "", 0, 0, false
	}
	contentStart := markerEnd
	for contentStart < len(listRest) && (listRest[contentStart] == ' ' || listRest[contentStart] == '\t') {
		contentStart++
	}
	listIndent = listStart + contentStart
	if quoteDepth == 0 {
		listIndent += indent
	}
	return listRest[contentStart:], quoteDepth, listIndent, true
}

func (f fenceContext) content(line string) (string, bool) {
	line = strings.TrimRight(line, "\r")
	if strings.TrimSpace(line) == "" {
		return line, true
	}
	if f.quoteDepth == 0 && f.listIndent == 0 {
		return line, true
	}

	rest := line
	if f.quoteDepth > 0 {
		indent := len(line) - len(strings.TrimLeft(line, " "))
		if indent > 3 {
			return "", false
		}
		rest = line[indent:]
		for i := 0; i < f.quoteDepth; i++ {
			if len(rest) == 0 || rest[0] != '>' {
				return "", false
			}
			rest = rest[1:]
			if len(rest) > 0 && (rest[0] == ' ' || rest[0] == '\t') {
				rest = rest[1:]
			}
		}
	}
	if f.listIndent == 0 {
		return rest, true
	}

	contentIndent := len(rest) - len(strings.TrimLeft(rest, " "))
	if contentIndent < f.listIndent {
		return "", false
	}
	return rest[f.listIndent:], true
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
