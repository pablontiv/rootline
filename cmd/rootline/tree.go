package main

import (
	"errors"
	"fmt"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var treeWhere []string

var treeCmd = &cobra.Command{
	Use:   "tree [path]",
	Short: "Hierarchical view with completion counts",
	Long:  "Display the document tree with completion counts\nderived from frontmatter lifecycle fields.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runTree,
}

func init() {
	treeCmd.Flags().StringArrayVar(&treeWhere, "where", nil, "filter expression (e.g. \"estado == 'Pending'\")")
	rootCmd.AddCommand(treeCmd)
}

// TreeResult is the versioned JSON output for tree.
type TreeResult struct {
	Version int       `json:"version"`
	Kind    string    `json:"kind"`
	Root    *treeNode `json:"root"`
}

// treeNode represents a directory or file in the tree.
type treeNode struct {
	Name        string         `json:"name"`
	Path        string         `json:"path"`
	Children    []*treeNode    `json:"children,omitempty"`
	Total       int            `json:"total"`
	IsLeaf      bool           `json:"is_leaf,omitempty"`
	Frontmatter map[string]any `json:"frontmatter,omitempty"`
}

// treeRenderContext holds schema information for rendering lifecycle field values.
type treeRenderContext struct {
	// lifecycleField is the name of the field that should be displayed as a status.
	// If empty, the renderer picks the lexically first enum-typed field name in the
	// schema — enum field names are sorted alphabetically and the first is selected.
	// Go provides no implicit ordering over a map, so the sort is what makes the
	// choice reproducible across runs.
	lifecycleField string
	// effectiveSchema maps field names to their schema definitions.
	effectiveSchema map[string]rules.SchemaField
}

// firstEnumField returns the lexically first enum-typed field name in schema,
// or "" when schema is nil or holds no enum-typed field.
//
// Enum fields are the schema's status-like fields, and the tree renderer has to
// display exactly one of them. Ranging the schema map and taking whatever comes
// first picks a different field on every run, so the names are sorted and the
// first is taken. Both decision sites — buildRenderContext and the getStatusValue
// fallback — call this one helper, so the two cannot disagree about which field
// the status column represents.
func firstEnumField(schema map[string]rules.SchemaField) string {
	names := make([]string, 0, len(schema))
	for name, field := range schema {
		if len(field.Values) > 0 {
			names = append(names, name)
		}
	}
	if len(names) == 0 {
		return ""
	}
	sort.Strings(names)
	return names[0]
}

func runTree(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	scanRoot := "."
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
	if errors.Is(err, rules.ErrNoSchemaFound) {
		// Nothing under absRoot resolved a schema. There is no scope to filter
		// by and no governance to report, but the directory still has a shape,
		// and tree describes structure rather than returning a verdict about
		// it — so it renders what is there instead of refusing.
		//
		// This retry is deliberately narrower than passing AllowUngoverned
		// outright: as soon as one record IS governed, the scan above keeps its
		// scope filtering, and files outside every .stem's reach stay excluded.
		records, err = index.Scan(ctx, absRoot, reg, index.WithScopeResolver(resolver), index.AllowUngoverned())
	}
	if err != nil {
		return fmt.Errorf("scanning %s: %w", scanRoot, err)
	}

	derive.DeriveAllSimple(ctx, records, absRoot)
	derive.EnrichBuiltinsSimple(ctx, records, absRoot)
	derive.AggregateAllSimple(ctx, records, absRoot)

	// Apply --where filter.
	records, err = filterRecords(ctx, records, treeWhere, nil)
	if err != nil {
		return fmt.Errorf("filtering records: %w", err)
	}

	root := buildTree(records, filepath.Base(absRoot))

	if outputFormat == "json" {
		result := &TreeResult{Version: 2, Kind: "rootline/tree", Root: root}
		return outputJSON(cmd, result, false)
	}

	// Build render context with schema information for ASCII output.
	renderCtx := buildRenderContext(absRoot)

	// ASCII output
	lines := renderASCII(root, "", renderCtx)
	for _, line := range lines {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
	}
	return nil
}

func buildTree(records []*extract.Record, rootName string) *treeNode {
	root := &treeNode{Name: rootName, Path: rootName}

	for _, rec := range records {
		parts := strings.Split(filepath.ToSlash(rec.Path), "/")
		node := root

		// Navigate/create directory nodes
		for i := 0; i < len(parts)-1; i++ {
			child := findChild(node, parts[i])
			if child == nil {
				child = &treeNode{Name: parts[i], Path: strings.Join(parts[:i+1], "/")}
				node.Children = append(node.Children, child)
			}
			node = child
		}

		// Create leaf node for the file with Frontmatter map
		frontmatter := make(map[string]any)

		// Add frontmatter fields
		for k, v := range rec.Frontmatter {
			frontmatter[k] = v
		}

		// Add derived fields
		for k, v := range rec.Derived {
			frontmatter[k] = v
		}

		leaf := &treeNode{
			Name:        parts[len(parts)-1],
			Path:        rec.Path,
			IsLeaf:      true,
			Frontmatter: frontmatter,
			Total:       1,
		}
		node.Children = append(node.Children, leaf)
	}

	// Sort children and propagate counts
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

	node.Total = 0
	for _, child := range node.Children {
		propagateCounts(child)
		node.Total += child.Total
	}
}

func renderASCII(node *treeNode, prefix string, ctx *treeRenderContext) []string {
	var lines []string

	// Root node
	if prefix == "" {
		lines = append(lines, fmt.Sprintf("%s [%d]", node.Name, node.Total))
		for i, child := range node.Children {
			isLast := i == len(node.Children)-1
			lines = append(lines, renderChild(child, "", isLast, ctx)...)
		}
		return lines
	}
	return lines
}

func renderChild(node *treeNode, prefix string, isLast bool, ctx *treeRenderContext) []string {
	var lines []string

	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}

	if node.IsLeaf {
		status := getStatusValue(node, ctx)
		lines = append(lines, fmt.Sprintf("%s%s%s [%s]", prefix, connector, node.Name, status))
	} else {
		lines = append(lines, fmt.Sprintf("%s%s%s [%d]", prefix, connector, node.Name, node.Total))
		for i, child := range node.Children {
			isChildLast := i == len(node.Children)-1
			lines = append(lines, renderChild(child, childPrefix, isChildLast, ctx)...)
		}
	}

	return lines
}

