package infer

import (
	"fmt"
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// DetectLinkTypes validates each record's links against its effective allowed list.
// Records without an allowed list contribute to a global ≥80% consensus suggestion.
func DetectLinkTypes(records []*extract.Record, schemas map[string]rules.LinkSchema) []Inference {
	typeCounts := make(map[string]int)
	totalLinks := 0
	var inferences []Inference
	seen := make(map[string]map[string]bool)

	for _, rec := range records {
		schema := schemas[rec.Path]
		for _, link := range rec.Links {
			if len(schema.Allowed) == 0 {
				typeCounts[link.Type]++
				totalLinks++
				continue
			}

			allowed := false
			for _, linkType := range schema.Allowed {
				if link.Type == linkType {
					allowed = true
					break
				}
			}
			if allowed {
				continue
			}
			if seen[rec.Path] == nil {
				seen[rec.Path] = make(map[string]bool)
			}
			if seen[rec.Path][link.Type] {
				continue
			}
			seen[rec.Path][link.Type] = true

			inferences = append(inferences, Inference{
				Type:    "link_type_violation",
				Source:  rec.Path,
				Field:   "links.allowed",
				Value:   link.Type,
				Message: fmt.Sprintf("link type %q is not in allowed list %v", link.Type, schema.Allowed),
			})
		}
	}

	if totalLinks > 0 {
		threshold := float64(totalLinks) * 0.8
		var suggested []string
		for linkType, count := range typeCounts {
			if float64(count) >= threshold {
				suggested = append(suggested, linkType)
			}
		}
		sort.Strings(suggested)

		if len(suggested) > 0 {
			inferences = append(inferences, Inference{
				Type:    "link_type_suggestion",
				Source:  "",
				Field:   "links.allowed",
				Value:   fmt.Sprintf("%v", suggested),
				Message: fmt.Sprintf("observed link types with ≥80%% consensus: %v", suggested),
			})
		}
	}

	return inferences
}
