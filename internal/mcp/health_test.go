package mcp

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestHealthTool_ReturnsStatus(t *testing.T) {
	s := NewServer("test-server", "v0.1.0")
	RegisterTools(s)

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := s.Inner().Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer func() { _ = serverSession.Close() }()

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer func() { _ = clientSession.Close() }()

	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "health",
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatal("expected no error in result")
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var status HealthStatus
	if err := json.Unmarshal([]byte(text.Text), &status); err != nil {
		t.Fatalf("unmarshal health: %v", err)
	}
	if status.Status != "ok" {
		t.Errorf("expected status 'ok', got %q", status.Status)
	}
	if status.Version != "v0.1.0" {
		t.Errorf("expected version 'v0.1.0', got %q", status.Version)
	}
	if status.GoRoutines == 0 {
		t.Error("expected non-zero goroutines")
	}
}
