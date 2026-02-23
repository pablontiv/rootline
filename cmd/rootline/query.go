package main

import (
	"fmt"
	"path/filepath"
	"sort"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/spf13/cobra"
)

// whereExamples shows the new expr-based syntax in the help text.
const whereExamples = `Examples:
  rootline query --where "estado == 'Pending'"
  rootline query --where "estado in ['Pending', 'Especificado']"
  rootline query --where "tipo == 'lxc' && estado == 'Pending'"
  rootline query --where "body contains 'migration'"
  rootline query --where 'tags != nil'`

var (
	queryWhere []string
	queryCount bool
	queryLimit int
	queryFrom  string
)

var queryCmd = &cobra.Command{
	Use:   "query [path]",
	Short: "Search and filter records",
	Long:  "Query documents matching filter expressions.\nMultiple --where flags are combined with AND.\n\n" + whereExamples,
	Args:  cobra.MaximumNArgs(1),
	RunE:  runQuery,
}

func init() {
	queryCmd.Flags().StringArrayVar(&queryWhere, "where", nil, "filter expression (e.g. \"estado == 'Pending'\")")
	queryCmd.Flags().BoolVar(&queryCount, "count", false, "return count instead of records")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 0, "limit number of results (0 = unlimited)")
	queryCmd.Flags().StringVar(&queryFrom, "from", ".", "root path to scan")
	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	// Determine scan root
	scanRoot := queryFrom
	if len(args) > 0 {
		scanRoot = args[0]
	}

	absRoot, err := filepath.Abs(scanRoot)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Scan records
	reg := extract.NewRegistry()
	records, err := index.Scan(absRoot, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", scanRoot, err)
	}

	// Run derivation and aggregation (best-effort, errors silently skipped).
	derive.DeriveAllSimple(records, absRoot)
	derive.AggregateAllSimple(records, absRoot)

	q := &query.Query{
		Count: queryCount,
		Limit: queryLimit,
	}

	// Filter records using shared helper.
	filtered, err := filterRecords(records, queryWhere)
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	// Execute query (count/limit) on filtered records.
	result, err := query.ExecuteExpr(filtered, "", q)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	if outputFormat == "table" {
		return renderQueryTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

func renderQueryTable(cmd *cobra.Command, result any) error {
	qr, ok := result.(*query.QueryResult)
	if !ok {
		return outputJSON(cmd, result, false)
	}

	// Collect all effective field keys across rows (frontmatter + derived)
	keySet := map[string]bool{}
	for _, row := range qr.Rows {
		for k := range row.Frontmatter {
			keySet[k] = true
		}
		for k := range row.Derived {
			keySet[k] = true
		}
	}
	var keys []string
	for k := range keySet {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	headers := append([]string{"Path"}, keys...)
	var rows [][]string
	for _, row := range qr.Rows {
		r := []string{row.Path}
		for _, k := range keys {
			if v, ok := row.EffectiveField(k); ok {
				r = append(r, fmt.Sprintf("%v", v))
			} else {
				r = append(r, "")
			}
		}
		rows = append(rows, r)
	}

	renderTable(cmd.OutOrStdout(), headers, rows)
	return nil
}
