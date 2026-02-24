package main

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"

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
	Version  int            `json:"version"`
	Kind     string         `json:"kind"`
	ByEstado map[string]int `json:"by_estado"`
	ByTipo   map[string]int `json:"by_tipo"`
	Total    int            `json:"total"`
}

func runStats(cmd *cobra.Command, args []string) error {
	scanRoot := statsFrom
	if len(args) > 0 {
		scanRoot = args[0]
	}

	absRoot, err := filepath.Abs(scanRoot)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(context.TODO(), absRoot, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", scanRoot, err)
	}

	// Run derivation and aggregation (best-effort, errors silently skipped).
	derive.DeriveAllSimple(context.TODO(), records, absRoot)
	derive.AggregateAllSimple(context.TODO(), records, absRoot)

	// Apply --where filter.
	records, err = filterRecords(records, statsWhere)
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	byEstado := make(map[string]int)
	byTipo := make(map[string]int)

	for _, rec := range records {
		if e, ok := rec.EffectiveField("estado"); ok {
			byEstado[fmt.Sprintf("%v", e)]++
		}
		if t, ok := rec.EffectiveField("tipo"); ok {
			byTipo[fmt.Sprintf("%v", t)]++
		}
	}

	result := &StatsResult{
		Version:  1,
		Kind:     "rootline/stats",
		ByEstado: byEstado,
		ByTipo:   byTipo,
		Total:    len(records),
	}

	if outputFormat == "table" {
		return renderStatsTable(cmd, result)
	}

	return outputJSON(cmd, result, false)
}

func renderStatsTable(cmd *cobra.Command, r *StatsResult) error {
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Total: %d records\n\n", r.Total)

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "By Estado:")
	for _, k := range sortedKeys(r.ByEstado) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-20s %d\n", k, r.ByEstado[k])
	}

	_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nBy Tipo:")
	for _, k := range sortedKeys(r.ByTipo) {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %-20s %d\n", k, r.ByTipo[k])
	}
	return nil
}

func sortedKeys(m map[string]int) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}
