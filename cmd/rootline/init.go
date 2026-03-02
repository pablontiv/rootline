package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/migrate"
	"github.com/spf13/cobra"
)

var (
	initDryRun bool
	initForce  bool
)

var initCmd = &cobra.Command{
	Use:   "init [path]",
	Short: "Generate .stem from existing documents",
	Long:  "Scan markdown files and infer a .stem schema from frontmatter patterns.\nUse --dry-run to preview without writing.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runInit,
}

func init() {
	initCmd.Flags().BoolVar(&initDryRun, "dry-run", false, "print to stdout without writing file")
	initCmd.Flags().BoolVar(&initForce, "force", false, "overwrite existing .stem file")
	rootCmd.AddCommand(initCmd)
}

func runInit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	target := "."
	if len(args) > 0 {
		target = args[0]
	}

	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Scan for records
	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, absTarget, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", target, err)
	}

	if len(records) == 0 {
		return fmt.Errorf("no markdown files found in %s", target)
	}

	// Try hierarchical detection first.
	hierarchy := infer.AnalyzeHierarchy(records, absTarget)
	if hierarchy.Detected {
		return runInitHierarchical(cmd, absTarget, target, hierarchy, records)
	}

	// Fall back to flat mode.
	return runInitFlat(cmd, absTarget, target, records)
}

func runInitFlat(cmd *cobra.Command, absTarget, target string, records []*extract.Record) error {
	stemPath := filepath.Join(absTarget, ".stem")

	// Check for existing .stem
	if !initForce && !initDryRun {
		if _, err := os.Stat(stemPath); err == nil {
			return fmt.Errorf(".stem already exists in %s (use --force to overwrite)", target)
		}
	}

	// Analyze
	schema := infer.Analyze(records)

	// Warn about mixed content
	if schema.TotalFiles > 0 && schema.FilesWithout > 0 {
		ratio := float64(schema.FilesWithout) / float64(schema.TotalFiles)
		if ratio > 0.2 {
			_, _ = fmt.Fprintf(cmd.ErrOrStderr(),
				"Warning: %d of %d files have no frontmatter. Consider running init on a more specific subdirectory.\n",
				schema.FilesWithout, schema.TotalFiles)
		}
	}

	// Generate YAML
	yaml := generateStemYAML(schema)

	if initDryRun {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), yaml)
		return nil
	}

	if err := os.WriteFile(stemPath, []byte(yaml), 0644); err != nil {
		return fmt.Errorf("writing .stem: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s (%d fields inferred from %d files)\n",
		stemPath, len(schema.Schema), len(records))
	return nil
}

func runInitHierarchical(cmd *cobra.Command, absTarget, target string, hierarchy *infer.HierarchyResult, records []*extract.Record) error {
	// Build the list of .stem files to write.
	stemFiles, generatedAggNames := buildHierarchicalStems(absTarget, hierarchy)

	// Emit notes for auto-generated aggregates.
	for _, name := range generatedAggNames {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Note: auto-generated aggregate for '%s'\n", name)
	}

	// Check for existing .stem files.
	if !initForce && !initDryRun {
		for _, sf := range stemFiles {
			if _, err := os.Stat(sf.path); err == nil {
				return fmt.Errorf(".stem already exists at %s (use --force to overwrite)", sf.path)
			}
		}
	}

	if initDryRun {
		for i, sf := range stemFiles {
			if i > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}
			relPath, _ := filepath.Rel(absTarget, sf.path)
			if relPath == "" {
				relPath = sf.path
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "# --- %s ---\n", relPath)
			_, _ = fmt.Fprint(cmd.OutOrStdout(), sf.content)
		}
		return nil
	}

	// Write all .stem files.
	for _, sf := range stemFiles {
		if err := os.WriteFile(sf.path, []byte(sf.content), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", sf.path, err)
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created .stem with %d levels (detected from %d files)\n",
		len(hierarchy.Levels), len(records))
	return nil
}

type stemFile struct {
	path    string
	content string
}

