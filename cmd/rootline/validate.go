package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var validateCmd = &cobra.Command{
	Use:   "validate [path...]",
	Short: "Check documents against .stem rules",
	Long:  "Validate one or more documents against the effective schema\ndefined by .stem files in the directory tree.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "not implemented yet")
		return nil
	},
}

func init() {
	validateCmd.Flags().String("path", ".", "root path to validate from")
	rootCmd.AddCommand(validateCmd)
}
