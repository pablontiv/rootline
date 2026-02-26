package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/pablontiv/rootline/internal/rules"
)

// RegisterTools registers all rootline MCP tools on the server.
func RegisterTools(s *Server) {
	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "query",
		Description: "Search and filter records by frontmatter fields using expr-lang expressions",
	}, handleQuery)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "validate",
		Description: "Check documents against .stem schema rules",
	}, handleValidate)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "describe",
		Description: "Show the effective .stem schema for a directory",
	}, handleDescribe)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "tree",
		Description: "Show hierarchical tree of records with completion counts",
	}, handleTree)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "stats",
		Description: "Show aggregate statistics (by estado, by tipo) for records",
	}, handleStats)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "explain",
		Description: "Trace why a document has a given state: field origins, derivation chain, and validation errors",
	}, handleExplain)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "fix",
		Description: "Analyze validation errors and return fix proposals (always dry-run, never modifies files)",
	}, handleFix)

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "graph",
		Description: "Build dependency graph from wiki-links with cycle detection and broken link analysis",
	}, handleGraph)
}

// Tool input types

// QueryInput is the input for the query tool.
type QueryInput struct {
	Path  string   `json:"path" jsonschema:"directory to scan (absolute path)"`
	Where []string `json:"where,omitempty" jsonschema:"filter expressions (expr-lang syntax)"`
	Count bool     `json:"count,omitempty" jsonschema:"return count instead of records"`
	Limit int      `json:"limit,omitempty" jsonschema:"limit number of results (0 = unlimited)"`
}

// ValidateInput is the input for the validate tool.
type ValidateInput struct {
	Path  string   `json:"path" jsonschema:"directory to validate (absolute path)"`
	Where []string `json:"where,omitempty" jsonschema:"filter expressions for validation scope"`
}

// DescribeInput is the input for the describe tool.
type DescribeInput struct {
	Path string `json:"path" jsonschema:"directory to describe (absolute path)"`
}

// TreeInput is the input for the tree tool.
type TreeInput struct {
	Path  string   `json:"path" jsonschema:"directory to scan (absolute path)"`
	Where []string `json:"where,omitempty" jsonschema:"filter expressions"`
}

// StatsInput is the input for the stats tool.
type StatsInput struct {
	Path  string   `json:"path" jsonschema:"directory to scan (absolute path)"`
	Where []string `json:"where,omitempty" jsonschema:"filter expressions"`
}

// ExplainInput is the input for the explain tool.
type ExplainInput struct {
	Path string `json:"path" jsonschema:"file to explain (absolute path)"`
}

// FixInput is the input for the fix tool.
type FixInput struct {
	Path   string `json:"path" jsonschema:"directory to scan (absolute path)"`
	DryRun bool   `json:"dry_run,omitempty" jsonschema:"always true for MCP (proposals only)"`
	All    bool   `json:"all,omitempty" jsonschema:"fix all files in scope"`
}

// GraphInput is the input for the graph tool.
type GraphInput struct {
	Path   string `json:"path" jsonschema:"directory to scan (absolute path)"`
	Check  bool   `json:"check,omitempty" jsonschema:"validate only (cycles + broken links)"`
	Format string `json:"format,omitempty" jsonschema:"output format: dot or mermaid (default: json)"`
}

func handleQuery(ctx context.Context, _ *mcp.CallToolRequest, input QueryInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	filtered, err := filterWhere(ctx, records, input.Where)
	if err != nil {
		return nil, nil, err
	}

	q := &query.Query{Count: input.Count, Limit: input.Limit}
	result, err := query.ExecuteExpr(ctx, filtered, "", q)
	if err != nil {
		return nil, nil, fmt.Errorf("executing query: %w", err)
	}

	return jsonResult(result)
}

func handleValidate(ctx context.Context, _ *mcp.CallToolRequest, input ValidateInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(ctx, absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	filtered, err := filterWhere(ctx, records, input.Where)
	if err != nil {
		return nil, nil, err
	}

	var results []*rules.ValidationResult
	for _, rec := range filtered {
		absPath := filepath.Join(absRoot, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			continue
		}
		errs := rules.Validate(ctx, rec, effective)
		results = append(results, rules.NewValidationResult(rec.Path, errs))
	}

	batch := rules.NewBatchValidationResult(results)
	return jsonResult(batch)
}

func handleDescribe(ctx context.Context, _ *mcp.CallToolRequest, input DescribeInput) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	entries, err := rules.WalkUp(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("walking up: %w", err)
	}

	effective := rules.MergeStemFiles(entries)
	result := rules.NewDescribeResult(input.Path, entries, effective)
	return jsonResult(result)
}

