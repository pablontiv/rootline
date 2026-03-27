package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/spf13/cobra"
)

var (
	traceReverse bool
	traceDepth   int
	traceType    string
	traceFormat  string
)

var traceCmd = &cobra.Command{
	Use:   "trace <file>",
	Short: "Follow reference chains through the document graph",
	Long:  "Performs BFS traversal starting from a file, following wiki-link references\nin frontmatter and body text. Use --reverse to find all documents that\nreference the given file.",
	Args:  cobra.ExactArgs(1),
	RunE:  runTrace,
}

func init() {
	traceCmd.Flags().BoolVar(&traceReverse, "reverse", false, "follow incoming references instead of outgoing")
	traceCmd.Flags().IntVar(&traceDepth, "depth", 0, "max traversal depth (0 = unlimited)")
	traceCmd.Flags().StringVar(&traceType, "type", "", "filter by edge type (field name)")
	traceCmd.Flags().StringVar(&traceFormat, "format", "tree", "output format: tree, json")
	rootCmd.AddCommand(traceCmd)
}

func runTrace(cmd *cobra.Command, args []string) error {
	filePath := args[0]

	absFile, err := filepath.Abs(filePath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Find the root directory by walking up to .git.
	absRoot := filepath.Dir(absFile)
	for {
		if _, err := os.Stat(filepath.Join(absRoot, ".git")); err == nil {
			break
		}
		parent := filepath.Dir(absRoot)
		if parent == absRoot {
			return fmt.Errorf("no .git directory found above %s", filePath)
		}
		absRoot = parent
	}

	relFile, err := filepath.Rel(absRoot, absFile)
	if err != nil {
		return fmt.Errorf("computing relative path: %w", err)
	}

	registry := extract.NewRegistry()
	records, err := index.Scan(cmd.Context(), absRoot, registry)
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	derive.EnrichBuiltinsSimple(cmd.Context(), records, absRoot)

	g := graph.Build(cmd.Context(), records)

	opts := graph.TraceOptions{
		Reverse:  traceReverse,
		MaxDepth: traceDepth,
		EdgeType: traceType,
	}

	result := g.Trace(relFile, opts)

	if outputFormat == "json" || traceFormat == "json" {
		data, err := json.Marshal(result)
		if err != nil {
			return fmt.Errorf("marshaling JSON: %w", err)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
		return nil
	}

	// Tree format output.
	cmd.Printf("%s", relFile)
	if rec, ok := g.Nodes[relFile]; ok {
		if estado, ok := rec.EffectiveField("estado"); ok {
			cmd.Printf(" (%s)", estado)
		}
	}
	cmd.Println()

	for _, node := range result.Nodes {
		indent := strings.Repeat("  ", node.Depth)
		arrow := "→"
		if traceReverse {
			arrow = "←"
		}
		cmd.Printf("%s%s %s", indent, arrow, node.Path)
		if node.Estado != "" {
			cmd.Printf(" (%s)", node.Estado)
		}
		if node.Via != "" {
			cmd.Printf(" [via %s]", node.Via)
		}
		cmd.Println()
	}

	if len(result.Nodes) == 0 {
		cmd.Println("  (no references found)")
	}

	return nil
}
