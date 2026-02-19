package main

import (
	"fmt"
	"path/filepath"

	"github.com/pablontiv/rootline/internal/rules"
	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe <path>",
	Short: "Show effective schema for a directory",
	Long:  "Display the merged .stem schema that applies to documents\nin the given directory, showing inherited and local rules.",
	Args:  cobra.ExactArgs(1),
	RunE:  runDescribe,
}

func init() {
	rootCmd.AddCommand(describeCmd)
}

func runDescribe(cmd *cobra.Command, args []string) error {
	targetPath, err := filepath.Abs(args[0])
	if err != nil {
		return fmt.Errorf("resolving path: %w", err)
	}

	entries, err := rules.WalkUp(targetPath)
	if err != nil {
		return fmt.Errorf("discovering .stem files: %w", err)
	}

	effective := rules.MergeStemFiles(entries)

	// Use relative path for display
	relPath := args[0]

	result := rules.NewDescribeResult(relPath, entries, effective)
	return outputJSON(cmd, result, false)
}
