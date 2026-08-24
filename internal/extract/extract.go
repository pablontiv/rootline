// Package extract implements metadata extraction from files.
//
// It provides an Extractor interface and built-in implementations
// for YAML frontmatter in Markdown files.
package extract

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/ast"
	gmtext "github.com/yuin/goldmark/text"
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
	Path               string                       `json:"path"`
	Type               string                       `json:"type"`
	Frontmatter        map[string]any               `json:"frontmatter"`
	FrontmatterScalars map[string]FrontmatterScalar `json:"-"`
	Body               string                       `json:"body"`
	Sections           map[string]string            `json:"sections,omitempty"`
	BodySections       []Section                    `json:"body_sections,omitempty"`
	AST                ast.Node                     `json:"-"`
	Links              []Link                       `json:"links,omitempty"`
	Derived            map[string]any               `json:"derived,omitempty"`
	Errors             []ExtractionError            `json:"errors,omitempty"`
}

// EffectiveField returns the effective value for a field name.
// Derived fields take precedence over frontmatter.
func (r *Record) EffectiveField(key string) (any, bool) {
	if r.Derived != nil {
		if v, ok := r.Derived[key]; ok {
			return v, true
		}
	}
	v, ok := r.Frontmatter[key]
	return v, ok
}

// ExtractionError represents a non-fatal issue during extraction.
type ExtractionError struct {
	Line    int    `json:"line"`
	Message string `json:"message"`
}

// MarkdownExtractor extracts YAML frontmatter from Markdown files.
// Set ParseAST to false to skip goldmark AST parsing (default: true).
type MarkdownExtractor struct {
	ParseAST *bool
}

func (m *MarkdownExtractor) shouldParseAST() bool {
	return m.ParseAST != nil && *m.ParseAST
}

func (m *MarkdownExtractor) Name() string         { return "markdown" }
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
		m.populateBodySections(record)
		return record, nil
	}

	// Find closing delimiter. Only the leading block is frontmatter.
	fmStart, fmEnd, bodyStart, ok := FrontmatterBounds(text)
	if !ok {
		record.Body = text
		record.Errors = append(record.Errors, ExtractionError{
			Line:    1,
			Message: "unclosed frontmatter delimiter",
		})
		return record, nil
	}

	// Parse YAML frontmatter.
	fmContent := text[fmStart:fmEnd]
	frontmatter, scalars, err := decodeFrontmatter(fmContent)
	if err != nil {
		record.Errors = append(record.Errors, ExtractionError{
			Line:    1,
			Message: fmt.Sprintf("malformed YAML frontmatter: %v", err),
		})
		// Line-by-line fallback.
		record.Frontmatter = fallbackParseFrontmatter(fmContent)
		record.FrontmatterScalars = nil
	} else {
		record.Frontmatter = frontmatter
		record.FrontmatterScalars = scalars
	}

	// Body is everything after closing delimiter.
	if bodyStart < len(text) {
		record.Body = strings.TrimLeft(text[bodyStart:], "\r\n")
	}

	// Extract wiki-links from body and frontmatter.
	record.Links = ParseLinks(record.Body)
	fmLinks := ParseFrontmatterLinks(record.Frontmatter)
	if len(fmLinks) > 0 {
		record.Links = append(fmLinks, record.Links...)
	}

	// Parse body structure after links/frontmatter are finalized.
	m.populateBodySections(record)

	return record, nil
}

func (m *MarkdownExtractor) populateBodySections(record *Record) {
	if record.Body == "" {
		return
	}
	if m.shouldParseAST() {
		source := []byte(record.Body)
		reader := gmtext.NewReader(source)
		parser := goldmark.DefaultParser()
		record.AST = parser.Parse(reader)
		record.BodySections = ExtractSections(record.AST, source)
	} else {
		record.BodySections = ExtractSectionsFromText(record.Body)
	}
	if len(record.BodySections) == 0 {
		return
	}
	record.Sections = make(map[string]string, len(record.BodySections))
	for _, sec := range record.BodySections {
		if sec.Level > 0 {
			record.Sections[sectionExactHeading(sec)] = strings.TrimSpace(sec.Content)
		}
	}
}

// stripBOM removes a UTF-8 BOM from the start of text.
func stripBOM(text string) string {
	if strings.HasPrefix(text, "\xef\xbb\xbf") {
		return text[3:]
	}
	return text
}

var boldColonRe = regexp.MustCompile(`\*\*([^*]+)\*\*:\s*(.+)`)

// ScanBodyFields detects bold-colon metadata patterns in markdown body text.
// It returns a map of lowercase keys to their original-case values.
func ScanBodyFields(body string) map[string]string {
	result := make(map[string]string)
	for _, match := range boldColonRe.FindAllStringSubmatch(body, -1) {
		key := strings.ToLower(match[1])
		result[key] = strings.TrimSpace(match[2])
	}
	return result
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
