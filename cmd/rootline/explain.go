package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var explainCmd = &cobra.Command{
	Use:   "explain <file>",
	Short: "Trace why a document has a given state",
	Long:  "Show the .stem rules and derivation chain that produced\nthe current state of a document. Every computed field is traceable.",
	Args:  cobra.ExactArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "not implemented yet")
		return nil
	},
}

func init() {
	rootCmd.AddCommand(explainCmd)
}
