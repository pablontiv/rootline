package main

import (
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
	"golang.org/x/text/cases"
	"golang.org/x/text/language"
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
	if newForce {
		if err := rejectForcedFinalCaseCollision(absTarget); err != nil {
			return err
		}
	}

	// Check if file exists
	if !newForce {
		if _, err := os.Stat(absTarget); err == nil {
			return fmt.Errorf("file already exists: %s (use --force to overwrite)", target)
		}
	}

	// Resolve effective schema using per-record resolution to apply match filtering.
	// This ensures that if the schema has match-scoped fields, only fields applicable
	// to this record type are included in the generated frontmatter.
	dir := filepath.Dir(absTarget)
	effective, err := rules.ResolveForRecord(dir, absTarget)
	if err != nil {
		return fmt.Errorf("resolving .stem for %s: %w", dir, err)
	}

	if effective == nil || len(effective.Schema) == 0 {
		return fmt.Errorf("no .stem schema found for %s", dir)
	}

	if schemaRoot := rules.SchemaRoot(absTarget); schemaRoot != "" {
		within, err := rules.PhysicalPathWithin(schemaRoot, absTarget)
		if err != nil {
			return fmt.Errorf("validating target boundary: %w", err)
		}
		if !within {
			return fmt.Errorf("target escapes schema root: %s", target)
		}
	}

	// Generate markdown content.
	content, err := generateMarkdown(absTarget, effective)
	if err != nil {
		return fmt.Errorf("generating markdown: %w", err)
	}

	if err := validateProspectiveNewContent(cmd, absTarget, content, effective); err != nil {
		return err
	}

	if newDryRun {
		_, _ = fmt.Fprint(cmd.OutOrStdout(), content)
		return nil
	}

	// Ensure parent directory exists
	if err := os.MkdirAll(dir, 0755); err != nil {
		return fmt.Errorf("creating directory: %w", err)
	}

	if err := os.WriteFile(absTarget, []byte(content), 0644); err != nil {
		return fmt.Errorf("writing file: %w", err)
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Created %s\n", target)
	return nil
}

func generateMarkdown(absPath string, effective *rules.StemFile) (string, error) {
	var b strings.Builder
	b.WriteString("---\n")

	// Sort fields for deterministic output.
	keys := make([]string, 0, len(effective.Schema))
	for k := range effective.Schema {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		field := effective.Schema[name]
		if field.Extract != "" {
			continue
		}

		// Determine value for frontmatter fields.
		// Only use field.Default when explicitly set; never fall back to Values[0].
		value := ""
		if field.Default != "" {
			value = field.Default
		}

		switch {
		case len(field.Values) > 0 && (value != "" || field.Required):
			fmt.Fprintf(&b, "%s: %s # [%s]\n", name, value, strings.Join(field.Values, ", "))
		case value != "":
			fmt.Fprintf(&b, "%s: %s\n", name, value)
		case field.Required:
			fmt.Fprintf(&b, "%s: \n", name)
		}
	}

	b.WriteString("---\n")

	// Title from filename.
	base := filepath.Base(absPath)
	title := strings.TrimSuffix(base, filepath.Ext(base))
	title = strings.ReplaceAll(title, "-", " ")
	title = strings.ReplaceAll(title, "_", " ")
	fmt.Fprintf(&b, "# %s\n", cases.Title(language.Und).String(title))

	record, err := extractProspectiveNewRecord(absPath, b.String())
	if err != nil {
		return "", err
	}
	sections, err := rules.RequiredSectionMaterializations(record, effective)
	if err != nil {
		return "", err
	}
	for _, section := range sections {
		b.WriteString("\n")
		b.WriteString(section.Heading)
		b.WriteString("\n\n")
		b.WriteString(section.Content)
		b.WriteString("\n")
	}

	return b.String(), nil
}

func validateProspectiveNewContent(cmd *cobra.Command, absPath, content string, effective *rules.StemFile) error {
	prospectiveBytes := []byte(content)
	result, err := validateProspectiveRecord(cmd.Context(), prospectiveRecordValidationInput{
		Path:                         absPath,
		AbsPath:                      absPath,
		Content:                      prospectiveBytes,
		Effective:                    effective,
		ProspectiveLinkTargetAbsPath: absPath,
		ProspectiveLinkTargetContent: prospectiveBytes,
	})
	if err != nil {
		return fmt.Errorf("prospective record validation failed: %w", err)
	}
	if !validationResultHasFailure(result, false) {
		return nil
	}

	msgs := make([]string, 0, len(result.Errors))
	for _, e := range result.Errors {
		field := e.Field
		if field == "" {
			field = e.Rule
		}
		msgs = append(msgs, fmt.Sprintf("%s: %s", field, e.Message))
	}
	return fmt.Errorf("prospective record validation failed: %s", strings.Join(msgs, "; "))
}

func extractProspectiveNewRecord(absPath, content string) (*extract.Record, error) {
	return extractProspectiveRecord(absPath, absPath, []byte(content))
}

func rejectForcedFinalCaseCollision(absTarget string) error {
	if _, err := os.Stat(absTarget); err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("checking existing target: %w", err)
	}

	baseName := filepath.Base(absTarget)
	entries, err := os.ReadDir(filepath.Dir(absTarget))
	if err != nil {
		return fmt.Errorf("checking target directory: %w", err)
	}
	caseFoldMatch := ""
	for _, entry := range entries {
		name := entry.Name()
		if name == baseName {
			return nil
		}
		if caseFoldMatch == "" && strings.EqualFold(name, baseName) {
			caseFoldMatch = name
		}
	}
	if caseFoldMatch != "" {
		return fmt.Errorf("target path %q collides with existing differently-cased entry %q", absTarget, caseFoldMatch)
	}
	return nil
}
