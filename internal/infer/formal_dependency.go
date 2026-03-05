package infer

import (
	"fmt"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// semanticLinkTypes are wiki-link prefixes that denote formal dependencies.
var semanticLinkTypes = map[string]bool{
	"blocks":  true,
	"relates": true,
	"extends": true,
}

// DetectFormalDependencies extracts formal dependencies from wiki-links
// with semantic prefixes (blocks:, relates:, extends:) and informal
// dependency candidates from "Dependencias" body sections.
func DetectFormalDependencies(records []*extract.Record) []Inference {
	var inferences []Inference

	for _, rec := range records {
		// Formal dependencies from wiki-links.
		for _, link := range rec.Links {
			if !semanticLinkTypes[link.Type] {
				continue
			}
			inferences = append(inferences, Inference{
				Type:    "formal_dependency",
				Source:  rec.Path,
				Field:   link.Type,
				Value:   link.Target,
				Message: fmt.Sprintf("formal dependency %s:%s in %s", link.Type, link.Target, rec.Path),
			})
		}

		// Informal dependency candidates from "Dependencias" section.
		if rec.AST == nil || rec.Body == "" {
			continue
		}

		source := []byte(rec.Body)
		sections := extract.ExtractSections(rec.AST, source)

		for _, sec := range sections {
			if sec.Heading != "Dependencias" {
				continue
			}
			// Extract list items from the section content.
			for _, line := range strings.Split(sec.Content, "\n") {
				trimmed := strings.TrimSpace(line)
				if !strings.HasPrefix(trimmed, "- ") && !strings.HasPrefix(trimmed, "* ") {
					continue
				}
				text := strings.TrimSpace(trimmed[2:])
				if text == "" {
					continue
				}
				inferences = append(inferences, Inference{
					Type:    "informal_dependency_candidate",
					Source:  rec.Path,
					Value:   text,
					Message: fmt.Sprintf("informal dependency candidate in %s: %s (requires_agent: true)", rec.Path, text),
				})
			}
		}
	}

	return inferences
}
