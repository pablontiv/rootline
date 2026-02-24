package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/infer"
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
	records, err := index.Scan(context.TODO(), absTarget, reg)
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
	stemFiles := buildHierarchicalStems(absTarget, hierarchy)

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

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %d .stem files (%d levels detected from %d files)\n",
		len(stemFiles), len(hierarchy.Levels), len(records))
	return nil
}

type stemFile struct {
	path    string
	content string
}

// buildHierarchicalStems generates the root and per-level .stem file contents.
func buildHierarchicalStems(absTarget string, hierarchy *infer.HierarchyResult) []stemFile {
	var files []stemFile

	// Root .stem: common fields + first level's sequence.
	rootYAML := generateHierarchicalRootYAML(hierarchy)
	files = append(files, stemFile{
		path:    filepath.Join(absTarget, ".stem"),
		content: rootYAML,
	})

	// Per-level .stems: each level's overrides go into parent dirs.
	for i := 1; i < len(hierarchy.Levels); i++ {
		ls := hierarchy.Levels[i]
		prevLevel := hierarchy.Levels[i-1]
		yaml := generateLevelStemYAML(&ls)

		// Write into each parent directory (previous level's dirs).
		for _, parentDir := range prevLevel.Level.DirPaths {
			stemPath := filepath.Join(absTarget, parentDir, ".stem")
			files = append(files, stemFile{
				path:    stemPath,
				content: yaml,
			})
		}
	}

	return files
}

// generateHierarchicalRootYAML generates the root .stem with common fields
// and the first level's sequence id.
func generateHierarchicalRootYAML(hierarchy *infer.HierarchyResult) string {
	var b strings.Builder
	b.WriteString("version: 1\nscope:\n  match: \"*.md\"\nschema:\n")

	// Add the first level's sequence id.
	if len(hierarchy.Levels) > 0 {
		ls := hierarchy.Levels[0]
		if idField, ok := ls.OnlyHere["id"]; ok {
			b.WriteString("  id:\n")
			fmt.Fprintf(&b, "    type: %s\n", idField.Type)
			if idField.Prefix != "" {
				fmt.Fprintf(&b, "    prefix: %s\n", idField.Prefix)
			}
			if idField.Digits > 0 {
				fmt.Fprintf(&b, "    digits: %d\n", idField.Digits)
			}
		}
	}

	// Add common fields from root schema.
	keys := make([]string, 0, len(hierarchy.Root.Schema))
	for k := range hierarchy.Root.Schema {
		if k == "id" {
			continue // already handled above
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		field := hierarchy.Root.Schema[name]
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

// generateLevelStemYAML generates a child .stem with only override fields.
func generateLevelStemYAML(ls *infer.LevelSchema) string {
	var b strings.Builder
	b.WriteString("schema:\n")

	// Always include the sequence id first.
	if idField, ok := ls.OnlyHere["id"]; ok {
		b.WriteString("  id:\n")
		fmt.Fprintf(&b, "    type: %s\n", idField.Type)
		if idField.Prefix != "" {
			fmt.Fprintf(&b, "    prefix: %s\n", idField.Prefix)
		}
		if idField.Digits > 0 {
			fmt.Fprintf(&b, "    digits: %d\n", idField.Digits)
		}
	}

	// Add non-id override fields.
	keys := make([]string, 0, len(ls.OnlyHere))
	for k := range ls.OnlyHere {
		if k == "id" {
			continue
		}
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		field := ls.OnlyHere[name]
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

func generateStemYAML(schema *infer.InferredSchema) string {
	var b strings.Builder
	b.WriteString("version: 1\nscope:\n  match: \"*.md\"\nschema:\n")

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
