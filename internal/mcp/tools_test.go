package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
)

func setupTestProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stem := `version: 1
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Completed]
    required: true
  tipo:
    type: string
`
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\nestado: Pending\ntipo: task\n---\n# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "b.md"), []byte("---\nestado: Completed\ntipo: task\n---\n# B\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func connectTestSession(t *testing.T, s *Server) *mcp.ClientSession {
	t.Helper()
	clientTransport, serverTransport := mcp.NewInMemoryTransports()

	serverSession, err := s.Inner().Connect(context.Background(), serverTransport, nil)
	if err != nil {
		t.Fatalf("server connect: %v", err)
	}
	t.Cleanup(func() { _ = serverSession.Close() })

	client := mcp.NewClient(&mcp.Implementation{Name: "test-client", Version: "v0.1.0"}, nil)
	clientSession, err := client.Connect(context.Background(), clientTransport, nil)
	if err != nil {
		t.Fatalf("client connect: %v", err)
	}
	t.Cleanup(func() { _ = clientSession.Close() })

	return clientSession
}

func TestToolsRegistration_ListTools(t *testing.T) {
	s := NewServer("test", "v0.1.0")
	RegisterTools(s)

	cs := connectTestSession(t, s)

	result, err := cs.ListTools(context.Background(), nil)
	if err != nil {
		t.Fatalf("list tools: %v", err)
	}

	names := make(map[string]bool)
	for _, tool := range result.Tools {
		names[tool.Name] = true
	}

	expected := []string{"query", "validate", "describe", "tree", "stats"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected tool %q not found in list", name)
		}
	}
}

func TestTool_Query(t *testing.T) {
	root := setupTestProject(t)

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "query",
		Arguments: map[string]any{"path": root},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error in result")
	}

	text, ok := result.Content[0].(*mcp.TextContent)
	if !ok {
		t.Fatalf("expected TextContent, got %T", result.Content[0])
	}

	var qr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &qr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if qr["kind"] != "rootline/query" {
		t.Errorf("kind = %v, want rootline/query", qr["kind"])
	}
}

func TestTool_QueryWithWhere(t *testing.T) {
	root := setupTestProject(t)

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "query",
		Arguments: map[string]any{
			"path":  root,
			"where": []any{"estado == 'Pending'"},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent)
	var qr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &qr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	meta := qr["meta"].(map[string]any)
	if meta["count"].(float64) != 1 {
		t.Errorf("expected 1 result for Pending, got %v", meta["count"])
	}
}

func TestTool_Validate(t *testing.T) {
	root := setupTestProject(t)

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "validate",
		Arguments: map[string]any{"path": root},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent)
	var vr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &vr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if vr["kind"] != "rootline/validate-batch" {
		t.Errorf("kind = %v, want rootline/validate-batch", vr["kind"])
	}
}

func TestTool_Describe(t *testing.T) {
	root := setupTestProject(t)

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "describe",
		Arguments: map[string]any{"path": root},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent)
	var dr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &dr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if dr["kind"] != "rootline/describe" {
		t.Errorf("kind = %v, want rootline/describe", dr["kind"])
	}
	schema := dr["schema"].(map[string]any)
	if _, ok := schema["estado"]; !ok {
		t.Error("expected estado field in schema")
	}
}

func TestTool_Tree(t *testing.T) {
	root := setupTestProject(t)

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "tree",
		Arguments: map[string]any{"path": root},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent)
	var tr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &tr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if tr["kind"] != "rootline/tree" {
		t.Errorf("kind = %v, want rootline/tree", tr["kind"])
	}
	rootNode := tr["root"].(map[string]any)
	if rootNode["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", rootNode["total"])
	}
}

func TestTool_Stats(t *testing.T) {
	root := setupTestProject(t)

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "stats",
		Arguments: map[string]any{"path": root},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}

	text := result.Content[0].(*mcp.TextContent)
	var sr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &sr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if sr["kind"] != "rootline/stats" {
		t.Errorf("kind = %v, want rootline/stats", sr["kind"])
	}
	if sr["total"].(float64) != 2 {
		t.Errorf("total = %v, want 2", sr["total"])
	}
	byEstado := sr["by_estado"].(map[string]any)
	if byEstado["Pending"].(float64) != 1 {
		t.Errorf("Pending count = %v, want 1", byEstado["Pending"])
	}
	if byEstado["Completed"].(float64) != 1 {
		t.Errorf("Completed count = %v, want 1", byEstado["Completed"])
	}
}
