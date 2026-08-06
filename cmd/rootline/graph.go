package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
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
	graphCmd.Flags().StringVar(&graphFormat, "format", "dot", "diagram format when -o table: dot or mermaid")
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

	// --check is a text-plus-exit-code validator (docs/graph.md); it returns
	// before any format dispatch. Accepting --output there and discarding it is
	// the silent-drop this command is being fixed for, so an explicit --output
	// is rejected. The test is on Changed, not on the value: the default
	// `-o json` is not a request, and `rootline graph <path> --check` — what CI
	// runs today — must keep working untouched.
	if graphCheck && cmd.Flags().Changed("output") {
		return fmt.Errorf("--check does not support --output: it emits a text report and an exit code; " +
			"drop --output, or drop --check to get the rootline/graph envelope, which carries the same cycles and broken_links")
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
	records, err := scanGoverned(ctx, absRoot, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", scanRoot, err)
	}

	rules.FilterLinksByStyles(records, absRoot)

	derive.EnrichBuiltinsSimple(ctx, records, absRoot)

	// Apply --where filter.
	records, err = filterRecords(ctx, records, graphWhere, knownWhereFields(records, absRoot), cmd.ErrOrStderr())
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	// Link checks read the target the author wrote, so they run before
	// PrepareLinks rewrites targets to resolved node keys. Resolution decodes
	// percent escapes, so checking afterwards would flag an already-correct
	// %20 target for the raw space its decoded form contains.
	var linkIssues []linkCheckIssue
	if graphCheck {
		linkIssues = collectLinkCheckIssues(records, absRoot)
	}

	rules.PrepareLinks(records, absRoot)

	// Typed-rule filtering and the cycle-failure opt-in both resolve per
	// record. Reading them from the chain above the scan root meant hardening
	// declared in a subdirectory evaporated the moment CI ran the command from
	// the repository root — the most likely invocation.
	//
	// A missing .stem is still not a missing graph: an ungoverned record
	// filters nothing and opts into nothing.
	rules.FilterLinksByTypedRules(records, absRoot)
	cycleScope := rules.CycleFailureScope(records, absRoot)

	g := graph.Build(ctx, records)
	cycles := g.DetectCycles()
	failCycles := anyCycleOptsIn(cycles, cycleScope)
	if cmd.Flags().Changed("fail-cycles") {
		failCycles = graphFailCycles
	}
	broken := g.BrokenLinks()

	// --check mode: report issues and exit.
	if graphCheck {
		hasProblems := (failCycles && len(cycles) > 0) || len(broken) > 0 || len(linkIssues) > 0
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
		if len(linkIssues) > 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Link check failures: %d\n", len(linkIssues))
			for _, li := range linkIssues {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s (%s)\n", li.path, li.message, li.rule)
			}
		}
		if len(cycles) == 0 && len(broken) == 0 && len(linkIssues) == 0 {
			_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No cycles or broken links found.")
		}
		if hasProblems {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return ErrValidationFailed
		}
		return nil
	}

	// The diagram belongs to -o table, and only to it — the command's own help
	// has said so all along. Testing for "json" and rendering DOT for
	// everything else handed `-o jsonl` a Graphviz document at exit 0.
	if outputFormat == "table" {
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

// linkCheckIssue is one links.checks failure attributed to its record.
type linkCheckIssue struct {
	path    string
	rule    string
	message string
}

// collectLinkCheckIssues runs the schema's declared links.checks over every
// record so `graph --check` sees the same anchor and encoding failures
// `validate` does.
//
// Resolution failures are deliberately skipped: the graph already reports an
// unresolvable target as a broken link, and reporting it twice under two names
// would be noise. Checks stay opt-in — a schema declaring none yields nothing.
func collectLinkCheckIssues(records []*extract.Record, root string) []linkCheckIssue {
	cache := rules.NewHeadingCache()
	var issues []linkCheckIssue
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		effective, err := rules.ResolveForRecord(filepath.Dir(absPath), rec.Path)
		if err != nil || effective == nil {
			continue
		}
		for _, e := range rules.CheckLinks(rec.Links, effective.Links, absPath, root, cache) {
			if e.Rule == "link_resolve" {
				continue
			}
			issues = append(issues, linkCheckIssue{path: rec.Path, rule: e.Rule, message: e.Message})
		}
	}
	return issues
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

// anyCycleOptsIn reports whether any detected cycle passes through a record
// whose schema asked for cycles to fail.
//
// Deciding per cycle rather than per run means hardening declared in one
// subtree fails the cycles that actually run through it, without failing
// unrelated cycles in a tree that never opted in.
func anyCycleOptsIn(cycles [][]string, scope map[string]bool) bool {
	for _, cycle := range cycles {
		for _, node := range cycle {
			if scope[node] {
				return true
			}
		}
	}
	return false
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
