package main

import (
	"encoding/json"
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/spf13/cobra"
)

var (
	graphFormat string
	graphCheck  bool
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
	scanRoot := "."
	if len(args) > 0 {
		scanRoot = args[0]
	}

	absRoot, err := filepath.Abs(scanRoot)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(absRoot, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", scanRoot, err)
	}

	g := graph.Build(records)
	cycles := g.DetectCycles()
	broken := g.BrokenLinks()

	// --check mode: report issues and exit.
	if graphCheck {
		hasProblems := len(cycles) > 0 || len(broken) > 0
		if len(cycles) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Cycles found: %d\n", len(cycles))
			for i, c := range cycles {
				fmt.Fprintf(cmd.OutOrStdout(), "  %d: %s\n", i+1, strings.Join(c, " → "))
			}
		}
		if len(broken) > 0 {
			fmt.Fprintf(cmd.OutOrStdout(), "Broken links: %d\n", len(broken))
			for _, b := range broken {
				fmt.Fprintf(cmd.OutOrStdout(), "  %s:%d → %s (%s)\n", b.Source, b.Line, b.Target, b.Type)
			}
		}
		if !hasProblems {
			fmt.Fprintln(cmd.OutOrStdout(), "No cycles or broken links found.")
		}
		if hasProblems {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return ErrValidationFailed
		}
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
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
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
	fmt.Fprintln(w, "digraph {")
	fmt.Fprintln(w, "  rankdir=LR;")
	for path := range g.Nodes {
		fmt.Fprintf(w, "  %q;\n", path)
	}
	for _, edges := range g.Edges {
		for _, e := range edges {
			fmt.Fprintf(w, "  %q -> %q [label=%q];\n", e.Source, e.Target, e.Type)
		}
	}
	fmt.Fprintln(w, "}")
}

func renderMermaid(cmd *cobra.Command, g *graph.Graph) {
	w := cmd.OutOrStdout()
	fmt.Fprintln(w, "graph TD;")

	// Sanitize node IDs for mermaid (replace special chars).
	id := func(path string) string {
		r := strings.NewReplacer("/", "_", ".", "_", "-", "_", " ", "_")
		return r.Replace(path)
	}

	for path := range g.Nodes {
		fmt.Fprintf(w, "  %s[%q];\n", id(path), path)
	}
	for _, edges := range g.Edges {
		for _, e := range edges {
			fmt.Fprintf(w, "  %s --> |%s| %s;\n", id(e.Source), e.Type, id(e.Target))
		}
	}
}
