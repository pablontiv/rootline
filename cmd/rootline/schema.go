package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var (
	schemaProposeIncremental bool
)

// SchemaProposal represents a single schema operation proposal.
type SchemaProposal struct {
	ID            string  `json:"id"`
	Operation     string  `json:"operation"` // "create_stem", "update_stem", etc.
	Target        string  `json:"target"`    // path to .stem file
	Confidence    float64 `json:"confidence"`
	RequiresAgent bool    `json:"requires_agent"`
	PatchPreview  string  `json:"patch_preview"`
}

// SchemaProposalsReport is the top-level output for schema propose.
type SchemaProposalsReport struct {
	Version     int              `json:"version"`
	Kind        string           `json:"kind"`
	Path        string           `json:"path"`
	Incremental bool             `json:"incremental"`
	Proposals   []SchemaProposal `json:"proposals"`
	Summary     ProposalsSummary `json:"summary"`
}

// ProposalsSummary provides aggregate counts for the schema proposals report.
type ProposalsSummary struct {
	Total          int `json:"total"`
	RequiresAgent  int `json:"requires_agent"`
	EngineResolved int `json:"engine_resolved"`
}

// SchemaApplyResult represents the result of applying schema proposals.
type SchemaApplyResult struct {
	Version           int                `json:"version"`
	Kind              string             `json:"kind"`
	Applied           []string           `json:"applied"`
	Skipped           []string           `json:"skipped"`
	DryRun            bool               `json:"dry_run,omitempty"`
	Errors            []string           `json:"errors,omitempty"`
	ValidationSummary *ValidationSummary `json:"validation_summary,omitempty"`
}

// ValidationSummary contains validation results after schema apply.
type ValidationSummary struct {
	TotalFiles   int `json:"total_files"`
	ValidFiles   int `json:"valid_files"`
	InvalidFiles int `json:"invalid_files"`
	TotalErrors  int `json:"total_errors"`
}

var schemaCmd = &cobra.Command{
	Use:   "schema",
	Short: "Schema operations and proposals",
	Long:  "Manage schema operations and generate schema proposals.",
}

