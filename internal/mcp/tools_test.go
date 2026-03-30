package mcp

import (
	"context"
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/modelcontextprotocol/go-sdk/mcp"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
)

func setupTestProject(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stem := `version: 2
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

	expected := []string{"query", "validate", "describe", "tree", "stats", "explain", "fix", "graph", "set", "health"}
	for _, name := range expected {
		if !names[name] {
			t.Errorf("expected tool %q not found in list", name)
		}
	}
	if len(result.Tools) != 11 {
		t.Errorf("tool count = %d, want 11", len(result.Tools))
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

func TestTool_Explain(t *testing.T) {
	root := setupTestProject(t)

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "explain",
		Arguments: map[string]any{"path": filepath.Join(root, "a.md")},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error in result")
	}

	text := result.Content[0].(*mcp.TextContent)
	var er map[string]any
	if err := json.Unmarshal([]byte(text.Text), &er); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if er["kind"] != "rootline/explain" {
		t.Errorf("kind = %v, want rootline/explain", er["kind"])
	}

	fields := er["fields"].([]any)
	found := false
	for _, f := range fields {
		fm := f.(map[string]any)
		if fm["name"] == "estado" && fm["origin"] == "frontmatter" {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected estado field with origin frontmatter")
	}
}

func TestTool_Fix(t *testing.T) {
	root := setupTestProject(t)

	// Add a file with invalid estado to trigger proposals.
	if err := os.WriteFile(filepath.Join(root, "c.md"), []byte("---\nestado: Invalid\ntipo: task\n---\n# C\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "fix",
		Arguments: map[string]any{"path": root, "dry_run": true},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error in result")
	}

	text := result.Content[0].(*mcp.TextContent)
	var fr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &fr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if fr["kind"] != "rootline/proposals" {
		t.Errorf("kind = %v, want rootline/proposals", fr["kind"])
	}

	// Verify the report has a summary with total count.
	summary := fr["summary"].(map[string]any)
	if summary["total"] == nil {
		t.Error("expected summary.total in fix report")
	}
}

func TestTool_Graph(t *testing.T) {
	root := setupTestProject(t)

	// Add wiki-link to a.md pointing at b.md.
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\nestado: Pending\ntipo: task\n---\n# A\n[[blocks:b]]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name:      "graph",
		Arguments: map[string]any{"path": root, "check": true},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatal("unexpected error in result")
	}

	text := result.Content[0].(*mcp.TextContent)
	var gr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &gr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if gr["kind"] != "rootline/graph" {
		t.Errorf("kind = %v, want rootline/graph", gr["kind"])
	}

	edges := gr["edges"].([]any)
	if len(edges) == 0 {
		t.Error("expected at least one edge from wiki-link")
	}
}

func TestTool_Set(t *testing.T) {
	root := setupTestProject(t)

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	filePath := filepath.Join(root, "a.md")

	// Call set tool: change estado from Pending to Completed.
	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "set",
		Arguments: map[string]any{
			"path":   filePath,
			"fields": map[string]any{"estado": "Completed"},
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error in result: %v", result.Content)
	}

	text := result.Content[0].(*mcp.TextContent)
	var sr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &sr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if sr["kind"] != "rootline/set" {
		t.Errorf("kind = %v, want rootline/set", sr["kind"])
	}

	applied := sr["applied"].([]any)
	if len(applied) == 0 {
		t.Error("expected at least one applied change")
	}

	// Verify the file was actually modified.
	data, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatalf("reading modified file: %v", err)
	}
	if !strings.Contains(string(data), "estado: Completed") {
		t.Errorf("expected file to contain 'estado: Completed', got:\n%s", string(data))
	}
}

func TestTool_Set_DryRun(t *testing.T) {
	root := setupTestProject(t)

	s := NewServer("test", "v0.1.0")
	RegisterTools(s)
	cs := connectTestSession(t, s)

	filePath := filepath.Join(root, "a.md")
	originalData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}

	result, err := cs.CallTool(context.Background(), &mcp.CallToolParams{
		Name: "set",
		Arguments: map[string]any{
			"path":    filePath,
			"fields":  map[string]any{"estado": "Completed"},
			"dry_run": true,
		},
	})
	if err != nil {
		t.Fatalf("call tool: %v", err)
	}
	if result.IsError {
		t.Fatalf("unexpected error in result: %v", result.Content)
	}

	text := result.Content[0].(*mcp.TextContent)
	var sr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &sr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if sr["dry_run"] != true {
		t.Errorf("dry_run = %v, want true", sr["dry_run"])
	}
	previews := sr["previews"].([]any)
	if len(previews) == 0 {
		t.Error("expected at least one preview")
	}

	// File must not have been modified.
	afterData, err := os.ReadFile(filePath)
	if err != nil {
		t.Fatal(err)
	}
	if string(afterData) != string(originalData) {
		t.Error("dry_run=true must not modify the file")
	}
}

// --- Direct handler unit tests for error/edge paths ---

// setupProjectWithSections creates a temp project with a section field in the schema.
func setupProjectWithSections(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Completed]
    required: true
  tipo:
    type: string
  notes:
    type: section
    heading: "## Notes"
    required: false
`
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\nestado: Pending\ntipo: task\n---\n# A\n\n## Notes\n\nInitial notes.\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	return root
}

