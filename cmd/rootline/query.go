package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var (
	queryWhere []string
	queryCount bool
	queryLimit int
)

var queryCmd = &cobra.Command{
	Use:   "query [path]",
	Short: "Search and filter records",
	Long:  "Query documents matching filter expressions.\nMultiple --where flags are combined with AND.",
	Args:  cobra.MaximumNArgs(1),
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "not implemented yet")
		return nil
	},
}

func init() {
	queryCmd.Flags().StringSliceVar(&queryWhere, "where", nil, "filter expression (repeatable, e.g. 'estado eq Pending')")
	queryCmd.Flags().BoolVar(&queryCount, "count", false, "return count instead of records")
	queryCmd.Flags().IntVar(&queryLimit, "limit", 0, "limit number of results (0 = unlimited)")
	rootCmd.AddCommand(queryCmd)
}
