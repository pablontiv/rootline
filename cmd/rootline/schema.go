package main

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fix"
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
	Patch         string  `json:"patch,omitempty"` // full YAML markup to be written
	PatchPreview  string  `json:"patch_preview"`
}

// SchemaProposalsReport is the top-level output for schema propose.
type SchemaProposalsReport struct {
	Version     int              `json:"version"`
	Kind        string           `json:"kind"`
	Path        string           `json:"path"`
	Root        string           `json:"root,omitempty"` // absolute path to scan root
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
	Version int    `json:"version"`
	Kind    string `json:"kind"`
	// Root is the directory the report's targets were resolved against, named
	// for the same reason repair apply names it: a root resolved from the wrong
	// rung produces a run that touches nothing and explains nothing.
	Root string `json:"root,omitempty"`
	// Complete is the run's own verdict on whether it carried through everything
	// it accepted: true exactly when Errors is empty, which is also exactly when
	// the command exits 0. It exists so a stored report artifact, read long after
	// the exit status is gone, does not require its consumer to re-implement the
	// rule. Rejections and skips do not make a run incomplete.
	Complete          bool                         `json:"complete"`
	Applied           []string                     `json:"applied"`
	Skipped           []string                     `json:"skipped"`
	Rejected          []string                     `json:"rejected,omitempty"`
	DryRun            bool                         `json:"dry_run,omitempty"`
	Errors            []string                     `json:"errors,omitempty"`
	ValidationSummary *ValidationSummary           `json:"validation_summary,omitempty"`
	StemHealth        []rules.StemHealthDiagnostic `json:"stem_health"`

	// ResolvedTargets is populated in dry-run only, where the caller cannot
	// inspect the outcome on disk and needs to see where each .stem would land.
	ResolvedTargets *fix.ResolvedTargetsBreakdown `json:"resolved_targets,omitempty"`
}

func newSchemaApplyResult(root string, dryRun bool) *SchemaApplyResult {
	return &SchemaApplyResult{
		Version:    1,
		Kind:       "rootline/schema-apply",
		Root:       root,
		DryRun:     dryRun,
		Applied:    []string{},
		Skipped:    []string{},
		Rejected:   []string{},
		Errors:     []string{},
		StemHealth: []rules.StemHealthDiagnostic{},
	}
}

// seal records the run's completeness verdict once every phase has had its say,
// so Complete can never disagree with the fields it summarizes. schema apply
// performs no post-validation rollback, so Errors is the whole story here.
func (r *SchemaApplyResult) seal() {
	r.Complete = len(r.Errors) == 0
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
	schemaApplyForce  bool
	schemaApplyRoot   string
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
	schemaApplyCmd.Flags().BoolVar(&schemaApplyForce, "force", false, "overwrite existing .stem files")
	schemaApplyCmd.Flags().StringVar(&schemaApplyRoot, "root", "", "absolute path to scan root (overrides report root)")
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
	if err := ensureRecordsResolve(ctx, records, root); err != nil {
		return fmt.Errorf("resolving .stem: %w", err)
	}

	if len(records) == 0 {
		// No records found, emit empty report
		report := &SchemaProposalsReport{
			Version:     1,
			Kind:        "rootline/schema-proposals",
			Path:        scanRoot,
			Root:        root,
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
	if err := derive.DeriveAllSimple(ctx, records, root); err != nil {
		return fmt.Errorf("deriving records: %w", err)
	}
	if err := derive.EnrichBuiltinsSimple(ctx, records, root); err != nil {
		return fmt.Errorf("enriching records: %w", err)
	}
	if err := derive.AggregateAllSimple(ctx, records, root); err != nil {
		return fmt.Errorf("aggregating records: %w", err)
	}

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
		Root:        root,
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

	// Determine the root directory using the four-level precedence chain.
	scanRoot, err := resolveReportRoot(schemaApplyRoot, report.Root, report.Path, schemaApplyReport)
	if err != nil {
		return fmt.Errorf("resolving report root: %w", err)
	}

	result := newSchemaApplyResult(scanRoot, schemaApplyDryRun)

	resolved := &fix.ResolvedTargetsBreakdown{
		Accepted: map[string]string{},
		Rejected: map[string]string{},
	}

	if err := preflightSchemaApplyRecords(ctx, scanRoot); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("resolving .stem: %v", err))
		result.seal()
		if outputFormat == "table" {
			if err := renderSchemaApplyTable(cmd, result); err != nil {
				return err
			}
		} else if err := outputJSON(cmd, result, false); err != nil {
			return err
		}
		return applyExitError(len(result.Errors), 0)
	}

	plan := planSchemaProposalApply(report.Proposals, scanRoot, schemaApplyForce, result, resolved)
	if schemaApplyDryRun {
		result.ResolvedTargets = resolved
	}
	if len(result.Errors) > 0 {
		result.seal()
		if outputFormat == "table" {
			if err := renderSchemaApplyTable(cmd, result); err != nil {
				return err
			}
		} else if err := outputJSON(cmd, result, false); err != nil {
			return err
		}
		return applyExitError(len(result.Errors), 0)
	}

	// Execute only after the whole report has been planned and prospectively
	// validated. Actual IO failures still use the existing no-rollback run-level
	// behavior below.
	for _, op := range plan {
		if !schemaApplyDryRun {
			if err := os.WriteFile(op.target, []byte(op.patch), 0o644); err != nil {
				result.Errors = append(result.Errors, fmt.Sprintf("write %s: %v", op.reportTarget, err))
			} else {
				result.Applied = append(result.Applied, fmt.Sprintf("%s: %s", op.action, op.target))
			}
		} else {
			result.Applied = append(result.Applied, fmt.Sprintf("%s: %s", op.action, op.target))
		}
	}

	// If not dry-run, run validation.
	if !schemaApplyDryRun {
		validationResult, validationErr := runPostApplyValidation(ctx, scanRoot)
		if validationErr != nil {
			result.Errors = append(result.Errors, validationErr.Error())
		} else {
			result.ValidationSummary = validationResult
		}
	}

	result.seal()

	// Emit the payload first, then let the run's own outcome decide the exit
	// status. schema apply performs no post-validation rollback, so it has no
	// rolled_back[] to count.
	if outputFormat == "table" {
		if err := renderSchemaApplyTable(cmd, result); err != nil {
			return err
		}
	} else if err := outputJSON(cmd, result, false); err != nil {
		return err
	}

	return applyExitError(len(result.Errors), 0)
}

