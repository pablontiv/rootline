package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe <path>",
	Short: "Show effective schema for a directory",
	Long:  "Display the merged .stem schema that applies to documents\nin the given directory, showing inherited and local rules.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDescribe,
}

func init() {
	rootCmd.AddCommand(describeCmd)
}

func runDescribe(cmd *cobra.Command, args []string) error {
	targetPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	entries, err := rules.WalkUp(targetPath)
	if err != nil {
		return fmt.Errorf("discovering .stem files: %w", err)
	}

	effective := rules.MergeStemFiles(entries)

	// Use relative path for display
	relPath := args[0]

	result := rules.NewDescribeResult(relPath, entries, effective)

	// Add hint when no schema is found
	if effective == nil || len(effective.Schema) == 0 {
		result.Hints = append(result.Hints, "No .stem schema found. Run 'rootline init <path>' to infer schema from existing files.")
	}

	if outputFormat == "table" {
		return renderDescribeTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

func renderDescribeTable(cmd *cobra.Command, r *rules.DescribeResult) error {
	headers := []string{"Field", "Type", "Required", "Values", "Source"}
	var rows [][]string

	keys := make([]string, 0, len(r.Schema))
	for k := range r.Schema {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, k := range keys {
		f := r.Schema[k]
		req := "no"
		if f.Required {
			req = "yes"
		}
		vals := ""
		if len(f.Values) > 0 {
			vals = strings.Join(f.Values, ", ")
		}
		rows = append(rows, []string{k, f.Type, req, vals, f.Source})
	}

	renderTable(cmd.OutOrStdout(), headers, rows)

	if len(r.Hints) > 0 {
		fmt.Fprintln(cmd.OutOrStdout())
		for _, h := range r.Hints {
			fmt.Fprintf(cmd.OutOrStdout(), "Hint: %s\n", h)
		}
	}

	return nil
}
