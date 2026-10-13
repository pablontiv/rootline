package main

import (
	"context"
	"encoding/json"
	"fmt"
	"os"

	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/spf13/cobra"
)

var (
	repairDryRun      bool
	repairFillMissing bool
	reportPath        string
	repairRoot        string
)

var repairCmd = &cobra.Command{
	Use:   "repair apply --report <file> [--dry-run]",
	Short: "Apply data-only repair fixes to document frontmatter",
	Long: `Apply repair proposals from a fix report (produced by 'rootline fix --all <dir> --dry-run -o json') to document frontmatter.

Only version 1 rootline/proposals reports are accepted; analyze reports contain
schema and diagnostic inferences rather than document repair proposals.

Repair-surface proposals (correct_value, add_field, migrate_value, etc.)
are applied. Schema proposals (extend_enum, add_aggregate, etc.) are rejected
and appear in the output under Rejected.

Use --dry-run to preview changes without modifying files.`,
	RunE: runRepairApply,
}

var repairApplyCmd = &cobra.Command{
	Use:   "apply --report <file> [--dry-run]",
	Short: "Apply repair proposals to document frontmatter",
	Long: `Apply repair proposals from a fix report (produced by 'rootline fix --all <dir> --dry-run -o json') to document frontmatter.

Only version 1 rootline/proposals reports are accepted. Repair-surface proposals
are applied and schema proposals are rejected. Analyze reports are not accepted.
Use --dry-run to preview changes without modifying files.`,
	Args: cobra.NoArgs,
	RunE: runRepairApply,
}

func init() {
	repairApplyCmd.Flags().StringVar(&reportPath, "report", "", "path to fix report JSON (produced by 'rootline fix --all <dir> --dry-run -o json'; required)")
	_ = repairApplyCmd.MarkFlagRequired("report")
	repairApplyCmd.Flags().BoolVar(&repairDryRun, "dry-run", false, "preview changes without modifying files")
	repairApplyCmd.Flags().BoolVar(&repairFillMissing, "fill-missing", false, "also apply add_field proposals whose value the engine chose rather than the schema declaring it")
	repairApplyCmd.Flags().StringVar(&repairRoot, "root", "", "absolute path to scan root (overrides report root)")

	repairCmd.AddCommand(repairApplyCmd)
	rootCmd.AddCommand(repairCmd)
}

type repairInvocationKey struct{}

// repairInvocation binds the report snapshot and absolute root authorized by
// preflight. RunE must not reread a report that could now name another tree.
type repairInvocation struct {
	report proposal.Report
	root   string
}

func prepareRepairInvocation(cmd *cobra.Command) (*repairInvocation, error) {
	if cmd.Flags().Lookup("report") == nil {
		return nil, fmt.Errorf("--report is required")
	}
	reportFile, err := cmd.Flags().GetString("report")
	if err != nil {
		return nil, err
	}
	if reportFile == "" {
		return nil, fmt.Errorf("--report is required")
	}

	// Read report from file.
	reportData, err := os.ReadFile(reportFile)
	if err != nil {
		return nil, fmt.Errorf("reading report %s: %w", reportFile, err)
	}

	var report proposal.Report
	if err := json.Unmarshal(reportData, &report); err != nil {
		return nil, fmt.Errorf("parsing report: %w", err)
	}

	// Validate report version and kind.
	if report.Version != 1 {
		return nil, fmt.Errorf("unsupported report version: %d (expected 1)", report.Version)
	}
	if report.Kind != "rootline/proposals" {
		return nil, fmt.Errorf("wrong report kind: %s (expected rootline/proposals)", report.Kind)
	}

	// Determine the root directory using the four-level precedence chain.
	explicitRoot, err := cmd.Flags().GetString("root")
	if err != nil {
		return nil, err
	}
	root, err := resolveReportRoot(explicitRoot, report.Root, report.Path, reportFile)
	if err != nil {
		return nil, fmt.Errorf("resolving report root: %w", err)
	}
	return &repairInvocation{report: report, root: root}, nil
}

func runRepairApply(cmd *cobra.Command, _ []string) error {
	invocation, _ := cmd.Context().Value(repairInvocationKey{}).(*repairInvocation)
	// Consume authorization even if applying or rendering fails.
	cmd.SetContext(context.WithValue(cmd.Context(), repairInvocationKey{}, (*repairInvocation)(nil)))
	if invocation == nil {
		return fmt.Errorf("repair invocation was not authorized by boundary preflight")
	}

	// Apply exactly the report and root inspected by preflight.
	result, err := fix.ApplyRepair(invocation.report.Proposals, repairDryRun, invocation.root, repairFillMissing)
	if err != nil {
		return fmt.Errorf("applying repair: %w", err)
	}

	// Emit the payload first, then let the run's own outcome decide the exit
	// status. A caller that sees a non-zero exit must still be able to read
	// what happened from stdout.
	if outputFormat == "table" {
		if err := renderRepairTable(cmd, result); err != nil {
			return err
		}
	} else if err := outputJSON(cmd, result, false); err != nil {
		return err
	}

	return applyExitError(len(result.Errors), len(result.RolledBack))
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

	if len(result.RolledBack) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nRolled back (%d):\n", len(result.RolledBack))
		for _, file := range result.RolledBack {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", file.Path)
			for _, reason := range file.Errors {
				_, _ = fmt.Fprintf(cmd.OutOrStdout(), "    %s\n", reason)
			}
		}
	}

	if len(result.Skipped) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nSkipped (%d):\n", len(result.Skipped))
		for _, s := range result.Skipped {
			_, _ = fmt.Fprintf(cmd.OutOrStdout(), "  %s\n", s)
		}
	}

	if len(result.Rejected) > 0 {
		_, _ = fmt.Fprintf(cmd.OutOrStdout(), "\nRejected (%d):\n", len(result.Rejected))
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

	if len(result.Changed) == 0 && len(result.RolledBack) == 0 && len(result.Errors) == 0 && len(result.Rejected) == 0 {
		_, _ = fmt.Fprintln(cmd.OutOrStdout(), "No repairs applied")
	}

	return nil
}
