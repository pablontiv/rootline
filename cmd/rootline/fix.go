package main

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/derive"
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
	Version           int          `json:"version"`
	Kind              string       `json:"kind"`
	Complete          *bool        `json:"complete,omitempty"`
	Results           []*FixResult `json:"results"`
	Summary           FixSummary   `json:"summary"`
	Errors            []string     `json:"errors,omitempty"`
	SchemaSuggestions int          `json:"schema_suggestions,omitempty"`
	// SkippedProposals holds add_field proposals declined because the engine,
	// not the schema, chose their value. Carried so the operator can see the
	// field was left unfilled deliberately.
	SkippedProposals []proposal.Proposal `json:"skipped_proposals,omitempty"`

	// LinkFindings reports link problems fix will not repair. Rewriting a
	// link body on a fuzzy guess is a destructive edit outside fix's
	// data-repair contract, but staying silent about a defect validate does
	// report made a clean fix followed by a failing validate read as a bug.
	LinkFindings []proposal.LinkFinding `json:"link_findings,omitempty"`

	// TypeFindings reports representation mismatches fix cannot repair safely.
	// They are observations, not failed writes; validate owns corpus validity.
	TypeFindings []proposal.TypeFinding `json:"type_findings,omitempty"`
}

// FixSummary holds aggregate counts for batch fix.
type FixSummary struct {
	Total   int `json:"total"`
	Fixed   int `json:"fixed"`
	Skipped int `json:"skipped"`
}

