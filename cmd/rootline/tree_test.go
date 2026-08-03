package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// setupTreeProject creates a temp directory with .git marker, .stem, and
// markdown files for tree command testing. It adds a boundary marker only
// if the caller provides one explicitly in the files map.
func setupTreeProject(t *testing.T, files map[string]string) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	for relPath, content := range files {
		absPath := filepath.Join(root, relPath)
		if err := os.MkdirAll(filepath.Dir(absPath), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(absPath, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}

	declareTestBoundary(t, root)
	return root
}

func TestTreeCmd_ScopeFiltering_ExcludesOutOfScope(t *testing.T) {
	// Root .stem scopes only to docs/ subdirectory patterns via a nested stem.
	// Files at root level without a matching .stem scope should be excluded.
	root := setupTreeProject(t, map[string]string{
		"docs/.stem":    "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n",
		"docs/task1.md": "---\nestado: Pending\n---\n# Task 1\n",
		"docs/task2.md": "---\nestado: Completed\n---\n# Task 2\n",
		// These files at root have no .stem => no scope => should be excluded
		// when scope resolver returns nil (no stem found)
		"CHANGELOG.md":    "---\n---\n# Changelog\n",
		"CONTRIBUTING.md": "---\n---\n# Contributing\n",
	})

	out, err := runCmd(t, "tree", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	// Only docs/task1.md and docs/task2.md should be included.
	if result.Root.Total != 2 {
		t.Errorf("total = %d, want 2 (only scoped files)", result.Root.Total)
	}
}

func TestTreeCmd_ScopeFiltering_SubdirOnly(t *testing.T) {
	// When running tree on a subdirectory, only files matching that
	// subdirectory's scope should appear.
	root := setupTreeProject(t, map[string]string{
		"docs/.stem":        "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n",
		"docs/epic1/F01.md": "---\nestado: Pending\n---\n# F01\n",
		"docs/epic1/F02.md": "---\nestado: Completed\n---\n# F02\n",
	})

	out, err := runCmd(t, "tree", filepath.Join(root, "docs"))
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	if result.Root.Total != 2 {
		t.Errorf("total = %d, want 2", result.Root.Total)
	}
}

func TestTreeCmd_ScopeFiltering_ASCIIOutput(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		"docs/.stem":    "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n",
		"docs/task1.md": "---\nestado: Pending\n---\n# Task 1\n",
		"CHANGELOG.md":  "---\n---\n# Changelog\n",
	})

	out, err := runCmd(t, "tree", root, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(out, "CHANGELOG.md") {
		t.Errorf("tree ASCII output should NOT contain CHANGELOG.md (out of scope)\noutput: %s", out)
	}
	if !strings.Contains(out, "task1.md") {
		t.Errorf("tree ASCII output should contain task1.md\noutput: %s", out)
	}
}

func TestTreeCmd_JSONStructure(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		".stem":    "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n",
		"a.md":     "---\nestado: Completed\n---\n",
		"sub/b.md": "---\nestado: Pending\n---\n",
	})

	out, err := runCmd(t, "tree", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	if result.Version != 2 {
		t.Errorf("version = %d, want 2", result.Version)
	}
	if result.Kind != "rootline/tree" {
		t.Errorf("kind = %q, want rootline/tree", result.Kind)
	}
	if result.Root.Total != 2 {
		t.Errorf("root total = %d, want 2", result.Root.Total)
	}
}

func TestTreeCmd_EmptyDirectory(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\n",
	})

	out, err := runCmd(t, "tree", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	if result.Root.Total != 0 {
		t.Errorf("total = %d, want 0 for empty directory", result.Root.Total)
	}
}

func TestTreeCmd_DefaultsToCurrentDir(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending]\n    required: true\n",
		"doc.md": "---\nestado: Pending\n---\n",
	})

	mustChdir(t, root)

	out, err := runCmd(t, "tree")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	if result.Root.Total != 1 {
		t.Errorf("total = %d, want 1", result.Root.Total)
	}
}

