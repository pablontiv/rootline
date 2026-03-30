// Package mcp implements the MCP server (JSON-RPC 2.0).
//
// It wraps the core engine and exposes it via the Model Context Protocol,
// supporting stdio and Streamable HTTP transports.
package mcp

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"runtime"
	"time"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// Server wraps the MCP SDK server with rootline-specific configuration.
type Server struct {
	inner     *mcp.Server
	startTime time.Time
	version   string
}

// NewServer creates a new MCP server with the given name and version.
func NewServer(name, version string) *Server {
	inner := mcp.NewServer(&mcp.Implementation{
		Name:    name,
		Version: version,
	}, nil)
	return &Server{inner: inner, startTime: time.Now(), version: version}
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

// HealthStatus is the response payload for the health endpoint and tool.
type HealthStatus struct {
	Status     string `json:"status"`
	Version    string `json:"version"`
	Uptime     string `json:"uptime"`
	GoRoutines int    `json:"go_routines"`
}

// Health returns the current health status.
func (s *Server) Health() HealthStatus {
	return HealthStatus{
		Status:     "ok",
		Version:    s.version,
		Uptime:     time.Since(s.startTime).Truncate(time.Second).String(),
		GoRoutines: runtime.NumGoroutine(),
	}
}

// RunHTTP starts the MCP server over Streamable HTTP transport (stateless mode).
// It blocks until the context is cancelled.
// If addrCh is non-nil, the actual listen address is sent after binding.
func (s *Server) RunHTTP(ctx context.Context, addr string, addrCh chan<- string) error {
	handler := mcp.NewStreamableHTTPHandler(
		func(r *http.Request) *mcp.Server { return s.inner },
		&mcp.StreamableHTTPOptions{Stateless: true},
	)

	mux := http.NewServeMux()
	mux.Handle("/mcp", handler)
	mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		_ = json.NewEncoder(w).Encode(s.Health())
	})

	ln, err := net.Listen("tcp", addr)
	if err != nil {
		return fmt.Errorf("listen %s: %w", addr, err)
	}

	if addrCh != nil {
		addrCh <- ln.Addr().String()
	}

	srv := &http.Server{Handler: mux, ReadHeaderTimeout: 10 * time.Second}

	// Graceful shutdown on context cancellation.
	go func() {
		<-ctx.Done()
		shutdownCtx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()
		_ = srv.Shutdown(shutdownCtx)
	}()

	if err := srv.Serve(ln); err != http.ErrServerClosed {
		return err
	}
	return nil
}
