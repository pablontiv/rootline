package infer

import (
	"os"
	"path/filepath"
	"testing"
)

func setupSchemaCoverageDir(t *testing.T) string {
	t.Helper()
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	governed := filepath.Join(root, "governed")
	if err := os.MkdirAll(governed, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(governed, ".stem"), []byte("version: 2\nschema:\n  estado:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(governed, "doc.md"), []byte("---\nestado: ok\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	ungoverned := filepath.Join(root, "ungoverned")
	if err := os.MkdirAll(ungoverned, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ungoverned, "doc1.md"), []byte("---\ntitle: one\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(ungoverned, "doc2.md"), []byte("---\ntitle: two\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	empty := filepath.Join(root, "assets")
	if err := os.MkdirAll(empty, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(empty, "logo.png"), []byte("binary"), 0o644); err != nil {
		t.Fatal(err)
	}

	return root
}

func TestDetectMissingSchemata_FindsUngoverned(t *testing.T) {
	root := setupSchemaCoverageDir(t)
	got := DetectMissingSchemata(root)

	var missing []Inference
	for _, inf := range got {
		if inf.Type == "missing_schema" {
			missing = append(missing, inf)
		}
	}
	if len(missing) != 1 {
		t.Fatalf("expected 1 missing_schema inference, got %d", len(missing))
	}
	if missing[0].Source != filepath.Join(root, "ungoverned") {
		t.Errorf("expected source %s, got %s", filepath.Join(root, "ungoverned"), missing[0].Source)
	}
}

func TestDetectMissingSchemata_IgnoresGoverned(t *testing.T) {
	root := setupSchemaCoverageDir(t)
	got := DetectMissingSchemata(root)

	for _, inf := range got {
		if inf.Source == filepath.Join(root, "governed") {
			t.Errorf("governed directory should not produce inferences, got: %s", inf.Message)
		}
	}
}

func TestDetectMissingSchemata_IgnoresNonMarkdown(t *testing.T) {
	root := setupSchemaCoverageDir(t)
	got := DetectMissingSchemata(root)

	for _, inf := range got {
		if inf.Source == filepath.Join(root, "assets") {
			t.Errorf("non-markdown directory should not produce inferences, got: %s", inf.Message)
		}
	}
}

func TestDetectMissingSchemata_ImplicitSchema(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte("version: 2\nschema:\n  x:\n    type: string\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	deep := filepath.Join(root, "a", "b", "c")
	if err := os.MkdirAll(deep, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(deep, "doc.md"), []byte("---\nx: val\n---\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	got := DetectMissingSchemata(root)
	found := false
	for _, inf := range got {
		if inf.Type == "implicit_schema" && inf.Source == deep {
			found = true
			break
		}
	}
	if !found {
		t.Error("expected implicit_schema inference for deeply nested directory")
	}
}
