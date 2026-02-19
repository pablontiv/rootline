package main

import (
	"fmt"

	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server (stdio or SSE)",
	Long:  "Launch the Rootline MCP server exposing the core engine\nvia JSON-RPC 2.0. Supports stdio (default) and SSE transports.",
	RunE: func(cmd *cobra.Command, args []string) error {
		fmt.Fprintln(cmd.ErrOrStderr(), "not implemented yet")
		return nil
	},
}

func init() {
	serveCmd.Flags().String("transport", "stdio", "transport mode (stdio|sse)")
	serveCmd.Flags().Int("port", 3000, "port for SSE transport")
	rootCmd.AddCommand(serveCmd)
}
