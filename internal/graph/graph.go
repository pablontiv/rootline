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

	// resolution carries extract.Link.Resolution for this edge. When the
	// links were prepared by rules.PrepareLinks it is authoritative and
	// BrokenLinks trusts it instead of the node set; when it is the zero
	// value nothing resolved this link and BrokenLinks falls back to node
	// membership, which is how in-memory callers have always behaved.
	resolution string

	// markdown excludes the edge from the unchecked basename fallback: a
	// markdown destination names a path literally, so it stays verbatim.
	markdown bool
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
//
// Targets are expected to be resolved already, by rules.PrepareLinks, which
// rewrites them to root-relative node keys and records whether each resolved.
// Build no longer does any resolution of its own beyond a directory join for
// links nothing has looked at, so graph and validate cannot disagree about
// which links are broken (issue #62).
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
			// An unprepared wikilink still gets the directory join it always
			// had; a prepared one already carries its root-relative key.
			if link.Resolution == extract.LinkUnchecked && link.Style != extract.StyleMarkdown {
				target = resolveTarget(rec.Path, link.Target)
			}
			g.Edges[rec.Path] = append(g.Edges[rec.Path], Edge{
				Source:     rec.Path,
				Target:     target,
				Type:       link.Type,
				Line:       link.Line,
				resolution: link.Resolution,
				markdown:   link.Style == extract.StyleMarkdown,
			})
		}
	}

	// Links nothing resolved keep the historical basename fallback, so callers
	// that build a graph without preparing links behave exactly as before.
	g.resolveUncheckedByBasename()

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

// DetectCycles finds cycles in the graph using DFS over a canonical spanning
// forest. Each cycle is returned as a path slice ending with the repeated node.
//
// The result is a function of the graph alone: it does not depend on Go map
// iteration order, nor on the order records were passed to Build. Three things
// are pinned to make that true (issue #114):
//
//   - DFS roots come from SortedNodes(), the same lexical order the renderers
//     use, so root order and rendered node order cannot drift apart.
//   - Adjacency is de-duplicated and sorted by sortedAdjacency, so traversal
//     follows the graph rather than the scan order that filled g.Edges.
//   - Every cycle is rotated to start at its lexicographically smallest member
//     by canonicalCycle, then the whole list is de-duplicated and sorted under
//     compareCycles.
//
// Rotation and outer ordering are representation, not semantics: [b a b] and
// [a b a] describe the same directed cycle, and picking one of them is what
// stops two runs printing the same graph two ways.
//
// LIMITATION: this detects back edges over one spanning forest, not every
// elementary circuit in the graph. Cycles that overlap are represented, not
// exhaustively enumerated, so len(cycles) is a stable count of detected back
// edges — not the number of distinct circuits.
func (g *Graph) DetectCycles() [][]string {
	const (
		white = 0 // unvisited
		gray  = 1 // in current DFS path
		black = 2 // fully processed
	)

	adjacency := g.sortedAdjacency()
	color := make(map[string]int, len(g.Nodes))
	parent := make(map[string]string)
	var cycles [][]string

	var dfs func(node string)
	dfs = func(node string) {
		color[node] = gray
		for _, target := range adjacency[node] {
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
				cycles = append(cycles, canonicalCycle(cycle))
			}
		}
		color[node] = black
	}

	for _, node := range g.SortedNodes() {
		if color[node] == white {
			dfs(node)
		}
	}

	sort.Slice(cycles, func(i, j int) bool {
		return compareCycles(cycles[i], cycles[j]) < 0
	})
	return dedupeCycles(cycles)
}

// sortedAdjacency returns, for each node, the distinct in-graph targets it
// links to, in lexical order. Broken links are skipped, exactly as the raw edge
// walk skipped them.
//
// Both operations matter for determinism. Sorting removes the dependency on the
// order records were scanned, which is the leak SortedNodes alone does not
// close: two scans that produce the same graph in a different order would
// otherwise explore it differently. De-duplication removes parallel links — a
// document linking twice to the same target fires the same back edge twice and
// yields two byte-identical cycles, which carry no information a consumer can
// act on.
func (g *Graph) sortedAdjacency() map[string][]string {
	adjacency := make(map[string][]string, len(g.Edges))
	for source, edges := range g.Edges {
		seen := make(map[string]bool, len(edges))
		targets := make([]string, 0, len(edges))
		for _, edge := range edges {
			if _, exists := g.Nodes[edge.Target]; !exists {
				continue // skip broken links
			}
			if seen[edge.Target] {
				continue
			}
			seen[edge.Target] = true
			targets = append(targets, edge.Target)
		}
		sort.Strings(targets)
		adjacency[source] = targets
	}
	return adjacency
}

