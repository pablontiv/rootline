package main

import (
	"context"
	"fmt"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var analyzeIncremental bool
var analyzeThreshold float64

var analyzeCmd = &cobra.Command{
	Use:   "analyze [directory]",
	Short: "Analyze documents and infer schema patterns",
	Long: `Run all inference detectors on documents in the given directory
and produce a structured report of findings.

The generated report can be processed by rootline schema apply to apply
schema-modifying inferences to .stem files.`,
	Args: cobra.MaximumNArgs(1),
	RunE: runAnalyze,
}

func init() {
	analyzeCmd.Flags().BoolVar(&analyzeIncremental, "incremental", false, "report only inferences not covered by existing .stem")
	analyzeCmd.Flags().Float64Var(&analyzeThreshold, "threshold", 0.60, "section pattern detection threshold (0.0-1.0)")
	rootCmd.AddCommand(analyzeCmd)
}

// agentRequiredTypes lists inference types that need agent disambiguation.
var agentRequiredTypes = map[string]bool{
	"informal_dependency_candidate": true,
	"unverified_traceability":       true,
	// Governance detectors
	"implicit_schema":         true,
	"naming_inconsistency":    true,
	"enum_without_values":     true,
	"required_understatement": true,
}

func runAnalyze(cmd *cobra.Command, args []string) error {
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
	resolver := stemScopeResolver()

	// Bootstrap scan: this command derives a schema from documents that may
	// not have one yet, so a missing schema must not stop it.
	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver), index.AllowUngoverned())
	if err != nil {
		return fmt.Errorf("scanning: %w", err)
	}

	gapsResolver := infer.DefaultStemResolver()

	derive.DeriveAllSimple(ctx, records, root)
	if err := derive.EnrichBuiltinsSimple(ctx, records, root); err != nil {
		return fmt.Errorf("enriching records: %w", err)
	}
	derive.AggregateAllSimple(ctx, records, root)

	// Load stem for link schema.
	linkSchema := resolveLinkSchema(root)

	// Build graph for back-reference analysis.
	g := graph.Build(ctx, records)

	// Build report.
	report := infer.NewAnalyzeReport(scanRoot)
	report.Root = root

	if analyzeIncremental {
		report.Incremental = true
	}

	// Run all detectors, catching panics per-category.
	type category struct {
		id   string
		name string
		run  func() ([]infer.Inference, error)
	}

	categories := []category{
		{"field_types", "Field Type Inference", func() ([]infer.Inference, error) {
			return fieldStatsToInferences(infer.Analyze(records)), nil
		}},
		{"required_fields", "Required Field Detection", func() ([]infer.Inference, error) {
			return requiredFieldInferences(infer.Analyze(records)), nil
		}},
		{"enum_values", "Enum Value Detection", func() ([]infer.Inference, error) {
			return enumInferences(infer.Analyze(records)), nil
		}},
		{"constant_fields", "Constant Field Detection", func() ([]infer.Inference, error) {
			return infer.DetectConstantFields(records), nil
		}},
		{"link_types", "Link Type Validation", func() ([]infer.Inference, error) {
			return infer.DetectLinkTypes(records, linkSchema), nil
		}},
		{"back_references", "Back Reference Detection", func() ([]infer.Inference, error) {
			return infer.DetectMissingBackReferences(g), nil
		}},
		{"cross_references", "Cross Reference Detection", func() ([]infer.Inference, error) {
			return infer.DetectCrossReferences(records, root), nil
		}},
		{"section_patterns", "Body Section Patterns", func() ([]infer.Inference, error) {
			return infer.DetectSectionPatterns(records, analyzeThreshold)
		}},
		{"invariants", "Invariant Extraction", func() ([]infer.Inference, error) {
			return infer.DetectInvariants(records), nil
		}},
		{"formal_dependencies", "Formal Dependency Extraction", func() ([]infer.Inference, error) {
			return infer.DetectFormalDependencies(records), nil
		}},
		{"traceability", "Traceability Link Extraction", func() ([]infer.Inference, error) {
			return infer.DetectTraceabilityLinks(records), nil
		}},
		{"structural", "Structural Rule Detection", func() ([]infer.Inference, error) {
			return infer.DetectStructural(root), nil
		}},
		{"schema_coverage", "Schema Coverage", func() ([]infer.Inference, error) {
			return infer.DetectMissingSchemata(root), nil
		}},
		{"validation_gaps", "Validation Gaps", func() ([]infer.Inference, error) {
			var prior []infer.Inference
			for _, cat := range report.Categories {
				for _, inf := range cat.Inferences {
					prior = append(prior, infer.Inference{
						Type: inf.Type, Field: inf.Field, Value: inf.Value,
					})
				}
			}
			return infer.DetectValidationGaps(records, prior, root, gapsResolver), nil
		}},
	}

	for _, cat := range categories {
		inferences, err := safeRunDetector(ctx, cat.id, cat.run)
		if err != nil {
			return fmt.Errorf("detector %s: %w", cat.id, err)
		}
		if analyzeIncremental {
			inferences = infer.FilterCoveredInferences(inferences, records, root, gapsResolver)
		}
		report.AddCategory(cat.id, cat.name, inferences, agentRequiredTypes)
	}

	report.Finalize()

	if outputFormat == "table" {
		return renderAnalyzeTable(cmd, report)
	}
	return outputJSON(cmd, report, false)
}