// treeNode represents a directory or file in the tree.
type treeNode struct {
	Name      string      `json:"name"`
	Path      string      `json:"path"`
	Children  []*treeNode `json:"children,omitempty"`
	Completed int         `json:"completed"`
	Total     int         `json:"total"`
	IsLeaf    bool        `json:"is_leaf,omitempty"`
	Estado    string      `json:"estado,omitempty"`
}

// treeResult is the versioned JSON output for tree.
type treeResult struct {
	Version int       `json:"version"`
	Kind    string    `json:"kind"`
	Root    *treeNode `json:"root"`
}

func handleTree(ctx context.Context, _ *mcp.CallToolRequest, input TreeInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(ctx, absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	filtered, err := filterWhere(ctx, records, input.Where)
	if err != nil {
		return nil, nil, err
	}

	root := buildTree(filtered, filepath.Base(absRoot))
	result := &treeResult{Version: 1, Kind: "rootline/tree", Root: root}
	return jsonResult(result)
}

// statsResult is the versioned JSON output for stats.
type statsResult struct {
	Version  int            `json:"version"`
	Kind     string         `json:"kind"`
	ByEstado map[string]int `json:"by_estado"`
	ByTipo   map[string]int `json:"by_tipo"`
	Total    int            `json:"total"`
}

func handleStats(ctx context.Context, _ *mcp.CallToolRequest, input StatsInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	filtered, err := filterWhere(ctx, records, input.Where)
	if err != nil {
		return nil, nil, err
	}

	byEstado := make(map[string]int)
	byTipo := make(map[string]int)
	for _, rec := range filtered {
		if v, ok := rec.EffectiveField("estado"); ok {
			byEstado[fmt.Sprintf("%v", v)]++
		}
		if v, ok := rec.EffectiveField("tipo"); ok {
			byTipo[fmt.Sprintf("%v", v)]++
		}
	}

	result := &statsResult{
		Version:  1,
		Kind:     "rootline/stats",
		ByEstado: byEstado,
		ByTipo:   byTipo,
		Total:    len(filtered),
	}
	return jsonResult(result)
}

// explainResult is the versioned JSON output for explain.
type explainResult struct {
	Version   int            `json:"version"`
	Kind      string         `json:"kind"`
	Path      string         `json:"path"`
	StemChain []string       `json:"stem_chain"`
	Fields    []explainField `json:"fields"`
	Errors    []explainError `json:"errors,omitempty"`
}

type explainField struct {
	Name       string `json:"name"`
	Value      any    `json:"value"`
	Origin     string `json:"origin"`
	Source     string `json:"source,omitempty"`
	Expression string `json:"expression,omitempty"`
}

type explainError struct {
	Rule     string `json:"rule"`
	Field    string `json:"field"`
	Message  string `json:"message"`
	Source   string `json:"source"`
	Severity string `json:"severity"`
}

func handleExplain(ctx context.Context, _ *mcp.CallToolRequest, input ExplainInput) (*mcp.CallToolResult, any, error) {
	absPath, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); err != nil {
		return nil, nil, fmt.Errorf("file not found: %s", input.Path)
	}

	reg := extract.NewRegistry()
	ext := reg.ForFile(absPath, "")
	if ext == nil {
		return nil, nil, fmt.Errorf("no extractor for %s", input.Path)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, nil, fmt.Errorf("reading %s: %w", input.Path, err)
	}

	record, err := ext.Extract(input.Path, content)
	if err != nil {
		return nil, nil, fmt.Errorf("extracting %s: %w", input.Path, err)
	}

	dir := filepath.Dir(absPath)
	entries, err := rules.WalkUp(dir)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving .stem for %s: %w", input.Path, err)
	}
	effective := rules.MergeStemFiles(entries)

	if effective != nil && len(effective.Derive) > 0 {
		_, _ = derive.DeriveRecord(ctx, record, effective, nil)
	}

	if effective != nil && len(effective.Aggregate) > 0 && filepath.Base(absPath) == "README.md" {
		scanDir := filepath.Dir(absPath)
		reg2 := extract.NewRegistry()
		siblings, scanErr := index.Scan(ctx, scanDir, reg2)
		if scanErr == nil && len(siblings) > 0 {
			derive.DeriveAllSimple(ctx, siblings, scanDir)
			derive.AggregateAllSimple(ctx, siblings, scanDir)
			for _, s := range siblings {
				sAbs := filepath.Join(scanDir, s.Path)
				if sAbs == absPath {
					record.Derived = s.Derived
					break
				}
			}
		}
	}

	var valErrs []rules.ValidationError
	if effective != nil {
		valErrs = rules.Validate(ctx, record, effective)
	}

	result := buildExplainResult(input.Path, entries, effective, record, valErrs)
	return jsonResult(result)
}