func TestHandleSet_FileNotFound(t *testing.T) {
	_, _, err := handleSet(context.Background(), nil, SetInput{
		Path:   "/nonexistent/path/file.md",
		Fields: map[string]string{"estado": "Completed"},
	})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' error, got: %v", err)
	}
}

func TestHandleSet_NoFieldsSpecified(t *testing.T) {
	root := setupTestProject(t)
	_, _, err := handleSet(context.Background(), nil, SetInput{
		Path:   filepath.Join(root, "a.md"),
		Fields: map[string]string{},
	})
	if err == nil {
		t.Fatal("expected error for no fields")
	}
	if !strings.Contains(err.Error(), "no fields specified") {
		t.Errorf("expected 'no fields specified' error, got: %v", err)
	}
}

func TestHandleSet_InvalidEnumValue(t *testing.T) {
	root := setupTestProject(t)
	_, _, err := handleSet(context.Background(), nil, SetInput{
		Path:   filepath.Join(root, "a.md"),
		Fields: map[string]string{"estado": "InvalidValue"},
	})
	if err == nil {
		t.Fatal("expected error for invalid enum value")
	}
	if !strings.Contains(err.Error(), "not in allowed values") {
		t.Errorf("expected 'not in allowed values' error, got: %v", err)
	}
}