func TestTreeCmd_ASCIIRendering(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		".stem":    "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n",
		"a.md":     "---\nestado: Completed\n---\n",
		"sub/b.md": "---\nestado: Pending\n---\n",
		"sub/c.md": "---\nestado: Completed\n---\n",
	})

	out, err := runCmd(t, "tree", root, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Check root line has total counts.
	lines := strings.Split(strings.TrimSpace(out), "\n")
	if len(lines) == 0 {
		t.Fatal("expected non-empty output")
	}
	if !strings.Contains(lines[0], "[3]") {
		t.Errorf("root line = %q, want [3]", lines[0])
	}

	// Should contain tree connectors.
	if !strings.Contains(out, "├──") && !strings.Contains(out, "└──") {
		t.Errorf("expected tree connectors in output:\n%s", out)
	}

	// Leaf nodes show estado.
	if !strings.Contains(out, "[Completed]") {
		t.Errorf("expected [Completed] in leaf output:\n%s", out)
	}
	if !strings.Contains(out, "[Pending]") {
		t.Errorf("expected [Pending] in leaf output:\n%s", out)
	}
}

func TestTreeCmd_NoEstadoShowsDash(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\n",
		"doc.md": "---\ntitle: Hello\n---\n",
	})

	out, err := runCmd(t, "tree", root, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	// Files without estado show "—" (em dash).
	if !strings.Contains(out, "[\xe2\x80\x94]") {
		t.Errorf("expected em-dash for missing estado, got:\n%s", out)
	}
}

func TestTreeCmd_DeepNesting(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		".stem":             "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n",
		"a/b/c/deep.md":     "---\nestado: Pending\n---\n",
		"a/b/c/d/deeper.md": "---\nestado: Completed\n---\n",
	})

	out, err := runCmd(t, "tree", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	if result.Root.Total != 2 {
		t.Errorf("total = %d, want 2", result.Root.Total)
	}

	// Verify nesting: root -> a -> b -> c -> ...
	if len(result.Root.Children) != 1 || result.Root.Children[0].Name != "a" {
		t.Errorf("expected single child 'a', got %v", result.Root.Children)
	}
}

// --- Unit tests for buildTree internals ---

