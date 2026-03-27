package extract

import (
	"regexp"
	"strings"
)

// Link represents a wiki-link extracted from document text.
type Link struct {
	Target string `json:"target"`
	Type   string `json:"type"`
	Line   int    `json:"line"`
	Source string `json:"source,omitempty"` // "body" or "frontmatter:<fieldname>"
}

var wikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// ParseLinks extracts wiki-links from body text.
// Supported formats: [[target]] (type "reference") and [[type:target]] (typed).
// Links inside fenced code blocks or inline code are ignored.
func ParseLinks(body string) []Link {
	lines := strings.Split(body, "\n")
	var links []Link
	inFenced := false

	for i, line := range lines {
		lineNum := i + 1

		// Track fenced code blocks.
		trimmed := strings.TrimSpace(line)
		if strings.HasPrefix(trimmed, "```") || strings.HasPrefix(trimmed, "~~~") {
			inFenced = !inFenced
			continue
		}
		if inFenced {
			continue
		}

		// Strip inline code spans before matching.
		cleaned := removeInlineCode(line)

		for _, match := range wikilinkRe.FindAllStringSubmatch(cleaned, -1) {
			inner := match[1]
			link := Link{Line: lineNum, Source: "body"}
			if idx := strings.Index(inner, ":"); idx > 0 {
				link.Type = inner[:idx]
				link.Target = inner[idx+1:]
			} else {
				link.Type = "reference"
				link.Target = inner
			}
			links = append(links, link)
		}
	}

	return links
}

// ParseFrontmatterLinks extracts wiki-links from frontmatter values.
// Scans string values and string slices for [[target]] patterns.
// Links get Source set to "frontmatter:<fieldname>".
func ParseFrontmatterLinks(frontmatter map[string]any) []Link {
	if frontmatter == nil {
		return nil
	}
	var links []Link
	for key, val := range frontmatter {
		source := "frontmatter:" + key
		switch v := val.(type) {
		case string:
			for _, link := range parseWikilinksFromString(v) {
				link.Source = source
				links = append(links, link)
			}
		case []any:
			for _, item := range v {
				s, ok := item.(string)
				if !ok {
					continue
				}
				for _, link := range parseWikilinksFromString(s) {
					link.Source = source
					links = append(links, link)
				}
			}
		}
	}
	return links
}

// parseWikilinksFromString extracts wiki-links from a single string value.
func parseWikilinksFromString(s string) []Link {
	var links []Link
	for _, match := range wikilinkRe.FindAllStringSubmatch(s, -1) {
		inner := match[1]
		link := Link{Line: 0}
		if idx := strings.Index(inner, ":"); idx > 0 {
			link.Type = inner[:idx]
			link.Target = inner[idx+1:]
		} else {
			link.Type = "reference"
			link.Target = inner
		}
		links = append(links, link)
	}
	return links
}

// ContainsWikilink checks if a string contains at least one [[target]] pattern.
func ContainsWikilink(s string) bool {
	return wikilinkRe.MatchString(s)
}

// removeInlineCode replaces inline code spans (backtick-delimited) with spaces,
// preserving string length so that match positions remain valid for the line.
func removeInlineCode(line string) string {
	var b strings.Builder
	b.Grow(len(line))
	i := 0
	for i < len(line) {
		if line[i] == '`' {
			// Count opening backticks.
			start := i
			for i < len(line) && line[i] == '`' {
				i++
			}
			ticks := i - start
			// Find matching closing backticks.
			closer := strings.Repeat("`", ticks)
			end := strings.Index(line[i:], closer)
			if end >= 0 {
				// Replace entire span (including backticks) with spaces.
				total := ticks + end + ticks
				for j := 0; j < total; j++ {
					b.WriteByte(' ')
				}
				i += end + ticks
			} else {
				// No closing backticks — keep as-is.
				b.WriteString(line[start:i])
			}
		} else {
			b.WriteByte(line[i])
			i++
		}
	}
	return b.String()
}
