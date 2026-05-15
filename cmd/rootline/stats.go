package main

import (
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
	records, err = filterRecords(ctx, records, statsWhere, nil)
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	// Use hardcoded field names (domain: field lookup was removed in O14 refactor)
	estadoField := "estado"
	tipoField := "tipo"

	byEstado := make(map[string]int)
	byTipo := make(map[string]int)

	for _, rec := range records {
		if e, ok := rec.EffectiveField(estadoField); ok {
			byEstado[fmt.Sprintf("%v", e)]++
		}
		if t, ok := rec.EffectiveField(tipoField); ok {
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
