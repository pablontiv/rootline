package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var (
	graphFormat string
	graphCheck  bool
	graphWhere  []string
	graphOpen   bool
)

var graphCmd = &cobra.Command{
	Use:   "graph [path]",
	Short: "Build and visualize the document dependency graph",
	Long:  "Scan documents, build a directed graph from wiki-links, and output in DOT or Mermaid format.\nUse --check to validate for cycles and broken links without generating a diagram.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runGraph,
}

func init() {
	graphCmd.Flags().StringVar(&graphFormat, "format", "dot", "output format: dot or mermaid")
	graphCmd.Flags().BoolVar(&graphCheck, "check", false, "validate only (cycles + broken links), no diagram")
	graphCmd.Flags().StringArrayVar(&graphWhere, "where", nil, "filter expression (e.g. \"tipo != 'feature'\")")
	graphCmd.Flags().BoolVar(&graphOpen, "open", false, "render diagram in browser")
	rootCmd.AddCommand(graphCmd)
}

// GraphResult is the JSON output for rootline graph.
type GraphResult struct {
	Version     int                `json:"version"`
	Kind        string             `json:"kind"`
	Nodes       []string           `json:"nodes"`
	Edges       []graph.Edge       `json:"edges"`
	Cycles      [][]string         `json:"cycles"`
	BrokenLinks []graph.BrokenLink `json:"broken_links"`
}

func runGraph(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Validate --open flag combinations up front.
	if graphOpen && graphCheck {
		return fmt.Errorf("cannot use --open with --check")
	}
	if graphOpen && graphFormat == "dot" {
		return fmt.Errorf("cannot use --open with --format dot")
	}

	scanRoot := "."
	if len(args) > 0 {
		scanRoot = args[0]
	}

	absRoot, err := filepath.Abs(scanRoot)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", scanRoot, err)
	}

	rules.FilterLinksByStyles(records, absRoot)

	derive.EnrichBuiltinsSimple(ctx, records, absRoot)

	// Apply --where filter.
	records, err = filterRecords(ctx, records, graphWhere, nil)
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	// Load .stem schema and filter links to only include structurally relevant ones.
	entries, err := rules.WalkUp(absRoot)
	if err == nil {
		stem := rules.MergeStemFiles(entries)
		if stem != nil {
			filterLinksBySchema(records, stem.Links)
		}
	}

	g := graph.Build(ctx, records)
	cycles := g.DetectCycles()
	broken := g.BrokenLinks()

	// --check mode: report issues and exit.
	if graphCheck {
		hasProblems := len(cycles) > 0 || len(broken) > 0
		if len(cycles) > 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Cycles found: %d\n", len(cycles))
			for i, c := range cycles {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d: %s\n", i+1, strings.Join(c, " → "))
			}
		}
		if len(broken) > 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Broken links: %d\n", len(broken))
			for _, b := range broken {
				msg := fmt.Sprintf("  %s:%d → %s (%s)", b.Source, b.Line, b.Target, b.Type)
				if len(b.Suggestions) > 0 {
					msg += fmt.Sprintf(" — did you mean: %s?", strings.Join(b.Suggestions, ", "))
				}
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), msg)
			}
		}
		if !hasProblems {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No cycles or broken links found.")
		}
		if hasProblems {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return ErrValidationFailed
		}
		return nil
	}

	// --open mode: render mermaid into temp HTML and open in browser.
	if graphOpen {
		mermaidText := mermaidGraphText(g)
		htmlPath, err := graph.RenderHTML(mermaidText)
		if err != nil {
			return fmt.Errorf("rendering HTML: %w", err)
		}
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Opened: %s\n", htmlPath)
		_ = graph.OpenBrowser(htmlPath)
		return nil
	}

	// JSON output.
	if outputFormat == "json" {
		nodes := make([]string, 0, len(g.Nodes))
		for path := range g.Nodes {
			nodes = append(nodes, path)
		}
		var allEdges []graph.Edge
		for _, edges := range g.Edges {
			allEdges = append(allEdges, edges...)
		}
		if allEdges == nil {
			allEdges = []graph.Edge{}
		}
		if broken == nil {
			broken = []graph.BrokenLink{}
		}
		if cycles == nil {
			cycles = [][]string{}
		}

		result := GraphResult{
			Version:     1,
			Kind:        "rootline/graph",
			Nodes:       nodes,
			Edges:       allEdges,
			Cycles:      cycles,
			BrokenLinks: broken,
		}
		return outputJSON(cmd, result, false)
	}

	// Diagram output.
	switch graphFormat {
	case "dot":
		renderDOT(cmd, g)
	case "mermaid":
		renderMermaid(cmd, g)
	default:
		return fmt.Errorf("unknown format %q (use dot or mermaid)", graphFormat)
	}

	return nil
}

func renderDOT(cmd *cobra.Command, g *graph.Graph) {
	w := cmd.OutOrStdout()
	_, _ = fmt.Fprintln(w, "digraph {")
	_, _ = fmt.Fprintln(w, "  rankdir=LR;")
	for path := range g.Nodes {
		_, _ = fmt.Fprintf(w, "  %q;\n", path)
	}
	for _, edges := range g.Edges {
		for _, e := range edges {
			_, _ = fmt.Fprintf(w, "  %q -> %q [label=%q];\n", e.Source, e.Target, e.Type)
		}
	}
	_, _ = fmt.Fprintln(w, "}")
}

// filterLinksBySchema removes links from records whose type has no rule in the schema.
// If the schema is empty, no filtering is performed (backward compatible).
func filterLinksBySchema(records []*extract.Record, schema rules.LinkSchema) {
	if schema.IsEmpty() {
		return
	}
	for _, rec := range records {
		filtered := rec.Links[:0]
		for _, link := range rec.Links {
			if _, ok := schema.Rules[link.Type]; ok {
				filtered = append(filtered, link)
			}
		}
		rec.Links = filtered
	}
}

func renderMermaid(cmd *cobra.Command, g *graph.Graph) {
	_, _ = fmt.Fprint(cmd.OutOrStdout(), mermaidGraphText(g))
}

// mermaidGraphText generates a Mermaid diagram string from a Graph.
func mermaidGraphText(g *graph.Graph) string {
	var sb strings.Builder

	// Sanitize node IDs for mermaid (replace special chars).
	id := func(path string) string {
		r := strings.NewReplacer("/", "_", ".", "_", "-", "_", " ", "_")
		return r.Replace(path)
	}

	_, _ = fmt.Fprintln(&sb, "graph TD;")
	for path := range g.Nodes {
		_, _ = fmt.Fprintf(&sb, "  %s[%q];\n", id(path), path)
	}
	for _, edges := range g.Edges {
		for _, e := range edges {
			_, _ = fmt.Fprintf(&sb, "  %s --> |%s| %s;\n", id(e.Source), e.Type, id(e.Target))
		}
	}
	return sb.String()
}
