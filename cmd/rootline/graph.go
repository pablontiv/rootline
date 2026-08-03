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
	graphFormat      string
	graphCheck       bool
	graphWhere       []string
	graphFailCycles  bool
	graphQuietCycles bool
)

var graphCmd = &cobra.Command{
	Use:   "graph [path]",
	Short: "Build and visualize the document dependency graph",
	Long:  "Scan documents and build a directed graph from wiki-links.\nOutput is JSON by default; pass -o table with --format dot|mermaid to render a diagram.\nUse --check to validate for cycles and broken links (text report + exit code).",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runGraph,
}

func init() {
	graphCmd.Flags().StringVar(&graphFormat, "format", "dot", "diagram format when -o is not json: dot or mermaid")
	graphCmd.Flags().BoolVar(&graphCheck, "check", false, "validate only (cycles + broken links), no diagram")
	graphCmd.Flags().StringArrayVar(&graphWhere, "where", nil, "filter expression (e.g. \"tipo != 'feature'\")")
	graphCmd.Flags().BoolVar(&graphFailCycles, "fail-cycles", false, "treat cycles as check failures (overrides .stem links.checks.cycles)")
	graphCmd.Flags().BoolVar(&graphQuietCycles, "quiet-cycles", false, "suppress per-cycle enumeration when informational (has no effect when --fail-cycles or .stem opt-in hardening)")
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
	rules.ResolveMarkdownTargets(records, absRoot)

	derive.EnrichBuiltinsSimple(ctx, records, absRoot)

	// Apply --where filter.
	records, err = filterRecords(ctx, records, graphWhere, nil)
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	// Load .stem schema: filter links and read the cycle-failure opt-in.
	failCycles := false
	entries, err := rules.WalkUp(absRoot)
	if err != nil {
		return fmt.Errorf("discovering schema for link filtering: %w", err)
	}
	stem := rules.MergeStemFiles(entries)
	if stem != nil {
		filterLinksBySchema(records, stem.Links)
		failCycles = stem.Links.Checks != nil && stem.Links.Checks.Cycles
	}
	if cmd.Flags().Changed("fail-cycles") {
		failCycles = graphFailCycles
	}

	g := graph.Build(ctx, records)
	cycles := g.DetectCycles()
	broken := g.BrokenLinks()

	// --check mode: report issues and exit.
	if graphCheck {
		hasProblems := (failCycles && len(cycles) > 0) || len(broken) > 0
		if len(cycles) > 0 {
			header := "Cycles found"
			if !failCycles {
				header = "Cycles found (informational)"
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: %d", header, len(cycles))
			// When informational and --quiet-cycles set: suppress per-cycle enumeration, single line only.
			if !failCycles && graphQuietCycles {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), " (use --fail-cycles or omit --quiet-cycles for details)\n")
			} else {
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
				for i, c := range cycles {
					_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %d: %s\n", i+1, strings.Join(c, " → "))
				}
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
		if len(cycles) == 0 && len(broken) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No cycles or broken links found.")
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
		nodes := g.SortedNodes()
		allEdges := g.SortedEdges()
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
	for _, path := range g.SortedNodes() {
		_, _ = fmt.Fprintf(w, "  %q;\n", path)
	}
	for _, e := range g.SortedEdges() {
		_, _ = fmt.Fprintf(w, "  %q -> %q [label=%q];\n", e.Source, e.Target, e.Type)
	}
	_, _ = fmt.Fprintln(w, "}")
}

// filterLinksBySchema removes links from records whose type has no rule in the schema.
// Typed-rule filtering applies only when typed rules are declared; a schema
// carrying only styles/checks/allowed must not suppress links (the graph
// showed zero edges on styles-only repos otherwise).
func filterLinksBySchema(records []*extract.Record, schema rules.LinkSchema) {
	if len(schema.Rules) == 0 {
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
	for _, path := range g.SortedNodes() {
		_, _ = fmt.Fprintf(&sb, "  %s[%q];\n", id(path), path)
	}
	for _, e := range g.SortedEdges() {
		_, _ = fmt.Fprintf(&sb, "  %s --> |%s| %s;\n", id(e.Source), e.Type, id(e.Target))
	}
	return sb.String()
}
