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
	migrateDryRun   bool
	migrateFrom     string
	migrateRename   string
	migrateSplit    bool
	migrateScaffold bool
	migrateForce    bool
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
	migrateCmd.Flags().BoolVar(&migrateScaffold, "scaffold", false, "scaffold missing required section fields in documents")
	migrateCmd.Flags().BoolVar(&migrateForce, "force", false, "with --split: overwrite existing child .stem files")
	rootCmd.AddCommand(migrateCmd)
}

func runMigrate(cmd *cobra.Command, args []string) error {
	if migrateSplit {
		return runMigrateSplit(cmd, args)
	}
	if migrateRename != "" {
		return runMigrateRename(cmd, args)
	}
	if migrateScaffold {
		return runMigrateScaffold(cmd, args)
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

// --- Scaffold operation ---

func runMigrateScaffold(cmd *cobra.Command, args []string) error {
	rootPath := "."
	if len(args) > 0 {
		rootPath = args[0]
	}

	absPath, err := filepath.Abs(rootPath)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return fmt.Errorf("stat %s: %w", rootPath, err)
	}
	if !info.IsDir() {
		return fmt.Errorf("--scaffold requires a directory path, got: %s", rootPath)
	}

	op := &migrate.ScaffoldOperation{
		RootPath: absPath,
		DryRun:   migrateDryRun,
	}

	result, err := op.Execute()
	if err != nil {
		return fmt.Errorf("scaffolding sections: %w", err)
	}

	if outputFormat != "table" {
		return outputJSON(cmd, result, false)
	}
	return renderMigrateScaffoldTable(cmd, result)
}

func renderMigrateScaffoldTable(cmd *cobra.Command, result *migrate.ScaffoldResult) error {
	w := cmd.OutOrStdout()

	if result.SectionsAdded == 0 {
		_, _ = fmt.Fprintln(w, "no missing required sections found")
		return nil
	}

	prefix := ""
	if migrateDryRun {
		prefix = "would "
	}

	_, _ = fmt.Fprintf(w, "Sections %sadded:\n\n", prefix)

	headers := []string{"File", "Heading"}
	var rows [][]string
	for _, d := range result.Details {
		rows = append(rows, []string{d.File, d.Heading})
	}
	renderTable(w, headers, rows)

	_, _ = fmt.Fprintf(w, "\nSummary: %d section(s) %sadded to %d file(s)\n",
		result.SectionsAdded, prefix, result.FilesScaffolded)
	return nil
}

// --- Split operation ---

func runMigrateSplit(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

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
	records, err := index.Scan(ctx, absTarget, reg)
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
	result := migrate.BuildSplitStems(absTarget, existing, hierarchy)

	// Emit notes for auto-generated aggregates.
	for _, name := range result.GeneratedAggs {
		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "Note: auto-generated aggregate for '%s'\n", name)
	}

	collisions := splitCollisions(absTarget, result.Stems)

	if migrateDryRun {
		for i, sf := range result.Stems {
			if i > 0 {
				_, _ = fmt.Fprintln(cmd.OutOrStdout())
			}
			relPath, _ := filepath.Rel(absTarget, sf.Path)
			if relPath == "" {
				relPath = sf.Path
			}
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "# --- %s ---\n", relPath)
			if collisions[sf.Path] {
				_, _ = fmt.Fprintln(cmd.OutOrStdout(), "# (existing file will be overwritten)")
			}
			_, _ = fmt.Fprint(cmd.OutOrStdout(), sf.Content)
		}
		return nil
	}

	// Pre-flight guard: refuse before any write so a split is all-or-nothing.
	if len(collisions) > 0 && !migrateForce {
		return errSplitCollisions(absTarget, collisions)
	}

	// Write all .stem files.
	for _, sf := range result.Stems {
		if writeErr := os.WriteFile(sf.Path, []byte(sf.Content), 0644); writeErr != nil {
			return fmt.Errorf("writing %s: %w", sf.Path, writeErr)
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(),
		"Split .stem into %d files across %d levels (derive/aggregate/links preserved at root)\n",
		len(result.Stems), len(hierarchy.Levels))
	return nil
}

// splitCollisions stats every child .stem target and reports which already exist.
// The root .stem at <absTarget>/.stem is the input being rewritten in place, so it
// is never a collision. The root is matched by path rather than by index so the
// check stays correct if BuildSplitStems ever reorders its output.
func splitCollisions(absTarget string, stems []migrate.StemOutput) map[string]bool {
	rootPath := filepath.Join(absTarget, ".stem")
	found := make(map[string]bool)
	for _, sf := range stems {
		if sf.Path == rootPath {
			continue
		}
		if _, err := os.Stat(sf.Path); err == nil {
			found[sf.Path] = true
		}
	}
	return found
}

// errSplitCollisions renders the refusal, naming every colliding target so the
// user sees the full scope in one error instead of discovering them one at a time.
func errSplitCollisions(absTarget string, collisions map[string]bool) error {
	rels := make([]string, 0, len(collisions))
	for path := range collisions {
		rel, err := filepath.Rel(absTarget, path)
		if err != nil {
			rel = path
		}
		rels = append(rels, rel)
	}
	sort.Strings(rels)

	if len(rels) == 1 {
		return fmt.Errorf("%s already exists (use --force to overwrite)", rels[0])
	}
	return fmt.Errorf("cannot split: %s already exist (use --force to overwrite)", strings.Join(rels, ", "))
}