type schemaProposalApplyPlan struct {
	reportTarget string
	target       string
	patch        string
	action       string
}

type schemaApplyStatFunc func(string) (os.FileInfo, error)

type schemaApplyTargetObservation struct {
	exists  bool
	statErr error
}

func planSchemaProposalApply(proposals []SchemaProposal, scanRoot string, force bool, result *SchemaApplyResult, resolved *fix.ResolvedTargetsBreakdown) []schemaProposalApplyPlan {
	return planSchemaProposalApplyWithStat(proposals, scanRoot, force, result, resolved, os.Stat)
}

func planSchemaProposalApplyWithStat(proposals []SchemaProposal, scanRoot string, force bool, result *SchemaApplyResult, resolved *fix.ResolvedTargetsBreakdown, stat schemaApplyStatFunc) []schemaProposalApplyPlan {
	plan := []schemaProposalApplyPlan{}
	virtualTargets := map[string]schemaApplyTargetObservation{}
	for _, proposal := range proposals {
		// Skip proposals that require agent intervention.
		if proposal.RequiresAgent {
			result.Skipped = append(result.Skipped, fmt.Sprintf("%s: %s (requires agent)", proposal.ID, proposal.Target))
			continue
		}

		if proposal.Operation != "create_stem" {
			// Unknown operation: reject with descriptive message.
			result.Rejected = append(result.Rejected, fmt.Sprintf("%s: unknown operation \"%s\"", proposal.ID, proposal.Operation))
			continue
		}

		// `schema propose` emits absolute targets under the scan root, so the
		// propose->apply contract depends on absolute paths staying valid — they
		// are accepted, then confined.
		target, err := fix.ContainPath(scanRoot, proposal.Target, fix.PolicyAcceptAbsolute)
		if err != nil {
			// A target outside the scan root is a policy refusal, not a failed write.
			result.Rejected = append(result.Rejected, err.Error())
			resolved.Rejected[proposal.Target] = fix.ContainmentReason(err)
			continue
		}
		resolved.Accepted[proposal.Target] = target

		// Check for empty patch before write attempt.
		if proposal.Patch == "" {
			result.Errors = append(result.Errors, fmt.Sprintf("create_stem: %s: patch content required; re-run 'schema propose' to generate it", proposal.Target))
			continue
		}
		if err := validateProspectiveSchemaPatch(target, proposal.Patch); err != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("validating proposed .stem %s: %v", proposal.Target, err))
			continue
		}

		// Overwrite guard: replacing a governed .stem discards root, scope, enum
		// values, required markers, derive and aggregate in one write, so it is a
		// policy refusal unless the caller opted in with --force. The planner has to
		// model earlier accepted operations in this same report because execution is
		// intentionally deferred until all validation has completed.
		observation, ok := virtualTargets[target]
		if !ok {
			observation = schemaApplyTargetObservationFromStat(stat(target))
			virtualTargets[target] = observation
		}
		if observation.statErr != nil {
			result.Errors = append(result.Errors, fmt.Sprintf("stat %s: %v", proposal.Target, observation.statErr))
			continue
		}
		targetExists := observation.exists
		if targetExists && !force {
			result.Rejected = append(result.Rejected,
				fmt.Sprintf(".stem already exists in %s (use --force to overwrite)", filepath.Dir(target)))
			continue
		}

		// Name the action after what it actually does. A dry run is the only chance
		// the caller has to notice that "create" means "replace".
		action := "create_stem"
		if targetExists {
			action = "overwrite_stem"
		}

		plan = append(plan, schemaProposalApplyPlan{
			reportTarget: proposal.Target,
			target:       target,
			patch:        proposal.Patch,
			action:       action,
		})
		virtualTargets[target] = schemaApplyTargetObservation{exists: true}
	}
	return plan
}