func buildExplainResult(
	path string,
	entries []rules.StemEntry,
	effective *rules.StemFile,
	record *extract.Record,
	valErrs []rules.ValidationError,
) *explainResult {
	chain := make([]string, len(entries))
	for i, e := range entries {
		chain[i] = e.Path
	}

	var fields []explainField

	fmKeys := sortedMapKeys(record.Frontmatter)
	for _, k := range fmKeys {
		f := explainField{
			Name:   k,
			Value:  record.Frontmatter[k],
			Origin: "frontmatter",
		}
		if effective != nil {
			if sf, ok := effective.Schema[k]; ok {
				f.Source = sf.Source
			}
		}
		fields = append(fields, f)
	}

	if effective != nil {
		for name, sf := range effective.Schema {
			if _, exists := record.Frontmatter[name]; exists {
				continue
			}
			f := explainField{
				Name:   name,
				Value:  nil,
				Origin: "schema",
				Source: sf.Source,
			}
			if sf.Default != "" {
				f.Value = sf.Default
			}
			fields = append(fields, f)
		}

		derivedKeys := sortedMapKeys(record.Derived)
		for _, k := range derivedKeys {
			f := explainField{
				Name:   k,
				Value:  record.Derived[k],
				Origin: "derived",
			}
			if exprVal, ok := effective.Derive[k]; ok {
				if exprStr, ok := exprVal.(string); ok {
					f.Expression = exprStr
				}
			} else if exprVal, ok := effective.Aggregate[k]; ok {
				f.Origin = "aggregate"
				if exprStr, ok := exprVal.(string); ok {
					f.Expression = exprStr
				}
			}
			fields = append(fields, f)
		}
	}

	var explainErrs []explainError
	for _, ve := range valErrs {
		explainErrs = append(explainErrs, explainError{
			Rule:     ve.Rule,
			Field:    ve.Field,
			Message:  ve.Message,
			Source:   ve.Source,
			Severity: ve.Severity,
		})
	}

	return &explainResult{
		Version:   1,
		Kind:      "rootline/explain",
		Path:      path,
		StemChain: chain,
		Fields:    fields,
		Errors:    explainErrs,
	}
}

func sortedMapKeys(m map[string]any) []string {
	keys := make([]string, 0, len(m))
	for k := range m {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

func handleFix(ctx context.Context, _ *mcp.CallToolRequest, input FixInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(ctx, absRoot, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	allErrs := make(map[string][]rules.ValidationError)
	effectiveStems := make(map[string]*rules.StemFile)

	for _, rec := range records {
		recAbsPath := filepath.Join(absRoot, rec.Path)
		dir := filepath.Dir(recAbsPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			continue
		}
		effectiveStems[rec.Path] = effective
		errs := rules.Validate(ctx, rec, effective)
		if len(errs) > 0 {
			allErrs[rec.Path] = errs
		}
	}

	var effective *rules.StemFile
	for _, s := range effectiveStems {
		if effective == nil || len(s.Schema) > len(effective.Schema) {
			effective = s
		}
	}

	report := proposal.Analyze(records, effective, allErrs)
	return jsonResult(report)
}

// graphResult is the versioned JSON output for graph.
type graphResult struct {
	Version     int                `json:"version"`
	Kind        string             `json:"kind"`
	Nodes       []string           `json:"nodes"`
	Edges       []graph.Edge       `json:"edges"`
	Cycles      [][]string         `json:"cycles"`
	BrokenLinks []graph.BrokenLink `json:"broken_links"`
}

func handleGraph(ctx context.Context, _ *mcp.CallToolRequest, input GraphInput) (*mcp.CallToolResult, any, error) {
	absRoot, err := filepath.Abs(input.Path)
	if err != nil {
		return nil, nil, fmt.Errorf("resolving path: %w", err)
	}

	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absRoot, reg)
	if err != nil {
		return nil, nil, fmt.Errorf("scanning: %w", err)
	}

	g := graph.Build(ctx, records)
	cycles := g.DetectCycles()
	broken := g.BrokenLinks()

	nodes := make([]string, 0, len(g.Nodes))
	for path := range g.Nodes {
		nodes = append(nodes, path)
	}
	sort.Strings(nodes)

	var allEdges []graph.Edge
	for _, edges := range g.Edges {
		allEdges = append(allEdges, edges...)
	}
	if allEdges == nil {
		allEdges = []graph.Edge{}
	}
	if broken == nil {
		broken = []graph.BrokenLink{}
	}
	if cycles == nil {
		cycles = [][]string{}
	}

	if input.Format == "dot" || input.Format == "mermaid" {
		var sb strings.Builder
		if input.Format == "dot" {
			sb.WriteString("digraph {\n  rankdir=LR;\n")
			for _, path := range nodes {
				fmt.Fprintf(&sb, "  %q;\n", path)
			}
			for _, e := range allEdges {
				fmt.Fprintf(&sb, "  %q -> %q [label=%q];\n", e.Source, e.Target, e.Type)
			}
			sb.WriteString("}\n")
		} else {
			sb.WriteString("graph TD;\n")
			replacer := strings.NewReplacer("/", "_", ".", "_", "-", "_", " ", "_")
			for _, path := range nodes {
				fmt.Fprintf(&sb, "  %s[%q];\n", replacer.Replace(path), path)
			}
			for _, e := range allEdges {
				fmt.Fprintf(&sb, "  %s --> |%s| %s;\n", replacer.Replace(e.Source), e.Type, replacer.Replace(e.Target))
			}
		}
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: sb.String()}},
		}, nil, nil
	}

	result := &graphResult{
		Version:     1,
		Kind:        "rootline/graph",
		Nodes:       nodes,
		Edges:       allEdges,
		Cycles:      cycles,
		BrokenLinks: broken,
	}
	return jsonResult(result)
}

