package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var treeCmd = &cobra.Command{
	Use:   "tree [path]",
	Short: "Hierarchical view with derived state",
	Long:  "Display the document tree with completion counts and\nderived state computed from .stem rules.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "not implemented yet")
		return nil
	},
}

func init() {
	treeCmd.Flags().Bool("derive", false, "compute derived state for display")
	rootCmd.AddCommand(treeCmd)
}
