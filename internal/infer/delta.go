package infer

import (
	"github.com/pablontiv/rootline/internal/rules"
)

// FilterCoveredInferences removes inferences that are already covered
// by the existing .stem schema. Returns only uncovered (delta) inferences.
func FilterCoveredInferences(inferences []Inference, stem *rules.StemFile) []Inference {
	if stem == nil || len(stem.Schema) == 0 {
		return inferences
	}

	var deltas []Inference
	for _, inf := range inferences {
		if isCovered(inf, stem) {
			continue
		}
		deltas = append(deltas, inf)
	}
	return deltas
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

	case "required_understatement":
		if sf, ok := stem.Schema[inf.Field]; ok && sf.Required {
			return true
		}
	}

	return false
}
