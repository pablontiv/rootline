package main

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/rules"
)

func TestIsIndexFile(t *testing.T) {
	tests := []struct {
		path string
		want bool
	}{
		{"/tmp/project/README.md", true},
		{"/tmp/project/sub/README.md", true},
		{"/tmp/project/doc1.md", false},
		{"/tmp/project/readme.md", false}, // case-sensitive
		{"/tmp/project/README.txt", false},
	}
	for _, tt := range tests {
		got := rules.IsIndexFile(tt.path, nil)
		if got != tt.want {
			t.Errorf("IsIndexFile(%q, nil) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestIsIndexFile_CustomRequireIndex(t *testing.T) {
	stem := &rules.StemFile{
		Structural: rules.StructuralRules{
			Subdirs: rules.SubdirRules{
				RequireIndex: "index.md",
			},
		},
	}
	tests := []struct {
		path string
		want bool
	}{
		{"/tmp/project/index.md", true},
		{"/tmp/project/README.md", false},
		{"/tmp/project/doc1.md", false},
	}
	for _, tt := range tests {
		got := rules.IsIndexFile(tt.path, stem)
		if got != tt.want {
			t.Errorf("IsIndexFile(%q, custom) = %v, want %v", tt.path, got, tt.want)
		}
	}
}

func TestSortedMapKeys(t *testing.T) {
	tests := []struct {
		name string
		m    map[string]any
		want []string
	}{
		{"nil map", nil, nil},
		{"empty map", map[string]any{}, nil},
		{"single key", map[string]any{"a": 1}, []string{"a"}},
		{"sorted order", map[string]any{"b": 2, "a": 1, "c": 3}, []string{"a", "b", "c"}},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := sortedMapKeys(tt.m)
			if len(got) != len(tt.want) {
				t.Fatalf("got %v, want %v", got, tt.want)
			}
			for i := range got {
				if got[i] != tt.want[i] {
					t.Errorf("index %d: got %q, want %q", i, got[i], tt.want[i])
				}
			}
		})
	}
}

// TEST-FIRST: Explain command should fail (non-zero exit) when schema resolution fails
// This test will check if explain propagates WalkUp errors properly
func TestExplainErrorPropagation_NoSchema(t *testing.T) {
	dir := t.TempDir()

	// Create a markdown file WITHOUT a .stem anywhere in the tree
	// This will cause WalkUp to return ErrNoSchemaFound
	filePath := filepath.Join(dir, "doc.md")
	if err := os.WriteFile(filePath, []byte("---\nfoo: bar\n---\n# Doc\n"), 0644); err != nil {
		t.Fatal(err)
	}

	// Run explain on a file with no schema in the tree
	_, err := runCmd(t, "explain", filePath)

	// EXPECTATION: explain should fail when no schema is found
	if err == nil {
		t.Fatalf("explain should fail when no schema is found, but got nil error (silent degradation)")
	}
}

// TestExplain_DirectoryArgumentError mirrors validate's directory message.
//
// explain traces one document, and has no --all to point at, so the message
// names the argument shape it does accept rather than a flag.
func TestExplain_DirectoryArgumentError(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":     "version: 2\nscope:\n  match: \"*.md\"\n",
		"docs/a.md": "---\ntitulo: x\n---\n# x\n",
	})
	target := filepath.Join(root, "docs")

	_, err := runCmd(t, "explain", target)
	if err == nil {
		t.Fatal("explain must reject a directory argument")
	}
	want := target + " is a directory; use 'rootline explain <file>'"
	if err.Error() != want {
		t.Errorf("error = %q, want %q", err.Error(), want)
	}
}
