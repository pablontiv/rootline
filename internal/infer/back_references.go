package infer

import (
	"fmt"

	"github.com/pablontiv/rootline/internal/graph"
)

// unidirectionalTypes are link types that do not require a back-reference.
var unidirectionalTypes = map[string]bool{
	"blocks":  true,
	"parent":  true,
	"depends": true,
}

// DetectMissingBackReferences checks that bidirectional links have reciprocal references.
// For each edge A→B with a reciprocal link type, it checks that B→A also exists.
// Unidirectional types (blocks, parent, depends) are skipped.
func DetectMissingBackReferences(g *graph.Graph) []Inference {
	// Build reverse index: target → set of sources, keyed by type.
	type edgeKey struct {
		source string
		target string
	}
	exists := make(map[edgeKey]bool)

	for _, edges := range g.Edges {
		for _, e := range edges {
			exists[edgeKey{source: e.Source, target: e.Target}] = true
		}
	}

	seen := make(map[edgeKey]bool)
	var inferences []Inference

	for _, edges := range g.Edges {
		for _, e := range edges {
			if unidirectionalTypes[e.Type] {
				continue
			}
			// Only check edges whose target is a known node.
			if _, isNode := g.Nodes[e.Target]; !isNode {
				continue
			}

			reverse := edgeKey{source: e.Target, target: e.Source}
			if exists[reverse] {
				continue
			}
			if seen[reverse] {
				continue
			}
			seen[reverse] = true

			inferences = append(inferences, Inference{
				Category: 7,
				Type:     "missing_back_reference",
				Source:   e.Target,
				Field:    "links",
				Value:    e.Source,
				Message:  fmt.Sprintf("%s is referenced by %s (type %q) but has no back-reference", e.Target, e.Source, e.Type),
			})
		}
	}

	return inferences
}
