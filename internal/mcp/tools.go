package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
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
		entries, walkErr := rules.WalkUp(dir)
		if walkErr != nil {
			continue
		}
		effective := rules.MergeStemFiles(entries)
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
