// Package graph builds a directed graph from document links
// and provides cycle detection and broken link analysis.
package graph

import (
	"context"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/picokit/fuzzy"
	"github.com/pablontiv/rootline/internal/extract"
)

// Graph represents a directed graph of document links.
type Graph struct {
	// Nodes maps record path → record.
	Nodes map[string]*extract.Record
	// Edges maps source path → list of edges.
	Edges map[string][]Edge
}

// Edge represents a directed link from one document to another.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Line   int    `json:"line"`

	// noFallback excludes the edge from basename fallback resolution.
	// Markdown targets are path-resolved upstream; an unresolved one must
	// stay verbatim so BrokenLinks reports it, matching validate.
	noFallback bool
}

// BrokenLink represents a link whose target doesn't match any record.
type BrokenLink struct {
	Source      string   `json:"source"`
	Target      string   `json:"target"`
	Type        string   `json:"type"`
	Line        int      `json:"line"`
	Suggestions []string `json:"suggestions,omitempty"`
}

// Build constructs a Graph from a slice of records.
// Link targets are resolved relative to the source record's directory.
// Unresolved targets get a basename fallback lookup across all nodes.
func Build(_ context.Context, records []*extract.Record) *Graph {
	g := &Graph{
		Nodes: make(map[string]*extract.Record, len(records)),
		Edges: make(map[string][]Edge),
	}

	for _, rec := range records {
		g.Nodes[rec.Path] = rec
	}

	for _, rec := range records {
		for _, link := range rec.Links {
			target := link.Target
			markdown := link.Style == extract.StyleMarkdown
			if !markdown {
				target = resolveTarget(rec.Path, link.Target)
			}
			g.Edges[rec.Path] = append(g.Edges[rec.Path], Edge{
				Source:     rec.Path,
				Target:     target,
				Type:       link.Type,
				Line:       link.Line,
				noFallback: markdown,
			})
		}
	}

	// Second pass: resolve unmatched targets by basename fallback.
	g.resolveByBasename()

	return g
}

// SortedNodes returns every node path in lexical order.
//
// Nodes live in a map, and Go randomizes map iteration order per range
// statement, so ranging g.Nodes directly leaks that randomization into any
// rendered output. Determinism is a property of the graph, not of a single
// renderer: every consumer must go through this accessor so the JSON, DOT and
// Mermaid renderers cannot drift apart. Returns an empty slice for an empty graph.
func (g *Graph) SortedNodes() []string {
	nodes := make([]string, 0, len(g.Nodes))
	for path := range g.Nodes {
		nodes = append(nodes, path)
	}
	sort.Strings(nodes)
	return nodes
}

// SortedEdges returns every edge from every source under the total order
// (source, target, line, type) defined by compareEdges.
// Returns an empty slice when the graph has no edges.
func (g *Graph) SortedEdges() []Edge {
	total := 0
	for _, edges := range g.Edges {
		total += len(edges)
	}
	all := make([]Edge, 0, total)
	for _, edges := range g.Edges {
		all = append(all, edges...)
	}
	sort.Slice(all, func(i, j int) bool {
		return compareEdges(all[i], all[j]) < 0
	})
	return all
}

// compareEdges orders two edges by source path, then target path, then line
// number, then link type. The four keys together are a total order: two edges
// agreeing on all of them are indistinguishable in the rendered output, so no
// further tiebreak is observable.
func compareEdges(a, b Edge) int {
	if c := strings.Compare(a.Source, b.Source); c != 0 {
		return c
	}
	if c := strings.Compare(a.Target, b.Target); c != 0 {
		return c
	}
	if a.Line != b.Line {
		return a.Line - b.Line
	}
	return strings.Compare(a.Type, b.Type)
}

// resolveByBasename rewrites edge targets that don't match any node
// by searching for a unique basename match (with and without .md).
func (g *Graph) resolveByBasename() {
	// Build basename index: basename → list of full paths.
	idx := make(map[string][]string, len(g.Nodes))
	for path := range g.Nodes {
		base := filepath.Base(path)
		idx[base] = append(idx[base], path)
		// Also index without .md extension.
		noExt := strings.TrimSuffix(base, ".md")
		if noExt != base {
			idx[noExt] = append(idx[noExt], path)
		}
	}

	for src, edges := range g.Edges {
		for i, edge := range edges {
			if edge.noFallback {
				continue
			}
			if _, exists := g.Nodes[edge.Target]; exists {
				continue // already resolved
			}
			// Try basename lookup.
			if matches, ok := idx[edge.Target]; ok && len(matches) == 1 {
				g.Edges[src][i].Target = matches[0]
			}
		}
	}
}

