package main

import (
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/spf13/cobra"
)

var (
	repairDryRun bool
	reportPath   string
)

var repairCmd = &cobra.Command{
	Use:   "repair apply --report <file> [--dry-run]",
	Short: "Apply data-only repair fixes from an analyze report",
	Long: `Apply repair proposals from an analyze report to document frontmatter.

Only repair-surface proposals (correct_value, add_field, migrate_value, etc.)
are applied. Schema proposals (extend_enum, add_aggregate, etc.) are silently
rejected.

Use --dry-run to preview changes without modifying files.`,
	RunE: runRepairApply,
}

var repairApplyCmd = &cobra.Command{
	Use:   "apply --report <file> [--dry-run]",
	Short: "Apply repair proposals from an analyze report",
	Long: `Apply repair proposals to document frontmatter.

Only repair-surface proposals are applied. Schema proposals are rejected.
Use --dry-run to preview changes without modifying files.`,
	Args: cobra.NoArgs,
	RunE: runRepairApply,
}

func init() {
	repairApplyCmd.Flags().StringVar(&reportPath, "report", "", "path to analyze report JSON (required)")
	_ = repairApplyCmd.MarkFlagRequired("report")
	repairApplyCmd.Flags().BoolVar(&repairDryRun, "dry-run", false, "preview changes without modifying files")

	repairCmd.AddCommand(repairApplyCmd)
	rootCmd.AddCommand(repairCmd)
}

func runRepairApply(cmd *cobra.Command, _ []string) error {
	if reportPath == "" {
		return fmt.Errorf("--report is required")
	}

	// Read report from file.
	reportData, err := os.ReadFile(reportPath)
	if err != nil {
		return fmt.Errorf("reading report %s: %w", reportPath, err)
	}

	var report proposal.Report
	if err := json.Unmarshal(reportData, &report); err != nil {
		return fmt.Errorf("parsing report: %w", err)
	}

	// Determine the root directory (directory containing the report).
	root, err := filepath.Abs(filepath.Dir(reportPath))
	if err != nil {
		return fmt.Errorf("resolving report directory: %w", err)
	}

	// Apply repair.
	result, err := fix.ApplyRepair(report.Proposals, repairDryRun, root)
	if err != nil {
		return fmt.Errorf("applying repair: %w", err)
	}

	// Output result.
	if outputFormat == "table" {
		return renderRepairTable(cmd, result)
	}
	return outputJSON(cmd, result, false)
}

// renderRepairTable renders the repair result in table format.
func renderRepairTable(cmd *cobra.Command, result *fix.RepairResult) error {
	if result.DryRun {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "[DRY RUN - no files modified]")
	}

	if len(result.Changed) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nChanged (%d):\n", len(result.Changed))
		for _, c := range result.Changed {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", c)
		}
	}

	if len(result.Skipped) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nSkipped (%d):\n", len(result.Skipped))
		for _, s := range result.Skipped {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", s)
		}
	}

	if len(result.Rejected) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nRejected (schema proposals, %d):\n", len(result.Rejected))
		for _, r := range result.Rejected {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", r)
		}
	}

	if len(result.Errors) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nErrors (%d):\n", len(result.Errors))
		for _, e := range result.Errors {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", e)
		}
	}

	if len(result.Changed) == 0 && len(result.Errors) == 0 && len(result.Rejected) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No repairs applied")
	}

	return nil
}
