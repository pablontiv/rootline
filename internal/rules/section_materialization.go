package rules

import (
	"fmt"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

type SectionMaterialization struct {
	Field           string
	Heading         string
	RenderedHeading string
	Content         string
}

func (m SectionMaterialization) MarkdownHeading() string {
	if m.RenderedHeading != "" {
		return m.RenderedHeading
	}
	return m.Heading
}

func RequiredSectionMaterializations(record *extract.Record, effective *StemFile) ([]SectionMaterialization, error) {
	if effective == nil || len(effective.Schema) == 0 {
		return nil, nil
	}

	fields := make([]string, 0, len(effective.Schema))
	for name := range effective.Schema {
		fields = append(fields, name)
	}
	sort.Strings(fields)

	for _, name := range fields {
		if issues := ValidateFieldDeclaration(name, effective.Schema[name]); len(issues) > 0 {
			return nil, fmt.Errorf("field %q: %s", name, issues[0].Message)
		}
	}

	recordPath := ""
	if record != nil {
		recordPath = record.Path
	}
	local := *effective
	var err error
	local.Schema, err = filterValueSchemaByMatch(effective.Schema, recordPath)
	if err != nil {
		return nil, err
	}

	fields = fields[:0]
	for name := range local.Schema {
		fields = append(fields, name)
	}
	sort.Strings(fields)

	out := make([]SectionMaterialization, 0)
	for _, name := range fields {
		field := local.Schema[name]
		if !field.Required || !requiredCheckApplies(record, &local, name, field) {
			continue
		}
		if field.Extract == "" || field.Type != "string" {
			continue
		}
		source, err := extract.ParseBodySource(field.Extract)
		if err != nil {
			return nil, err
		}
		if source.Kind != extract.BodySourceSection {
			continue
		}
		_, present, err := ResolveFieldValue(record, name, field)
		if err != nil {
			return nil, err
		}
		if present {
			continue
		}
		if len(source.Selector) > 1 {
			canonical, err := extract.CanonicalSectionSelectorSource(source.Selector)
			if err != nil {
				return nil, err
			}
			return nil, fmt.Errorf("cannot materialize required field %q: qualified section selector %s is missing", name, canonical)
		}
		renderedHeading, err := extract.MaterializeHeading(source.Selector[0])
		if err != nil {
			return nil, fmt.Errorf("cannot materialize required field %q: %w", name, err)
		}
		content := field.Default
		if content == "" {
			content = "<!-- TODO -->"
		}
		materialization := SectionMaterialization{Field: name, Heading: source.Heading, Content: content}
		if renderedHeading != source.Heading {
			materialization.RenderedHeading = renderedHeading
		}
		out = append(out, materialization)
	}

	sort.Slice(out, func(i, j int) bool {
		if out[i].Heading != out[j].Heading {
			return out[i].Heading < out[j].Heading
		}
		return out[i].Field < out[j].Field
	})
	if err := rejectDuplicateMaterializationHeadings(out); err != nil {
		return nil, err
	}
	return out, nil
}

func rejectDuplicateMaterializationHeadings(materializations []SectionMaterialization) error {
	for i := 0; i < len(materializations); {
		j := i + 1
		for j < len(materializations) && materializations[j].Heading == materializations[i].Heading {
			j++
		}
		if j-i > 1 {
			fields := make([]string, 0, j-i)
			for _, materialization := range materializations[i:j] {
				fields = append(fields, materialization.Field)
			}
			return fmt.Errorf("multiple required fields %s materialize missing section %q ambiguously", strings.Join(fields, ", "), materializations[i].Heading)
		}
		i = j
	}
	return nil
}
