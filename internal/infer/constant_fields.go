package infer

import (
	"fmt"

	"github.com/pablontiv/rootline/internal/extract"
)

// DetectConstantFields finds fields with identical values across all records.
// A field that has the same value across 100% of records (minimum 3) is a constant.
func DetectConstantFields(records []*extract.Record) []Inference {
	if len(records) < 3 {
		return nil
	}

	// Count field presence and value frequency.
	type fieldInfo struct {
		values map[string]int
		count  int
	}
	fields := make(map[string]*fieldInfo)

	for _, rec := range records {
		for key, val := range rec.Frontmatter {
			fi, ok := fields[key]
			if !ok {
				fi = &fieldInfo{values: make(map[string]int)}
				fields[key] = fi
			}
			fi.count++
			fi.values[fmt.Sprintf("%v", val)]++
		}
	}

	total := len(records)
	var inferences []Inference

	for name, fi := range fields {
		// Must be present in all records.
		if fi.count != total {
			continue
		}
		// Must have exactly one unique value.
		if len(fi.values) != 1 {
			continue
		}
		// Get the single value.
		var value string
		for v := range fi.values {
			value = v
		}
		inferences = append(inferences, Inference{
			Type:    "constant_field",
			Source:  "",
			Field:   name,
			Value:   value,
			Message: fmt.Sprintf("field %q has constant value %q across all %d records", name, value, total),
		})
	}

	return inferences
}
