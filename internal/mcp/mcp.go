// Package mcp implements the MCP server (JSON-RPC 2.0).
//
// It wraps the core engine and exposes it via the Model Context Protocol,
// supporting stdio transport.
package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP SDK server with rootline-specific configuration.
type Server struct {
	inner *mcp.Server
}

// NewServer creates a new MCP server with the given name and version.
func NewServer(name, version string) *Server {
	inner := mcp.NewServer(&mcp.Implementation{
		Name:    name,
		Version: version,
	}, nil)
	return &Server{inner: inner}
}

// Inner returns the underlying MCP SDK server for tool registration.
func (s *Server) Inner() *mcp.Server {
	return s.inner
}

// RunStdio starts the server over stdin/stdout transport.
// It blocks until the client disconnects or the context is cancelled.
func (s *Server) RunStdio(ctx context.Context) error {
	return s.inner.Run(ctx, &mcp.StdioTransport{})
}
