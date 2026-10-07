package extract

import (
	"fmt"
	"strconv"
	"strings"
)

type BodySourceKind string

const (
	BodySourceH1      BodySourceKind = "h1"
	BodySourceSection BodySourceKind = "section"
)

type BodySource struct {
	Kind BodySourceKind

	// Heading retains the final exact heading for callers that use simple selectors.
	Heading  string
	Selector SectionSelector
}

func ParseBodySource(directive string) (BodySource, error) {
	switch {
	case directive == "body.h1":
		return BodySource{Kind: BodySourceH1}, nil
	case strings.HasPrefix(directive, "body.section"):
		selector, err := parseSectionSelector(directive)
		if err != nil {
			return BodySource{}, err
		}
		return BodySource{
			Kind:     BodySourceSection,
			Heading:  exactHeading(selector[len(selector)-1]),
			Selector: selector,
		}, nil
	default:
		return BodySource{}, fmt.Errorf("unsupported body source %q", directive)
	}
}

func parseSectionSelector(directive string) (SectionSelector, error) {
	const prefix = "body.section"
	if !strings.HasPrefix(directive, prefix) {
		return nil, fmt.Errorf("unsupported body source %q", directive)
	}

	rest := strings.TrimPrefix(directive, prefix)
	selector := make(SectionSelector, 0, 1)
	for rest != "" {
		if len(rest) < 4 || rest[0] != '[' || (rest[1] != '"' && rest[1] != '`') {
			return nil, fmt.Errorf("malformed body section source %q", directive)
		}
		quoteEnd := quotedStringEnd(rest[1:])
		if quoteEnd < 0 || quoteEnd+2 >= len(rest) || rest[quoteEnd+2] != ']' {
			return nil, fmt.Errorf("malformed body section source %q", directive)
		}
		quoted := rest[1 : quoteEnd+2]
		heading, err := strconv.Unquote(quoted)
		if err != nil {
			return nil, fmt.Errorf("malformed body section source %q", directive)
		}
		key, err := parseHeadingKey(heading)
		if err != nil {
			return nil, err
		}
		if len(selector) > 0 && key.Level <= selector[len(selector)-1].Level {
			return nil, fmt.Errorf("section source heading levels must increase in %q", directive)
		}
		selector = append(selector, key)
		rest = rest[quoteEnd+3:]
	}
	if len(selector) == 0 {
		return nil, fmt.Errorf("malformed body section source %q", directive)
	}
	return selector, nil
}

// quotedStringEnd returns the index of the closing quote in a quoted string.
func quotedStringEnd(quoted string) int {
	if len(quoted) == 0 || (quoted[0] != '"' && quoted[0] != '`') {
		return -1
	}
	delimiter := quoted[0]
	escaped := false
	for i := 1; i < len(quoted); i++ {
		if delimiter == '`' {
			if quoted[i] == delimiter {
				return i
			}
			continue
		}
		if escaped {
			escaped = false
			continue
		}
		switch quoted[i] {
		case '\\':
			escaped = true
		case delimiter:
			return i
		}
	}
	return -1
}

func CanonicalSectionSource(exactHeading string) (string, error) {
	key, err := parseHeadingKey(exactHeading)
	if err != nil {
		return "", err
	}
	return CanonicalSectionSelectorSource(SectionSelector{key})
}

// CanonicalSectionSelectorSource serializes a section selector.
func CanonicalSectionSelectorSource(selector SectionSelector) (string, error) {
	if err := validateSectionSelector(selector); err != nil {
		return "", err
	}
	var source strings.Builder
	source.WriteString("body.section")
	for _, key := range selector {
		source.WriteByte('[')
		source.WriteString(strconv.Quote(exactHeading(key)))
		source.WriteByte(']')
	}
	return source.String(), nil
}

func ResolveBodyValue(record *Record, directive string) (string, bool, error) {
	if record == nil {
		return "", false, nil
	}
	source, err := ParseBodySource(directive)
	if err != nil {
		return "", false, err
	}
	switch source.Kind {
	case BodySourceH1:
		h1 := resolveH1(record)
		return h1, h1 != "", nil
	case BodySourceSection:
		return resolveUniqueSection(sectionsForRecord(record), source.Selector)
	default:
		return "", false, fmt.Errorf("unsupported body source %q", directive)
	}
}

