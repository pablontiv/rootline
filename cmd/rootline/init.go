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

	stemPath := filepath.Join(absTarget, ".stem")

	// Check for existing .stem
	if !initForce && !initDryRun {
		if _, err := os.Stat(stemPath); err == nil {
			return fmt.Errorf(".stem already exists in %s (use --force to overwrite)", target)
		}
	}

	// Scan for records
	reg := extract.NewRegistry()
	records, err := index.Scan(absTarget, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", target, err)
	}

	if len(records) == 0 {
		return fmt.Errorf("no markdown files found in %s", target)
	}

	// Analyze
	schema := infer.Analyze(records)

	// Warn about mixed content
	if schema.TotalFiles > 0 && schema.FilesWithout > 0 {
		ratio := float64(schema.FilesWithout) / float64(schema.TotalFiles)
		if ratio > 0.2 {
			fmt.Fprintf(cmd.ErrOrStderr(),
				"Warning: %d of %d files have no frontmatter. Consider running init on a more specific subdirectory.\n",
				schema.FilesWithout, schema.TotalFiles)
		}
	}

	// Generate YAML
	yaml := generateStemYAML(schema)

	if initDryRun {
		fmt.Fprint(cmd.OutOrStdout(), yaml)
		return nil
	}

	if err := os.WriteFile(stemPath, []byte(yaml), 0644); err != nil {
		return fmt.Errorf("writing .stem: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created %s (%d fields inferred from %d files)\n",
		stemPath, len(schema.Schema), len(records))
	return nil
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
		b.WriteString(fmt.Sprintf("  %s:\n", name))
		b.WriteString(fmt.Sprintf("    type: %s\n", field.Type))
		if field.Required {
			b.WriteString("    required: true\n")
		}
		if len(field.Values) > 0 {
			b.WriteString(fmt.Sprintf("    values: [%s]\n", strings.Join(field.Values, ", ")))
		}
	}

	return b.String()
}