func schemaApplyTargetObservationFromStat(_ os.FileInfo, err error) schemaApplyTargetObservation {
	if err == nil {
		return schemaApplyTargetObservation{exists: true}
	}
	if os.IsNotExist(err) {
		return schemaApplyTargetObservation{exists: false}
	}
	return schemaApplyTargetObservation{statErr: err}
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

	// Same precedence chain as the proposals path and as repair apply, so all
	// three report-consuming paths answer "which directory?" identically and
	// --root reaches every one of them.
	root, err := resolveReportRoot(schemaApplyRoot, report.Root, report.Path, schemaApplyReport)
	if err != nil {
		return fmt.Errorf("resolving report root: %w", err)
	}

	result := newSchemaApplyResult(root, schemaApplyDryRun)

	if err := preflightSchemaApplyRecords(ctx, root); err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("resolving .stem: %v", err))
		result.seal()
		if outputFormat == "table" {
			if err := renderSchemaApplyTable(cmd, result); err != nil {
				return err
			}
		} else if err := outputJSON(cmd, result, false); err != nil {
			return err
		}
		return applyExitError(len(result.Errors), 0)
	}

	// Resolve closest .stem for the report path. A malformed governed overlay
	// is an apply result failure, not a bare command error: callers depend on
	// this existing envelope to learn why no report operation was performed.
	res, resolveErr := rules.Resolve(root, root)
	if resolveErr != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("resolving stems for %s: %v", report.Path, resolveErr))
		result.seal()
		if outputFormat == "table" {
			if err := renderSchemaApplyTable(cmd, result); err != nil {
				return err
			}
		} else if err := outputJSON(cmd, result, false); err != nil {
			return err
		}
		return applyExitError(len(result.Errors), 0)
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
	// Schema-modifying: enum_values, required_field, constant_field, field_type, untyped_field, sequence_incomplete,
	// required_section, optional_section.
	// Data-correction: migrate_value, correct_value, add_field (these go to repair apply, not here)
	// Routing filter: emits only schema-modifying types. Drift guard in apply.go:default catches divergence.
	var schemaInferences []infer.ReportInference
	for _, cat := range report.Categories {
		for _, inf := range cat.Inferences {
			switch inf.Type {
			case "enum_values", "required_field", "constant_field", "field_type", "untyped_field", "sequence_incomplete", "required_section", "optional_section":
				schemaInferences = append(schemaInferences, inf)
			}
		}
	}

	// Apply schema modifications to .stem
	schemaResult, err := infer.ApplySchemaInferences(stemPath, schemaInferences, schemaApplyDryRun)
	if err != nil {
		result.Errors = append(result.Errors, fmt.Sprintf("applying schema: %v", err))
		result.seal()
		if outputFormat == "table" {
			if err := renderSchemaApplyTable(cmd, result); err != nil {
				return err
			}
		} else if err := outputJSON(cmd, result, false); err != nil {
			return err
		}
		return applyExitError(len(result.Errors), 0)
	}
	result.Applied = schemaResult.Applied
	result.Skipped = schemaResult.Skipped
	result.Rejected = schemaResult.Rejected

	// If not dry-run, run validation.
	if !schemaApplyDryRun {
		validationResult, validationErr := runPostApplyValidation(ctx, root)
		if validationErr != nil {
			result.Errors = append(result.Errors, validationErr.Error())
		} else {
			result.ValidationSummary = validationResult
		}
	}

	// Emit the payload first, then let the run's own outcome decide the exit
	// status. The analyze path writes through ApplySchemaInferences, which has
	// no rollback of its own, so there is no rolled_back[] to count.
	result.seal()

	if outputFormat == "table" {
		if err := renderSchemaApplyTable(cmd, result); err != nil {
			return err
		}
	} else if err := outputJSON(cmd, result, false); err != nil {
		return err
	}

	return applyExitError(len(result.Errors), 0)
}

