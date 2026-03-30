package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStemCache_CachesRepeatedLookups(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".git"), []byte(""), 0o644); err != nil {
		t.Fatal(err)
	}
	stemContent := []byte("version: 2\nschema:\n  estado:\n    type: enum\n    values: [pendiente, activo]\n    required: true\n")
	if err := os.WriteFile(filepath.Join(root, ".stem"), stemContent, 0o644); err != nil {
		t.Fatal(err)
	}

	cache := NewStemCache()

	entries1, err := cache.WalkUp(filepath.Join(root, "doc.md"))
	if err != nil {
		t.Fatalf("first WalkUp: %v", err)
	}
	if len(entries1) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries1))
	}
	if cache.Hits() != 0 {
		t.Errorf("expected 0 hits on first call, got %d", cache.Hits())
	}

	entries2, err := cache.WalkUp(filepath.Join(root, "other.md"))
	if err != nil {
		t.Fatalf("second WalkUp: %v", err)
	}
	if len(entries2) != 1 {
		t.Fatalf("expected 1 entry, got %d", len(entries2))
	}
	if cache.Hits() != 1 {
		t.Errorf("expected 1 hit on second call, got %d", cache.Hits())
	}

	if entries1[0].Stem.Schema["estado"].Type != entries2[0].Stem.Schema["estado"].Type {
		t.Error("cached schema differs from original")
	}
}
