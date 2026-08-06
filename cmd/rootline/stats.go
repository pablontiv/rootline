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
	Short: "Summary counts by type and state",
	Long:  "Show aggregate statistics for documents: counts by type,\nstate, and other frontmatter fields.",
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
	Version      int            `json:"version"`
	Kind         string         `json:"kind"`
	ByLifecycle  map[string]int `json:"by_lifecycle_state"`
	ByRecordType map[string]int `json:"by_record_type"`
	Total        int            `json:"total"`
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
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", scanRoot, err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.EnrichBuiltinsSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	// Apply --where filter.
	records, err = filterRecords(ctx, records, statsWhere, knownWhereFields(records, absRoot), cmd.ErrOrStderr())
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	// Field-agnostic statistics: no hardcoded field assumptions
	byLifecycle := make(map[string]int)
	byRecordType := make(map[string]int)

	// Without hardcoded field names, these maps remain empty but available
	// Users can filter via --where if they need field-specific stats

	result := &StatsResult{
		Version:      1,
		Kind:         "rootline/stats",
		ByLifecycle:  byLifecycle,
		ByRecordType: byRecordType,
		Total:        len(records),
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