// preflightSchemaApplyRecords resolves the actual record population before schema
// apply writes or publishes a dry-run result. A root-directory resolution cannot
// exercise match overlays that only apply to record path components.
func preflightSchemaApplyRecords(ctx context.Context, root string) error {
	reg := extract.NewRegistry()
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(stemScopeResolver()), index.AllowUngoverned())
	if err != nil {
		if errors.Is(err, rules.ErrNoSchemaFound) {
			return nil
		}
		return fmt.Errorf("post-apply validation scan of %s: %w", root, err)
	}
	if err := ensureRecordsResolve(ctx, records, root); err != nil {
		return err
	}
	return nil
}

// validateProspectiveSchemaPatch verifies a typed create_stem operation in
// memory. It uses the declaration owner rather than a temporary file or a
// second resolver, so an invalid overlay cannot be written then normalized by
// a later read path.
func validateProspectiveSchemaPatch(path, content string) error {
	stem, err := rules.ParseStem(path, []byte(content))
	if err != nil {
		return err
	}
	names := make([]string, 0, len(stem.Schema))
	for name := range stem.Schema {
		names = append(names, name)
	}
	sort.Strings(names)
	for _, name := range names {
		if issues := rules.ValidateFieldDeclaration(name, stem.Schema[name]); len(issues) > 0 {
			return fmt.Errorf("field %q: %s", name, issues[0].Message)
		}
	}
	return nil
}

// runPostApplyValidation runs validate --all on the root and returns a summary.
//
// A failed scan returns an error rather than an empty summary. An all-zero
// ValidationSummary is indistinguishable from a clean run, so swallowing the
// scan error here would report a green result for a root that was never read.
func runPostApplyValidation(ctx context.Context, root string) (*ValidationSummary, error) {
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
		return nil, fmt.Errorf("post-apply validation scan of %s: %w", root, err)
	}
	if err := ensureRecordsResolve(ctx, records, root); err != nil {
		return nil, fmt.Errorf("resolving .stem: %w", err)
	}

	if err := derive.DeriveAllSimple(ctx, records, root); err != nil {
		return nil, fmt.Errorf("deriving records: %w", err)
	}
	if err := derive.EnrichBuiltinsSimple(ctx, records, root); err != nil {
		return nil, fmt.Errorf("enriching records: %w", err)
	}
	if err := derive.AggregateAllSimple(ctx, records, root); err != nil {
		return nil, fmt.Errorf("aggregating records: %w", err)
	}

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
	}, nil
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
			yaml, err := stemFileToYAML(rootStem, root)
			if err != nil {
				return nil, fmt.Errorf("serializing hierarchical schema: %w", err)
			}
			proposal := SchemaProposal{
				ID:            "bootstrap-hierarchical",
				Operation:     "create_stem",
				Target:        stemPath,
				Confidence:    0.9,
				RequiresAgent: false,
				Patch:         yaml,
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
		yaml, err := stemFileToYAML(stemFile, root)
		if err != nil {
			return nil, fmt.Errorf("serializing flat schema: %w", err)
		}
		proposal := SchemaProposal{
			ID:            "bootstrap-flat",
			Operation:     "create_stem",
			Target:        stemPath,
			Confidence:    0.85,
			RequiresAgent: false,
			Patch:         yaml,
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
		if sf.Extract != "" && sf.Type == "string" {
			if source, err := extract.ParseBodySource(sf.Extract); err == nil && source.Kind == extract.BodySourceSection {
				if canonical, err := extract.CanonicalSectionSource(source.Heading); err == nil && canonical == sf.Extract {
					infType := "optional_section"
					if sf.Required {
						infType = "required_section"
					}
					inferences = append(inferences, infer.Inference{
						Type:            infType,
						Field:           fieldName,
						SourceDirective: sf.Extract,
					})
					continue
				}
			}
		}
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

	if len(result.Rejected) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Rejected (policy):")
		for _, r := range result.Rejected {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  - %s\n", r)
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

	if len(result.Applied) == 0 && len(result.Skipped) == 0 && len(result.Rejected) == 0 && len(result.Errors) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No modifications to apply.")
	}

	return nil
}