// buildRenderContext constructs a treeRenderContext with schema information.
// It attempts to discover and merge .stem files from the given root to identify
// lifecycle-related fields for display in the ASCII tree.
func buildRenderContext(root string) *treeRenderContext {
	ctx := &treeRenderContext{
		effectiveSchema: make(map[string]rules.SchemaField),
	}

	// Attempt to discover and merge stems from the root.
	entries, err := rules.WalkUp(root)
	if err == nil && len(entries) > 0 {
		merged := rules.MergeStemFiles(entries)
		if merged != nil && len(merged.Schema) > 0 {
			ctx.effectiveSchema = merged.Schema

			// Enum fields are typically used for status/state values.
			ctx.lifecycleField = firstEnumField(merged.Schema)
		}
	}

	return ctx
}

// getStatusValue extracts the status/lifecycle value from a leaf node's frontmatter.
// Strategy:
// 1. If schema is available with a known lifecycle field, use it.
// 2. Otherwise, look for the first enum-typed field in schema.
// 3. If context is provided but has no enum fields, use em-dash.
// 4. If no context provided, try first enum-like field or any available string value.
func getStatusValue(node *treeNode, ctx *treeRenderContext) string {
	if len(node.Frontmatter) == 0 {
		return "—"
	}

	// If we have schema info with a known lifecycle field, use it.
	if ctx != nil && ctx.lifecycleField != "" {
		if val, ok := node.Frontmatter[ctx.lifecycleField]; ok {
			return fmt.Sprintf("%v", val)
		}
	}

	// Otherwise, pick the lexically first enum-typed field from schema if available.
	// This is the same selection buildRenderContext makes, so a context built
	// without a pre-resolved lifecycleField still lands on the same column.
	if ctx != nil && ctx.effectiveSchema != nil {
		if name := firstEnumField(ctx.effectiveSchema); name != "" {
			if val, ok := node.Frontmatter[name]; ok {
				return fmt.Sprintf("%v", val)
			}
		}
	}

	// If context was provided (schema-aware), but no enum field found, use em-dash.
	if ctx != nil {
		return "—"
	}

	// Fallback for tests without context: return first non-empty string value.
	for _, val := range node.Frontmatter {
		if valStr, ok := val.(string); ok && valStr != "" {
			return valStr
		}
	}

	// If all else fails, use em-dash.
	return "—"
}