// DetectCycles finds all cycles in the graph using DFS.
// Each cycle is returned as a path slice ending with the repeated node.
func (g *Graph) DetectCycles() [][]string {
	const (
		white = 0 // unvisited
		gray  = 1 // in current DFS path
		black = 2 // fully processed
	)

	color := make(map[string]int, len(g.Nodes))
	parent := make(map[string]string)
	var cycles [][]string

	var dfs func(node string)
	dfs = func(node string) {
		color[node] = gray
		for _, edge := range g.Edges[node] {
			target := edge.Target
			if _, exists := g.Nodes[target]; !exists {
				continue // skip broken links
			}
			switch color[target] {
			case white:
				parent[target] = node
				dfs(target)
			case gray:
				// Back edge found — extract cycle by walking
				// parent chain from node back to target.
				var cycle []string
				cur := node
				for cur != target {
					cycle = append(cycle, cur)
					cur = parent[cur]
				}
				cycle = append(cycle, target)
				// Reverse to get forward order.
				for i, j := 0, len(cycle)-1; i < j; i, j = i+1, j-1 {
					cycle[i], cycle[j] = cycle[j], cycle[i]
				}
				// Close the cycle.
				cycle = append(cycle, cycle[0])
				cycles = append(cycles, cycle)
			}
		}
		color[node] = black
	}

	for node := range g.Nodes {
		if color[node] == white {
			dfs(node)
		}
	}

	return cycles
}

// BrokenLinks returns links whose targets don't match any record in the graph.
func (g *Graph) BrokenLinks() []BrokenLink {
	nodeNames := make([]string, 0, len(g.Nodes))
	for name := range g.Nodes {
		nodeNames = append(nodeNames, name)
	}

	var broken []BrokenLink
	for _, edges := range g.Edges {
		for _, edge := range edges {
			if _, exists := g.Nodes[edge.Target]; !exists {
				bl := BrokenLink{
					Source: edge.Source,
					Target: edge.Target,
					Type:   edge.Type,
					Line:   edge.Line,
				}
				if suggestions := fuzzy.MatchN(edge.Target, nodeNames, 3); len(suggestions) > 0 {
					bl.Suggestions = suggestions
				}
				broken = append(broken, bl)
			}
		}
	}
	return broken
}

// TraceOptions controls trace traversal behavior.
type TraceOptions struct {
	Reverse  bool   // follow incoming edges instead of outgoing
	MaxDepth int    // 0 = unlimited
	EdgeType string // filter by edge type (field name); empty = all
}

// TraceNode represents a node discovered during trace traversal.
type TraceNode struct {
	Path  string `json:"path"`
	Depth int    `json:"depth"`
	Via   string `json:"via,omitempty"`  // edge type that led here
	From  string `json:"from,omitempty"` // source record path
}

// TraceResult is the output of a trace traversal.
type TraceResult struct {
	Version int         `json:"version"`
	Kind    string      `json:"kind"`
	Start   string      `json:"start"`
	Nodes   []TraceNode `json:"nodes"`
}

// Trace performs BFS through the graph from a starting node,
// following outgoing edges (or incoming if Reverse is true).
func (g *Graph) Trace(start string, opts TraceOptions) *TraceResult {
	result := &TraceResult{
		Version: 1,
		Kind:    "rootline/trace",
		Start:   start,
	}

	if _, exists := g.Nodes[start]; !exists {
		return result
	}

	// Build reverse edge index if needed.
	var reverseEdges map[string][]Edge
	if opts.Reverse {
		reverseEdges = make(map[string][]Edge)
		for src, edges := range g.Edges {
			for _, e := range edges {
				if _, ok := g.Nodes[e.Target]; ok {
					reverseEdges[e.Target] = append(reverseEdges[e.Target], Edge{
						Source: e.Target,
						Target: src,
						Type:   e.Type,
						Line:   e.Line,
					})
				}
			}
		}
	}

	type queueItem struct {
		path  string
		depth int
		via   string
		from  string
	}

	visited := map[string]bool{start: true}
	queue := []queueItem{{path: start, depth: 0}}

	for len(queue) > 0 {
		item := queue[0]
		queue = queue[1:]

		// Add to result (skip start node itself).
		if item.depth > 0 {
			node := TraceNode{
				Path:  item.path,
				Depth: item.depth,
				Via:   item.via,
				From:  item.from,
			}
			result.Nodes = append(result.Nodes, node)
		}

		// Check depth limit.
		if opts.MaxDepth > 0 && item.depth >= opts.MaxDepth {
			continue
		}

		// Get edges to follow.
		var edges []Edge
		if opts.Reverse {
			edges = reverseEdges[item.path]
		} else {
			edges = g.Edges[item.path]
		}

		for _, e := range edges {
			target := e.Target
			if _, ok := g.Nodes[target]; !ok {
				continue // skip broken links
			}
			if opts.EdgeType != "" && e.Type != opts.EdgeType {
				continue
			}
			if visited[target] {
				continue
			}
			visited[target] = true
			queue = append(queue, queueItem{
				path:  target,
				depth: item.depth + 1,
				via:   e.Type,
				from:  item.path,
			})
		}
	}

	return result
}

// resolveTarget resolves a link target relative to the source record's directory.
// If the target already looks like a path (contains / or .), it's resolved
// relative to the source's directory. Otherwise it's used as-is.
func resolveTarget(sourcePath, target string) string {
	if strings.Contains(target, "/") || strings.Contains(target, "..") {
		dir := filepath.Dir(sourcePath)
		return filepath.Clean(filepath.Join(dir, target))
	}
	return target
}
