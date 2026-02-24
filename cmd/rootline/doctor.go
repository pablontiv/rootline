package main

import (
	"fmt"
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/doctor"
	"github.com/spf13/cobra"
)

var doctorCmd = &cobra.Command{
	Use:   "doctor [path]",
	Short: "Check .stem configuration health",
	Long:  "Inspect .stem files and run diagnostic checks to detect\nconfiguration problems before they cause confusing validation errors.",
	Args:  cobra.MaximumNArgs(1),
	RunE:  runDoctor,
}

func init() {
	rootCmd.AddCommand(doctorCmd)
}

// DoctorResult is the versioned JSON output for doctor.
type DoctorResult = doctor.Result

// DoctorCheck represents a single diagnostic check result.
type DoctorCheck = doctor.Check

// DoctorSummary holds aggregate counts.
type DoctorSummary = doctor.Summary

func runDoctor(cmd *cobra.Command, args []string) error {
	root := "."
	if len(args) > 0 {
		root = args[0]
	}

	absRoot, err := filepath.Abs(root)
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	result, err := doctor.RunChecks(absRoot)
	if err != nil {
		return err
	}

	if outputFormat != "table" {
		return outputJSON(cmd, result, result.Summary.Failed > 0)
	}

	// Table / human-readable output
	w := cmd.OutOrStdout()
	for _, c := range result.Checks {
		icon := "✓"
		switch c.Status {
		case "fail":
			icon = "✗"
		case "warn":
			icon = "!"
		}
		line := fmt.Sprintf("  %s %s", icon, c.Name)
		if c.Path != "" {
			line += fmt.Sprintf(" [%s]", c.Path)
		}
		if c.Message != "" {
			line += fmt.Sprintf(": %s", c.Message)
		}
		_, _ = fmt.Fprintln(w, line)
	}

	var parts []string
	if result.Summary.Passed > 0 {
		parts = append(parts, fmt.Sprintf("%d passed", result.Summary.Passed))
	}
	if result.Summary.Warnings > 0 {
		parts = append(parts, fmt.Sprintf("%d warnings", result.Summary.Warnings))
	}
	if result.Summary.Failed > 0 {
		parts = append(parts, fmt.Sprintf("%d failed", result.Summary.Failed))
	}
	_, _ = fmt.Fprintf(w, "\n%d checks: %s\n", result.Summary.Total, strings.Join(parts, ", "))

	if result.Summary.Failed > 0 {
		cmd.SilenceUsage = true
		cmd.SilenceErrors = true
		return ErrValidationFailed
	}
	return nil
}
