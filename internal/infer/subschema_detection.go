package infer

import (
	"fmt"

	"github.com/pablontiv/rootline/internal/extract"
)

// DetectSubSchemas groups records by a discriminator field and detects
// fields that are exclusive to a specific group (type_specific_field)
// versus common across all groups (common_field).
// Groups with fewer than 2 records are excluded (insufficient data).
func DetectSubSchemas(records []*extract.Record, discriminator string) []Inference {
	if discriminator == "" {
		discriminator = "tipo"
	}

	// Group records by discriminator value.
	groups := make(map[string][]*extract.Record)
	for _, rec := range records {
		val, ok := rec.Frontmatter[discriminator]
		if !ok {
			continue
		}
		key := fmt.Sprintf("%v", val)
		groups[key] = append(groups[key], rec)
	}

	if len(groups) < 2 {
		return nil
	}

	// Analyze each group to get field sets.
	groupFields := make(map[string]map[string]bool) // group → set of field names
	for groupName, recs := range groups {
		if len(recs) < 2 {
			continue
		}
		schema := Analyze(recs)
		fields := make(map[string]bool)
		for name := range schema.Fields {
			if name == discriminator {
				continue
			}
			fields[name] = true
		}
		groupFields[groupName] = fields
	}

	if len(groupFields) < 2 {
		return nil
	}

	// Find common fields (present in ALL groups).
	common := make(map[string]bool)
	first := true
	for _, fields := range groupFields {
		if first {
			for f := range fields {
				common[f] = true
			}
			first = false
			continue
		}
		for f := range common {
			if !fields[f] {
				delete(common, f)
			}
		}
	}

	var inferences []Inference

	// Detect type-specific fields.
	for groupName, fields := range groupFields {
		for field := range fields {
			if common[field] {
				continue
			}
			inferences = append(inferences, Inference{
				Type:    "type_specific_field",
				Source:  discriminator,
				Field:   field,
				Value:   groupName,
				Message: fmt.Sprintf("field %q is specific to %s=%q (not present in all groups)", field, discriminator, groupName),
			})
		}
	}

	return inferences
}