func resolveH1(record *Record) string {
	for _, sec := range sectionsForRecord(record) {
		if sec.Level == 1 {
			return sec.Heading
		}
	}
	return ""
}

func resolveUniqueSection(sections []Section, selector SectionSelector) (string, bool, error) {
	match := -1
	for i := range sections {
		if !SectionSelectorMatches(sections[i].Path, selector) {
			continue
		}
		if match >= 0 {
			description := exactHeading(selector[0])
			if len(selector) > 1 {
				description, _ = CanonicalSectionSelectorSource(selector)
			}
			return "", false, fmt.Errorf("ambiguous body section source %q", description)
		}
		match = i
	}
	if match < 0 {
		return "", false, nil
	}
	return sections[match].Content, true, nil
}

// SectionSelectorMatches reports whether selector is a contiguous path suffix.
func SectionSelectorMatches(path SectionPath, selector SectionSelector) bool {
	if len(selector) == 0 || len(selector) > len(path) {
		return false
	}
	offset := len(path) - len(selector)
	for i := range selector {
		if path[offset+i] != selector[i] {
			return false
		}
	}
	return true
}

func sectionsForRecord(record *Record) []Section {
	if record.BodySections != nil {
		sections := append([]Section(nil), record.BodySections...)
		return sectionsWithPaths(sections)
	}
	if record.AST != nil {
		return ExtractSections(record.AST, []byte(record.Body))
	}
	if record.Body == "" {
		return nil
	}
	return ExtractSectionsFromText(record.Body)
}

func sectionExactHeading(sec Section) string {
	if sec.Level <= 0 {
		return sec.Heading
	}
	return exactHeading(HeadingKey{Level: sec.Level, Text: sec.Heading})
}

func exactHeading(key HeadingKey) string {
	return strings.Repeat("#", key.Level) + " " + key.Text
}

func parseHeadingKey(heading string) (HeadingKey, error) {
	level := 0
	for level < len(heading) && heading[level] == '#' {
		level++
	}
	if level < 1 || level > 6 || level >= len(heading) || heading[level] != ' ' {
		return HeadingKey{}, fmt.Errorf("section source heading must contain 1 to 6 hashes and one space, got %q", heading)
	}
	return HeadingKey{Level: level, Text: heading[level+1:]}, nil
}

// MaterializeHeading returns markdown that extracts as key without data loss.
func MaterializeHeading(key HeadingKey) (string, error) {
	if err := validateHeadingKey(key); err != nil {
		return "", err
	}

	atx := exactHeading(key)
	if materializedHeadingMatches(atx, key) {
		return atx, nil
	}
	if key.Level <= 2 {
		underline := "---"
		if key.Level == 1 {
			underline = "==="
		}
		setext := key.Text + "\n" + underline
		if materializedHeadingMatches(setext, key) {
			return setext, nil
		}
	}
	return "", fmt.Errorf("heading level %d with text %q cannot be materialized without data loss", key.Level, key.Text)
}

func materializedHeadingMatches(markdown string, key HeadingKey) bool {
	sections := ExtractSectionsFromText(markdown)
	return len(sections) == 1 && sections[0].Level == key.Level && sections[0].Heading == key.Text
}

func validateHeadingKey(key HeadingKey) error {
	if key.Level < 1 || key.Level > 6 {
		return fmt.Errorf("heading level must be from 1 to 6, got %d", key.Level)
	}
	return nil
}

func validateSectionSelector(selector SectionSelector) error {
	if len(selector) == 0 {
		return fmt.Errorf("section selector must contain one heading")
	}
	for i, key := range selector {
		if err := validateHeadingKey(key); err != nil {
			return fmt.Errorf("section selector has an invalid heading at index %d", i)
		}
		parsed, err := parseHeadingKey(exactHeading(key))
		if err != nil || parsed != key {
			return fmt.Errorf("section selector has an invalid heading at index %d", i)
		}
		if i > 0 && key.Level <= selector[i-1].Level {
			return fmt.Errorf("section selector heading levels must increase")
		}
	}
	return nil
}
