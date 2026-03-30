package mcp

import (
	"context"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

// HealthInput is the input for the health tool (no parameters required).
type HealthInput struct{}

// serverRef holds a reference to the Server for the health handler.
// Set during RegisterTools.
var serverRef *Server

func handleHealth(ctx context.Context, _ *mcp.CallToolRequest, input HealthInput) (*mcp.CallToolResult, any, error) {
	if serverRef == nil {
		return jsonResult(HealthStatus{Status: "ok"})
	}
	return jsonResult(serverRef.Health())
}
