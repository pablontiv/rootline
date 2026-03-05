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

var applyCmd = &cobra.Command{
	Use:   "apply [report.json]",
	Short: "Apply inference results to .stem files",
	Long:  "Read an analyze report (from file or stdin) and apply\nschema-modifying inferences to the appropriate .stem files.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runApply,
}

func init() {
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

	// Resolve stem path from report path.
	root, err := filepath.Abs(report.Path)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	entries, walkErr := rules.WalkUp(root)
	if walkErr != nil || len(entries) == 0 {
		return fmt.Errorf("no .stem found for %s", report.Path)
	}

	// Use the closest stem file.
	stemPath := entries[0].Path

	// Collect all schema-modifying inferences.
	var allInferences []infer.ReportInference
	for _, cat := range report.Categories {
		allInferences = append(allInferences, cat.Inferences...)
	}

	result, err := infer.ApplySchemaInferences(stemPath, allInferences)
	if err != nil {
		return fmt.Errorf("applying: %w", err)
	}

	if outputFormat == "table" {
		return renderApplyTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

func renderApplyTable(cmd *cobra.Command, result *infer.ApplyResult) error {
	if len(result.Applied) > 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "Applied:")
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
