package infer

import (
	"fmt"
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

const requiredUsageThreshold = 0.80

// detectGapsForScope checks schema fields for insufficient validation rules.
// priorInferences is used for deduplication against data-inference detectors.
func detectGapsForScope(stem *rules.StemFile, records []*extract.Record, priorInferences []Inference) []Inference {
	if stem == nil || len(stem.Schema) == 0 {
		return nil
	}

	coveredByEnum := make(map[string]bool)
	coveredByRequired := make(map[string]bool)
	for _, inf := range priorInferences {
		switch inf.Type {
		case "enum_values":
			coveredByEnum[inf.Field] = true
		case "required_field":
			coveredByRequired[inf.Field] = true
		}
	}

	var inferences []Inference

	names := make([]string, 0, len(stem.Schema))
	for name := range stem.Schema {
		names = append(names, name)
	}
	sort.Strings(names)

	for _, name := range names {
		sf := stem.Schema[name]

		if sf.Type == "enum" && len(sf.Values) == 0 && !coveredByEnum[name] {
			inferences = append(inferences, Inference{
				Type:    "enum_without_values",
				Source:  sf.Source,
				Field:   name,
				Message: fmt.Sprintf("Field %q is declared as enum but has no values list — cannot validate", name),
			})
		}

		if sf.Type == "" {
			inferences = append(inferences, Inference{
				Type:    "untyped_field",
				Source:  sf.Source,
				Field:   name,
				Message: fmt.Sprintf("Field %q has no type — rootline cannot validate it", name),
			})
		}

		if sf.Type == "sequence" && (sf.Prefix == "" || sf.Digits == 0) {
			var missing string
			switch {
			case sf.Prefix == "" && sf.Digits == 0:
				missing = "prefix and digits"
			case sf.Prefix == "":
				missing = "prefix"
			default:
				missing = "digits"
			}
			inferences = append(inferences, Inference{
				Type:    "sequence_incomplete",
				Source:  sf.Source,
				Field:   name,
				Message: fmt.Sprintf("Field %q is type sequence but missing %s", name, missing),
			})
		}
	}

	if len(records) >= 3 {
		total := len(records)
		for _, name := range names {
			sf := stem.Schema[name]
			if sf.Required || coveredByRequired[name] {
				continue
			}
			count := 0
			for _, rec := range records {
				if _, ok := rec.Frontmatter[name]; ok {
					count++
				}
			}
			ratio := float64(count) / float64(total)
			if ratio >= requiredUsageThreshold {
				inferences = append(inferences, Inference{
					Type:    "required_understatement",
					Source:  sf.Source,
					Field:   name,
					Message: fmt.Sprintf("Field %q is used in %d/%d records (%.0f%%) but not declared required", name, count, total, ratio*100),
				})
			}
		}
	}

	return inferences
}

// DetectValidationGaps runs the per-scope schema-gap checks across every .stem
// scope in the tree (grouped by GroupByScope), deduplicating findings by
// (Type, Field, Source) since inherited fields appear in multiple scopes.
func DetectValidationGaps(records []*extract.Record, prior []Inference, root string, resolve StemResolver) []Inference {
	groups := GroupByScope(records, root, resolve)
	seen := make(map[string]bool)
	var out []Inference
	for _, g := range groups {
		if g.Stem == nil || len(g.Stem.Schema) == 0 {
			continue
		}
		for _, inf := range detectGapsForScope(g.Stem, g.Records, prior) {
			k := inf.Type + "|" + inf.Field + "|" + inf.Source
			if seen[k] {
				continue
			}
			seen[k] = true
			out = append(out, inf)
		}
	}
	return out
}
