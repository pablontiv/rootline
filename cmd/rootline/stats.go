package main

import (
	"fmt"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/spf13/cobra"
)

var (
	statsFrom  string
	statsWhere []string
)

var statsCmd = &cobra.Command{
	Use:   "stats [path]",
	Short: "Count records, optionally filtered by --where",
	Long:  "Count records in a directory. Use --where to filter the records included in the total.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runStats,
}

func init() {
	statsCmd.Flags().StringVar(&statsFrom, "from", ".", "root path to scan")
	statsCmd.Flags().StringArrayVar(&statsWhere, "where", nil, "filter expression (e.g. \"estado == 'Pending'\")")
	rootCmd.AddCommand(statsCmd)
}

// StatsResult is the versioned JSON output for stats.
type StatsResult struct {
	Version int    `json:"version"`
	Kind    string `json:"kind"`
	Total   int    `json:"total"`
}

func runStats(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	scanRoot := statsFrom
	if len(args) > 0 {
		scanRoot = args[0]
	}

	absRoot, err := filepath.Abs(scanRoot)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	resolver := stemScopeResolver()
	records, err := index.Scan(ctx, absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return fmt.Errorf("scanning %s: %w", scanRoot, err)
	}
	if err := ensureRecordsResolve(ctx, records, absRoot); err != nil {
		return fmt.Errorf("resolving .stem: %w", err)
	}

	if err := derive.DeriveAllSimple(ctx, records, absRoot); err != nil {
		return fmt.Errorf("deriving records: %w", err)
	}
	if err := derive.EnrichBuiltinsSimple(ctx, records, absRoot); err != nil {
		return fmt.Errorf("enriching records: %w", err)
	}
	if err := derive.AggregateAllSimple(ctx, records, absRoot); err != nil {
		return fmt.Errorf("aggregating records: %w", err)
	}

	// Apply --where filter.
	records, err = filterRecords(ctx, records, statsWhere, knownWhereFields(records, absRoot), cmd.ErrOrStderr())
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	result := &StatsResult{
		Version: 2,
		Kind:    "rootline/stats",
		Total:   len(records),
	}

	if outputFormat == "table" {
		return renderStatsTable(cmd, result)
	}

	return outputJSON(cmd, result, false)
}

func renderStatsTable(cmd *cobra.Command, r *StatsResult) error {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Total: %d records\n", r.Total)
	return nil
}