func TestBuildTree_Basic(t *testing.T) {
	records := []*extract.Record{
		{Path: "doc1.md", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "doc2.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	root := buildTree(records, "project")
	if root.Name != "project" {
		t.Errorf("root name = %q, want project", root.Name)
	}
	if root.Total != 2 {
		t.Errorf("total = %d, want 2", root.Total)
	}
	if len(root.Children) != 2 {
		t.Errorf("children = %d, want 2", len(root.Children))
	}
}

func TestBuildTree_WithSubdirs(t *testing.T) {
	records := []*extract.Record{
		{Path: "sub/a.md", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "sub/b.md", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "other/c.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	root := buildTree(records, "root")
	if root.Total != 3 {
		t.Errorf("total = %d, want 3", root.Total)
	}

	// Should have 2 directory children: other and sub (sorted).
	if len(root.Children) != 2 {
		t.Fatalf("children = %d, want 2", len(root.Children))
	}
	if root.Children[0].Name != "other" {
		t.Errorf("first child = %q, want other", root.Children[0].Name)
	}
	if root.Children[1].Name != "sub" {
		t.Errorf("second child = %q, want sub", root.Children[1].Name)
	}
}

func TestBuildTree_NoEstado(t *testing.T) {
	records := []*extract.Record{
		{Path: "doc.md", Frontmatter: map[string]any{"title": "Hello"}},
	}

	root := buildTree(records, "root")
	if root.Total != 1 {
		t.Errorf("total = %d, want 1", root.Total)
	}
}

func TestBuildTree_EmptyRecords(t *testing.T) {
	root := buildTree(nil, "empty")
	if root.Total != 0 {
		t.Errorf("total = %d, want 0", root.Total)
	}
	if len(root.Children) != 0 {
		t.Errorf("children = %d, want 0", len(root.Children))
	}
}

func TestFindChild_Found(t *testing.T) {
	parent := &treeNode{
		Children: []*treeNode{
			{Name: "dir1"},
			{Name: "dir2"},
		},
	}
	child := findChild(parent, "dir1")
	if child == nil {
		t.Fatal("expected to find child dir1")
	}
	if child.Name != "dir1" {
		t.Errorf("name = %q, want dir1", child.Name)
	}
}

func TestFindChild_NotFound(t *testing.T) {
	parent := &treeNode{
		Children: []*treeNode{
			{Name: "dir1"},
		},
	}
	child := findChild(parent, "nonexistent")
	if child != nil {
		t.Errorf("expected nil for nonexistent child, got %v", child)
	}
}

func TestFindChild_SkipsLeaves(t *testing.T) {
	parent := &treeNode{
		Children: []*treeNode{
			{Name: "file.md", IsLeaf: true},
		},
	}
	child := findChild(parent, "file.md")
	if child != nil {
		t.Errorf("expected nil for leaf node, got %v", child)
	}
}

func TestRenderASCII_EmptyRoot(t *testing.T) {
	root := &treeNode{Name: "empty", Total: 0}
	lines := renderASCII(root, "", nil)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if lines[0] != "empty [0]" {
		t.Errorf("line = %q, want 'empty [0]'", lines[0])
	}
}

func TestRenderASCII_NonRootPrefix(t *testing.T) {
	node := &treeNode{Name: "sub", Total: 1}
	lines := renderASCII(node, "  ", nil)
	// Non-root calls return empty since prefix != ""
	if len(lines) != 0 {
		t.Errorf("expected 0 lines for non-root prefix, got %d", len(lines))
	}
}

func TestRenderChild_Leaf(t *testing.T) {
	leaf := &treeNode{Name: "doc.md", IsLeaf: true, Frontmatter: map[string]any{"estado": "Pending"}}
	lines := renderChild(leaf, "", false, nil)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "doc.md") {
		t.Errorf("line = %q, missing doc.md", lines[0])
	}
	if !strings.Contains(lines[0], "[Pending]") {
		t.Errorf("line = %q, missing [Pending]", lines[0])
	}
	if !strings.Contains(lines[0], "├──") {
		t.Errorf("line = %q, expected ├── connector", lines[0])
	}
}

func TestRenderChild_LastLeaf(t *testing.T) {
	leaf := &treeNode{Name: "last.md", IsLeaf: true, Frontmatter: map[string]any{"estado": "Completed"}}
	lines := renderChild(leaf, "", true, nil)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	if !strings.Contains(lines[0], "└──") {
		t.Errorf("line = %q, expected └── connector for last child", lines[0])
	}
}

func TestRenderChild_Directory(t *testing.T) {
	dir := &treeNode{
		Name:  "subdir",
		Total: 2,
		Children: []*treeNode{
			{Name: "a.md", IsLeaf: true, Frontmatter: map[string]any{"estado": "Completed"}, Total: 1},
			{Name: "b.md", IsLeaf: true, Frontmatter: map[string]any{"estado": "Pending"}, Total: 1},
		},
	}
	lines := renderChild(dir, "", false, nil)
	if len(lines) != 3 {
		t.Fatalf("expected 3 lines, got %d: %v", len(lines), lines)
	}
	if !strings.Contains(lines[0], "subdir [2]") {
		t.Errorf("dir line = %q, want subdir [2]", lines[0])
	}
}

func TestRenderChild_LeafNoEstado(t *testing.T) {
	leaf := &treeNode{Name: "doc.md", IsLeaf: true, Frontmatter: map[string]any{}}
	lines := renderChild(leaf, "", true, nil)
	if len(lines) != 1 {
		t.Fatalf("expected 1 line, got %d", len(lines))
	}
	// Empty estado shows em-dash.
	if !strings.Contains(lines[0], "[\xe2\x80\x94]") {
		t.Errorf("line = %q, expected em-dash for missing estado", lines[0])
	}
}

func TestPropagateCounts_NestedDirs(t *testing.T) {
	root := &treeNode{
		Name: "root",
		Children: []*treeNode{
			{
				Name: "sub",
				Children: []*treeNode{
					{Name: "a.md", IsLeaf: true, Total: 1},
					{Name: "b.md", IsLeaf: true, Total: 1},
				},
			},
			{Name: "c.md", IsLeaf: true, Total: 1},
		},
	}

	propagateCounts(root)

	if root.Total != 3 {
		t.Errorf("root total = %d, want 3", root.Total)
	}

	// Find the sub node (sorting puts "c.md" before "sub" alphabetically).
	var subNode *treeNode
	for _, c := range root.Children {
		if c.Name == "sub" {
			subNode = c
		}
	}
	if subNode == nil {
		t.Fatal("sub node not found")
	}
	if subNode.Total != 2 {
		t.Errorf("sub total = %d, want 2", subNode.Total)
	}
}

func TestPropagateCounts_SortsByName(t *testing.T) {
	root := &treeNode{
		Name: "root",
		Children: []*treeNode{
			{Name: "zebra.md", IsLeaf: true, Total: 1},
			{Name: "alpha.md", IsLeaf: true, Total: 1},
			{Name: "mid.md", IsLeaf: true, Total: 1},
		},
	}

	propagateCounts(root)

	if root.Children[0].Name != "alpha.md" {
		t.Errorf("first child = %q, want alpha.md", root.Children[0].Name)
	}
	if root.Children[1].Name != "mid.md" {
		t.Errorf("second child = %q, want mid.md", root.Children[1].Name)
	}
	if root.Children[2].Name != "zebra.md" {
		t.Errorf("third child = %q, want zebra.md", root.Children[2].Name)
	}
}

func TestTreeCmd_FieldExtraction(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending]\n    required: true\n",
		"doc.md": "---\nestado: Pending\n---\n",
	})

	out, err := runCmd(t, "tree", root, "--field", "root.total")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.TrimSpace(out) != "1" {
		t.Errorf("output = %q, want 1", strings.TrimSpace(out))
	}
}

