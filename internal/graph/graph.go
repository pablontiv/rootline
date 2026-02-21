// Package graph builds a directed graph from document links
// and provides cycle detection and broken link analysis.
package graph

import (
	"path/filepath"
	"strings"

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
}

// BrokenLink represents a link whose target doesn't match any record.
type BrokenLink struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Line   int    `json:"line"`
}

// Build constructs a Graph from a slice of records.
// Link targets are resolved relative to the source record's directory.
// Unresolved targets get a basename fallback lookup across all nodes.
func Build(records []*extract.Record) *Graph {
	g := &Graph{
		Nodes: make(map[string]*extract.Record, len(records)),
		Edges: make(map[string][]Edge),
	}

	for _, rec := range records {
		g.Nodes[rec.Path] = rec
	}

	for _, rec := range records {
		for _, link := range rec.Links {
			target := resolveTarget(rec.Path, link.Target)
			g.Edges[rec.Path] = append(g.Edges[rec.Path], Edge{
				Source: rec.Path,
				Target: target,
				Type:   link.Type,
				Line:   link.Line,
			})
		}
	}

	// Second pass: resolve unmatched targets by basename fallback.
	g.resolveByBasename()

	return g
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
	var broken []BrokenLink
	for _, edges := range g.Edges {
		for _, edge := range edges {
			if _, exists := g.Nodes[edge.Target]; !exists {
				broken = append(broken, BrokenLink{
					Source: edge.Source,
					Target: edge.Target,
					Type:   edge.Type,
					Line:   edge.Line,
				})
			}
		}
	}
	return broken
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
