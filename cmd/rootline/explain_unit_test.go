package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
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

func TestExplain_SiblingScanFailure(t *testing.T) {
	root := setupValidateProject(t, map[string]string{
		".stem":     "version: 2\naggregate:\n  total: len(descendants)\n",
		"README.md": "# Root\n",
		"good.md":   "# Readable sibling\n",
	})
	brokenPath := filepath.Join(root, "broken.md")
	// A dangling link makes os.ReadFile fail without relying on permission bits
	// (which do not prevent reads when the tests run as a privileged user).
	if err := os.Symlink("missing.md", brokenPath); err != nil {
		t.Skipf("symlink unavailable: %v", err)
	}

	for _, format := range []string{"json", "table"} {
		t.Run(format, func(t *testing.T) {
			out, err := runTask9CmdStdoutOnly(t, "explain", filepath.Join(root, "README.md"), "-o", format)
			if err == nil {
				t.Fatal("explain must fail when a sibling cannot be read")
			}
			if !strings.Contains(err.Error(), "scanning aggregate input records in "+root) {
				t.Fatalf("error lacks aggregate scan context: %v", err)
			}
			if !errors.Is(err, os.ErrNotExist) {
				t.Fatalf("error must preserve the missing sibling cause: %v", err)
			}
			var pathErr *os.PathError
			if !errors.As(err, &pathErr) || pathErr.Path != brokenPath {
				t.Fatalf("error must identify the unreadable sibling %s: %v", brokenPath, err)
			}
			if out != "" {
				t.Fatalf("failed scan must emit no stdout, got %q", out)
			}
		})
	}
}

func TestExplain_SiblingExtractionSemantics(t *testing.T) {
	tests := []struct {
		name    string
		sibling string
		want    float64
	}{
		{
			name:    "readable",
			sibling: "---\nstatus: pending\n---\n# Sibling\n",
			want:    1,
		},
		{
			name:    "malformed frontmatter retains fallback values",
			sibling: "---\nstatus: done\nbroken: [\n---\n# Sibling\n",
			want:    2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			root := setupValidateProject(t, map[string]string{
				".stem":     "version: 2\nschema:\n  status:\n    type: string\naggregate:\n  done_total: 'len(filter(descendants, {.status == \"done\"}))'\n",
				"README.md": "# Root\n",
				"good.md":   "---\nstatus: done\n---\n# Readable sibling\n",
				"other.md":  tt.sibling,
			})
			out, err := runTask9CmdStdoutOnly(t, "explain", filepath.Join(root, "README.md"), "-o", "json")
			if err != nil {
				t.Fatalf("sibling extraction must not fail the scan: %v", err)
			}
			var result struct {
				Fields []map[string]any `json:"fields"`
			}
			if err := json.Unmarshal([]byte(out), &result); err != nil {
				t.Fatalf("invalid explain JSON: %v\n%s", err, out)
			}
			for _, field := range result.Fields {
				if field["name"] == "done_total" {
					if field["value"] != tt.want || field["origin"] != "aggregate" {
						t.Fatalf("done_total = %#v, want aggregate count %v", field, tt.want)
					}
					return
				}
			}
			t.Fatalf("explain fields missing done_total: %#v", result.Fields)
		})
	}
}
