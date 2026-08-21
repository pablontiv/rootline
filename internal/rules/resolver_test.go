package rules

import (
	"os"
	"path/filepath"
	"reflect"
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

// ====== Monotonic Constraint Tests ======

func TestResolveLayered_PopulatesLayers(t *testing.T) {
	root := setupResolverTest(t)
	targetPath := filepath.Join(root, "docs", "roadmap", "plan.md")

	lr, err := ResolveLayered(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	// Layers should be populated and compatible chains should have no conflicts.
	if len(lr.Layers) == 0 {
		t.Fatal("expected Layers to be populated")
	}
	if len(lr.Conflicts) != 0 {
		t.Fatalf("expected no Conflicts for compatible chain, got %d", len(lr.Conflicts))
	}
}

func TestResolveLayered_MonotonicStringToEnumNarrowing(t *testing.T) {
	root := t.TempDir()

	// Create .git marker.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create root .stem with string type.
	rootStem := filepath.Join(root, ".stem")
	rootContent := []byte(`version: 2
schema:
  status:
    type: string
`)
	if err := os.WriteFile(rootStem, rootContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create child .stem with enum type (narrowing).
	subDir := filepath.Join(root, "tasks")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subStem := filepath.Join(subDir, ".stem")
	subContent := []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, active, completed]
`)
	if err := os.WriteFile(subStem, subContent, 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(subDir, "task.md")

	// This narrowing should be valid (string→enum is allowed narrowing).
	lr, err := ResolveLayered(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	// Should have no conflicts for this valid narrowing.
	if len(lr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts for valid string→enum narrowing, got %d", len(lr.Conflicts))
	}
}

func TestResolveLayered_MonotonicEnumExtensionRejected(t *testing.T) {
	root := t.TempDir()

	// Create .git marker.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create root .stem with enum values.
	rootStem := filepath.Join(root, ".stem")
	rootContent := []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, completed]
`)
	if err := os.WriteFile(rootStem, rootContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create child .stem that extends enum (invalid in monotonic mode).
	subDir := filepath.Join(root, "tasks")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subStem := filepath.Join(subDir, ".stem")
	subContent := []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, in_progress, completed]
`)
	if err := os.WriteFile(subStem, subContent, 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(subDir, "task.md")

	lr, err := ResolveLayered(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	// Should have a conflict for enum extension.
	if len(lr.Conflicts) == 0 {
		t.Fatal("expected conflict for enum extension in monotonic mode")
	}

	// Find the conflict.
	found := false
	for _, conflict := range lr.Conflicts {
		if conflict.Field == "status.values" && conflict.Operation == "extension" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected status.values extension conflict, got conflicts: %+v", lr.Conflicts)
	}
}

func TestResolveLayered_MonotonicEnumNarrowing(t *testing.T) {
	root := t.TempDir()

	// Create .git marker.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create root .stem with enum values.
	rootStem := filepath.Join(root, ".stem")
	rootContent := []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, active, completed, archived]
`)
	if err := os.WriteFile(rootStem, rootContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create child .stem that narrows enum (valid in monotonic mode).
	subDir := filepath.Join(root, "tasks")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subStem := filepath.Join(subDir, ".stem")
	subContent := []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, active, completed]
`)
	if err := os.WriteFile(subStem, subContent, 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(subDir, "task.md")

	lr, err := ResolveLayered(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	// Should have no conflicts for valid enum narrowing.
	if len(lr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts for enum narrowing, got %d: %+v", len(lr.Conflicts), lr.Conflicts)
	}
}

func TestResolveLayered_MonotonicRequiredLoosening(t *testing.T) {
	root := t.TempDir()

	// Create .git marker.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create root .stem with required=true.
	rootStem := filepath.Join(root, ".stem")
	rootContent := []byte(`version: 2
schema:
  title:
    type: string
    required: true
`)
	if err := os.WriteFile(rootStem, rootContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create child .stem that loosens required (invalid in monotonic mode).
	subDir := filepath.Join(root, "docs")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subStem := filepath.Join(subDir, ".stem")
	subContent := []byte(`version: 2
schema:
  title:
    type: string
    required: false
`)
	if err := os.WriteFile(subStem, subContent, 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(subDir, "doc.md")

	lr, err := ResolveLayered(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	// Should have a conflict for required loosening.
	if len(lr.Conflicts) == 0 {
		t.Fatal("expected conflict for required loosening in monotonic mode")
	}

	// Find the conflict.
	found := false
	for _, conflict := range lr.Conflicts {
		if conflict.Field == "title.required" && conflict.Operation == "conflict" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected title.required conflict, got conflicts: %+v", lr.Conflicts)
	}
}

func TestResolveLayered_MonotonicSeverityLoosening(t *testing.T) {
	root := t.TempDir()

	// Create .git marker.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create root .stem with severity=error.
	rootStem := filepath.Join(root, ".stem")
	rootContent := []byte(`version: 2
schema:
  id:
    type: string
    severity: error
`)
	if err := os.WriteFile(rootStem, rootContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create child .stem that loosens severity (invalid in monotonic mode).
	subDir := filepath.Join(root, "tasks")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subStem := filepath.Join(subDir, ".stem")
	subContent := []byte(`version: 2
schema:
  id:
    type: string
    severity: warn
`)
	if err := os.WriteFile(subStem, subContent, 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(subDir, "task.md")

	lr, err := ResolveLayered(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	// Should have a conflict for severity loosening.
	if len(lr.Conflicts) == 0 {
		t.Fatal("expected conflict for severity loosening in monotonic mode")
	}

	found := false
	for _, conflict := range lr.Conflicts {
		if conflict.Field == "id.severity" && conflict.Operation == "conflict" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected id.severity conflict, got conflicts: %+v", lr.Conflicts)
	}
}

func TestResolveLayered_MonotonicStructuralTightening(t *testing.T) {
	root := t.TempDir()

	// Create .git marker.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create root .stem with min_children=1.
	rootStem := filepath.Join(root, ".stem")
	rootContent := []byte(`version: 2
structural:
  subdirs:
    min_children: 1
`)
	if err := os.WriteFile(rootStem, rootContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create child .stem that tightens min_children (valid in monotonic mode).
	subDir := filepath.Join(root, "epics")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subStem := filepath.Join(subDir, ".stem")
	subContent := []byte(`version: 2
structural:
  subdirs:
    min_children: 2
`)
	if err := os.WriteFile(subStem, subContent, 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(subDir, "epic.md")

	lr, err := ResolveLayered(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	// Should have no conflicts for structural tightening.
	if len(lr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts for structural tightening, got %d: %+v", len(lr.Conflicts), lr.Conflicts)
	}
}

func TestResolveLayered_MonotonicStructuralLoosening(t *testing.T) {
	root := t.TempDir()

	// Create .git marker.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Create root .stem with min_children=2.
	rootStem := filepath.Join(root, ".stem")
	rootContent := []byte(`version: 2
structural:
  subdirs:
    min_children: 2
`)
	if err := os.WriteFile(rootStem, rootContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Create child .stem that loosens min_children (invalid in monotonic mode).
	subDir := filepath.Join(root, "epics")
	if err := os.MkdirAll(subDir, 0o755); err != nil {
		t.Fatal(err)
	}
	subStem := filepath.Join(subDir, ".stem")
	subContent := []byte(`version: 2
structural:
  subdirs:
    min_children: 1
`)
	if err := os.WriteFile(subStem, subContent, 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(subDir, "epic.md")

	lr, err := ResolveLayered(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	// Should have a conflict for structural loosening.
	if len(lr.Conflicts) == 0 {
		t.Fatal("expected conflict for structural loosening in monotonic mode")
	}

	found := false
	for _, conflict := range lr.Conflicts {
		if conflict.Field == "structural.subdirs.min_children" && conflict.Operation == "conflict" {
			found = true
			break
		}
	}
	if !found {
		t.Errorf("expected structural.subdirs.min_children conflict, got conflicts: %+v", lr.Conflicts)
	}
}

func TestResolveLayered_CumulativeSourceChangeRejected(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
schema:
  summary:
    type: string
    source: body.section["## Summary"]
`))
	middle := filepath.Join(root, "middle")
	mustWriteStemTestFile(t, filepath.Join(middle, ".stem"), []byte(`version: 2
schema:
  summary:
    type: string
`))
	leaf := filepath.Join(middle, "leaf")
	mustWriteStemTestFile(t, filepath.Join(leaf, ".stem"), []byte(`version: 2
schema:
  summary:
    type: string
    source: body.section["## Context"]
`))

	lr, err := ResolveLayered(filepath.Join(leaf, "doc.md"), root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	want := LayerConstraint{
		StemPath:  filepath.Join(leaf, ".stem"),
		Field:     "summary.source",
		Value:     `source changes from "body.section[\"## Summary\"]" to "body.section[\"## Context\"]"`,
		Operation: "conflict",
	}
	assertLayerConflicts(t, lr.Conflicts, []LayerConstraint{want})
}

