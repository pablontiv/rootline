package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/pablontiv/rootline/internal/rules"
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
	queryWhere  []string
	queryCount  bool
	queryLimit  int
	queryFrom   string
	querySort   string
	querySelect string
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
	queryCmd.Flags().StringVar(&querySort, "sort", "", `sort by fields (e.g. "prioridad:asc,impact_score:desc")`)
	queryCmd.Flags().StringVar(&querySelect, "select", "", "comma-separated field names to include in each row (e.g. path,estado,title,links)")
	rootCmd.AddCommand(queryCmd)
}

func runQuery(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Determine scan root
	scanRoot := queryFrom
	if len(args) > 0 {
		scanRoot = args[0]
	}

	absRoot, err := filepath.Abs(scanRoot)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Parse sort keys early to fail fast on invalid input.
	sortKeys, err := query.ParseSortKeys(querySort)
	if err != nil {
		return fmt.Errorf("parsing --sort: %w", err)
	}

	// Scan records
	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", scanRoot, err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.EnrichBuiltinsSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	q := &query.Query{
		Count: queryCount,
		Limit: queryLimit,
	}

	// Filter records using shared helper.
	filtered, err := filterRecords(ctx, records, queryWhere, nil)
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	// Sort AFTER filtering, BEFORE limit/output.
	if len(sortKeys) > 0 {
		var schema map[string]rules.SchemaField
		entries, walkErr := rules.WalkUp(absRoot)
		if walkErr == nil && len(entries) > 0 {
			merged := rules.MergeStemFiles(entries)
			schema = merged.Schema
		}
		query.SortRecords(filtered, sortKeys, schema)
	}

	// Execute query (count/limit) on sorted records.
	result, err := query.ExecuteExpr(ctx, filtered, "", q)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	// Apply projection if --select is given
	if querySelect != "" {
		fields := parseSelectFields(querySelect)
		result, err = projectQueryResult(result, fields)
		if err != nil {
			return fmt.Errorf("applying projection: %w", err)
		}
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

// parseSelectFields splits a comma-separated string into field names.
func parseSelectFields(selectStr string) []string {
	var fields []string
	for _, f := range strings.Split(selectStr, ",") {
		f = strings.TrimSpace(f)
		if f != "" {
			fields = append(fields, f)
		}
	}
	return fields
}

// extractTitle returns the first heading from the body (e.g., "# Heading" -> "Heading").
// Falls back to the first non-empty line if no heading is found.
func extractTitle(body string) string {
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "# ") {
			return strings.TrimPrefix(line, "# ")
		}
	}
	// Fallback: return first non-empty line
	for _, line := range strings.Split(body, "\n") {
		line = strings.TrimSpace(line)
		if line != "" && !strings.HasPrefix(line, "---") {
			return line
		}
	}
	return ""
}

// projectQueryResult projects rows to include only selected fields.
// It converts QueryResult rows to projected format (path, title, and selected fields).
func projectQueryResult(result any, fields []string) (any, error) {
	qr, ok := result.(*query.QueryResult)
	if !ok {
		return result, nil
	}

	// Build projected rows: each row becomes a map[string]any with only selected fields
	projectedRows := make([]map[string]any, len(qr.Rows))
	for i, row := range qr.Rows {
		projected := make(map[string]any)
		for _, field := range fields {
			switch field {
			case "path":
				projected["path"] = row.Path
			case "title":
				title := extractTitle(row.Body)
				if title != "" {
					projected["title"] = title
				}
			case "links":
				if len(row.Links) > 0 {
					projected["links"] = row.Links
				}
			default:
				// Try derived fields first, then frontmatter
				if row.Derived != nil {
					if v, ok := row.Derived[field]; ok {
						projected[field] = v
						continue
					}
				}
				if v, ok := row.Frontmatter[field]; ok {
					projected[field] = v
				}
			}
		}
		projectedRows[i] = projected
	}

	// Return a projected QueryResult with map rows instead of Record pointers
	return &query.ProjectedQueryResult{
		Version: 1,
		Kind:    "rootline/query",
		Meta:    query.QueryMeta{Count: len(projectedRows)},
		Rows:    projectedRows,
	}, nil
}
