package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

// FixResult represents the fix outcome for a single file.
type FixResult struct {
	Path            string   `json:"path"`
	Fixed           bool     `json:"fixed"`
	FieldsAdded     int      `json:"fields_added"`
	ValuesCorrected int      `json:"values_corrected"`
	Changes         []string `json:"changes"`
}

// BatchFixResult is the versioned JSON output for multi-file fix.
type BatchFixResult struct {
	Version int          `json:"version"`
	Kind    string       `json:"kind"`
	Results []*FixResult `json:"results"`
	Summary FixSummary   `json:"summary"`
}

// FixSummary holds aggregate counts for batch fix.
type FixSummary struct {
	Total   int `json:"total"`
	Fixed   int `json:"fixed"`
	Skipped int `json:"skipped"`
}

var (
	fixDryRun bool
	fixAll    bool
)

var fixCmd = &cobra.Command{
	Use:   "fix [file...]",
	Short: "Auto-repair validation errors",
	Long:  "Fix validation errors by adding missing required fields\nand correcting invalid enum values.",
	Args: func(cmd *cobra.Command, args []string) error {
		if fixAll {
			return nil
		}
		if len(args) == 0 {
			return fmt.Errorf("specify file(s) to fix or use --all")
		}
		return nil
	},
	RunE: runFix,
}

func init() {
	fixCmd.Flags().BoolVar(&fixDryRun, "dry-run", false, "show proposed changes without modifying files")
	fixCmd.Flags().BoolVar(&fixAll, "all", false, "fix all files in scope from current directory")
	rootCmd.AddCommand(fixCmd)
}

func runFix(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	if fixAll {
		return runFixAll(ctx, cmd, args)
	}

	reg := extract.NewRegistry()
	totalAdded := 0
	totalCorrected := 0

	for _, file := range args {
		absPath, err := filepath.Abs(file)
		if err != nil {
			return fmt.Errorf("resolving path %s: %w", file, err)
		}

		content, err := os.ReadFile(absPath)
		if err != nil {
			return fmt.Errorf("reading %s: %w", file, err)
		}

		ext := reg.ForFile(absPath, "")
		if ext == nil {
			return fmt.Errorf("no extractor for %s", file)
		}

		record, err := ext.Extract(file, content)
		if err != nil {
			return fmt.Errorf("extracting %s: %w", file, err)
		}

		// Resolve effective stem (with levels expansion for hierarchical schemas)
		dir := filepath.Dir(absPath)
		effective, err := rules.ResolveForRecord(dir, file)
		if err != nil {
			return fmt.Errorf("resolving .stem for %s: %w", file, err)
		}

		// Validate
		errs := rules.Validate(ctx, record, effective)

		if len(errs) == 0 {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: no errors to fix\n", file)
			continue
		}

		// Apply fixes to frontmatter
		added, corrected := fix.ApplyFixes(ctx, record, effective, errs)

		if fixDryRun {
			for _, a := range added {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: would add %s\n", file, a)
			}
			for _, c := range corrected {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: would correct %s\n", file, c)
			}
			totalAdded += len(added)
			totalCorrected += len(corrected)
			continue
		}

		// Rewrite file
		newContent := fix.RewriteFrontmatter(string(content), record.Frontmatter)
		if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil {
			return fmt.Errorf("writing %s: %w", file, err)
		}

		totalAdded += len(added)
		totalCorrected += len(corrected)

		for _, a := range added {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: added %s\n", file, a)
		}
		for _, c := range corrected {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: corrected %s\n", file, c)
		}
	}

	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nFixed: %d fields added, %d values corrected\n", totalAdded, totalCorrected)
	return nil
}

func runFixAll(ctx context.Context, cmd *cobra.Command, args []string) error {
	scanRoot := "."
	if len(args) > 0 {
		scanRoot = args[0]
	}
	root, err := filepath.Abs(scanRoot)
	if err != nil {
		return err
	}

	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	// First pass: collect all records, their effective stems, and errors.
	allErrs := make(map[string][]rules.ValidationError)
	effectiveStems := make(map[string]*rules.StemFile)

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			continue
		}
		effectiveStems[rec.Path] = effective

		errs := rules.Validate(ctx, rec, effective)

		if len(errs) > 0 {
			allErrs[rec.Path] = errs
		}
	}

	// Use the effective stem with the richest schema (most fields defined).
	var effective *rules.StemFile
	for _, s := range effectiveStems {
		if effective == nil || len(s.Schema) > len(effective.Schema) {
			effective = s
		}
	}

	report := proposal.Analyze(records, effective, allErrs)
	appendAggregateProposals(report, root, records, effective)
	appendStemHealthProposals(report, ctx, root)

	// In dry-run mode, use proposal engine for richer output.
	if fixDryRun {
		if outputFormat == "table" {
			return renderProposalTable(cmd, report)
		}
		return outputJSON(cmd, report, false)
	}

	// Apply proposals if any.
	if len(report.Proposals) > 0 {
		if err := fix.ApplyProposals(ctx, report, root, records); err != nil {
			return err
		}
	}

	// Convert proposals to BatchFixResult for output compatibility.
	results := proposalsToFixResults(report, records)
	batch := newBatchFixResult(results)

	if outputFormat == "table" {
		return renderFixTable(cmd, batch)
	}
	return outputJSON(cmd, batch, false)
}