func TestResolveLayered_CumulativeSourceRemovalOwnedByMiddle(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
schema:
  summary:
    type: string
    source: body.section["## Summary"]
`))
	middle := filepath.Join(root, "middle")
	mustWriteStemTestFile(t, filepath.Join(middle, ".stem"), []byte(`version: 2
schema:
  summary:
    type: string
    source: null
`))
	leaf := filepath.Join(middle, "leaf")
	mustWriteStemTestFile(t, filepath.Join(leaf, ".stem"), []byte(`version: 2
schema:
  summary:
    type: string
`))

	lr, err := ResolveLayered(filepath.Join(leaf, "doc.md"), root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	want := LayerConstraint{
		StemPath:  filepath.Join(middle, ".stem"),
		Field:     "summary.source",
		Value:     `source removes inherited source "body.section[\"## Summary\"]"`,
		Operation: "conflict",
	}
	assertLayerConflicts(t, lr.Conflicts, []LayerConstraint{want})
	if got := lr.EffectiveSchema["summary"].Extract; got != "" {
		t.Fatalf("effective summary source = %q, want explicit removal to persist", got)
	}
}

func TestResolveLayered_CumulativeRequiredLooseningRejected(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
schema:
  title:
    type: string
    required: true
`))
	middle := filepath.Join(root, "middle")
	mustWriteStemTestFile(t, filepath.Join(middle, ".stem"), []byte(`version: 2
schema:
  other:
    type: string
`))
	leaf := filepath.Join(middle, "leaf")
	mustWriteStemTestFile(t, filepath.Join(leaf, ".stem"), []byte(`version: 2
schema:
  title:
    type: string
    required: false
`))

	lr, err := ResolveLayered(filepath.Join(leaf, "doc.md"), root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	if !hasLayerConflict(lr, "title.required", "conflict") {
		t.Fatalf("expected cumulative required conflict, got %+v", lr.Conflicts)
	}
}

