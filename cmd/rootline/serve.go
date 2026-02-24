package main

import (
	mcpserver "github.com/pablontiv/rootline/internal/mcp"
	"github.com/spf13/cobra"
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server (stdio)",
	Long:  "Launch the Rootline MCP server exposing the core engine\nvia JSON-RPC 2.0 over stdio transport.",
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mcpserver.NewServer("rootline", version)
		mcpserver.RegisterTools(s)
		return s.RunStdio(cmd.Context())
	},
}

func init() {
	rootCmd.AddCommand(serveCmd)
}
