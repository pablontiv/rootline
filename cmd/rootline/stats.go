package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var statsCmd = &cobra.Command{
	Use:   "stats [path]",
	Short: "Summary counts by type and state",
	Long:  "Show aggregate statistics for documents: counts by type,\nstate, and other frontmatter fields.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "not implemented yet")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(statsCmd)
}
