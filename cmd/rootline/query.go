package main

import (
	"context"
	"encoding/csv"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
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
	queryWhere        []string
	queryCount        bool
	queryLimit        int
	queryFrom         string
	querySort         string
	querySelect       string
	queryHasInbound   string
	queryHasOutbound  string
	queryInboundType  string
	queryOutboundType string
	queryGraphRoot    string
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
	queryCmd.Flags().StringVar(&queryHasInbound, "has-inbound", "", "keep records with an inbound link from a record matching this expression (empty = any)")
	queryCmd.Flags().StringVar(&queryHasOutbound, "has-outbound", "", "keep records with an outbound link to a record matching this expression (empty = any)")
	queryCmd.Flags().StringVar(&queryInboundType, "inbound-type", "", "restrict --has-inbound to this link type")
	queryCmd.Flags().StringVar(&queryOutboundType, "outbound-type", "", "restrict --has-outbound to this link type")
	queryCmd.Flags().StringVar(&queryGraphRoot, "graph-root", "", "root for the link-graph scan (default: the query path)")
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

	// Traversal predicates: validate flag combinations before scanning.
	hasInbound := cmd.Flags().Changed("has-inbound")
	hasOutbound := cmd.Flags().Changed("has-outbound")
	traversalActive := hasInbound || hasOutbound
	if queryInboundType != "" && !hasInbound {
		return fmt.Errorf("--inbound-type requires --has-inbound")
	}
	if queryOutboundType != "" && !hasOutbound {
		return fmt.Errorf("--outbound-type requires --has-outbound")
	}
	if cmd.Flags().Changed("graph-root") && !traversalActive {
		return fmt.Errorf("--graph-root requires --has-inbound or --has-outbound")
	}

	// Scan records. With traversal predicates the scan runs from the graph
	// root (a superset of the query path) so inbound edges are visible.
	var records []*extract.Record
	var linkGraph *graph.Graph
	var graphPrefix string
	if traversalActive {
		records, linkGraph, graphPrefix, err = scanForTraversal(ctx, absRoot)
		if err != nil {
			return err
		}
	} else {
		reg := extract.NewRegistry()
		records, err = index.Scan(ctx, absRoot, reg)
		if err != nil {
			return fmt.Errorf("scanning %s: %w", scanRoot, err)
		}
		derive.DeriveAllSimple(ctx, records, absRoot)
		derive.EnrichBuiltinsSimple(ctx, records, absRoot)
		derive.AggregateAllSimple(ctx, records, absRoot)
	}

	q := &query.Query{
		Count: queryCount,
		Limit: queryLimit,
	}

	// Filter records using shared helper.
	filtered, err := filterRecords(ctx, records, queryWhere, nil)
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	if traversalActive {
		opts := query.TraversalOptions{
			InboundType:  queryInboundType,
			OutboundType: queryOutboundType,
		}
		if hasInbound {
			opts.HasInbound = &queryHasInbound
		}
		if hasOutbound {
			opts.HasOutbound = &queryHasOutbound
		}
		filtered, err = query.FilterByTraversal(ctx, filtered, linkGraph, opts)
		if err != nil {
			return fmt.Errorf("applying traversal predicates: %w", err)
		}
		// Rebase paths from graph-root-relative to query-path-relative so
		// rows keep the same path format as a non-traversal query. This must
		// run after FilterByTraversal: the graph indexes nodes and edges by
		// graph-root-relative paths.
		if graphPrefix != "" {
			for _, rec := range filtered {
				rec.Path = filepath.FromSlash(strings.TrimPrefix(filepath.ToSlash(rec.Path), graphPrefix))
			}
		}
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

	// Handle output formats
	if outputFormat == "table" {
		return renderQueryTable(cmd, result)
	}
	if outputFormat == "jsonl" {
		if querySelect == "" {
			return fmt.Errorf("jsonl output requires --select flag")
		}
		return outputQueryJSONL(cmd, result)
	}
	if outputFormat == "csv" {
		if querySelect == "" {
			return fmt.Errorf("csv output requires --select flag")
		}
		fields := parseSelectFields(querySelect)
		return outputQueryCSV(cmd, result, fields)
	}
	return outputJSON(cmd, result, false)
}

// scanForTraversal scans from the graph root (--graph-root, default: the
// query path), prepares links the same way `rootline graph` does, builds the
// link graph over the full universe, and returns the records under the query
// path as candidates. Record paths are relative to the graph root; the
// returned prefix ("" when both roots match) lets the caller rebase them to
// query-path-relative after traversal filtering.
func scanForTraversal(ctx context.Context, absQueryRoot string) ([]*extract.Record, *graph.Graph, string, error) {
	graphRoot := queryGraphRoot
	if graphRoot == "" {
		graphRoot = absQueryRoot
	}
	absGraphRoot, err := filepath.Abs(graphRoot)
	if err != nil {
		return nil, nil, "", fmt.Errorf("resolving --graph-root: %w", err)
	}
	rel, err := filepath.Rel(absGraphRoot, absQueryRoot)
	if err != nil || rel == ".." || strings.HasPrefix(rel, ".."+string(filepath.Separator)) {
		return nil, nil, "", fmt.Errorf("query path %s is outside --graph-root %s", absQueryRoot, absGraphRoot)
	}

	reg := extract.NewRegistry()
	all, err := index.Scan(ctx, absGraphRoot, reg)
	if err != nil {
		return nil, nil, "", fmt.Errorf("scanning graph root %s: %w", absGraphRoot, err)
	}

	// Mirror the `graph` command's link preparation so both commands see
	// the same edge universe for the same root.
	rules.FilterLinksByStyles(all, absGraphRoot)
	rules.ResolveMarkdownTargets(all, absGraphRoot)

	derive.DeriveAllSimple(ctx, all, absGraphRoot)
	derive.EnrichBuiltinsSimple(ctx, all, absGraphRoot)
	derive.AggregateAllSimple(ctx, all, absGraphRoot)

	g := graph.Build(ctx, all)

	if rel == "." {
		return all, g, "", nil
	}
	prefix := filepath.ToSlash(rel) + "/"
	var candidates []*extract.Record
	for _, r := range all {
		if strings.HasPrefix(filepath.ToSlash(r.Path), prefix) {
			candidates = append(candidates, r)
		}
	}
	return candidates, g, prefix, nil
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

// outputQueryJSONL outputs a ProjectedQueryResult as JSON Lines (one JSON object per line).
func outputQueryJSONL(cmd *cobra.Command, result any) error {
	pqr, ok := result.(*query.ProjectedQueryResult)
	if !ok {
		return fmt.Errorf("expected ProjectedQueryResult for jsonl output")
	}

	for _, row := range pqr.Rows {
		b, err := json.Marshal(row)
		if err != nil {
			return fmt.Errorf("marshaling row to JSON: %w", err)
		}
		if _, err := fmt.Fprintln(cmd.OutOrStdout(), string(b)); err != nil {
			return fmt.Errorf("writing JSONL line: %w", err)
		}
	}
	return nil
}

// outputQueryCSV outputs a ProjectedQueryResult as CSV with header row.
// The columns are derived from the --select fields in order.
func outputQueryCSV(cmd *cobra.Command, result any, fields []string) error {
	pqr, ok := result.(*query.ProjectedQueryResult)
	if !ok {
		return fmt.Errorf("expected ProjectedQueryResult for csv output")
	}

	w := csv.NewWriter(cmd.OutOrStdout())
	defer w.Flush()

	// Write header row
	if err := w.Write(fields); err != nil {
		return fmt.Errorf("writing CSV header: %w", err)
	}

	// Write data rows
	for _, row := range pqr.Rows {
		record := make([]string, len(fields))
		for i, field := range fields {
			if v, ok := row[field]; ok && v != nil {
				record[i] = fmt.Sprintf("%v", v)
			}
			// If field is missing or nil, leave as empty string
		}
		if err := w.Write(record); err != nil {
			return fmt.Errorf("writing CSV row: %w", err)
		}
	}

	return nil
}