// safeRunDetector runs a detector function, recovering from panics.
func safeRunDetector(_ context.Context, id string, fn func() ([]infer.Inference, error)) (result []infer.Inference, err error) {
	defer func() {
		if r := recover(); r != nil {
			result = []infer.Inference{{
				Type:    "detector_error",
				Source:  id,
				Message: fmt.Sprintf("detector %s panicked: %v", id, r),
			}}
			err = nil
		}
	}()
	return fn()
}

// fieldStatsToInferences converts Analyze() field stats into Inference values.
func fieldStatsToInferences(schema *infer.InferredSchema) []infer.Inference {
	var inferences []infer.Inference
	for name, stats := range schema.Fields {
		inferences = append(inferences, infer.Inference{
			Type:    "field_type",
			Field:   name,
			Value:   stats.InferredType,
			Message: fmt.Sprintf("field %q inferred as %s (%d/%d records)", name, stats.InferredType, stats.Count, stats.TotalRecords),
		})
	}
	return inferences
}

// requiredFieldInferences extracts required field inferences from schema.
func requiredFieldInferences(schema *infer.InferredSchema) []infer.Inference {
	var inferences []infer.Inference
	for name, sf := range schema.Schema {
		if sf.Required {
			inferences = append(inferences, infer.Inference{
				Type:    "required_field",
				Field:   name,
				Message: fmt.Sprintf("field %q appears in >80%% of records — required", name),
			})
		}
	}
	return inferences
}

// enumInferences extracts enum value inferences from schema.
func enumInferences(schema *infer.InferredSchema) []infer.Inference {
	var inferences []infer.Inference
	for name, sf := range schema.Schema {
		if sf.Type == "enum" && len(sf.Values) > 0 {
			inferences = append(inferences, infer.Inference{
				Type:    "enum_values",
				Field:   name,
				Value:   fmt.Sprintf("%v", sf.Values),
				Message: fmt.Sprintf("field %q has enum values: %v", name, sf.Values),
			})
		}
	}
	return inferences
}

// resolveLinkSchema loads the link schema from the root's effective stem.
func resolveLinkSchema(root string) rules.LinkSchema {
	entries, err := rules.WalkUp(root)
	if err != nil || len(entries) == 0 {
		return rules.LinkSchema{}
	}
	effective := rules.MergeStemFiles(entries)
	if effective == nil {
		return rules.LinkSchema{}
	}
	return effective.Links
}

// renderAnalyzeTable renders the analyze report in table format.
func renderAnalyzeTable(cmd *cobra.Command, report *infer.AnalyzeReport) error {
	headers := []string{"Category", "Inferences", "Agent Required"}
	var rows [][]string
	for _, cat := range report.Categories {
		agentCount := 0
		for _, inf := range cat.Inferences {
			if inf.RequiresAgent {
				agentCount++
			}
		}
		rows = append(rows, []string{
			cat.Name,
			fmt.Sprintf("%d", cat.InferenceCount),
			fmt.Sprintf("%d", agentCount),
		})
	}
	renderTable(cmd.OutOrStdout(), headers, rows)

	// Summary line.
	_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nTotal: %d inferences (%d engine, %d agent)\n",
		report.Summary.TotalInferences, report.Summary.EngineResolved, report.Summary.AgentRequired)

	return nil
}