var schemaProposeCmd = &cobra.Command{
	Use:   "propose [directory]",
	Short: "Propose schema changes without writing files",
	Long: `Analyze documents and propose schema changes as versioned JSON.
Use --incremental to only show proposals not already covered by existing .stem files.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runSchemaPropose,
}

var (
	schemaApplyReport string
	schemaApplyDryRun bool
)

var schemaApplyCmd = &cobra.Command{
	Use:   "apply --report <proposals.json> [--dry-run]",
	Short: "Apply schema proposals to .stem files",
	Long: `Read a schema proposals report and apply schema-modifying proposals to .stem files.
Skips proposals with requires_agent: true.
After apply (non-dry-run), runs validation and includes results in output.`,
	Args: cobra.NoArgs,
	RunE: runSchemaApply,
}

func init() {
	schemaProposeCmd.Flags().BoolVar(&schemaProposeIncremental, "incremental", false, "only show proposals not covered by existing .stem")
	schemaCmd.AddCommand(schemaProposeCmd)

	schemaApplyCmd.Flags().StringVar(&schemaApplyReport, "report", "", "path to schema proposals report JSON file (required)")
	_ = schemaApplyCmd.MarkFlagRequired("report")
	schemaApplyCmd.Flags().BoolVar(&schemaApplyDryRun, "dry-run", false, "show changes without applying them")
	schemaCmd.AddCommand(schemaApplyCmd)

	rootCmd.AddCommand(schemaCmd)
}

func runSchemaPropose(cmd *cobra.Command, args []string) error {
	scanRoot := "."
	if len(args) > 0 {
		scanRoot = args[0]
	}

	root, err := filepath.Abs(scanRoot)
	if err != nil {
		return err
	}

	ctx := cmd.Context()

	// Index and extract records.
	reg := extract.NewASTRegistry()
	resolver := func(dir string) (*rules.StemFile, error) {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil, err
		}
		return rules.MergeStemFiles(entries), nil
	}

	// Bootstrap scan: this command derives a schema from documents that may
	// not have one yet, so a missing schema must not stop it.
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver), index.AllowUngoverned())
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	if len(records) == 0 {
		// No records found, emit empty report
		report := &SchemaProposalsReport{
			Version:     1,
			Kind:        "rootline/schema-proposals",
			Path:        scanRoot,
			Incremental: schemaProposeIncremental,
			Proposals:   []SchemaProposal{},
			Summary:     ProposalsSummary{},
		}
		if outputFormat == "table" {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No records found in %s\n", scanRoot)
			return nil
		}
		return outputJSON(cmd, report, false)
	}

	// Derive and aggregate
	derive.DeriveAllSimple(ctx, records, root)
	derive.EnrichBuiltinsSimple(ctx, records, root)
	derive.AggregateAllSimple(ctx, records, root)

	// Check for existing stem
	var existingStem *rules.StemFile
	entries, walkErr := rules.WalkUp(root)
	if walkErr == nil && len(entries) > 0 {
		existingStem = rules.MergeStemFiles(entries)
	}

	// Create per-scope resolver for incremental filtering
	var resolve infer.StemResolver
	if schemaProposeIncremental {
		resolve = infer.DefaultStemResolver()
	}

	// Generate schema proposals
	proposals, err := generateSchemaProposals(ctx, root, records, existingStem, schemaProposeIncremental, resolve)
	if err != nil {
		return fmt.Errorf("generating proposals: %w", err)
	}

	// Build report
	report := &SchemaProposalsReport{
		Version:     1,
		Kind:        "rootline/schema-proposals",
		Path:        scanRoot,
		Incremental: schemaProposeIncremental,
		Proposals:   proposals,
	}

	// Calculate summary
	agentReq := 0
	for _, p := range proposals {
		if p.RequiresAgent {
			agentReq++
		}
	}
	report.Summary = ProposalsSummary{
		Total:          len(proposals),
		RequiresAgent:  agentReq,
		EngineResolved: len(proposals) - agentReq,
	}

	if outputFormat == "table" {
		return renderSchemaProposeTable(cmd, report)
	}
	return outputJSON(cmd, report, false)
}

func runSchemaApply(cmd *cobra.Command, args []string) error {
	ctx := cmd.Context()

	// Read report data
	data, err := os.ReadFile(schemaApplyReport)
	if err != nil {
		return fmt.Errorf("reading report file: %w", err)
	}

	// Probe report kind to dispatch
	var probe struct {
		Kind string `json:"kind"`
	}
	_ = json.Unmarshal(data, &probe)
	if probe.Kind == "analyze" || probe.Kind == "rootline/analyze" {
		return runSchemaApplyFromAnalyze(cmd, data)
	}

	// Parse as SchemaProposalsReport for the proposals path
	var report SchemaProposalsReport
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("parsing report: %w", err)
	}

	// Validate report version and kind.
	if report.Version != 1 {
		return fmt.Errorf("unsupported report version: %d (expected 1)", report.Version)
	}
	if report.Kind != "rootline/schema-proposals" {
		return fmt.Errorf("wrong report kind: %s (expected rootline/schema-proposals)", report.Kind)
	}

	result := &SchemaApplyResult{
		Version: 1,
		Kind:    "rootline/schema-apply",
		DryRun:  schemaApplyDryRun,
		Applied: []string{},
		Skipped: []string{},
		Errors:  []string{},
	}

	// Resolve absolute path from report.
	scanRoot := report.Path
	if !filepath.IsAbs(scanRoot) {
		absRoot, err := filepath.Abs(scanRoot)
		if err != nil {
			return fmt.Errorf("resolving path: %w", err)
		}
		scanRoot = absRoot
	}

	// Process each proposal.
	for _, proposal := range report.Proposals {
		// Skip proposals that require agent intervention.
		if proposal.RequiresAgent {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s: %s (requires agent)", proposal.ID, proposal.Target))
			continue
		}

		// Process based on operation type.
		if proposal.Operation == "create_stem" {
			// Create a new .stem file at target.
			if err := infer.ScaffoldSchema(filepath.Dir(proposal.Target), schemaApplyDryRun); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("scaffold %s: %v", proposal.Target, err))
			} else {
				result.Applied = append(result.Applied, fmt.Sprintf("create_stem: %s", proposal.Target))
			}
		}
	}

	// If not dry-run, run validation.
	if !schemaApplyDryRun {
		validationResult := runPostApplyValidation(ctx, scanRoot)
		result.ValidationSummary = validationResult
	}

	// Output result.
	if outputFormat == "table" {
		return renderSchemaApplyTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

// runSchemaApplyFromAnalyze processes an analyze report and applies schema-modifying inferences to .stem files.
// Mirrors apply.go's schema half: resolves closest .stem, filters schema-modifying inferences, calls ApplySchemaInferences.
func runSchemaApplyFromAnalyze(cmd *cobra.Command, data []byte) error {
	ctx := cmd.Context()

	var report infer.AnalyzeReport
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("parsing analyze report: %w", err)
	}

	// Validate report version and kind.
	if report.Version != 1 {
		return fmt.Errorf("unsupported report version: %d (expected 1)", report.Version)
	}
	if report.Kind != "analyze" && report.Kind != "rootline/analyze" {
		return fmt.Errorf("wrong report kind: %s (expected analyze or rootline/analyze)", report.Kind)
	}

	// Resolve root path from report path.
	root, err := filepath.Abs(report.Path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Resolve closest .stem for the report path.
	res, resolveErr := rules.Resolve(root, root)
	if resolveErr != nil {
		return fmt.Errorf("resolving stems for %s: %w", report.Path, resolveErr)
	}

	if len(res.Chain) == 0 {
		return fmt.Errorf("no .stem found for %s", report.Path)
	}

	// Use the closest (leaf-most) stem file for schema modifications.
	closestStem := res.ClosestStem()
	if closestStem == nil {
		return fmt.Errorf("no stem available for %s", report.Path)
	}
	stemPath := closestStem.Path

	// Separate schema-modifying inferences from data-correction inferences.
	// Schema-modifying: enum_values, required_field, constant_field, field_type, untyped_field, sequence_incomplete
	// Data-correction: migrate_value, correct_value, add_field (these go to repair apply, not here)
	var schemaInferences []infer.ReportInference
	for _, cat := range report.Categories {
		for _, inf := range cat.Inferences {
			switch inf.Type {
			case "enum_values", "required_field", "constant_field", "field_type", "untyped_field", "sequence_incomplete":
				schemaInferences = append(schemaInferences, inf)
			}
		}
	}

	// Apply schema modifications to .stem
	schemaResult, err := infer.ApplySchemaInferences(stemPath, schemaInferences, schemaApplyDryRun)
	if err != nil {
		return fmt.Errorf("applying schema: %w", err)
	}

	// Format result as SchemaApplyResult
	result := &SchemaApplyResult{
		Version: 1,
		Kind:    "rootline/schema-apply",
		DryRun:  schemaApplyDryRun,
		Applied: schemaResult.Applied,
		Skipped: schemaResult.Skipped,
		Errors:  []string{},
	}

	// If not dry-run, run validation.
	if !schemaApplyDryRun {
		validationResult := runPostApplyValidation(ctx, root)
		result.ValidationSummary = validationResult
	}

	// Output result.
	if outputFormat == "table" {
		return renderSchemaApplyTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

// runPostApplyValidation runs validate --all on the root and returns a summary.
func runPostApplyValidation(ctx context.Context, root string) *ValidationSummary {
	reg := extract.NewRegistry()
	resolver := func(dir string) (*rules.StemFile, error) {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil, err
		}
		return rules.MergeStemFiles(entries), nil
	}

	// Bootstrap scan: this command derives a schema from documents that may
	// not have one yet, so a missing schema must not stop it.
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver), index.AllowUngoverned())
	if err != nil {
		return &ValidationSummary{}
	}

	derive.DeriveAllSimple(ctx, records, root)
	derive.EnrichBuiltinsSimple(ctx, records, root)
	derive.AggregateAllSimple(ctx, records, root)

	validCount := 0
	invalidCount := 0
	totalErrors := 0

	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, resolveErr := rules.ResolveForRecord(dir, rec.Path)
		if resolveErr != nil {
			invalidCount++
			continue
		}
		errs := rules.Validate(ctx, rec, effective)
		if len(errs) > 0 {
			invalidCount++
			totalErrors += len(errs)
		} else {
			validCount++
		}
	}

	return &ValidationSummary{
		TotalFiles:   len(records),
		ValidFiles:   validCount,
		InvalidFiles: invalidCount,
		TotalErrors:  totalErrors,
	}
}

// generateSchemaProposals analyzes records and generates schema proposals.
// When incremental is true, uses per-scope stem resolution to filter proposals
// that are already covered by existing scopes.
func generateSchemaProposals(ctx context.Context, root string, records []*extract.Record, existingStem *rules.StemFile, incremental bool, resolve infer.StemResolver) ([]SchemaProposal, error) {
	var proposals []SchemaProposal

	// Try hierarchical detection first
	hierarchy := infer.AnalyzeHierarchy(records, root)

	stemPath := filepath.Join(root, ".stem")
	var generatedStem *rules.StemFile

	if hierarchy.Detected {
		// Hierarchical case
		opts := infer.DefaultInferOptions()
		stemMap, err := infer.GenerateHierarchicalSchema(ctx, root, records, opts)
		if err != nil {
			return nil, fmt.Errorf("generating hierarchical schema: %w", err)
		}

		for _, rootStem := range stemMap {
			generatedStem = rootStem
			yaml := stemFileToYAML(rootStem, root)
			proposal := SchemaProposal{
				ID:            "bootstrap-hierarchical",
				Operation:     "create_stem",
				Target:        stemPath,
				Confidence:    0.9,
				RequiresAgent: false,
				PatchPreview:  truncatePreview(yaml, 200),
			}
			proposals = append(proposals, proposal)
		}
	} else {
		// Flat case
		opts := infer.DefaultInferOptions()
		stemFile, err := infer.GenerateFlatSchema(ctx, root, records, opts)
		if err != nil {
			return nil, fmt.Errorf("generating flat schema: %w", err)
		}

		generatedStem = stemFile
		yaml := stemFileToYAML(stemFile, root)
		proposal := SchemaProposal{
			ID:            "bootstrap-flat",
			Operation:     "create_stem",
			Target:        stemPath,
			Confidence:    0.85,
			RequiresAgent: false,
			PatchPreview:  truncatePreview(yaml, 200),
		}
		proposals = append(proposals, proposal)
	}

	// If incremental mode, filter proposals using per-scope stems
	if incremental && resolve != nil && generatedStem != nil {
		// Convert schema fields to inferences for filtering
		inferences := schemaToInferences(generatedStem)
		if len(inferences) > 0 {
			// Filter inferences using per-scope resolver
			filtered := infer.FilterCoveredInferences(inferences, records, root, resolve)
			// If all inferences are covered, don't propose
			if len(filtered) == 0 {
				proposals = nil
			}
		}
	}

	return proposals, nil
}

// schemaToInferences converts schema fields to inference objects for filtering.
func schemaToInferences(stem *rules.StemFile) []infer.Inference {
	var inferences []infer.Inference
	for fieldName, sf := range stem.Schema {
		// Create inferences based on field properties
		if sf.Required {
			inferences = append(inferences, infer.Inference{
				Type:  "required_field",
				Field: fieldName,
			})
		}
		if sf.Type != "" {
			inferences = append(inferences, infer.Inference{
				Type:  "field_type",
				Field: fieldName,
				Value: sf.Type,
			})
		}
		if sf.Type == "enum" && len(sf.Values) > 0 {
			inferences = append(inferences, infer.Inference{
				Type:  "enum_values",
				Field: fieldName,
			})
		}
	}
	return inferences
}

// truncatePreview limits preview length for JSON output.
func truncatePreview(content string, maxLen int) string {
	if len(content) <= maxLen {
		return content
	}
	return content[:maxLen] + "..."
}

// renderSchemaProposeTable renders schema proposals in table format.
func renderSchemaProposeTable(cmd *cobra.Command, report *SchemaProposalsReport) error {
	if len(report.Proposals) == 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "No schema proposals generated for %s\n", report.Path)
		return nil
	}

	headers := []string{"ID", "Operation", "Target", "Confidence", "Requires Agent"}
	var rows [][]string
	for _, p := range report.Proposals {
		row := []string{
			p.ID,
			p.Operation,
			p.Target,
			fmt.Sprintf("%.2f", p.Confidence),
			fmt.Sprintf("%v", p.RequiresAgent),
		}
		rows = append(rows, row)
	}

	renderTable(cmd.OutOrStdout(), headers, rows)
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nSummary: %d total, %d requires agent, %d engine-resolved\n",
		report.Summary.Total, report.Summary.RequiresAgent, report.Summary.EngineResolved)
	return nil
}

// renderSchemaApplyTable renders schema apply results in table format.
func renderSchemaApplyTable(cmd *cobra.Command, result *SchemaApplyResult) error {
	if result.DryRun {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Dry run (no changes applied):")
	}

	if len(result.Applied) > 0 {
		label := "Applied:"
		if result.DryRun {
			label = "Would apply:"
		}
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), label)
		for _, a := range result.Applied {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  * %s\n", a)
		}
	}

	if len(result.Skipped) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Skipped (requires agent):")
		for _, s := range result.Skipped {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", s)
		}
	}

	if len(result.Errors) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Errors:")
		for _, e := range result.Errors {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  ! %s\n", e)
		}
	}

	if result.ValidationSummary != nil {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "\nValidation Summary:")
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Total files: %d\n", result.ValidationSummary.TotalFiles)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Valid: %d\n", result.ValidationSummary.ValidFiles)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Invalid: %d\n", result.ValidationSummary.InvalidFiles)
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  Total errors: %d\n", result.ValidationSummary.TotalErrors)
	}

	if len(result.Applied) == 0 && len(result.Skipped) == 0 && len(result.Errors) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No modifications to apply.")
	}

	return nil
}