// buildHierarchicalStems generates a single root .stem file with a levels: section.
// Returns the stem files and a sorted list of field names for which aggregates were generated.
func buildHierarchicalStems(absTarget string, hierarchy *infer.HierarchyResult) ([]stemFile, []string) {
	var files []stemFile

	// Generate aggregate expressions for enum fields in root schema.
	aggregates := migrate.GenerateAggregates(hierarchy.Root.Schema, nil)
	aggNames := make([]string, 0, len(aggregates))
	for k := range aggregates {
		aggNames = append(aggNames, k)
	}
	sort.Strings(aggNames)

	// Single root .stem: common fields + levels section + aggregates.
	rootYAML := generateHierarchicalRootYAML(hierarchy, aggregates)
	files = append(files, stemFile{
		path:    filepath.Join(absTarget, ".stem"),
		content: rootYAML,
	})

	return files, aggNames
}

// generateHierarchicalRootYAML generates the root .stem with common fields,
// match-based per-level schema, and aggregates.
func generateHierarchicalRootYAML(hierarchy *infer.HierarchyResult, aggregates map[string]string) string {
	var b strings.Builder
	b.WriteString("version: 2\nscope:\n  match: \"*.md\"\nschema:\n")

	// Merge root and per-level fields into a single schema with match annotations.
	matchSchema := hierarchy.ToMatchSchema()

	keys := make([]string, 0, len(matchSchema))
	for k := range matchSchema {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		field := matchSchema[name]
		fmt.Fprintf(&b, "  %s:\n", name)
		fmt.Fprintf(&b, "    type: %s\n", field.Type)
		if field.Required {
			b.WriteString("    required: true\n")
		}
		if len(field.Values) > 0 {
			fmt.Fprintf(&b, "    values: [%s]\n", strings.Join(field.Values, ", "))
		}
		if field.Prefix != "" {
			fmt.Fprintf(&b, "    prefix: %s\n", field.Prefix)
		}
		if field.Digits > 0 {
			fmt.Fprintf(&b, "    digits: %d\n", field.Digits)
		}
		if field.Match != nil {
			switch {
			case len(field.Match.Configs) > 0:
				b.WriteString("    match:\n")
				matchKeys := make([]string, 0, len(field.Match.Configs))
				for k := range field.Match.Configs {
					matchKeys = append(matchKeys, k)
				}
				sort.Strings(matchKeys)
				for _, mk := range matchKeys {
					cfg := field.Match.Configs[mk]
					if cfgMap, ok := cfg.(map[string]any); ok {
						fmt.Fprintf(&b, "      \"%s\": {", mk)
						first := true
						if p, ok := cfgMap["prefix"]; ok {
							fmt.Fprintf(&b, "prefix: %s", p)
							first = false
						}
						if d, ok := cfgMap["digits"]; ok {
							if !first {
								b.WriteString(", ")
							}
							fmt.Fprintf(&b, "digits: %v", d)
						}
						b.WriteString("}\n")
					}
				}
			case len(field.Match.Patterns) == 1:
				fmt.Fprintf(&b, "    match: \"%s\"\n", field.Match.Patterns[0])
			case len(field.Match.Patterns) > 1:
				fmt.Fprintf(&b, "    match: [%s]\n", strings.Join(field.Match.Patterns, ", "))
			}
		}
	}

	// Emit aggregate section for auto-generated expressions.
	if len(aggregates) > 0 {
		aggKeys := make([]string, 0, len(aggregates))
		for k := range aggregates {
			aggKeys = append(aggKeys, k)
		}
		sort.Strings(aggKeys)

		b.WriteString("aggregate:\n")
		for _, name := range aggKeys {
			fmt.Fprintf(&b, "  %s: |\n", name)
			for _, line := range strings.Split(aggregates[name], "\n") {
				fmt.Fprintf(&b, "    %s\n", line)
			}
		}
	}

	return b.String()
}

func generateStemYAML(schema *infer.InferredSchema) string {
	var b strings.Builder
	b.WriteString("version: 2\nscope:\n  match: \"*.md\"\nschema:\n")

	keys := make([]string, 0, len(schema.Schema))
	for k := range schema.Schema {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		field := schema.Schema[name]
		fmt.Fprintf(&b, "  %s:\n", name)
		fmt.Fprintf(&b, "    type: %s\n", field.Type)
		if field.Required {
			b.WriteString("    required: true\n")
		}
		if len(field.Values) > 0 {
			fmt.Fprintf(&b, "    values: [%s]\n", strings.Join(field.Values, ", "))
		}
	}

	return b.String()
}