func TestTreeCmd_RestrictedScopePattern(t *testing.T) {
	// .stem with a restrictive scope pattern that only matches T*.md files.
	root := setupTreeProject(t, map[string]string{
		".stem":        "version: 2\nscope:\n  match: \"T*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n",
		"T001.md":      "---\nestado: Pending\n---\n",
		"T002.md":      "---\nestado: Completed\n---\n",
		"README.md":    "---\nestado: Pending\n---\n",
		"CHANGELOG.md": "---\n---\n# Changelog\n",
	})

	out, err := runCmd(t, "tree", root)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	// Only T001.md and T002.md should match the T*.md scope.
	if result.Root.Total != 2 {
		t.Errorf("total = %d, want 2 (only T*.md files)", result.Root.Total)
	}

	// Verify in ASCII output as well.
	asciiOut, err := runCmd(t, "tree", root, "-o", "table")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if strings.Contains(asciiOut, "README.md") {
		t.Errorf("ASCII output should NOT contain README.md:\n%s", asciiOut)
	}
	if strings.Contains(asciiOut, "CHANGELOG.md") {
		t.Errorf("ASCII output should NOT contain CHANGELOG.md:\n%s", asciiOut)
	}
	if !strings.Contains(asciiOut, "T001.md") {
		t.Errorf("ASCII output should contain T001.md:\n%s", asciiOut)
	}
}

// --- Tree --where tests ---

func TestTreeCmd_WhereFilters(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n  tipo:\n    type: string\n",
		"a.md":  "---\nestado: Completed\ntipo: test\n---\n",
		"b.md":  "---\nestado: Pending\ntipo: prod\n---\n",
		"c.md":  "---\nestado: Completed\ntipo: prod\n---\n",
	})

	out, err := runCmd(t, "tree", root, "--where", "estado == 'Pending'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	if result.Root.Total != 1 {
		t.Errorf("total = %d, want 1 (only Pending)", result.Root.Total)
	}
}

func TestTreeCmd_WhereInvalidExpr(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		".stem":  "version: 2\nscope:\n  match: \"*.md\"\n",
		"doc.md": "---\ntitle: Test\n---\n",
	})

	_, err := runCmd(t, "tree", root, "--where", "== bad syntax")
	if err == nil {
		t.Fatal("expected error for invalid where expression")
	}
}

func TestTreeCmd_WhereMultiple(t *testing.T) {
	root := setupTreeProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n  estado:\n    type: enum\n    values: [Pending, Completed]\n    required: true\n  tipo:\n    type: string\n",
		"a.md":  "---\nestado: Pending\ntipo: test\n---\n",
		"b.md":  "---\nestado: Pending\ntipo: prod\n---\n",
		"c.md":  "---\nestado: Completed\ntipo: test\n---\n",
	})

	out, err := runCmd(t, "tree", root, "--where", "estado == 'Pending'", "--where", "tipo == 'test'")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	var result TreeResult
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\noutput: %s", err, out)
	}

	if result.Root.Total != 1 {
		t.Errorf("total = %d, want 1 (Pending + test)", result.Root.Total)
	}
}

// TestTree_SchemalessToleratesMissing replaces TestTreeErrorPropagation_NoSchema,
// which asserted the opposite.
//
// tree renders a directory hierarchy and whatever frontmatter it finds. A .stem
// only tells it which field to show as a status, so its absence narrows the
// output rather than invalidating it — the same tolerance `query` already has
// on the same tree.
func TestTree_SchemalessToleratesMissing(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "sub"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, "doc1.md"), []byte("---\nstatus: Pending\n---\n# Doc1\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "sub", "doc2.md"), []byte("---\nstatus: Completed\n---\n# Doc2\n"), 0644)

	out, err := runCmd(t, "tree", dir)
	if err != nil {
		t.Fatalf("tree must tolerate a tree with no .stem, got: %v", err)
	}

	var result TreeResult
	if jsonErr := json.Unmarshal([]byte(out), &result); jsonErr != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", jsonErr, out)
	}
	if result.Root == nil {
		t.Fatal("root node missing")
	}
	if result.Root.Total != 2 {
		t.Errorf("total = %d, want 2 (both documents are still indexed)", result.Root.Total)
	}
}