func TestHandleSet_AppendSection(t *testing.T) {
	root := setupProjectWithSections(t)
	filePath := filepath.Join(root, "a.md")
	originalData, _ := os.ReadFile(filePath)

	_, _, err := handleSet(context.Background(), nil, SetInput{
		Path:         filePath,
		AppendFields: map[string]string{"notes": "Additional notes."},
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(filePath)
	if string(data) == string(originalData) {
		t.Error("expected file to be modified")
	}
	if !strings.Contains(string(data), "Additional notes.") {
		t.Errorf("expected appended content in file, got:\n%s", string(data))
	}
	if !strings.Contains(string(data), "Initial notes.") {
		t.Errorf("expected original content preserved, got:\n%s", string(data))
	}
}

func TestHandleSet_SectionDryRun(t *testing.T) {
	root := setupProjectWithSections(t)
	filePath := filepath.Join(root, "a.md")

	res, _, err := handleSet(context.Background(), nil, SetInput{
		Path:   filePath,
		Fields: map[string]string{"notes": "New notes content."},
		DryRun: true,
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := res.Content[0].(*mcp.TextContent)
	var sr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &sr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	if sr["dry_run"] != true {
		t.Errorf("expected dry_run=true in result")
	}
	previews := sr["previews"].([]any)
	if len(previews) == 0 {
		t.Error("expected previews for section dry run")
	}
	found := false
	for _, p := range previews {
		if strings.Contains(p.(string), "section") {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected 'section' in preview, got: %v", previews)
	}
}

func TestHandleSet_NoGitRoot(t *testing.T) {
	// Create a directory WITHOUT .git
	dir := t.TempDir()
	filePath := filepath.Join(dir, "file.md")
	if err := os.WriteFile(filePath, []byte("---\nestado: Pending\n---\n# File\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := handleSet(context.Background(), nil, SetInput{
		Path:   filePath,
		Fields: map[string]string{"estado": "Completed"},
	})
	if err == nil {
		t.Fatal("expected error when no git root")
	}
	if !strings.Contains(err.Error(), "finding git root") {
		t.Errorf("expected 'finding git root' error, got: %v", err)
	}
}

func TestFindGitRoot_NoGit(t *testing.T) {
	dir := t.TempDir()
	_, err := findGitRoot(dir)
	if err == nil {
		t.Fatal("expected error when no .git directory")
	}
	if !strings.Contains(err.Error(), "no .git directory") {
		t.Errorf("expected 'no .git directory' error, got: %v", err)
	}
}

func TestFindGitRoot_Found(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	subDir := filepath.Join(dir, "sub", "dir")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}

	got, err := findGitRoot(subDir)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if got != dir {
		t.Errorf("expected root %s, got %s", dir, got)
	}
}

func TestHandleGraph_DotFormat(t *testing.T) {
	root := setupTestProject(t)
	// Add wiki-link
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("---\nestado: Pending\ntipo: task\n---\n# A\n[[blocks:b]]\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := handleGraph(context.Background(), nil, GraphInput{
		Path:   root,
		Format: "dot",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := res.Content[0].(*mcp.TextContent)
	if !strings.Contains(text.Text, "digraph {") {
		t.Errorf("expected DOT format output, got: %s", text.Text)
	}
}

func TestHandleGraph_MermaidFormat(t *testing.T) {
	root := setupTestProject(t)

	res, _, err := handleGraph(context.Background(), nil, GraphInput{
		Path:   root,
		Format: "mermaid",
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := res.Content[0].(*mcp.TextContent)
	if !strings.Contains(text.Text, "graph TD") {
		t.Errorf("expected Mermaid format output, got: %s", text.Text)
	}
}

func TestHandleExplain_FileNotFound(t *testing.T) {
	_, _, err := handleExplain(context.Background(), nil, ExplainInput{
		Path: "/nonexistent/path/file.md",
	})
	if err == nil {
		t.Fatal("expected error for nonexistent file")
	}
	if !strings.Contains(err.Error(), "file not found") {
		t.Errorf("expected 'file not found' error, got: %v", err)
	}
}

func TestHandleExplain_NoExtractor(t *testing.T) {
	// Create a non-markdown file that has no extractor
	dir := t.TempDir()
	binFile := filepath.Join(dir, "test.xyz")
	if err := os.WriteFile(binFile, []byte("binary content"), 0o644); err != nil {
		t.Fatal(err)
	}

	_, _, err := handleExplain(context.Background(), nil, ExplainInput{
		Path: binFile,
	})
	if err == nil {
		t.Fatal("expected error for file with no extractor")
	}
	if !strings.Contains(err.Error(), "no extractor") {
		t.Errorf("expected 'no extractor' error, got: %v", err)
	}
}

func TestHandleQuery_BadWhere(t *testing.T) {
	root := setupTestProject(t)
	_, _, err := handleQuery(context.Background(), nil, QueryInput{
		Path:  root,
		Where: []string{"invalid @@@ expression"},
	})
	if err == nil {
		t.Fatal("expected error for invalid where expression")
	}
}

func TestHandleValidate_BadWhere(t *testing.T) {
	root := setupTestProject(t)
	_, _, err := handleValidate(context.Background(), nil, ValidateInput{
		Path:  root,
		Where: []string{"invalid @@@ expression"},
	})
	if err == nil {
		t.Fatal("expected error for invalid where expression")
	}
}

func TestHandleStats_BadWhere(t *testing.T) {
	root := setupTestProject(t)
	_, _, err := handleStats(context.Background(), nil, StatsInput{
		Path:  root,
		Where: []string{"invalid @@@ expression"},
	})
	if err == nil {
		t.Fatal("expected error for invalid where expression")
	}
}

func TestBuildSetProposal_AppendSection(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"notes": {Type: "section", Heading: "## Notes"},
		},
	}

	p, err := buildSetProposal("notes", "new content", true, "file.md", effective, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Type != proposal.SetSection {
		t.Errorf("expected SetSection, got %s", p.Type)
	}
	if p.Mode != "append" {
		t.Errorf("expected mode 'append', got %q", p.Mode)
	}
}

func TestBuildSetProposal_AppendNonSection(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}},
		},
	}

	_, err := buildSetProposal("estado", "Completed", true, "file.md", effective, false)
	if err == nil {
		t.Fatal("expected error for append on non-section field")
	}
	if !strings.Contains(err.Error(), "append is only supported for section") {
		t.Errorf("expected 'append is only supported for section' error, got: %v", err)
	}
}

func TestBuildSetProposal_CreateSection(t *testing.T) {
	effective := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"notes": {Type: "section"}, // no heading set
		},
	}

	p, err := buildSetProposal("notes", "content", false, "file.md", effective, true)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Type != proposal.SetSection {
		t.Errorf("expected SetSection, got %s", p.Type)
	}
	if p.Mode != "create" {
		t.Errorf("expected mode 'create', got %q", p.Mode)
	}
	// When no heading set, should default to "## <field>"
	if p.Heading != "## notes" {
		t.Errorf("expected heading '## notes', got %q", p.Heading)
	}
}

