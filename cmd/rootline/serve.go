package main

import (
	"fmt"
	"os/signal"
	"syscall"

	mcpserver "github.com/pablontiv/rootline/internal/mcp"
	"github.com/spf13/cobra"
)

var (
	serveAddr  string
	serveStdio bool
)

var serveCmd = &cobra.Command{
	Use:   "serve",
	Short: "Start MCP server",
	Long: `Launch the Rootline MCP server.

Default: Streamable HTTP on --addr (stateless mode, multi-consumer).
Legacy:  --stdio for Claude Code MCP client configuration.`,
	RunE: func(cmd *cobra.Command, args []string) error {
		s := mcpserver.NewServer("rootline", version)
		mcpserver.RegisterTools(s)

		ctx, stop := signal.NotifyContext(cmd.Context(), syscall.SIGTERM, syscall.SIGINT)
		defer stop()

		if serveStdio {
			return s.RunStdio(ctx)
		}

		_, _ = fmt.Fprintf(cmd.ErrOrStderr(), "rootline MCP server listening on %s\n", serveAddr)
		return s.RunHTTP(ctx, serveAddr, nil)
	},
}

func init() {
	serveCmd.Flags().StringVar(&serveAddr, "addr", "127.0.0.1:9200", "HTTP listen address")
	serveCmd.Flags().BoolVar(&serveStdio, "stdio", false, "use stdio transport (legacy)")
	rootCmd.AddCommand(serveCmd)
}
