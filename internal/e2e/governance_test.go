package e2e

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/infer"
)

func TestGovernance_AnalyzePipeline(t *testing.T) {
	root := setupProject(t, map[string]string{
		// Governed root with .stem but missing domains on fields.
		".stem":   "version: 2\nschema:\n  estado:\n    type: enum\n    values: [draft, active, closed]\n  tipo:\n    type: string\n",
		"doc1.md": "---\nestado: draft\ntipo: feature\n---\n# Doc 1\n",
		"doc2.md": "---\nestado: active\ntipo: feature\n---\n# Doc 2\n",
		"doc3.md": "---\nestado: closed\ntipo: feature\n---\n# Doc 3\n",
		// Deep ungoverned path: a/b/notes/ is 3 levels under root where .stem lives.
		// depth = 2 path separators → implicit_schema inference.
		"a/b/notes/note1.md": "---\ntitle: Note 1\n---\n# Note\n",
		"a/b/notes/note2.md": "---\ntitle: Note 2\n---\n# Note\n",
	})

	report := runAnalyze(t, root)

	// Note: domain_coverage is no longer available after O14 refactor (domain: field removed)

	// Verify schema_coverage category exists.
	schemaCat := findCategory(report, "schema_coverage")
	if schemaCat == nil {
		t.Fatal("expected schema_coverage category in report")
	}
	// The deep a/b/notes/ dir should produce an implicit_schema inference
	// (it inherits from root .stem which is 3 levels up, depth >= 2).
	foundCoverage := false
	for _, inf := range schemaCat.Inferences {
		if inf.Type == "implicit_schema" || inf.Type == "missing_schema" {
			foundCoverage = true
			break
		}
	}
	if !foundCoverage {
		t.Error("expected implicit_schema or missing_schema inference for deep ungoverned directory")
	}

	// Verify validation_gaps category exists.
	valCat := findCategory(report, "validation_gaps")
	if valCat == nil {
		t.Fatal("expected validation_gaps category in report")
	}

	// Verify summary includes governance counts.
	if report.Summary.TotalInferences == 0 {
		t.Error("expected non-zero total inferences in summary")
	}
}

func TestGovernance_ScaffoldSchema(t *testing.T) {
	dir := t.TempDir()
	// Create markdown files with frontmatter
	files := map[string]string{
		"doc1.md": "---\ntitle: Hello\nestado: draft\n---\n# Doc\n",
		"doc2.md": "---\ntitle: World\nestado: active\n---\n# Doc 2\n",
	}
	for name, content := range files {
		govWriteFile(t, filepath.Join(dir, name), content)
	}

	err := infer.ScaffoldSchema(dir, false)
	if err != nil {
		t.Fatal(err)
	}

	// Verify .stem was created
	stemPath := filepath.Join(dir, ".stem")
	data := govReadFile(t, stemPath)
	if len(data) == 0 {
		t.Fatal("expected non-empty .stem file")
	}
}

func findCategory(report *infer.AnalyzeReport, id string) *infer.CategoryResult {
	for i := range report.Categories {
		if report.Categories[i].ID == id {
			return &report.Categories[i]
		}
	}
	return nil
}

// govWriteFile is a test helper for creating files at an absolute path.
func govWriteFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

// govReadFile is a test helper for reading a file at an absolute path.
func govReadFile(t *testing.T, path string) []byte {
	t.Helper()
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	return data
}