func TestFirstEnumField(t *testing.T) {
	tests := []struct {
		name   string
		schema map[string]rules.SchemaField
		want   string
	}{
		{
			name: "two enums picks lexically first",
			schema: map[string]rules.SchemaField{
				"tipo":   {Type: "enum", Values: []string{"task", "goal"}},
				"estado": {Type: "enum", Values: []string{"Pending", "Completed"}},
			},
			want: "estado",
		},
		{
			name: "single enum",
			schema: map[string]rules.SchemaField{
				"status": {Type: "enum", Values: []string{"Active", "Inactive"}},
				"title":  {Type: "string"},
			},
			want: "status",
		},
		{
			name: "no enum fields",
			schema: map[string]rules.SchemaField{
				"title": {Type: "string"},
				"count": {Type: "number"},
			},
			want: "",
		},
		{
			name:   "nil schema",
			schema: nil,
			want:   "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Repeat: map iteration is randomized per range, so one pass can pass by luck.
			for run := 1; run <= 10; run++ {
				if got := firstEnumField(tt.schema); got != tt.want {
					t.Fatalf("run %d: firstEnumField() = %q, want %q", run, got, tt.want)
				}
			}
		})
	}
}

func TestTreeStatusDeterminism_TwoEnumFields(t *testing.T) {
	// estado and tipo are both enum-typed; "estado" wins because e < t.
	root := setupTreeProject(t, map[string]string{
		".stem": "version: 2\nscope:\n  match: \"*.md\"\nschema:\n" +
			"  estado:\n    type: enum\n    values: [Pending, Completed]\n" +
			"  tipo:\n    type: enum\n    values: [task, goal]\n",
		"a.md": "---\nestado: Completed\ntipo: task\n---\n",
		"b.md": "---\nestado: Pending\ntipo: goal\n---\n",
	})

	var first string
	for run := 1; run <= 5; run++ {
		out, err := runCmd(t, "tree", root, "-o", "table")
		if err != nil {
			t.Fatalf("run %d: unexpected error: %v", run, err)
		}
		if run == 1 {
			first = out
		} else if out != first {
			t.Fatalf("run %d output differs from run 1:\n--- run 1 ---\n%s\n--- run %d ---\n%s",
				run, first, run, out)
		}
		if !strings.Contains(out, "[Completed]") || !strings.Contains(out, "[Pending]") {
			t.Fatalf("run %d: want estado values in status column, got:\n%s", run, out)
		}
		if strings.Contains(out, "[task]") || strings.Contains(out, "[goal]") {
			t.Fatalf("run %d: tipo values leaked into status column:\n%s", run, out)
		}
	}
}

func TestTreeStatusFallback_UsesSameEnumField(t *testing.T) {
	// A context with no pre-selected lifecycleField exercises the getStatusValue
	// fallback. It must resolve the same field the primary path would pick.
	ctx := &treeRenderContext{
		effectiveSchema: map[string]rules.SchemaField{
			"tipo":   {Type: "enum", Values: []string{"task", "goal"}},
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}},
		},
	}
	node := &treeNode{
		Name:        "a.md",
		IsLeaf:      true,
		Frontmatter: map[string]any{"estado": "Completed", "tipo": "task"},
	}

	for run := 1; run <= 10; run++ {
		if got := getStatusValue(node, ctx); got != "Completed" {
			t.Fatalf("run %d: getStatusValue() = %q, want %q (estado, not tipo)", run, got, "Completed")
		}
	}
}

func TestTreeStatusFallback_MissingSelectedFieldIsDash(t *testing.T) {
	// The record has a value for tipo but not for estado. Because both decision
	// sites select estado, the deterministic answer is the em-dash — never tipo.
	ctx := &treeRenderContext{
		effectiveSchema: map[string]rules.SchemaField{
			"tipo":   {Type: "enum", Values: []string{"task", "goal"}},
			"estado": {Type: "enum", Values: []string{"Pending", "Completed"}},
		},
	}
	node := &treeNode{
		Name:        "a.md",
		IsLeaf:      true,
		Frontmatter: map[string]any{"tipo": "task"},
	}

	for run := 1; run <= 10; run++ {
		if got := getStatusValue(node, ctx); got != "—" {
			t.Fatalf("run %d: getStatusValue() = %q, want em-dash", run, got)
		}
	}
}
