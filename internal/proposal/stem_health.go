package proposal

import (
	"fmt"

	"github.com/pablontiv/rootline/internal/rules"
)

// DetectRemoveStemField generates remove_stem_field proposals from stem-health
// check results. It targets field-override and type-consistency errors where
// the child .stem redundantly redefines a field inherited from a parent.
//
// Exception: fields with type "sequence" (like id) are skipped because each
// level intentionally uses different prefix/digits.
func DetectRemoveStemField(checks []rules.StemHealthCheck) []Proposal {
	var proposals []Proposal

	for _, c := range checks {
		if c.Status == "pass" {
			continue
		}
		if c.Field == "" || c.Path == "" {
			continue
		}

		switch c.Name {
		case "field-override":
			// Skip sequence fields (id) — each level needs its own prefix/digits.
			if isSequenceField(c) {
				continue
			}
			proposals = append(proposals, Proposal{
				Type:        RemoveStemField,
				Field:       c.Field,
				Description: fmt.Sprintf("remove redundant field %q from child .stem (inherited from parent)", c.Field),
				Paths:       []string{c.Path},
			})

		case "type-consistency":
			// Child changes the type of an inherited field — remove from child.
			proposals = append(proposals, Proposal{
				Type:        RemoveStemField,
				Field:       c.Field,
				Description: fmt.Sprintf("remove field %q from child .stem to resolve type conflict with parent", c.Field),
				Paths:       []string{c.Path},
			})
		}
	}

	return deduplicateProposals(proposals)
}

// isSequenceField returns true if the check is about a field that uses
// sequence auto-numbering (like "id"). These overrides are intentional.
func isSequenceField(c rules.StemHealthCheck) bool {
	return c.Field == "id"
}

// deduplicateProposals merges proposals that target the same path+field.
// A field can generate both field-override and type-consistency — we only
// need one remove proposal.
func deduplicateProposals(proposals []Proposal) []Proposal {
	type key struct {
		path  string
		field string
	}
	seen := make(map[key]bool)
	var result []Proposal

	for _, p := range proposals {
		for _, path := range p.Paths {
			k := key{path: path, field: p.Field}
			if seen[k] {
				continue
			}
			seen[k] = true
			result = append(result, p)
		}
	}
	return result
}
