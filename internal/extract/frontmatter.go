package extract

import "strings"

// FrontmatterBounds locates the leading YAML frontmatter block of a document.
//
// Only the leading block is frontmatter. The scan starts after the opening
// "---" line and stops at the first later line that is exactly "---" and is not
// inside a fenced code block, so thematic breaks and fence content in the
// Markdown body are never mistaken for a delimiter.
//
// It returns the offset of the first byte of the frontmatter content (start),
// the offset of the first byte of the closing delimiter line (end), and the
// offset of the first byte after that line (next). ok is false when the text
// does not open with a "---" line or the block is never closed.
func FrontmatterBounds(text string) (start, end, next int, ok bool) {
	lineEnd, lineNext := scanLine(text, 0)
	if trimCR(text[0:lineEnd]) != "---" {
		return 0, 0, 0, false
	}
	start = lineNext

	var fenceChar byte
	var fenceLen int
	for pos := lineNext; pos < len(text); {
		lineEnd, lineNext = scanLine(text, pos)
		line := trimCR(text[pos:lineEnd])

		switch char, runLen, isFence := fenceInfo(line); {
		case fenceChar != 0:
			if isFence && char == fenceChar && runLen >= fenceLen {
				fenceChar, fenceLen = 0, 0
			}
		case isFence:
			fenceChar, fenceLen = char, runLen
		case line == "---":
			return start, pos, lineNext, true
		}
		pos = lineNext
	}
	return 0, 0, 0, false
}

// scanLine returns the offset of the line terminator at or after pos and the
// offset of the next line. Both are len(text) for a final line without one.
func scanLine(text string, pos int) (lineEnd, next int) {
	nl := strings.IndexByte(text[pos:], '\n')
	if nl < 0 {
		return len(text), len(text)
	}
	return pos + nl, pos + nl + 1
}

// trimCR drops a trailing carriage return so CRLF input compares like LF input.
func trimCR(line string) string {
	return strings.TrimSuffix(line, "\r")
}

// fenceInfo reports whether line opens or closes a fenced code block,
// returning the fence character and the length of its run.
//
// Only column-0 fences count, which is stricter than CommonMark's "up to three
// leading spaces". Frontmatter block scalars are always indented, so an indented
// fence belongs to a YAML value — treating it as a fence would hide the real
// closing delimiter behind it. A body fence that could shadow a delimiter is
// written at column 0 in practice.
func fenceInfo(line string) (char byte, runLen int, ok bool) {
	if line == "" || (line[0] != '`' && line[0] != '~') {
		return 0, 0, false
	}
	char = line[0]
	for runLen < len(line) && line[runLen] == char {
		runLen++
	}
	if runLen < 3 {
		return 0, 0, false
	}
	return char, runLen, true
}
