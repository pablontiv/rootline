package main

import (
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
	Long:  "Display the document tree with completion counts\nderived from frontmatter.estado fields.",
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
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}
	records, err := index.Scan(ctx, absRoot, reg, index.WithScopeResolver(resolver))
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

	// ASCII output
	lines := renderASCII(root, "")
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

func renderASCII(node *treeNode, prefix string) []string {
	var lines []string

	// Root node
	if prefix == "" {
		lines = append(lines, fmt.Sprintf("%s [%d]", node.Name, node.Total))
		for i, child := range node.Children {
			isLast := i == len(node.Children)-1
			lines = append(lines, renderChild(child, "", isLast)...)
		}
		return lines
	}
	return lines
}

func renderChild(node *treeNode, prefix string, isLast bool) []string {
	var lines []string

	connector := "├── "
	childPrefix := prefix + "│   "
	if isLast {
		connector = "└── "
		childPrefix = prefix + "    "
	}

	if node.IsLeaf {
		estado := ""
		if node.Frontmatter != nil {
			if e, ok := node.Frontmatter["estado"]; ok {
				estado = fmt.Sprintf("%v", e)
			}
		}
		if estado == "" {
			estado = "—"
		}
		lines = append(lines, fmt.Sprintf("%s%s%s [%s]", prefix, connector, node.Name, estado))
	} else {
		lines = append(lines, fmt.Sprintf("%s%s%s [%d]", prefix, connector, node.Name, node.Total))
		for i, child := range node.Children {
			isChildLast := i == len(node.Children)-1
			lines = append(lines, renderChild(child, childPrefix, isChildLast)...)
		}
	}

	return lines
}
