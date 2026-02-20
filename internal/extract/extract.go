// Package extract implements metadata extraction from files.
//
// It provides an Extractor interface and built-in implementations
// for YAML frontmatter in Markdown files.
package extract

import (
	"fmt"
	"strings"

	"gopkg.in/yaml.v3"
)

// Extractor extracts structured metadata and body text from file content.
// Implementations are format-specific (Markdown built-in; others via plugins).
// Extractors receive content as bytes — they do NOT perform file I/O.
type Extractor interface {
	Extract(path string, content []byte) (*Record, error)
	Extensions() []string
	Name() string
}

// Record is the universal output of all extractors and the universal input
// to validation, derivation, and query.
type Record struct {
	Path        string            `json:"path"`
	Type        string            `json:"type"`
	Frontmatter map[string]any   `json:"frontmatter"`
	Body        string            `json:"body"`
	Links       []Link            `json:"links,omitempty"`
	Errors      []ExtractionError `json:"errors,omitempty"`
}

// ExtractionError represents a non-fatal issue during extraction.
type ExtractionError struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// MarkdownExtractor extracts YAML frontmatter from Markdown files.
type MarkdownExtractor struct{}

func (m *MarkdownExtractor) Name() string        { return "markdown" }
func (m *MarkdownExtractor) Extensions() []string { return []string{".md", ".markdown"} }

func (m *MarkdownExtractor) Extract(path string, content []byte) (*Record, error) {
	record := &Record{
		Path:        path,
		Type:        m.Name(),
		Frontmatter: make(map[string]any),
	}

	text := string(content)
	text = stripBOM(text)

	// No frontmatter — entire content is body.
	if !strings.HasPrefix(text, "---\n") && !strings.HasPrefix(text, "---\r\n") {
		record.Body = text
		record.Links = ParseLinks(record.Body)
		return record, nil
	}

	// Find closing delimiter.
	closeIdx := findClosingDelimiter(text)
	if closeIdx < 0 {
		record.Body = text
		record.Errors = append(record.Errors, ExtractionError{
			Line:    1,
			Message: "unclosed frontmatter delimiter",
		})
		return record, nil
	}

	// Parse YAML frontmatter.
	fmContent := text[4:closeIdx]
	if err := yaml.Unmarshal([]byte(fmContent), &record.Frontmatter); err != nil {
		record.Errors = append(record.Errors, ExtractionError{
			Line:    1,
			Message: fmt.Sprintf("malformed YAML frontmatter: %v", err),
		})
		// Line-by-line fallback.
		record.Frontmatter = fallbackParseFrontmatter(fmContent)
	}

	// Body is everything after closing delimiter.
	bodyStart := closeIdx + 4 // len("---\n")
	if bodyStart < len(text) {
		record.Body = strings.TrimLeft(text[bodyStart:], "\r\n")
	}

	// Extract wiki-links from body.
	record.Links = ParseLinks(record.Body)

	return record, nil
}

// stripBOM removes a UTF-8 BOM from the start of text.
func stripBOM(text string) string {
	if strings.HasPrefix(text, "\xef\xbb\xbf") {
		return text[3:]
	}
	return text
}

// findClosingDelimiter finds the index of the closing "---" delimiter.
// Returns -1 if not found.
func findClosingDelimiter(text string) int {
	// Skip the opening "---\n".
	search := text[4:]
	for i := 0; i < len(search); {
		nl := strings.Index(search[i:], "\n")
		if nl < 0 {
			break
		}
		lineStart := i
		i += nl + 1

		line := search[lineStart : lineStart+nl]
		line = strings.TrimRight(line, "\r")
		if line == "---" {
			return lineStart + 4 // offset from full text
		}
	}
	return -1
}

// fallbackParseFrontmatter does a best-effort line-by-line parse of
// malformed YAML frontmatter, extracting simple "key: value" pairs.
func fallbackParseFrontmatter(content string) map[string]any {
	result := make(map[string]any)
	for _, line := range strings.Split(content, "\n") {
		line = strings.TrimSpace(line)
		if line == "" || strings.HasPrefix(line, "#") {
			continue
		}
		idx := strings.Index(line, ":")
		if idx <= 0 {
			continue
		}
		key := strings.TrimSpace(line[:idx])
		val := strings.TrimSpace(line[idx+1:])
		if val != "" {
			result[key] = val
		}
	}
	return result
}
