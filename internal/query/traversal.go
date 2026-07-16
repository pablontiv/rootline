package query

import (
	"context"

	"github.com/expr-lang/expr/vm"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
)

// TraversalOptions configures link-traversal filtering of query candidates.
// A nil sub-where disables that direction; an empty string matches any
// linked record (pure existence or type check).
type TraversalOptions struct {
	// HasInbound retains candidates with at least one inbound edge whose
	// SOURCE record matches this sub-where.
	HasInbound *string
	// HasOutbound retains candidates with at least one outbound edge whose
	// TARGET record matches this sub-where.
	HasOutbound *string
	// InboundType restricts inbound edges to this link type ("" = any).
	InboundType string
	// OutboundType restricts outbound edges to this link type ("" = any).
	OutboundType string
}

// Active reports whether any traversal predicate is set.
func (o TraversalOptions) Active() bool {
	return o.HasInbound != nil || o.HasOutbound != nil
}

// FilterByTraversal retains candidates that satisfy the traversal predicates
// against the given graph. Sub-wheres reuse the --where grammar and are
// evaluated against the linked record. Edges whose counterpart record is not
// a graph node (broken links) never satisfy a predicate.
func FilterByTraversal(ctx context.Context, candidates []*extract.Record, g *graph.Graph, opts TraversalOptions) ([]*extract.Record, error) {
	inboundProgram, err := compileSubWhere(opts.HasInbound)
	if err != nil {
		return nil, err
	}
	outboundProgram, err := compileSubWhere(opts.HasOutbound)
	if err != nil {
		return nil, err
	}

	// Reverse index: target path → inbound edges.
	var inboundEdges map[string][]graph.Edge
	if opts.HasInbound != nil {
		inboundEdges = make(map[string][]graph.Edge)
		for _, edges := range g.Edges {
			for _, e := range edges {
				inboundEdges[e.Target] = append(inboundEdges[e.Target], e)
			}
		}
	}

	filtered := []*extract.Record{}
	for _, rec := range candidates {
		if opts.HasInbound != nil {
			ok, err := matchesAnyEdge(ctx, inboundEdges[rec.Path], opts.InboundType, inboundProgram, g, edgeSource)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
		}
		if opts.HasOutbound != nil {
			ok, err := matchesAnyEdge(ctx, g.Edges[rec.Path], opts.OutboundType, outboundProgram, g, edgeTarget)
			if err != nil {
				return nil, err
			}
			if !ok {
				continue
			}
		}
		filtered = append(filtered, rec)
	}
	return filtered, nil
}

// compileSubWhere compiles a traversal sub-where. Nil or empty means
// "match any linked record" and yields a nil program.
func compileSubWhere(subWhere *string) (*vm.Program, error) {
	if subWhere == nil || *subWhere == "" {
		return nil, nil
	}
	return CompileWhere(*subWhere)
}

// edgeSource and edgeTarget select which end of an edge the sub-where
// is evaluated against.
func edgeSource(e graph.Edge) string { return e.Source }
func edgeTarget(e graph.Edge) string { return e.Target }

// matchesAnyEdge reports whether any edge passes the type filter and has a
// counterpart record (selected by end) that exists and matches the program.
func matchesAnyEdge(ctx context.Context, edges []graph.Edge, linkType string, program *vm.Program, g *graph.Graph, end func(graph.Edge) string) (bool, error) {
	for _, e := range edges {
		if linkType != "" && e.Type != linkType {
			continue
		}
		linked, exists := g.Nodes[end(e)]
		if !exists {
			continue
		}
		if program == nil {
			return true, nil
		}
		match, err := MatchRecord(ctx, program, linked, nil)
		if err != nil {
			return false, err
		}
		if match {
			return true, nil
		}
	}
	return false, nil
}
