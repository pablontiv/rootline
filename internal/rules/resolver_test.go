package rules

import (
	"os"
	"path/filepath"
	"testing"
)

// setupResolverTest creates a test directory tree with multiple .stem files
// at different levels, suitable for testing the resolver.
func setupResolverTest(t *testing.T) string {
	t.Helper()
	root := t.TempDir()

	// Create .git marker at root.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create root .stem with a base field.
	rootStem := filepath.Join(root, ".stem")
	rootContent := []byte(`version: 2
schema:
  title:
    type: string
    required: true
  status:
    type: enum
    values: [draft, published]
`)
	if err := os.WriteFile(rootStem, rootContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create docs/ directory with its own .stem that adds fields.
	docsDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(docsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	docsStem := filepath.Join(docsDir, ".stem")
	docsContent := []byte(`version: 2
schema:
  author:
    type: string
    required: true
  created_date:
    type: string
`)
	if err := os.WriteFile(docsStem, docsContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create docs/roadmap/ directory with its own .stem.
	roadmapDir := filepath.Join(docsDir, "roadmap")
	if err := os.MkdirAll(roadmapDir, 0o755); err != nil {
		t.Fatal(err)
	}
	roadmapStem := filepath.Join(roadmapDir, ".stem")
	roadmapContent := []byte(`version: 2
schema:
  milestone:
    type: string
  owner:
    type: string
    required: true
`)
	if err := os.WriteFile(roadmapStem, roadmapContent, 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestStemChain_RootToLeafOrdering(t *testing.T) {
	root := setupResolverTest(t)

	// Resolve a file in the deepest directory.
	targetPath := filepath.Join(root, "docs", "roadmap", "plan.md")

	entries, err := StemChain(targetPath, root)
	if err != nil {
		t.Fatalf("StemChain() error: %v", err)
	}

	// Should find 3 stems: root, docs, and roadmap (in that order).
	if len(entries) != 3 {
		t.Fatalf("got %d stems, want 3", len(entries))
	}

	// Verify root-to-leaf ordering.
	// entries[0] should be root .stem
	rootStemPath := filepath.Join(root, ".stem")
	if entries[0].Path != rootStemPath {
		t.Errorf("entries[0] = %s, want %s", entries[0].Path, rootStemPath)
	}

	// entries[1] should be docs/.stem
	docsPath := filepath.Join(root, "docs", ".stem")
	if entries[1].Path != docsPath {
		t.Errorf("entries[1] = %s, want %s", entries[1].Path, docsPath)
	}

	// entries[2] should be docs/roadmap/.stem
	roadmapPath := filepath.Join(root, "docs", "roadmap", ".stem")
	if entries[2].Path != roadmapPath {
		t.Errorf("entries[2] = %s, want %s", entries[2].Path, roadmapPath)
	}
}

func TestEffectiveSchema_MergesAllLevels(t *testing.T) {
	root := setupResolverTest(t)
	targetPath := filepath.Join(root, "docs", "roadmap", "plan.md")

	schema, err := EffectiveSchema(targetPath, root)
	if err != nil {
		t.Fatalf("EffectiveSchema() error: %v", err)
	}

	// Should have fields from all three stems.
	expectedFields := map[string]bool{
		"title":        true, // from root
		"status":       true, // from root
		"author":       true, // from docs
		"created_date": true, // from docs
		"milestone":    true, // from roadmap
		"owner":        true, // from roadmap
	}

	for name := range expectedFields {
		if _, ok := schema[name]; !ok {
			t.Errorf("schema missing field %q", name)
		}
	}

	for name := range schema {
		if !expectedFields[name] {
			t.Errorf("unexpected field in schema: %q", name)
		}
	}
}

func TestResolveForRecord_ProvenianceTracking(t *testing.T) {
	root := setupResolverTest(t)
	targetPath := filepath.Join(root, "docs", "roadmap", "plan.md")

	res, err := Resolve(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveForRecord() error: %v", err)
	}

	// Verify chain is root-to-leaf.
	if len(res.Chain) != 3 {
		t.Fatalf("got %d stems in chain, want 3", len(res.Chain))
	}

	// Check provenance: each field should map to its source .stem.
	rootStemPath := filepath.Join(root, ".stem")
	docsStemPath := filepath.Join(root, "docs", ".stem")
	roadmapStemPath := filepath.Join(root, "docs", "roadmap", ".stem")

	testCases := []struct {
		field  string
		source string
	}{
		{"title", rootStemPath},
		{"status", rootStemPath},
		{"author", docsStemPath},
		{"created_date", docsStemPath},
		{"milestone", roadmapStemPath},
		{"owner", roadmapStemPath},
	}

	for _, tc := range testCases {
		if got := res.Provenance[tc.field]; got != tc.source {
			t.Errorf("provenance[%q] = %s, want %s", tc.field, got, tc.source)
		}
	}
}

func TestResolveForRecord_ClosestAndRootMost(t *testing.T) {
	root := setupResolverTest(t)
	targetPath := filepath.Join(root, "docs", "roadmap", "plan.md")

	res, err := Resolve(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveForRecord() error: %v", err)
	}

	// ClosestStem should be roadmap/.stem (leaf-most).
	closest := res.ClosestStem()
	if closest == nil {
		t.Fatal("ClosestStem() returned nil")
	}
	roadmapStemPath := filepath.Join(root, "docs", "roadmap", ".stem")
	if closest.Path != roadmapStemPath {
		t.Errorf("ClosestStem().Path = %s, want %s", closest.Path, roadmapStemPath)
	}

	// RootMostStem should be root/.stem (root-most).
	rootmost := res.RootMostStem()
	if rootmost == nil {
		t.Fatal("RootMostStem() returned nil")
	}
	rootStemPath := filepath.Join(root, ".stem")
	if rootmost.Path != rootStemPath {
		t.Errorf("RootMostStem().Path = %s, want %s", rootmost.Path, rootStemPath)
	}
}

func TestResolveForRecord_EmptyChain(t *testing.T) {
	// Test that ClosestStem/RootMostStem handle empty chains.
	res := &Resolution{
		Path:  "/some/path",
		Chain: []StemEntry{},
	}

	if closest := res.ClosestStem(); closest != nil {
		t.Error("ClosestStem() on empty chain should return nil")
	}
	if rootmost := res.RootMostStem(); rootmost != nil {
		t.Error("RootMostStem() on empty chain should return nil")
	}
}

func TestResolveForRecord_SingleStem(t *testing.T) {
	root := t.TempDir()

	// Create .git marker.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create only one .stem at root.
	stemPath := filepath.Join(root, ".stem")
	stemContent := []byte(`version: 2
schema:
  title:
    type: string
`)
	if err := os.WriteFile(stemPath, stemContent, 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(root, "doc.md")

	res, err := Resolve(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveForRecord() error: %v", err)
	}

	// Should have exactly one stem.
	if len(res.Chain) != 1 {
		t.Fatalf("got %d stems, want 1", len(res.Chain))
	}

	// ClosestStem and RootMostStem should both point to the same stem.
	closest := res.ClosestStem()
	rootmost := res.RootMostStem()

	if closest.Path != rootmost.Path {
		t.Errorf("with single stem, ClosestStem and RootMostStem should be the same")
	}
}

func TestResolveForRecord_FieldOverride(t *testing.T) {
	root := t.TempDir()

	// Create .git marker.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create root .stem with a field.
	rootStem := filepath.Join(root, ".stem")
	rootContent := []byte(`version: 2
schema:
  status:
    type: enum
    values: [draft, published]
`)
	if err := os.WriteFile(rootStem, rootContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create sub/.stem that overrides the field with different values.
	subDir := filepath.Join(root, "sub")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subStem := filepath.Join(subDir, ".stem")
	subContent := []byte(`version: 2
schema:
  status:
    type: enum
    values: [open, closed, merged]
`)
	if err := os.WriteFile(subStem, subContent, 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(subDir, "file.md")

	res, err := Resolve(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveForRecord() error: %v", err)
	}

	// Provenance should point to sub/.stem (the override).
	subStemPath := filepath.Join(subDir, ".stem")
	if got := res.Provenance["status"]; got != subStemPath {
		t.Errorf("provenance[status] = %s, want %s", got, subStemPath)
	}

	// Effective schema should have the overridden values.
	if len(res.EffectiveSchema["status"].Values) != 3 {
		t.Errorf("expected 3 enum values after override, got %d", len(res.EffectiveSchema["status"].Values))
	}
}
