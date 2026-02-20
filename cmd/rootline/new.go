package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var (
	newForce  bool
	newDryRun bool
)

var newCmd = &cobra.Command{
	Use:   "new <filepath>",
	Short: "Scaffold a document from effective schema",
	Long:  "Create a new markdown file with frontmatter pre-populated\nfrom the effective .stem schema of the target directory.",
	Args:  cobra.ExactArgs(1),
	RunE:  runNew,
}

func init() {
	newCmd.Flags().BoolVar(&newForce, "force", false, "overwrite existing file")
	newCmd.Flags().BoolVar(&newDryRun, "dry-run", false, "show generated content without writing file")
	rootCmd.AddCommand(newCmd)
}

func runNew(cmd *cobra.Command, args []string) error {
	target := args[0]
	absTarget, err := filepath.Abs(target)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Check if file exists
	if !newForce {
		if _, err := os.Stat(absTarget); err == nil {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", target)
		}
	}

	// Resolve effective schema from parent directory
	dir := filepath.Dir(absTarget)
	entries, err := rules.WalkUp(dir)
	if err != nil {
		return fmt.Errorf("discovering .stem files: %w", err)
	}

	effective := rules.MergeStemFiles(entries)
	if effective == nil || len(effective.Schema) == 0 {
		return fmt.Errorf("no .stem schema found for %s", dir)
	}

	// Generate markdown content
	content := generateMarkdown(absTarget, effective)

	if newDryRun {
		fmt.Fprint(cmd.OutOrStdout(), content)
		return nil
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(absTarget, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", target)
	return nil
}

func generateMarkdown(absPath string, effective *rules.StemFile) string {
	var b strings.Builder
	b.WriteString("---\n")

	// Sort fields for deterministic output
	keys := make([]string, 0, len(effective.Schema))
	for k := range effective.Schema {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		field := effective.Schema[name]

		// Determine value
		value := ""
		if field.Default != "" {
			value = field.Default
		} else if field.Type == "enum" && len(field.Values) > 0 {
			value = field.Values[0]
		}

		if len(field.Values) > 0 {
			b.WriteString(fmt.Sprintf("%s: %s # [%s]\n", name, value, strings.Join(field.Values, ", ")))
		} else if value != "" {
			b.WriteString(fmt.Sprintf("%s: %s\n", name, value))
		} else if field.Required {
			b.WriteString(fmt.Sprintf("%s: \n", name))
		}
	}

	b.WriteString("---\n")

	// Title from filename
	base := filepath.Base(absPath)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	b.WriteString(fmt.Sprintf("# %s\n", strings.Title(title)))

	return b.String()
}
