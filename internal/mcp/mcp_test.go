package mcp

import (
	"context"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func TestNewServer(t *testing.T) {
	s := NewServer("test-server", "v0.1.0")
	if s == nil {
		t.Fatal("expected non-nil server")
	}
	if s.Inner() == nil {
		t.Fatal("expected non-nil inner server")
	}
}

func TestServer_PingPong(t *testing.T) {
	s := NewServer("test-server", "v0.1.0")

	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	// Connect server.
	serverSession, err := s.Inner().Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	defer func() { _ = serverSession.Close() }()

	// Connect client.
	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	defer func() { _ = clientSession.Close() }()

	// Ping should work (built-in JSON-RPC method).
	if err := clientSession.Ping(context.Background(), nil); err != nil {
		t.Fatalf("ping failed: %v", err)
	}
}

type echoInput struct {
	Message string `json:"message"`
}

func TestServer_ToolCall(t *testing.T) {
	s := NewServer("test-server", "v0.1.0")

	// Register a simple echo tool.
	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "echo",
		Description: "echo a message",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input echoInput) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{
			Content: []mcp.Content{&mcp.TextContent{Text: input.Message}},
		}, nil, nil
	})

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

	// Call the echo tool.
	result, err := clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "echo",
		Arguments: map[string]any{"message": "hello"},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatal("expected no error in result")
	}
	if len(result.Content) == 0 {
		t.Fatal("expected content in result")
	}
	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected *TextContent, got %T", result.Content[0])
	}
	if text.Text != "hello" {
		t.Errorf("expected 'hello', got %q", text.Text)
	}
}

func TestServer_ToolNotFound(t *testing.T) {
	s := NewServer("test-server", "v0.1.0")

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

	// Call a non-existent tool — SDK should return an error.
	_, err = clientSession.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "nonexistent",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent tool")
	}
}

func TestServer_ListTools(t *testing.T) {
	s := NewServer("test-server", "v0.1.0")

	mcp.AddTool(s.Inner(), &mcp.Tool{
		Name:        "tool-a",
		Description: "first tool",
	}, func(ctx context.Context, req *mcp.CallToolRequest, input any) (*mcp.CallToolResult, any, error) {
		return &mcp.CallToolResult{}, nil, nil
	})

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

	result, err := clientSession.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}
	if len(result.Tools) != 1 {
		t.Fatalf("expected 1 tool, got %d", len(result.Tools))
	}
	if result.Tools[0].Name != "tool-a" {
		t.Errorf("expected tool name 'tool-a', got %q", result.Tools[0].Name)
	}
}
