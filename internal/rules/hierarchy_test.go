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

	stemContent := []byte("version: 1\nschema:\n  estado:\n    type: string\n")
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