// canonicalCycle rotates an open cycle so it starts at its lexicographically
// smallest member, then closes it by repeating that member.
//
// Rotating a directed cycle preserves its edge set, so every rotation describes
// the same cycle and only one of them can be the printed answer. The members
// are distinct — the parent chain walks strictly upward until it reaches the
// back-edge target — so the minimum is unique and the rotation unambiguous.
func canonicalCycle(open []string) []string {
	if len(open) == 0 {
		return nil
	}
	smallest := 0
	for i, node := range open {
		if node < open[smallest] {
			smallest = i
		}
	}
	rotated := make([]string, 0, len(open)+1)
	rotated = append(rotated, open[smallest:]...)
	rotated = append(rotated, open[:smallest]...)
	return append(rotated, rotated[0])
}

// compareCycles orders two cycles element by element, shorter first when one is
// a prefix of the other. Cycles are already canonically rotated, so this is a
// total order over the printed representations.
func compareCycles(a, b []string) int {
	for i := 0; i < len(a) && i < len(b); i++ {
		if c := strings.Compare(a[i], b[i]); c != 0 {
			return c
		}
	}
	return len(a) - len(b)
}

// dedupeCycles drops adjacent duplicates from a sorted cycle list. Two
// canonically rotated cycles that compare equal are indistinguishable in every
// rendered surface, so keeping both would only inflate the count.
func dedupeCycles(sorted [][]string) [][]string {
	if len(sorted) == 0 {
		return sorted
	}
	unique := sorted[:1]
	for _, cycle := range sorted[1:] {
		if compareCycles(unique[len(unique)-1], cycle) != 0 {
			unique = append(unique, cycle)
		}
	}
	return unique
}

// BrokenLinks returns links whose targets do not resolve.
//
// Brokenness comes from the resolver, not from node membership. A target that
// exists on disk but is not a governed record resolved fine — the schema
// declares what is governed, not what exists — so it is an edge to a non-node
// rather than a broken link. Reporting it broken is what made graph and
// validate disagree in issue #62. Edges nothing resolved keep the historical
// node-membership check.
func (g *Graph) BrokenLinks() []BrokenLink {
	nodeNames := g.SortedNodes()

	var broken []BrokenLink
	for _, edge := range g.SortedEdges() {
		if edge.isBroken(g.Nodes) {
			bl := BrokenLink{
				Source: edge.Source,
				Target: edge.Target,
				Type:   edge.Type,
				Line:   edge.Line,
			}
			if suggestions := brokenLinkSuggestions(edge.Target, nodeNames); len(suggestions) > 0 {
				bl.Suggestions = suggestions
			}
			broken = append(broken, bl)
		}
	}
	return broken
}

// brokenLinkSuggestions preserves fuzzy.MatchN's threshold while adding the
// total order its distance-only ranking lacks. Asking for every in-threshold
// candidate before truncating means an equal-distance tie is resolved
// lexically rather than by the map order that supplied the candidates.
func brokenLinkSuggestions(target string, nodeNames []string) []string {
	suggestions := fuzzy.MatchN(target, nodeNames, len(nodeNames))
	sort.Slice(suggestions, func(i, j int) bool {
		iDistance := fuzzy.Distance(target, suggestions[i])
		jDistance := fuzzy.Distance(target, suggestions[j])
		if iDistance != jDistance {
			return iDistance < jDistance
		}
		return suggestions[i] < suggestions[j]
	})
	if len(suggestions) > 3 {
		suggestions = suggestions[:3]
	}
	return suggestions
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

// resolveUncheckedByBasename rewrites unchecked edge targets that match no node
// by searching for a unique basename match, with and without ".md".
//
// Prepared edges are excluded: rules.PrepareLinks already applied the same
// fallback against the same record set, and re-running it here could rewrite a
// target the resolver deliberately left alone.
func (g *Graph) resolveUncheckedByBasename() {
	idx := make(map[string][]string, len(g.Nodes))
	for path := range g.Nodes {
		base := filepath.Base(path)
		idx[base] = append(idx[base], path)
		if noExt := strings.TrimSuffix(base, ".md"); noExt != base {
			idx[noExt] = append(idx[noExt], path)
		}
	}

	for src, edges := range g.Edges {
		for i, edge := range edges {
			if edge.resolution != extract.LinkUnchecked || edge.markdown {
				continue
			}
			if _, exists := g.Nodes[edge.Target]; exists {
				continue
			}
			if matches, ok := idx[edge.Target]; ok && len(matches) == 1 {
				g.Edges[src][i].Target = matches[0]
			}
		}
	}
}

// isBroken reports whether an edge should be listed as a broken link.
//
// A prepared edge answers from its own resolution outcome. An unprepared one
// (resolution is the zero value) falls back to node membership, preserving the
// behavior every in-memory caller relied on before resolution existed.
func (e Edge) isBroken(nodes map[string]*extract.Record) bool {
	switch e.resolution {
	case extract.LinkResolved:
		return false
	case extract.LinkUnresolved:
		return true
	default:
		_, exists := nodes[e.Target]
		return !exists
	}
}