func hasLayerConflict(lr *LayeredResolution, field, operation string) bool {
	for _, conflict := range lr.Conflicts {
		if conflict.Field == field && conflict.Operation == operation {
			return true
		}
	}
	return false
}

func assertLayerConflicts(t *testing.T, got, want []LayerConstraint) {
	t.Helper()
	if len(got) != len(want) {
		t.Fatalf("conflict count = %d, want %d\ngot:  %+v\nwant: %+v", len(got), len(want), got, want)
	}
	for i := range want {
		if got[i].StemPath != want[i].StemPath || got[i].Field != want[i].Field || got[i].Operation != want[i].Operation || got[i].Value != want[i].Value {
			t.Fatalf("conflict[%d] = %+v, want %+v", i, got[i], want[i])
		}
	}
}

func TestResolveLayered_ThreeLevelHierarchy(t *testing.T) {
	root := t.TempDir()

	// Create .git marker.
	gitDir := filepath.Join(root, ".git")
	if err := os.MkdirAll(gitDir, 0o755); err != nil {
		t.Fatal(err)
	}

	// Level 1: root .stem with string type.
	rootStem := filepath.Join(root, ".stem")
	rootContent := []byte(`version: 2
schema:
  status:
    type: string
`)
	if err := os.WriteFile(rootStem, rootContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Level 2: epics .stem narrows to enum.
	epicsDir := filepath.Join(root, "epics")
	if err := os.MkdirAll(epicsDir, 0o755); err != nil {
		t.Fatal(err)
	}
	epicsStem := filepath.Join(epicsDir, ".stem")
	epicsContent := []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, active, completed, archived]
`)
	if err := os.WriteFile(epicsStem, epicsContent, 0o644); err != nil {
		t.Fatal(err)
	}

	// Level 3: tasks .stem further narrows enum (valid narrowing).
	tasksDir := filepath.Join(epicsDir, "tasks")
	if err := os.MkdirAll(tasksDir, 0o755); err != nil {
		t.Fatal(err)
	}
	tasksStem := filepath.Join(tasksDir, ".stem")
	tasksContent := []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, active, completed]
`)
	if err := os.WriteFile(tasksStem, tasksContent, 0o644); err != nil {
		t.Fatal(err)
	}

	targetPath := filepath.Join(tasksDir, "task.md")

	lr, err := ResolveLayered(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}

	// Should have 3 entries in chain.
	if len(lr.Chain) != 3 {
		t.Fatalf("expected 3 stems in chain, got %d", len(lr.Chain))
	}

	// Should have no conflicts (all narrowing operations are valid).
	if len(lr.Conflicts) != 0 {
		t.Fatalf("expected no conflicts for three-level valid narrowing, got %d: %+v", len(lr.Conflicts), lr.Conflicts)
	}
}

