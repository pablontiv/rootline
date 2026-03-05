package infer

import (
	"fmt"
	"sort"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// DetectLinkTypes validates observed link types against a schema's allowed list.
// If LinkSchema.Allowed is defined, it validates observed link types against the list.
// If Allowed is empty, it infers allowed types from observed links using consensus ≥80%.
func DetectLinkTypes(records []*extract.Record, schema rules.LinkSchema) []Inference {
	// Collect all link types across all records.
	typeCounts := make(map[string]int)
	totalLinks := 0
	type linkOccurrence struct {
		source   string
		linkType string
	}
	var allLinks []linkOccurrence

	for _, rec := range records {
		for _, link := range rec.Links {
			typeCounts[link.Type]++
			totalLinks++
			allLinks = append(allLinks, linkOccurrence{
				source:   rec.Path,
				linkType: link.Type,
			})
		}
	}

	if totalLinks == 0 {
		return nil
	}

	var inferences []Inference

	if len(schema.Allowed) > 0 {
		// Validate mode: check observed types against allowed list.
		allowed := make(map[string]bool, len(schema.Allowed))
		for _, a := range schema.Allowed {
			allowed[a] = true
		}

		// Track violations per source to avoid duplicates.
		seen := make(map[string]map[string]bool)
		for _, occ := range allLinks {
			if allowed[occ.linkType] {
				continue
			}
			if seen[occ.source] == nil {
				seen[occ.source] = make(map[string]bool)
			}
			if seen[occ.source][occ.linkType] {
				continue
			}
			seen[occ.source][occ.linkType] = true

			inferences = append(inferences, Inference{
				Type:    "link_type_violation",
				Source:  occ.source,
				Field:   "links.allowed",
				Value:   occ.linkType,
				Message: fmt.Sprintf("link type %q is not in allowed list %v", occ.linkType, schema.Allowed),
			})
		}
	} else {
		// Inference mode: suggest allowed types based on consensus ≥80%.
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
