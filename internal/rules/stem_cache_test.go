package rules

import (
	"os"
	"path/filepath"
	"testing"
)

func TestStemCache_CachesRepeatedLookups(t *testing.T) {
	root := t.TempDir()
	stemContent := []byte("version: 2\nroot: true\nschema:\n  estado:\n    type: enum\n    values: [pendiente, activo]\n    required: true\n")
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

// TestStemCache_NoStemFound tests cache behavior when no .stem exists.
func TestStemCache_NoStemFound(t *testing.T) {
	root := t.TempDir()
	orphanDir := filepath.Join(root, "orphan")
	if err := os.MkdirAll(orphanDir, 0o755); err != nil {
		t.Fatal(err)
	}

	cache := NewStemCache()

	// First call: no .stem found
	entries1, err := cache.WalkUp(orphanDir)
	if err != ErrNoSchemaFound {
		t.Fatalf("expected ErrNoSchemaFound, got %v", err)
	}
	if len(entries1) > 0 {
		t.Errorf("expected empty entries, got %d", len(entries1))
	}

	// Second call: should also fail (no caching of positive results on negative hits)
	entries2, err := cache.WalkUp(orphanDir)
	if err != ErrNoSchemaFound {
		t.Fatalf("expected ErrNoSchemaFound on second call, got %v", err)
	}
	if len(entries2) > 0 {
		t.Errorf("expected empty entries on second call, got %d", len(entries2))
	}
}

// TestStemCache_MultiLevel tests caching with nested directories.
func TestStemCache_MultiLevel(t *testing.T) {
	root := t.TempDir()

	// Create nested structure
	level1 := filepath.Join(root, "l1")
	level2 := filepath.Join(level1, "l2")
	level3 := filepath.Join(level2, "l3")

	if err := os.MkdirAll(level3, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create .stem at level 1
	stemContent := []byte("version: 2\nroot: true\n")
	if err := os.WriteFile(filepath.Join(level1, ".stem"), stemContent, 0o644); err != nil {
		t.Fatal(err)
	}

	cache := NewStemCache()

	// Walk from level 3
	entries, err := cache.WalkUp(level3)
	if err != nil {
		t.Fatalf("WalkUp from level3: %v", err)
	}
	if len(entries) != 1 {
		t.Errorf("expected 1 entry from level3, got %d", len(entries))
	}

	initialHits := cache.Hits()

	// Walk again from level 2 (should hit cache for level2 and level1)
	entries2, err := cache.WalkUp(level2)
	if err != nil {
		t.Fatalf("WalkUp from level2: %v", err)
	}

	// Should have at least 1 more hit
	finalHits := cache.Hits()
	if finalHits <= initialHits {
		t.Errorf("expected cache hits to increase, was %d, now %d", initialHits, finalHits)
	}

	if len(entries2) != 1 {
		t.Errorf("expected 1 entry from level2, got %d", len(entries2))
	}
}

// TestStemCache_ParseError tests cache behavior when .stem has invalid YAML.
func TestStemCache_ParseError(t *testing.T) {
	root := t.TempDir()

	// Create invalid .stem
	badStem := []byte("version: 2\nroot: true\ninvalid yaml: [")
	if err := os.WriteFile(filepath.Join(root, ".stem"), badStem, 0o644); err != nil {
		t.Fatal(err)
	}

	cache := NewStemCache()

	// First call should error
	_, err := cache.WalkUp(root)
	if err == nil {
		t.Fatal("expected parse error")
	}

	// Second call should also error (errors are not cached as "found")
	_, err = cache.WalkUp(root)
	if err == nil {
		t.Fatal("expected parse error on second call")
	}
}
