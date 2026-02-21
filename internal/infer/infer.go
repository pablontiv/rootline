// Package infer analyzes frontmatter records to infer a .stem schema.
package infer

import (
	"fmt"
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// FieldStats holds statistics about a single frontmatter field.
type FieldStats struct {
	Name         string   `json:"name"`
	Count        int      `json:"count"`
	TotalRecords int      `json:"total_records"`
	UniqueValues []string `json:"unique_values"`
	InferredType string   `json:"inferred_type"`
	HasList      bool     `json:"has_list"`
}

// InferredSchema is the result of analyzing a set of records.
type InferredSchema struct {
	Fields               map[string]*FieldStats       `json:"fields"`
	Schema               map[string]rules.SchemaField `json:"schema"`
	TotalFiles           int                          `json:"total_files"`
	FilesWithFrontmatter int                          `json:"files_with_frontmatter"`
	FilesWithout         int                          `json:"files_without"`
}

// Analyze processes records and infers a schema from frontmatter patterns.
func Analyze(records []*extract.Record) *InferredSchema {
	result := &InferredSchema{
		Fields:     make(map[string]*FieldStats),
		Schema:     make(map[string]rules.SchemaField),
		TotalFiles: len(records),
	}

	if len(records) == 0 {
		return result
	}

	total := len(records)

	// Count files with and without frontmatter
	for _, rec := range records {
		if len(rec.Frontmatter) > 0 {
			result.FilesWithFrontmatter++
		} else {
			result.FilesWithout++
		}
	}

	// Collect field statistics
	valueSets := make(map[string]map[string]int) // field -> value -> count
	listFields := make(map[string]bool)

	for _, rec := range records {
		for key, val := range rec.Frontmatter {
			if _, exists := result.Fields[key]; !exists {
				result.Fields[key] = &FieldStats{
					Name:         key,
					TotalRecords: total,
				}
				valueSets[key] = make(map[string]int)
			}
			result.Fields[key].Count++

			// Detect list type
			switch v := val.(type) {
			case []any:
				listFields[key] = true
				for _, item := range v {
					valueSets[key][fmt.Sprintf("%v", item)]++
				}
			default:
				strVal := fmt.Sprintf("%v", v)
				valueSets[key][strVal]++
			}
		}
	}

	// Analyze each field and build schema
	for name, stats := range result.Fields {
		values := valueSets[name]

		// Collect unique values sorted
		uniq := make([]string, 0, len(values))
		for v := range values {
			uniq = append(uniq, v)
		}
		sort.Strings(uniq)
		stats.UniqueValues = uniq

		// Check for list
		switch {
		case listFields[name]:
			stats.HasList = true
			stats.InferredType = "list"
		case len(uniq) <= 20 && float64(stats.Count)/float64(total) > 0.5:
			// Enum: few unique values and present in >50% of records
			stats.InferredType = "enum"
		default:
			stats.InferredType = "string"
		}

		// Build SchemaField
		sf := rules.SchemaField{
			Type: stats.InferredType,
		}

		// Required: present in >80% of records
		if float64(stats.Count)/float64(total) > 0.8 {
			sf.Required = true
		}

		// Enum values
		if stats.InferredType == "enum" {
			sf.Values = uniq
		}

		result.Schema[name] = sf
	}

	return result
}