// Helpers

func filterWhere(ctx context.Context, records []*extract.Record, wheres []string) ([]*extract.Record, error) {
	var cleaned []string
	for _, w := range wheres {
		if w != "" {
			cleaned = append(cleaned, w)
		}
	}
	if len(cleaned) == 0 {
		return records, nil
	}

	combined := strings.Join(cleaned, " && ")
	program, err := query.CompileWhere(combined)
	if err != nil {
		return nil, fmt.Errorf("compiling where: %w", err)
	}

	var filtered []*extract.Record
	for _, rec := range records {
		match, matchErr := query.MatchRecord(ctx, program, rec)
		if matchErr != nil {
			continue
		}
		if match {
			filtered = append(filtered, rec)
		}
	}
	return filtered, nil
}

func jsonResult(v any) (*mcp.CallToolResult, any, error) {
	data, err := json.Marshal(v)
	if err != nil {
		return nil, nil, fmt.Errorf("marshaling result: %w", err)
	}
	return &mcp.CallToolResult{
		Content: []mcp.Content{&mcp.TextContent{Text: string(data)}},
	}, nil, nil
}

func buildTree(records []*extract.Record, rootName string) *treeNode {
	root := &treeNode{Name: rootName, Path: rootName}

	for _, rec := range records {
		parts := strings.Split(filepath.ToSlash(rec.Path), "/")
		node := root

		for i := 0; i < len(parts)-1; i++ {
			child := findChild(node, parts[i])
			if child == nil {
				child = &treeNode{Name: parts[i], Path: strings.Join(parts[:i+1], "/")}
				node.Children = append(node.Children, child)
			}
			node = child
		}

		estado := ""
		if e, ok := rec.EffectiveField("estado"); ok {
			estado = fmt.Sprintf("%v", e)
		}
		leaf := &treeNode{
			Name:   parts[len(parts)-1],
			Path:   rec.Path,
			IsLeaf: true,
			Estado: estado,
		}
		if estado == "Completed" {
			leaf.Completed = 1
		}
		leaf.Total = 1
		node.Children = append(node.Children, leaf)
	}

	propagateCounts(root)
	return root
}

func findChild(node *treeNode, name string) *treeNode {
	for _, c := range node.Children {
		if c.Name == name && !c.IsLeaf {
			return c
		}
	}
	return nil
}

func propagateCounts(node *treeNode) {
	if node.IsLeaf {
		return
	}

	sort.Slice(node.Children, func(i, j int) bool {
		return node.Children[i].Name < node.Children[j].Name
	})

	node.Completed = 0
	node.Total = 0
	for _, child := range node.Children {
		propagateCounts(child)
		node.Completed += child.Completed
		node.Total += child.Total
	}
}