func TestBuildSetProposal_NilEffective(t *testing.T) {
	p, err := buildSetProposal("title", "value", false, "file.md", nil, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if p.Type != proposal.SetField {
		t.Errorf("expected SetField, got %s", p.Type)
	}
}

func TestBuildSetProposal_NilEffectiveAppend(t *testing.T) {
	// append=true with nil effective → no section found → error
	_, err := buildSetProposal("notes", "content", true, "file.md", nil, false)
	if err == nil {
		t.Fatal("expected error for append with nil effective")
	}
}

func TestHandleTree_WithSubdirs(t *testing.T) {
	// Create a project with nested files to exercise findChild
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	stem := `version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Completed]
`
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}

	// Create nested structure: sub/a.md, sub/b.md (shares parent "sub")
	subDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "a.md"), []byte("---\nestado: Pending\n---\n# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(subDir, "b.md"), []byte("---\nestado: Completed\n---\n# B\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	res, _, err := handleTree(context.Background(), nil, TreeInput{Path: root})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	text := res.Content[0].(*mcp.TextContent)
	var tr map[string]any
	if err := json.Unmarshal([]byte(text.Text), &tr); err != nil {
		t.Fatalf("invalid JSON: %v", err)
	}
	rootNode := tr["root"].(map[string]any)
	if rootNode["total"].(float64) != 2 {
		t.Errorf("expected 2 total, got %v", rootNode["total"])
	}
}

func TestHandleSet_AppendNonSection(t *testing.T) {
	root := setupTestProject(t)
	// Use AppendFields on a non-section field → should fail
	_, _, err := handleSet(context.Background(), nil, SetInput{
		Path:         filepath.Join(root, "a.md"),
		AppendFields: map[string]string{"estado": "Completed"},
	})
	if err == nil {
		t.Fatal("expected error for append on non-section field")
	}
}