var (
	fixDryRun      bool
	fixAll         bool
	fixNoPropagate bool
	fixFillMissing bool
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
	fixCmd.Flags().BoolVar(&fixNoPropagate, "no-propagate", false, "skip aggregate propagation proposals")
	fixCmd.Flags().BoolVar(&fixFillMissing, "fill-missing", false, "also write values the engine chose for required fields the schema gave no default: for")
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
		added, corrected, skipped := fix.ApplyFixes(ctx, record, effective, errs, fixFillMissing)

		for _, sk := range skipped {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "%s: skipped %s\n", file, sk)
		}

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
		if err := os.WriteFile(absPath, []byte(newContent), 0644); err != nil { //nolint:gosec // fix intentionally rewrites the user-selected validated document path
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
	resolver := stemScopeResolver()

	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}
	// First pass: resolve each record independently. Broken governed records are
	// classified in the batch envelope; valid independent records still proceed.
	allErrs := make(map[string][]rules.ValidationError)
	effectiveStems := make(map[string]*rules.StemFile)
	linkCache := rules.NewHeadingCache()
	var linkFindings []proposal.LinkFinding
	var validRecords []*extract.Record
	var resolutionErrors []string
	unresolvedPopulation := false

	for _, rec := range records {
		if err := ctx.Err(); err != nil {
			return emitFixResult(cmd, failedBatchFixResult(fmt.Sprintf("resolving .stem: %v", err)))
		}
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			unresolvedPopulation = true
			resolutionErrors = append(resolutionErrors, fmt.Sprintf("resolving .stem for %s: %v", rec.Path, resolveErr))
			continue
		}
		validRecords = append(validRecords, rec)
		effectiveStems[rec.Path] = effective
	}

	// Run derive pipeline for valid records only so aggregate values are available
	// without manufacturing cross-record facts from an unresolved corpus.
	if err := derive.DeriveAllSimple(ctx, validRecords, root); err != nil {
		return emitFixResult(cmd, failedBatchFixResult(fmt.Sprintf("deriving records: %v", err)))
	}
	if err := derive.EnrichBuiltinsSimple(ctx, validRecords, root); err != nil {
		return emitFixResult(cmd, failedBatchFixResult(fmt.Sprintf("enriching records: %v", err)))
	}
	if !unresolvedPopulation {
		if err := derive.AggregateAllSimple(ctx, validRecords, root); err != nil {
			return emitFixResult(cmd, failedBatchFixResult(fmt.Sprintf("aggregating records: %v", err)))
		}
	}

	for _, rec := range validRecords {
		absPath := filepath.Join(root, rec.Path)
		effective := effectiveStems[rec.Path]
		errs := rules.Validate(ctx, rec, effective)

		// Link checks never reached fix, so a record validate failed with
		// link_resolve came back from fix as a clean run — with no warning,
		// even though the error already carries a fuzzy suggestion.
		for _, e := range rules.CheckLinks(rec.Links, effective.Links, absPath, root, linkCache) {
			linkFindings = append(linkFindings, proposal.LinkFinding{
				Path: rec.Path, Rule: e.Rule, Message: e.Message, Suggestion: e.Suggestion,
			})
		}

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

	report := proposal.Analyze(validRecords, effective, allErrs)
	report.Path = scanRoot
	report.Root = root
	if !unresolvedPopulation {
		appendAggregateProposals(report, root, validRecords, effective)
		if !fixNoPropagate {
			appendPropagateProposals(report, validRecords, effective)
		}
	}
	appendStemHealthProposals(report, ctx, root)

	// Separate schema proposals from data proposals (for dry-run and apply both).
	// In dry-run, we show what would be applied vs. what would be skipped.
	// In normal apply, we only apply data proposals.
	separateSchemaAndDataProposals(report)

	report.LinkFindings = linkFindings

	// An unresolved record makes a dry-run proposal corpus incomplete and thus
	// unsafe to publish as a reusable rootline/proposals report. Reuse the
	// established batch failure protocol while retaining valid-record previews.
	if fixDryRun && len(resolutionErrors) > 0 {
		results := proposalsToFixResults(report, validRecords)
		for _, result := range results {
			result.Fixed = false
		}
		results = appendInvalidFixResults(results, records, resolutionErrors)
		batch := newBatchFixResultWithSuggestions(results, len(report.SchemaSuggestions))
		complete := false
		batch.Complete = &complete
		batch.Errors = append(batch.Errors, resolutionErrors...)
		batch.LinkFindings = linkFindings
		batch.TypeFindings = report.TypeFindings
		return emitFixResult(cmd, batch)
	}

	// In dry-run mode, use proposal engine for richer output.
	if fixDryRun {
		if outputFormat == "table" {
			if err := renderProposalTable(cmd, report); err != nil {
				return err
			}
			renderLinkFindings(cmd, report.LinkFindings)
			renderTypeFindings(cmd, report.TypeFindings)
			return nil
		}
		return outputJSON(cmd, report, false)
	}

	// Apply proposals if any.
	var skippedProposals []proposal.Proposal
	if len(report.Proposals) > 0 {
		var err error
		skippedProposals, err = fix.ApplyProposals(ctx, report, root, validRecords, fixFillMissing)
		if err != nil {
			return err
		}
	}

	// Convert proposals to BatchFixResult for output compatibility.
	results := proposalsToFixResults(report, validRecords)
	results = appendInvalidFixResults(results, records, resolutionErrors)
	batch := newBatchFixResultWithSuggestions(results, len(report.SchemaSuggestions))
	if len(resolutionErrors) > 0 {
		complete := false
		batch.Complete = &complete
		batch.Errors = append(batch.Errors, resolutionErrors...)
	}
	batch.SkippedProposals = skippedProposals
	batch.LinkFindings = linkFindings
	batch.TypeFindings = report.TypeFindings

	return emitFixResult(cmd, batch)
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

// separateSchemaAndDataProposals splits proposals into data-only (applied) and
// schema-only (suggestions). This allows dry-run to show both, and apply to
// skip schema proposals.
func separateSchemaAndDataProposals(report *proposal.Report) {
	var dataProposals []proposal.Proposal
	var schemaSuggestions []proposal.Proposal

	for _, p := range report.Proposals {
		if p.Surface() == proposal.SurfaceSchema {
			schemaSuggestions = append(schemaSuggestions, p)
		} else {
			dataProposals = append(dataProposals, p)
		}
	}

	report.Proposals = dataProposals
	report.SchemaSuggestions = schemaSuggestions
}

// appendPropagateProposals detects stale aggregate values in index files and appends proposals to the report.
func appendPropagateProposals(report *proposal.Report, records []*extract.Record, effective *rules.StemFile) {
	propProposals := proposal.DetectPropagateAggregate(records, effective)
	if len(propProposals) > 0 {
		report.Proposals = append(report.Proposals, propProposals...)
		report.Summary.PropagateAggregate += len(propProposals)
		report.Summary.Total += len(propProposals)
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
			case proposal.CorrectValue, proposal.MigrateValue, proposal.CorrectLink, proposal.CorrectOutlier, proposal.PropagateAggregate:
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

func appendInvalidFixResults(results []*FixResult, records []*extract.Record, resolutionErrors []string) []*FixResult {
	if len(resolutionErrors) == 0 {
		return results
	}
	already := make(map[string]bool, len(results))
	for _, result := range results {
		already[result.Path] = true
	}
	for _, rec := range records {
		if already[rec.Path] {
			continue
		}
		for _, msg := range resolutionErrors {
			if strings.Contains(msg, rec.Path) {
				results = append(results, &FixResult{Path: rec.Path, Fixed: false, Changes: []string{"skipped: schema resolution failed"}})
				break
			}
		}
	}
	return results
}

func newBatchFixResultWithSuggestions(results []*FixResult, schemaSuggestionsCount int) *BatchFixResult {
	summary := FixSummary{Total: len(results)}
	for _, r := range results {
		if r.Fixed || r.FieldsAdded > 0 || r.ValuesCorrected > 0 {
			summary.Fixed++
		} else {
			summary.Skipped++
		}
	}
	return &BatchFixResult{
		Version:           1,
		Kind:              "rootline/fix-batch",
		Results:           results,
		Summary:           summary,
		SchemaSuggestions: schemaSuggestionsCount,
	}
}

func failedBatchFixResult(message string) *BatchFixResult {
	complete := false
	return &BatchFixResult{
		Version:  1,
		Kind:     "rootline/fix-batch",
		Complete: &complete,
		Results:  []*FixResult{},
		Summary:  FixSummary{},
		Errors:   []string{message},
	}
}

func fixBatchFailed(batch *BatchFixResult) bool {
	return (batch.Complete != nil && !*batch.Complete) || len(batch.Errors) > 0
}

func emitFixResult(cmd *cobra.Command, batch *BatchFixResult) error {
	if outputFormat == "table" {
		if err := renderFixTable(cmd, batch); err != nil {
			return err
		}
		if fixBatchFailed(batch) {
			cmd.SilenceUsage = true
			cmd.SilenceErrors = true
			return ErrValidationFailed
		}
		return nil
	}
	return outputJSON(cmd, batch, fixBatchFailed(batch))
}

// renderLinkFindings prints the link problems fix reported but did not repair.
// Shared by the dry-run preview and the applied run so the preview cannot go
// quiet about something the real run reports.
func renderLinkFindings(cmd *cobra.Command, findings []proposal.LinkFinding) {
	if len(findings) == 0 {
		return
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Link findings: %d (reported, not repaired)\n", len(findings))
	for _, f := range findings {
		line := fmt.Sprintf("  %s: %s (%s)", f.Path, f.Message, f.Rule)
		if f.Suggestion != "" {
			line += fmt.Sprintf(" — did you mean %q?", f.Suggestion)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), line)
	}
}

// renderTypeFindings prints representation problems fix reported but did not repair.
// Shared by the dry-run preview and the applied run so operators can distinguish
// successful repair execution from remaining corpus validity issues.
func renderTypeFindings(cmd *cobra.Command, findings []proposal.TypeFinding) {
	if len(findings) == 0 {
		return
	}
	_, _ = fmt.Fprintln(cmd.OutOrStdout())
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Type findings: %d (reported, not repaired)\n", len(findings))
	for _, finding := range findings {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s: %s (%s)\n",
			finding.Path, finding.Field, finding.Message, finding.ActualRepresentation)
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
	} else {
		renderTable(cmd.OutOrStdout(), headers, rows)
	}

	// Show schema suggestions as a separate note
	if len(report.SchemaSuggestions) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Schema suggestions (not applied):\n")
		for _, p := range report.SchemaSuggestions {
			files := strings.Join(p.Paths, ", ")
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s: %s (%s)\n", p.Type, p.Field, files)
		}
	}

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

	if len(batch.SkippedProposals) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Note: %d required field(s) left unfilled; schema declares no default:.\n", len(batch.SkippedProposals))
		for _, sp := range batch.SkippedProposals {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %v: %s (engine would have written %q)\n", sp.Paths, sp.Field, sp.Value)
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Invalid on purpose. Pass --fill-missing to write them.")
	}

	// Note about schema suggestions if present
	if batch.SchemaSuggestions > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "Note: %d schema proposals were skipped (data-only repairs applied).\n", batch.SchemaSuggestions)
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "To apply schema changes, use 'rootline schema propose' or manually edit .stem files.")
	}
	renderLinkFindings(cmd, batch.LinkFindings)
	renderTypeFindings(cmd, batch.TypeFindings)

	if len(batch.Errors) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout())
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Errors")
		var rows [][]string
		for _, message := range batch.Errors {
			rows = append(rows, []string{message})
		}
		renderTable(cmd.OutOrStdout(), []string{"Message"}, rows)
	}

	return nil
}
