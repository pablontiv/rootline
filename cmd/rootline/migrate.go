package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/migrate"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var (
	migrateDryRun bool
	migrateFrom   string
	migrateRename string
	migrateSplit  bool
)

var migrateCmd = &cobra.Command{
	Use:   "migrate [path]",
	Short: "Detect and apply schema changes in .stem files",
	Long: `Compare current .stem files against a previous version and report changes,
or perform bulk migration operations like field renaming.

By default, compares against the git HEAD version. Use --from to compare
against a specific file. The --dry-run flag reports changes without modifying
any files. Use --rename old=new to rename a field across all documents.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runMigrate,
}

func init() {
	migrateCmd.Flags().BoolVar(&migrateDryRun, "dry-run", false, "report changes without modifying files")
	migrateCmd.Flags().StringVar(&migrateFrom, "from", "", "compare against specified .stem file instead of git HEAD")
	migrateCmd.Flags().StringVar(&migrateRename, "rename", "", "rename a field: old_field=new_field")
	migrateCmd.Flags().BoolVar(&migrateSplit, "split", false, "split a flat .stem into hierarchical .stem files per level")
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	if migrateSplit {
		return runMigrateSplit(cmd, args)
	}
	if migrateRename != "" {
		return runMigrateRename(cmd, args)
	}
	return runMigrateDiff(cmd, args)
}

// --- Rename operation ---

func runMigrateRename(cmd *cobra.Command, args []string) error {
	parts := strings.SplitN(migrateRename, "=", 2)
	if len(parts) != 2 || parts[0] == "" || parts[1] == "" {
		return fmt.Errorf("invalid --rename format; expected old_field=new_field")
	}

	rootPath := "."
	if len(args) > 0 {
		rootPath = args[0]
	}

	op := &migrate.RenameOperation{
		OldField: parts[0],
		NewField: parts[1],
		RootPath: rootPath,
		DryRun:   migrateDryRun,
	}

	result, err := op.Execute()
	if err != nil {
		return err
	}

	// Append to migration log (skip for dry-run).
	if !migrateDryRun && result.Summary.FilesUpdated+result.Summary.StemsUpdated > 0 {
		absRoot, absErr := filepath.Abs(rootPath)
		if absErr != nil {
			return fmt.Errorf("resolving root path for log: %w", absErr)
		}
		ml := migrate.NewMigrationLog(absRoot)
		entry := migrate.NewRenameEntry(parts[0], parts[1], result.Summary.FilesUpdated+result.Summary.StemsUpdated)
		if logErr := ml.Append(entry); logErr != nil {
			return fmt.Errorf("writing migration log: %w", logErr)
		}
	}

	if outputFormat == "table" {
		return renderMigrateRenameTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

func renderMigrateRenameTable(cmd *cobra.Command, result *migrate.RenameResult) error {
	w := cmd.OutOrStdout()

	if result.Summary.FilesUpdated == 0 && result.Summary.StemsUpdated == 0 {
		_, _ = fmt.Fprintf(w, "No files affected (field %q not found in any documents)\n", result.OldField)
		return nil
	}

	prefix := ""
	if migrateDryRun {
		prefix = "would "
	}

	if len(result.FilesUpdated) > 0 {
		_, _ = fmt.Fprintf(w, "Files %supdated (%s → %s):\n", prefix, result.OldField, result.NewField)
		for _, f := range result.FilesUpdated {
			_, _ = fmt.Fprintf(w, "  %s\n", f)
		}
	}

	if len(result.StemsUpdated) > 0 {
		_, _ = fmt.Fprintf(w, "\n.stem schemas %supdated:\n", prefix)
		for _, s := range result.StemsUpdated {
			_, _ = fmt.Fprintf(w, "  %s\n", s)
		}
	}

	_, _ = fmt.Fprintf(w, "\nSummary: %d files %supdated, %d stems %supdated (of %d scanned)\n",
		result.Summary.FilesUpdated, prefix,
		result.Summary.StemsUpdated, prefix,
		result.Summary.FilesScanned)

	return nil
}

// --- Diff operation ---

func runMigrateDiff(cmd *cobra.Command, args []string) error {
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	absPath, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", targetPath, err)
	}

	var stemPaths []string
	if info.IsDir() {
		stemPaths, err = migrate.FindStemFiles(absPath)
		if err != nil {
			return fmt.Errorf("finding .stem files: %w", err)
		}
		if len(stemPaths) == 0 {
			return fmt.Errorf("no .stem files found in %s", targetPath)
		}
	} else {
		stemPaths = []string{absPath}
	}

	if migrateFrom != "" && len(stemPaths) > 1 {
		return fmt.Errorf("--from can only be used when targeting a single .stem file")
	}

	var results []*migrate.DiffResult
	for _, stemPath := range stemPaths {
		result, diffErr := diffSingleStem(stemPath)
		if diffErr != nil {
			current, parseErr := rules.ParseStemFile(stemPath)
			if parseErr != nil {
				return fmt.Errorf("parsing %s: %w", stemPath, parseErr)
			}
			result = migrate.Diff(stemPath, nil, current)
		}
		results = append(results, result)
	}

	if len(results) == 1 {
		return renderMigrateResult(cmd, results[0])
	}
	return renderMigrateBatch(cmd, results)
}

func diffSingleStem(stemPath string) (*migrate.DiffResult, error) {
	current, err := rules.ParseStemFile(stemPath)
	if err != nil {
		return nil, fmt.Errorf("parsing current %s: %w", stemPath, err)
	}

	previous, err := migrate.LoadPreviousStem(stemPath, migrateFrom)
	if err != nil {
		return nil, err
	}

	return migrate.Diff(stemPath, previous, current), nil
}

// MigrateBatchResult wraps multiple diff results for JSON output.
type MigrateBatchResult struct {
	Version int                   `json:"version"`
	Kind    string                `json:"kind"`
	Results []*migrate.DiffResult `json:"results"`
	Summary MigrateBatchSummary   `json:"summary"`
}

// MigrateBatchSummary holds aggregate counts.
type MigrateBatchSummary struct {
	StemsChecked  int `json:"stems_checked"`
	TotalChanges  int `json:"total_changes"`
	BreakingCount int `json:"breaking_count"`
}

func renderMigrateResult(cmd *cobra.Command, result *migrate.DiffResult) error {
	if outputFormat == "table" {
		return renderMigrateTable(cmd, result)
	}
	return renderMigrateJSON(cmd, result)
}

func renderMigrateBatch(cmd *cobra.Command, results []*migrate.DiffResult) error {
	batch := &MigrateBatchResult{
		Version: 1,
		Kind:    "rootline/migrate-batch",
		Results: results,
	}
	for _, r := range results {
		batch.Summary.StemsChecked++
		batch.Summary.TotalChanges += r.TotalCount
		batch.Summary.BreakingCount += r.BreakingCount
	}

	if outputFormat == "table" {
		return renderMigrateBatchTable(cmd, batch)
	}
	data, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	if len(fieldPath) > 0 {
		data, err = extractField(data, fieldPath[0])
		if err != nil {
			return err
		}
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func renderMigrateJSON(cmd *cobra.Command, result *migrate.DiffResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("marshaling JSON: %w", err)
	}
	if len(fieldPath) > 0 {
		data, err = extractField(data, fieldPath[0])
		if err != nil {
			return err
		}
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout(), string(data))
	return nil
}

func renderMigrateTable(cmd *cobra.Command, result *migrate.DiffResult) error {
	if len(result.Changes) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no changes detected")
		return nil
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Schema changes in %s:\n\n", result.StemPath)

	headers := []string{"Kind", "Field", "Breaking", "Message"}
	var rows [][]string
	for _, c := range result.Changes {
		breakingStr := "no"
		if c.Breaking {
			breakingStr = "YES"
		}
		rows = append(rows, []string{string(c.Kind), c.Field, breakingStr, c.Message})
	}

	renderTable(cmd.OutOrStdout(), headers, rows)

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d changes (%d breaking)\n", result.TotalCount, result.BreakingCount)
	return nil
}

func renderMigrateBatchTable(cmd *cobra.Command, batch *MigrateBatchResult) error {
	if batch.Summary.TotalChanges == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "no changes detected")
		return nil
	}

	for _, result := range batch.Results {
		if len(result.Changes) == 0 {
			continue
		}
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Schema changes in %s:\n\n", result.StemPath)

		headers := []string{"Kind", "Field", "Breaking", "Message"}
		var rows [][]string
		for _, c := range result.Changes {
			breakingStr := "no"
			if c.Breaking {
				breakingStr = "YES"
			}
			rows = append(rows, []string{string(c.Kind), c.Field, breakingStr, c.Message})
		}
		renderTable(cmd.OutOrStdout(), headers, rows)
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Summary: %d stems checked, %d changes (%d breaking)\n",
		batch.Summary.StemsChecked, batch.Summary.TotalChanges, batch.Summary.BreakingCount)
	return nil
}

// --- Split operation ---

func runMigrateSplit(cmd *cobra.Command, args []string) error {
	targetPath := "."
	if len(args) > 0 {
		targetPath = args[0]
	}

	absTarget, err := filepath.Abs(targetPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Parse existing .stem.
	stemPath := filepath.Join(absTarget, ".stem")
	existing, err := rules.ParseStemFile(stemPath)
	if err != nil {
		return fmt.Errorf("reading existing .stem: %w", err)
	}

	// Scan records.
	reg := extract.NewRegistry()
	records, err := index.Scan(absTarget, reg)
	if err != nil {
		return fmt.Errorf("scanning %s: %w", targetPath, err)
	}

	if len(records) == 0 {
		return fmt.Errorf("no markdown files found in %s", targetPath)
	}

	// Detect hierarchy.
	hierarchy := infer.AnalyzeHierarchy(records, absTarget)
	if !hierarchy.Detected {
		return fmt.Errorf("no hierarchy detected in %s; nothing to split", targetPath)
	}

	// Build split stems using field distribution from existing .stem.
	stemFiles := buildSplitStems(absTarget, existing, hierarchy)

	if migrateDryRun {
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
		if writeErr := os.WriteFile(sf.path, []byte(sf.content), 0644); writeErr != nil {
			return fmt.Errorf("writing %s: %w", sf.path, writeErr)
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(),
		"Split .stem into %d files across %d levels (derive/aggregate/links preserved at root)\n",
		len(stemFiles), len(hierarchy.Levels))
	return nil
}

// buildSplitStems distributes fields from existing .stem across hierarchy levels.
// Derive, aggregate, links, structural, and validate stay at root.
func buildSplitStems(absTarget string, existing *rules.StemFile, hierarchy *infer.HierarchyResult) []stemFile {
	var files []stemFile

	// Determine which fields from existing schema belong at root vs per-level.
	rootFields := make(map[string]rules.SchemaField)
	levelFields := make(map[int]map[string]rules.SchemaField)

	for i := range hierarchy.Levels {
		levelFields[i] = make(map[string]rules.SchemaField)
	}

	for name, sf := range existing.Schema {
		if name == "id" {
			continue // handled per-level via sequence detection
		}

		// Check if this field is in root (all levels) or specific levels.
		if _, inRoot := hierarchy.Root.Schema[name]; inRoot {
			rootFields[name] = sf
		} else {
			// Assign to levels that have it in OnlyHere.
			for i, ls := range hierarchy.Levels {
				if _, ok := ls.OnlyHere[name]; ok {
					levelFields[i][name] = sf
				}
			}
		}
	}

	// Build root .stem YAML preserving derive/aggregate/links/structural.
	rootYAML := buildSplitRootYAML(existing, rootFields, hierarchy)
	files = append(files, stemFile{
		path:    filepath.Join(absTarget, ".stem"),
		content: rootYAML,
	})

	// Build child .stem files.
	for i := 1; i < len(hierarchy.Levels); i++ {
		ls := hierarchy.Levels[i]
		prevLevel := hierarchy.Levels[i-1]

		childYAML := buildSplitChildYAML(&ls, levelFields[i])

		for _, parentDir := range prevLevel.Level.DirPaths {
			files = append(files, stemFile{
				path:    filepath.Join(absTarget, parentDir, ".stem"),
				content: childYAML,
			})
		}
	}

	return files
}

// buildSplitRootYAML generates the root .stem preserving non-schema sections.
func buildSplitRootYAML(existing *rules.StemFile, rootFields map[string]rules.SchemaField, hierarchy *infer.HierarchyResult) string {
	var b strings.Builder

	b.WriteString("version: 1\n")
	if existing.Scope.Match != "" {
		fmt.Fprintf(&b, "scope:\n  match: %q\n", existing.Scope.Match)
	}

	b.WriteString("\nschema:\n")

	// Add first level's sequence id.
	if len(hierarchy.Levels) > 0 {
		if idField, ok := hierarchy.Levels[0].OnlyHere["id"]; ok {
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

	// Add root fields sorted.
	keys := make([]string, 0, len(rootFields))
	for k := range rootFields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		sf := rootFields[name]
		fmt.Fprintf(&b, "  %s:\n", name)
		fmt.Fprintf(&b, "    type: %s\n", sf.Type)
		if sf.Required {
			b.WriteString("    required: true\n")
		}
		if len(sf.Values) > 0 {
			fmt.Fprintf(&b, "    values: [%s]\n", strings.Join(sf.Values, ", "))
		}
		if sf.Severity != "" {
			fmt.Fprintf(&b, "    severity: %s\n", sf.Severity)
		}
	}

	// Preserve structural.
	if existing.Structural.Subdirs.RequireIndex != "" || existing.Structural.Subdirs.MinChildren > 0 {
		b.WriteString("\nstructural:\n  subdirs:\n")
		if existing.Structural.Subdirs.RequireIndex != "" {
			fmt.Fprintf(&b, "    require_index: %s\n", existing.Structural.Subdirs.RequireIndex)
		}
		if existing.Structural.Subdirs.MinChildren > 0 {
			fmt.Fprintf(&b, "    min_children: %d\n", existing.Structural.Subdirs.MinChildren)
		}
		if existing.Structural.Subdirs.Severity != "" {
			fmt.Fprintf(&b, "    severity: %s\n", existing.Structural.Subdirs.Severity)
		}
	}

	// Preserve links.
	if len(existing.Links.Allowed) > 0 {
		fmt.Fprintf(&b, "\nlinks:\n  allowed: [%s]\n", strings.Join(existing.Links.Allowed, ", "))
		for name, rule := range existing.Links.Rules {
			fmt.Fprintf(&b, "  %s:\n", name)
			if rule.Target != "" {
				fmt.Fprintf(&b, "    target: %q\n", rule.Target)
			}
			if rule.Field != "" {
				fmt.Fprintf(&b, "    field: %s\n", rule.Field)
			}
		}
	}

	// Preserve derive.
	if len(existing.Derive) > 0 {
		b.WriteString("\nderive:\n")
		for name, expr := range existing.Derive {
			exprStr := fmt.Sprintf("%v", expr)
			if strings.Contains(exprStr, "\n") {
				fmt.Fprintf(&b, "  %s: |\n    %s\n", name, strings.ReplaceAll(strings.TrimSpace(exprStr), "\n", "\n    "))
			} else {
				fmt.Fprintf(&b, "  %s: %q\n", name, exprStr)
			}
		}
	}

	// Preserve aggregate.
	if len(existing.Aggregate) > 0 {
		b.WriteString("\naggregate:\n")
		for name, expr := range existing.Aggregate {
			exprStr := fmt.Sprintf("%v", expr)
			if strings.Contains(exprStr, "\n") {
				fmt.Fprintf(&b, "  %s: |\n    %s\n", name, strings.ReplaceAll(strings.TrimSpace(exprStr), "\n", "\n    "))
			} else {
				fmt.Fprintf(&b, "  %s: %q\n", name, exprStr)
			}
		}
	}

	// Preserve validate.
	if len(existing.Validate) > 0 {
		b.WriteString("\nvalidate:\n")
		for _, v := range existing.Validate {
			fmt.Fprintf(&b, "  - rule: %s\n", v.Rule)
			if v.Field != "" {
				fmt.Fprintf(&b, "    field: %s\n", v.Field)
			}
			if v.Severity != "" {
				fmt.Fprintf(&b, "    severity: %s\n", v.Severity)
			}
		}
	}

	return b.String()
}

// buildSplitChildYAML generates a child .stem with level-specific overrides.
func buildSplitChildYAML(ls *infer.LevelSchema, extraFields map[string]rules.SchemaField) string {
	var b strings.Builder
	b.WriteString("schema:\n")

	// Sequence id.
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

	// Level-specific fields from existing schema.
	keys := make([]string, 0, len(extraFields))
	for k := range extraFields {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	for _, name := range keys {
		sf := extraFields[name]
		fmt.Fprintf(&b, "  %s:\n", name)
		fmt.Fprintf(&b, "    type: %s\n", sf.Type)
		if sf.Required {
			b.WriteString("    required: true\n")
		}
		if len(sf.Values) > 0 {
			fmt.Fprintf(&b, "    values: [%s]\n", strings.Join(sf.Values, ", "))
		}
		if sf.Severity != "" {
			fmt.Fprintf(&b, "    severity: %s\n", sf.Severity)
		}
	}

	return b.String()
}
