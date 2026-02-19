package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var describeCmd = &cobra.Command{
	Use:   "describe <path>",
	Short: "Show effective schema for a directory",
	Long:  "Display the merged .stem schema that applies to documents\nin the given directory, showing inherited and local rules.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "not implemented yet")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(describeCmd)
}
