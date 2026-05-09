package main

import (
	"encoding/json"
	"fmt"
	"io"
	"os"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var applyDryRun bool

var applyCmd = &cobra.Command{
	Use:   "apply [report.json]",
	Short: "Apply inference results to .stem and document files",
	Long:  "Read an analyze report (from file or stdin) and apply\nschema-modifying inferences to .stem files and data corrections to documents.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runApply,
}

func init() {
	applyCmd.Flags().BoolVar(&applyDryRun, "dry-run", false, "Show changes without applying them")
	rootCmd.AddCommand(applyCmd)
}

func runApply(cmd *cobra.Command, args []string) error {
	// Read report from file or stdin.
	var data []byte
	var err error

	if len(args) > 0 {
		data, err = os.ReadFile(args[0])
	} else {
		data, err = io.ReadAll(os.Stdin)
	}
	if err != nil {
		return fmt.Errorf("reading report: %w", err)
	}

	var report infer.AnalyzeReport
	if err := json.Unmarshal(data, &report); err != nil {
		return fmt.Errorf("parsing report: %w", err)
	}

	// Resolve root path from report path.
	root, err := filepath.Abs(report.Path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	// Pre-phase: scaffold .stem for directories that have none.
	for _, cat := range report.Categories {
		for _, inf := range cat.Inferences {
			if inf.Type != "missing_schema" || inf.RequiresAgent {
				continue
			}
			if scaffoldErr := infer.ScaffoldSchema(inf.Source); scaffoldErr != nil {
				_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "scaffold %s: %v\n", inf.Source, scaffoldErr)
			} else {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "scaffolded %s/.stem\n", inf.Source)
			}
		}
	}

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

	// Separate schema-modifying and data-correction inferences.
	var schemaInferences []infer.ReportInference
	var dataInferences []infer.ReportInference
	for _, cat := range report.Categories {
		for _, inf := range cat.Inferences {
			switch inf.Type {
			case "migrate_value", "correct_value", "add_field":
				dataInferences = append(dataInferences, inf)
			default:
				schemaInferences = append(schemaInferences, inf)
			}
		}
	}

	// Apply schema modifications (to .stem).
	schemaResult, err := infer.ApplySchemaInferences(stemPath, schemaInferences)
	if err != nil {
		return fmt.Errorf("applying schema: %w", err)
	}

	// Apply data corrections (to documents).
	opts := infer.ApplyOptions{
		DryRun: applyDryRun,
		Root:   root,
	}
	dataResult, err := infer.ApplyDataCorrections(dataInferences, opts)
	if err != nil {
		return fmt.Errorf("applying data: %w", err)
	}

	// Merge results.
	result := &infer.ApplyResult{
		Applied: append(schemaResult.Applied, dataResult.Applied...),
		Skipped: append(schemaResult.Skipped, dataResult.Skipped...),
		DryRun:  applyDryRun,
	}

	if outputFormat == "table" {
		return renderApplyTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

func renderApplyTable(cmd *cobra.Command, result *infer.ApplyResult) error {
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
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  ✓ %s\n", a)
		}
	}
	if len(result.Skipped) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Skipped (requires agent):")
		for _, s := range result.Skipped {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  ⚠ %s\n", s)
		}
	}
	if len(result.Applied) == 0 && len(result.Skipped) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No modifications to apply.")
	}
	return nil
}
