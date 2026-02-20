package main

import (
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/spf13/cobra"
)

var (
	queryWhere []string
	queryCount bool
	queryLimit int
	queryFrom  string
)

var queryCmd = &cobra.Command{
	Use:   "query [path]",
	Short: "Search and filter records",
	Long:  "Query documents matching filter expressions.\nMultiple --where flags are combined with AND.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runQuery,
}

func init() {
	queryCmd.Flags().StringSliceVar(&queryWhere, "where", nil, "filter expression (repeatable, e.g. 'estado eq Pending')")
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

	// Parse --where flags into conditions
	q := &query.Query{
		Count: queryCount,
		Limit: queryLimit,
	}

	if len(queryWhere) > 0 {
		conditions, err := parseWhereFlags(queryWhere)
		if err != nil {
			return err
		}
		if len(conditions) == 1 {
			q.Where = &conditions[0]
		} else {
			q.Where = &query.Condition{
				Op:       query.OpAnd,
				Children: conditions,
			}
		}
	}

	// Execute query
	result, err := query.Execute(records, q)
	if err != nil {
		return fmt.Errorf("executing query: %w", err)
	}

	if outputFormat == "table" {
		return renderQueryTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

// parseWhereFlags parses --where flag strings into Conditions.
// Format: "field op value" where op is eq|ne|contains|exists|in.
// For "in", value is comma-separated.
func parseWhereFlags(wheres []string) ([]query.Condition, error) {
	var conditions []query.Condition
	for _, w := range wheres {
		cond, err := parseWhereExpr(w)
		if err != nil {
			return nil, fmt.Errorf("invalid --where %q: %w", w, err)
		}
		conditions = append(conditions, cond)
	}
	return conditions, nil
}

func parseWhereExpr(expr string) (query.Condition, error) {
	parts := strings.SplitN(expr, " ", 3)
	if len(parts) < 2 {
		return query.Condition{}, fmt.Errorf("expected 'field op [value]', got %q", expr)
	}

	field := parts[0]
	op := parts[1]

	switch op {
	case "eq":
		if len(parts) < 3 {
			return query.Condition{}, fmt.Errorf("eq requires a value")
		}
		return query.Condition{Op: query.OpEq, Field: field, Value: parts[2]}, nil
	case "ne":
		if len(parts) < 3 {
			return query.Condition{}, fmt.Errorf("ne requires a value")
		}
		return query.Condition{Op: query.OpNe, Field: field, Value: parts[2]}, nil
	case "contains":
		if len(parts) < 3 {
			return query.Condition{}, fmt.Errorf("contains requires a value")
		}
		return query.Condition{Op: query.OpContains, Field: field, Value: parts[2]}, nil
	case "in":
		if len(parts) < 3 {
			return query.Condition{}, fmt.Errorf("in requires comma-separated values")
		}
		values := strings.Split(parts[2], ",")
		return query.Condition{Op: query.OpIn, Field: field, Values: values}, nil
	case "exists":
		return query.Condition{Op: query.OpExists, Field: field}, nil
	default:
		return query.Condition{}, fmt.Errorf("unknown operator %q (expected eq|ne|contains|in|exists)", op)
	}
}

func renderQueryTable(cmd *cobra.Command, result any) error {
	qr, ok := result.(*query.QueryResult)
	if !ok {
		return outputJSON(cmd, result, false)
	}

	// Collect all frontmatter keys across rows
	keySet := map[string]bool{}
	for _, row := range qr.Rows {
		for k := range row.Frontmatter {
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
			if v, ok := row.Frontmatter[k]; ok {
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
