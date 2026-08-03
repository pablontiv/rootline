package extract

import (
	"regexp"
	"strings"
)

// Link represents a link extracted from document text.
type Link struct {
	Target string `json:"target"`
	Type   string `json:"type"`
	Line   int    `json:"line"`
	Source string `json:"source,omitempty"` // "body" or "frontmatter:<fieldname>"
	Style  string `json:"style,omitempty"`  // StyleWikilink or StyleMarkdown
	Anchor string `json:"anchor,omitempty"` // fragment part of markdown targets, without '#'
}

// Link styles produced by extraction.
const (
	StyleWikilink = "wikilink"
	StyleMarkdown = "markdown"
)

var wikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// markdownLinkRe captures an optional image marker and the destination of
// inline markdown links. Wikilinks don't match: their brackets are doubled
// and have no (...) destination. The destination group allows one level of
// nested parentheses to handle targets like foo(1).md.
var markdownLinkRe = regexp.MustCompile(`(!?)\[[^\]]*\]\(([^()]*(?:\([^()]*\)[^()]*)*)\)`)

// parseMarkdownDestination converts an inline-link destination into a Link.
// Returns false for destinations that aren't local paths: external schemes,
// mailto, and pure fragments. Quoted titles (`foo.md "Title"`) and angle
// brackets (`<foo.md>`) are stripped; a raw space with no quoted title after
// it is kept so the encoding check can flag it.
func parseMarkdownDestination(dest string) (Link, bool) {
	dest = strings.TrimSpace(dest)
	if i := strings.IndexAny(dest, " \t"); i >= 0 {
		rest := strings.TrimSpace(dest[i+1:])
		if strings.HasPrefix(rest, `"`) || strings.HasPrefix(rest, "'") || strings.HasPrefix(rest, "(") {
			dest = dest[:i]
		}
	}
	dest = strings.TrimPrefix(dest, "<")
	dest = strings.TrimSuffix(dest, ">")
	if dest == "" || strings.HasPrefix(dest, "#") {
		return Link{}, false
	}
	if strings.Contains(dest, "://") || strings.HasPrefix(dest, "mailto:") {
		return Link{}, false
	}
	link := Link{Type: "reference", Style: StyleMarkdown, Target: dest}
	target, anchor, found := strings.Cut(dest, "#")
	if found {
		if target == "" {
			return Link{}, false
		}
		link.Target = target
		link.Anchor = anchor
	}
	return link, true
}

// parseWikilinkInner converts the text between [[ and ]] into a Link.
// The optional "type:" prefix selects the link type; the optional "#anchor"
// suffix is split off into Link.Anchor exactly as parseMarkdownDestination
// does for markdown destinations, so anchor-aware checks fire on both styles.
// Returns false for a pure fragment ([[#heading]]), which names no target.
func parseWikilinkInner(inner string) (Link, bool) {
	link := Link{Type: "reference", Target: inner, Style: StyleWikilink}
	if idx := strings.Index(inner, ":"); idx > 0 {
		link.Type = inner[:idx]
		link.Target = inner[idx+1:]
	}
	if target, anchor, found := strings.Cut(link.Target, "#"); found {
		if target == "" {
			return Link{}, false
		}
		link.Target = target
		link.Anchor = anchor
	}
	return link, true
}

// ParseLinks extracts wiki-links and markdown links from body text.
// Wiki-links: [[target]] (type "reference") and [[type:target]] (typed).
// Markdown links: [text](target) with optional fragment (#anchor).
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
			link, ok := parseWikilinkInner(match[1])
			if !ok {
				continue
			}
			link.Line = lineNum
			link.Source = "body"
			links = append(links, link)
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
		link, ok := parseWikilinkInner(match[1])
		if !ok {
			continue
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
