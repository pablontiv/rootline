package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestResolveForRecord(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := []byte("version: 2\nschema:\n  estado:\n    type: string\n")
	if err := os.WriteFile(filepath.Join(dir, ".stem"), stemContent, 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ResolveForRecord(dir, "file.md")
	if err != nil {
		t.Fatalf("ResolveForRecord failed: %v", err)
	}
	if result.Schema["estado"].Type != "string" {
		t.Error("expected estado field from .stem")
	}
}

func TestResolveForRecord_V2MatchFiltering(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// v2 stem with match-scoped fields
	stemContent := []byte(`version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    values: [Pending, Completed]
  tipo:
    type: enum
    values: [modulo-sistema, test]
    match: ["F*", "T*"]
    required:
      match: ["T*"]
`)

	// Create directory structure: root/E01/F01/S001/
	taskDir := filepath.Join(root, "E01", "F01", "S001")
	if err := os.MkdirAll(taskDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".stem"), stemContent, 0o644); err != nil {
		t.Fatal(err)
	}

	t.Run("T-level includes tipo and required", func(t *testing.T) {
		recordPath := "E01/F01/S001/T001-task.md"
		result, err := ResolveForRecord(taskDir, recordPath)
		if err != nil {
			t.Fatalf("ResolveForRecord failed: %v", err)
		}
		tipo, ok := result.Schema["tipo"]
		if !ok {
			t.Fatal("expected tipo in schema for T-level")
		}
		if !tipo.Required {
			t.Error("expected tipo.Required=true at T-level")
		}
		if _, ok := result.Schema["estado"]; !ok {
			t.Error("expected estado in schema (global field)")
		}
	})

	t.Run("F-level includes tipo but not required", func(t *testing.T) {
		featureDir := filepath.Join(root, "E01", "F01")
		// Use relative path to avoid temp dir name matching patterns
		recordPath := "E01/F01/F01-feature/README.md"
		result, err := ResolveForRecord(featureDir, recordPath)
		if err != nil {
			t.Fatalf("ResolveForRecord failed: %v", err)
		}
		tipo, ok := result.Schema["tipo"]
		if !ok {
			t.Fatal("expected tipo in schema for F-level")
		}
		if tipo.Required {
			t.Error("expected tipo.Required=false at F-level")
		}
	})

	t.Run("E-level excludes tipo", func(t *testing.T) {
		epicDir := filepath.Join(root, "E01")
		recordPath := "E01/README.md"
		result, err := ResolveForRecord(epicDir, recordPath)
		if err != nil {
			t.Fatalf("ResolveForRecord failed: %v", err)
		}
		if _, ok := result.Schema["tipo"]; ok {
			t.Error("expected tipo absent at E-level (match restricts to F* and T*)")
		}
	})
}

func TestResolveForRecord_V1Unchanged(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	// v1 stem — should NOT apply match filtering
	stemContent := []byte(`version: 2
schema:
  estado:
    type: enum
    values: [Pending, Completed]
    required: true
  tipo:
    type: enum
    values: [a, b]
    match: ["T*"]
`)
	if err := os.WriteFile(filepath.Join(root, ".stem"), stemContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// v1 should return all fields without filtering
	result, err := ResolveForRecord(root, filepath.Join(root, "README.md"))
	if err != nil {
		t.Fatalf("ResolveForRecord failed: %v", err)
	}
	// For v1, tipo should still be in schema even though match doesn't match
	if _, ok := result.Schema["tipo"]; !ok {
		t.Error("v1 should not filter by match — expected tipo in schema")
	}
}
