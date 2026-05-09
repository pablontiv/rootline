package main

import (
	"context"
	"encoding/json"
	"fmt"
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

func init() {
	schemaProposeCmd.Flags().BoolVar(&schemaProposeIncremental, "incremental", false, "only show proposals not covered by existing .stem")
	schemaCmd.AddCommand(schemaProposeCmd)
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
			fmt.Fprintf(cmd.OutOrStdout(), "No records found in %s\n", scanRoot)
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

	// Generate schema proposals
	proposals, err := generateSchemaProposals(ctx, root, records, existingStem, schemaProposeIncremental)
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

// generateSchemaProposals analyzes records and generates schema proposals.
func generateSchemaProposals(ctx context.Context, root string, records []*extract.Record, existingStem *rules.StemFile, incremental bool) ([]SchemaProposal, error) {
	var proposals []SchemaProposal

	// Try hierarchical detection first
	hierarchy := infer.AnalyzeHierarchy(records, root)

	stemPath := filepath.Join(root, ".stem")
	if hierarchy.Detected {
		// Hierarchical case
		opts := infer.DefaultInferOptions()
		stemMap, err := infer.GenerateHierarchicalSchema(ctx, root, records, opts)
		if err != nil {
			return nil, fmt.Errorf("generating hierarchical schema: %w", err)
		}

		for _, rootStem := range stemMap {
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

	// If incremental mode and stem exists, filter out covered proposals
	if incremental && existingStem != nil {
		// In incremental mode, if a .stem already exists, we don't propose
		// a bootstrap operation (it's already covered)
		proposals = nil
	}

	return proposals, nil
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
		fmt.Fprintf(cmd.OutOrStdout(), "No schema proposals generated for %s\n", report.Path)
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
	fmt.Fprintf(cmd.OutOrStdout(), "\nSummary: %d total, %d requires agent, %d engine-resolved\n",
		report.Summary.Total, report.Summary.RequiresAgent, report.Summary.EngineResolved)
	return nil
}
