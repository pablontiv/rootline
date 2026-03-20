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

var analyzeCmd = &cobra.Command{
	Use:   "analyze [directory]",
	Short: "Analyze documents and infer schema patterns",
	Long:  "Run all inference detectors on documents in the given directory\nand produce a structured report of findings.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runAnalyze,
}

func init() {
	analyzeCmd.Flags().BoolVar(&analyzeIncremental, "incremental", false, "report only inferences not covered by existing .stem")
	rootCmd.AddCommand(analyzeCmd)
}

// agentRequiredTypes lists inference types that need agent disambiguation.
var agentRequiredTypes = map[string]bool{
	"informal_dependency_candidate": true,
	"unverified_traceability":       true,
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

	derive.DeriveAllSimple(ctx, records, root)
	derive.EnrichBuiltinsSimple(ctx, records, root)
	derive.AggregateAllSimple(ctx, records, root)

	// Load stem for link schema.
	linkSchema := resolveLinkSchema(root)

	// Build graph for back-reference analysis.
	g := graph.Build(ctx, records)

	// Build report.
	report := infer.NewAnalyzeReport(scanRoot)

	// Run all detectors, catching panics per-category.
	type category struct {
		id   string
		name string
		run  func() []infer.Inference
	}

	categories := []category{
		{"field_types", "Field Type Inference", func() []infer.Inference {
			return fieldStatsToInferences(infer.Analyze(records))
		}},
		{"required_fields", "Required Field Detection", func() []infer.Inference {
			return requiredFieldInferences(infer.Analyze(records))
		}},
		{"enum_values", "Enum Value Detection", func() []infer.Inference {
			return enumInferences(infer.Analyze(records))
		}},
		{"constant_fields", "Constant Field Detection", func() []infer.Inference {
			return infer.DetectConstantFields(records)
		}},
		{"link_types", "Link Type Validation", func() []infer.Inference {
			return infer.DetectLinkTypes(records, linkSchema)
		}},
		{"back_references", "Back Reference Detection", func() []infer.Inference {
			return infer.DetectMissingBackReferences(g)
		}},
		{"cross_references", "Cross Reference Detection", func() []infer.Inference {
			return infer.DetectCrossReferences(records, root)
		}},
		{"section_patterns", "Body Section Patterns", func() []infer.Inference {
			return infer.DetectSectionPatterns(records)
		}},
		{"invariants", "Invariant Extraction", func() []infer.Inference {
			return infer.DetectInvariants(records)
		}},
		{"sub_schemas", "Sub-Schema Detection", func() []infer.Inference {
			return infer.DetectSubSchemas(records, "tipo")
		}},
		{"formal_dependencies", "Formal Dependency Extraction", func() []infer.Inference {
			return infer.DetectFormalDependencies(records)
		}},
		{"traceability", "Traceability Link Extraction", func() []infer.Inference {
			return infer.DetectTraceabilityLinks(records)
		}},
		{"structural", "Structural Rule Detection", func() []infer.Inference {
			return infer.DetectStructural(root)
		}},
	}

	// Load stem for incremental filtering.
	var stem *rules.StemFile
	if analyzeIncremental {
		entries, walkErr := rules.WalkUp(root)
		if walkErr == nil && len(entries) > 0 {
			stem = rules.MergeStemFiles(entries)
		}
		report.Incremental = true
	}

	for _, cat := range categories {
		inferences := safeRunDetector(ctx, cat.id, cat.run)
		if analyzeIncremental && stem != nil {
			inferences = infer.FilterCoveredInferences(inferences, stem)
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
func safeRunDetector(_ context.Context, id string, fn func() []infer.Inference) (result []infer.Inference) {
	defer func() {
		if r := recover(); r != nil {
			result = []infer.Inference{{
				Type:    "detector_error",
				Source:  id,
				Message: fmt.Sprintf("detector %s panicked: %v", id, r),
			}}
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
