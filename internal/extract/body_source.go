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
	Kind    BodySourceKind
	Heading string
}

func ParseBodySource(directive string) (BodySource, error) {
	switch {
	case directive == "body.h1":
		return BodySource{Kind: BodySourceH1}, nil
	case strings.HasPrefix(directive, "body.section["):
		if !strings.HasSuffix(directive, "]") {
			return BodySource{}, fmt.Errorf("malformed body section source %q", directive)
		}
		quoted := strings.TrimSuffix(strings.TrimPrefix(directive, "body.section["), "]")
		heading, err := strconv.Unquote(quoted)
		if err != nil {
			return BodySource{}, fmt.Errorf("malformed body section source %q", directive)
		}
		if err := validateExactHeading(heading); err != nil {
			return BodySource{}, err
		}
		return BodySource{Kind: BodySourceSection, Heading: heading}, nil
	default:
		return BodySource{}, fmt.Errorf("unsupported body source %q", directive)
	}
}

func CanonicalSectionSource(exactHeading string) (string, error) {
	if err := validateExactHeading(exactHeading); err != nil {
		return "", err
	}
	return "body.section[" + strconv.Quote(exactHeading) + "]", nil
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
		return resolveUniqueSection(sectionsForRecord(record), source.Heading)
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

func resolveUniqueSection(sections []Section, exactHeading string) (string, bool, error) {
	var match *Section
	for i := range sections {
		if sectionExactHeading(sections[i]) != exactHeading {
			continue
		}
		if match != nil {
			return "", false, fmt.Errorf("ambiguous body section source %q", exactHeading)
		}
		match = &sections[i]
	}
	if match == nil {
		return "", false, nil
	}
	return match.Content, true, nil
}

func sectionsForRecord(record *Record) []Section {
	if len(record.BodySections) > 0 {
		return record.BodySections
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
	return strings.Repeat("#", sec.Level) + " " + sec.Heading
}

func validateExactHeading(heading string) error {
	level, text, ok := parseATXHeading(heading)
	if !ok || level == 0 || sectionExactHeading(Section{Heading: text, Level: level}) != heading {
		return fmt.Errorf("section source heading must be an exact markdown heading, got %q", heading)
	}
	return nil
}
