package infer

import (
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// FilterCoveredInferences removes inferences already covered by the existing
// per-scope .stem schema. An inference for a field is covered iff EVERY scope
// that is relevant to that field (the field is in the scope's schema or in one
// of its records) covers it via isCovered. Conservative: if any relevant scope
// is uncovered, the inference is kept. Reduces to the single-stem behavior when
// there is one scope.
func FilterCoveredInferences(inferences []Inference, records []*extract.Record, root string, resolve StemResolver) []Inference {
	groups := GroupByScope(records, root, resolve)
	var deltas []Inference
	for _, inf := range inferences {
		if coveredEverywhere(inf, groups) {
			continue
		}
		deltas = append(deltas, inf)
	}
	return deltas
}

func coveredEverywhere(inf Inference, groups []ScopeGroup) bool {
	relevant := false
	for _, g := range groups {
		if g.Stem == nil {
			continue
		}
		_, inSchema := g.Stem.Schema[inf.Field]
		inRecords := false
		for _, rec := range g.Records {
			if _, ok := rec.Frontmatter[inf.Field]; ok {
				inRecords = true
				break
			}
		}
		if !inSchema && !inRecords {
			continue
		}
		relevant = true
		if !isCovered(inf, g.Stem) {
			return false
		}
	}
	return relevant
}

func isCovered(inf Inference, stem *rules.StemFile) bool {
	switch inf.Type {
	case "required_field":
		// Covered if stem already marks this field as required.
		if sf, ok := stem.Schema[inf.Field]; ok && sf.Required {
			return true
		}

	case "field_type":
		// Covered if stem already defines this field with matching type.
		if sf, ok := stem.Schema[inf.Field]; ok && sf.Type == inf.Value {
			return true
		}

	case "enum_values":
		// Covered if stem already has enum values for this field.
		if sf, ok := stem.Schema[inf.Field]; ok && sf.Type == "enum" && len(sf.Values) > 0 {
			return true
		}

	case "constant_field":
		// Covered if stem has the field as required with a default matching the value,
		// or if the field exists and is required.
		if sf, ok := stem.Schema[inf.Field]; ok && sf.Required {
			return true
		}

	case "enum_without_values":
		if sf, ok := stem.Schema[inf.Field]; ok && sf.Type == "enum" && len(sf.Values) > 0 {
			return true
		}

	case "untyped_field":
		if sf, ok := stem.Schema[inf.Field]; ok && sf.Type != "" {
			return true
		}

	case "sequence_incomplete":
		if sf, ok := stem.Schema[inf.Field]; ok && sf.Prefix != "" && sf.Digits > 0 {
			return true
		}

	case "required_section", "optional_section":
		if sf, ok := stem.Schema[inf.Field]; ok && sectionInferenceCovered(inf, sf) {
			return true
		}

	case "required_understatement":
		if sf, ok := stem.Schema[inf.Field]; ok && sf.Required {
			return true
		}
	}

	return false
}

func sectionInferenceCovered(inf Inference, sf rules.SchemaField) bool {
	if sf.Type != "string" || inf.SourceDirective == "" || sf.Extract != inf.SourceDirective {
		return false
	}
	if inf.Type == "required_section" {
		return sf.Required
	}
	return true
}