// appendStemHealthProposals runs stem-health checks and appends remove_stem_field proposals.
func appendStemHealthProposals(report *proposal.Report, ctx context.Context, root string) {
	stemHealth, err := rules.ValidateStemHealth(ctx, root)
	if err != nil {
		return
	}
	stemProposals := proposal.DetectRemoveStemField(stemHealth.Checks)
	if len(stemProposals) > 0 {
		report.Proposals = append(report.Proposals, stemProposals...)
		report.Summary.RemoveStemField += len(stemProposals)
		report.Summary.Total += len(stemProposals)
	}
}

// appendAggregateProposals detects missing aggregate expressions and appends proposals to the report.
func appendAggregateProposals(report *proposal.Report, root string, records []*extract.Record, effective *rules.StemFile) {
	aggProposals := proposal.DetectMissingAggregates(root, records, effective)
	if len(aggProposals) > 0 {
		report.Proposals = append(report.Proposals, aggProposals...)
		report.Summary.AddAggregate += len(aggProposals)
		report.Summary.Total += len(aggProposals)
	}
}

func proposalsToFixResults(report *proposal.Report, records []*extract.Record) []*FixResult {
	// Group proposals by path.
	pathProposals := make(map[string][]proposal.Proposal)
	for _, p := range report.Proposals {
		for _, path := range p.Paths {
			pathProposals[path] = append(pathProposals[path], p)
		}
	}

	var results []*FixResult
	for _, rec := range records {
		proposals, hasProposals := pathProposals[rec.Path]
		if !hasProposals {
			results = append(results, &FixResult{
				Path: rec.Path, Fixed: false, Changes: []string{},
			})
			continue
		}

		var changes []string
		fieldsAdded := 0
		valuesCorrected := 0
		for _, p := range proposals {
			switch p.Type {
			case proposal.AddField, proposal.ExtractBody, proposal.InferFromChildren, proposal.InferFromSiblings:
				fieldsAdded++
				changes = append(changes, fmt.Sprintf("add %s=%q", p.Field, p.Value))
			case proposal.CorrectValue, proposal.MigrateValue, proposal.CorrectLink, proposal.CorrectOutlier:
				valuesCorrected++
				changes = append(changes, fmt.Sprintf("correct %s: %q -> %q", p.Field, p.From, p.To))
			case proposal.ExtendEnum:
				changes = append(changes, fmt.Sprintf("extend enum %s += %q", p.Field, p.Value))
			case proposal.RemoveStemField:
				valuesCorrected++
				changes = append(changes, fmt.Sprintf("remove %s from .stem schema", p.Field))
			}
		}

		results = append(results, &FixResult{
			Path:            rec.Path,
			Fixed:           true,
			FieldsAdded:     fieldsAdded,
			ValuesCorrected: valuesCorrected,
			Changes:         changes,
		})
	}

	// Add results for .stem file proposals not covered by record paths.
	stemResults := make(map[string]*FixResult)
	for _, p := range report.Proposals {
		if p.Type != proposal.RemoveStemField {
			continue
		}
		for _, path := range p.Paths {
			sr, ok := stemResults[path]
			if !ok {
				sr = &FixResult{Path: path, Fixed: true, Changes: []string{}}
				stemResults[path] = sr
			}
			sr.ValuesCorrected++
			sr.Changes = append(sr.Changes, fmt.Sprintf("remove %s from .stem schema", p.Field))
		}
	}
	for _, sr := range stemResults {
		results = append(results, sr)
	}

	return results
}

func newBatchFixResult(results []*FixResult) *BatchFixResult {
	summary := FixSummary{Total: len(results)}
	for _, r := range results {
		if r.Fixed || r.FieldsAdded > 0 || r.ValuesCorrected > 0 {
			summary.Fixed++
		} else {
			summary.Skipped++
		}
	}
	return &BatchFixResult{
		Version: 1,
		Kind:    "rootline/fix-batch",
		Results: results,
		Summary: summary,
	}
}

func renderProposalTable(cmd *cobra.Command, report *proposal.Report) error {
	headers := []string{"Type", "Field", "Description", "Files"}
	var rows [][]string
	for _, p := range report.Proposals {
		files := strings.Join(p.Paths, ", ")
		rows = append(rows, []string{string(p.Type), p.Field, p.Description, files})
	}
	if len(rows) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No proposals generated (0 fixable errors)")
		return nil
	}
	renderTable(cmd.OutOrStdout(), headers, rows)
	return nil
}

func renderFixTable(cmd *cobra.Command, batch *BatchFixResult) error {
	headers := []string{"File", "Fixed", "Changes"}
	var rows [][]string
	for _, r := range batch.Results {
		fixed := "no"
		if r.Fixed {
			fixed = "yes"
		}
		changesStr := ""
		if len(r.Changes) > 0 {
			changesStr = strings.Join(r.Changes, "; ")
		}
		rows = append(rows, []string{r.Path, fixed, changesStr})
	}
	renderTable(cmd.OutOrStdout(), headers, rows)
	return nil
}