func TestResolveFromEntriesMatchesFilesystem(t *testing.T) {
	root := setupResolverTest(t)
	targetPath := filepath.Join(root, "docs", "roadmap", "plan.md")

	entries, err := WalkUp(targetPath)
	if err != nil {
		t.Fatalf("WalkUp() error: %v", err)
	}
	filesystem, err := Resolve(targetPath, root)
	if err != nil {
		t.Fatalf("Resolve() error: %v", err)
	}
	explicit, err := resolveFromEntries(targetPath, entries)
	if err != nil {
		t.Fatalf("resolveFromEntries() error: %v", err)
	}

	assertResolutionEquivalent(t, explicit, filesystem)
}

func TestResolveLayeredFromEntriesMatchesFilesystem(t *testing.T) {
	root := t.TempDir()
	mustWriteStemTestFile(t, filepath.Join(root, ".stem"), []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, done]
  title:
    type: string
    required: true
`))
	childDir := filepath.Join(root, "child")
	mustWriteStemTestFile(t, filepath.Join(childDir, ".stem"), []byte(`version: 2
schema:
  status:
    type: enum
    values: [pending, blocked, done]
  title:
    type: string
    required: false
`))
	targetPath := filepath.Join(childDir, "record.md")

	entries, err := WalkUp(targetPath)
	if err != nil {
		t.Fatalf("WalkUp() error: %v", err)
	}
	filesystem, err := ResolveLayered(targetPath, root)
	if err != nil {
		t.Fatalf("ResolveLayered() error: %v", err)
	}
	explicit, err := resolveLayeredFromEntries(targetPath, entries)
	if err != nil {
		t.Fatalf("resolveLayeredFromEntries() error: %v", err)
	}

	assertResolutionEquivalent(t, explicit.Resolution, filesystem.Resolution)
	if !reflect.DeepEqual(explicit.Layers, filesystem.Layers) {
		t.Fatalf("layers mismatch\ngot:  %+v\nwant: %+v", explicit.Layers, filesystem.Layers)
	}
	if !reflect.DeepEqual(explicit.Conflicts, filesystem.Conflicts) {
		t.Fatalf("conflicts mismatch\ngot:  %+v\nwant: %+v", explicit.Conflicts, filesystem.Conflicts)
	}
}

func assertResolutionEquivalent(t *testing.T, got, want *Resolution) {
	t.Helper()
	if got.Path != want.Path {
		t.Fatalf("path = %q, want %q", got.Path, want.Path)
	}
	if !reflect.DeepEqual(stemEntryPaths(got.Chain), stemEntryPaths(want.Chain)) {
		t.Fatalf("chain paths = %+v, want %+v", stemEntryPaths(got.Chain), stemEntryPaths(want.Chain))
	}
	if !reflect.DeepEqual(got.EffectiveSchema, want.EffectiveSchema) {
		t.Fatalf("effective schema mismatch\ngot:  %+v\nwant: %+v", got.EffectiveSchema, want.EffectiveSchema)
	}
	if !reflect.DeepEqual(got.Provenance, want.Provenance) {
		t.Fatalf("provenance mismatch\ngot:  %+v\nwant: %+v", got.Provenance, want.Provenance)
	}
	if !reflect.DeepEqual(got.EffectiveStem, want.EffectiveStem) {
		t.Fatalf("effective stem mismatch\ngot:  %+v\nwant: %+v", got.EffectiveStem, want.EffectiveStem)
	}
}

func stemEntryPaths(entries []StemEntry) []string {
	paths := make([]string, 0, len(entries))
	for _, entry := range entries {
		paths = append(paths, entry.Path)
	}
	return paths
}
