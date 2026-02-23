package migrate

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// setupTestDir creates a temporary directory with a .git marker,
// a .stem file, and markdown files for testing.
func setupTestDir(t *testing.T, stemContent string, files map[string]string) string {
	t.Helper()
	dir := t.TempDir()

	// Create .git marker so WalkUp has a boundary.
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0755); err != nil {
		t.Fatal(err)
	}

	// Write .stem file.
	if stemContent != "" {
		if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(stemContent), 0644); err != nil {
			t.Fatal(err)
		}
	}

	// Write test files.
	for name, content := range files {
		path := filepath.Join(dir, name)
		if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(path, []byte(content), 0644); err != nil {
			t.Fatal(err)
		}
	}

	return dir
}

func TestRenameExistingField(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
    required: true
  estado:
    type: enum
    values: [Pending, Done]
`
	files := map[string]string{
		"doc1.md": "---\ntitulo: Hello World\nestado: Pending\n---\n# Content\n",
		"doc2.md": "---\ntitulo: Another Doc\nestado: Done\n---\n# More content\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &RenameOperation{
		OldField: "titulo",
		NewField: "title",
		RootPath: dir,
		DryRun:   false,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Verify result counts.
	if result.Summary.FilesUpdated != 2 {
		t.Errorf("expected 2 files updated, got %d", result.Summary.FilesUpdated)
	}
	if result.Summary.StemsUpdated != 1 {
		t.Errorf("expected 1 stem updated, got %d", result.Summary.StemsUpdated)
	}

	// Verify frontmatter was renamed in files.
	for _, name := range []string{"doc1.md", "doc2.md"} {
		content, err := os.ReadFile(filepath.Join(dir, name))
		if err != nil {
			t.Fatal(err)
		}
		text := string(content)
		if strings.Contains(text, "titulo:") {
			t.Errorf("%s still contains 'titulo:'", name)
		}
		if !strings.Contains(text, "title:") {
			t.Errorf("%s missing 'title:'", name)
		}
	}

	// Verify .stem was updated.
	stemContent, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	stemText := string(stemContent)
	if strings.Contains(stemText, "titulo:") {
		t.Error(".stem still contains 'titulo:'")
	}
	if !strings.Contains(stemText, "title:") {
		t.Error(".stem missing 'title:'")
	}

	// Verify JSON version.
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
	if result.Kind != "rootline/migrate-rename" {
		t.Errorf("expected kind rootline/migrate-rename, got %s", result.Kind)
	}
}

func TestRenameNonexistentField(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
    required: true
`
	files := map[string]string{
		"doc1.md": "---\ntitulo: Hello\n---\n# Content\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &RenameOperation{
		OldField: "nonexistent",
		NewField: "something",
		RootPath: dir,
		DryRun:   false,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Summary.FilesUpdated != 0 {
		t.Errorf("expected 0 files updated, got %d", result.Summary.FilesUpdated)
	}
	if result.Summary.StemsUpdated != 0 {
		t.Errorf("expected 0 stems updated, got %d", result.Summary.StemsUpdated)
	}
}

func TestRenameDryRun(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
    required: true
`
	files := map[string]string{
		"doc1.md": "---\ntitulo: Hello\n---\n# Content\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &RenameOperation{
		OldField: "titulo",
		NewField: "title",
		RootPath: dir,
		DryRun:   true,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Dry-run should report files but not modify.
	if result.Summary.FilesUpdated != 1 {
		t.Errorf("expected 1 file reported, got %d", result.Summary.FilesUpdated)
	}

	// Verify files are NOT modified.
	content, err := os.ReadFile(filepath.Join(dir, "doc1.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), "titulo:") {
		t.Error("dry-run should not modify files")
	}

	// Verify .stem is NOT modified.
	stemContent, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(stemContent), "titulo:") {
		t.Error("dry-run should not modify .stem")
	}
}

func TestRenameStemSchemaUpdated(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
    required: true
  estado:
    type: enum
    values: [Pending, Done]
`
	files := map[string]string{
		"doc1.md": "---\ntitulo: Test\nestado: Pending\n---\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &RenameOperation{
		OldField: "titulo",
		NewField: "title",
		RootPath: dir,
		DryRun:   false,
	}

	_, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	// Re-read and verify .stem structure.
	stemContent, err := os.ReadFile(filepath.Join(dir, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(stemContent)

	// "title" should appear but not "titulo" in schema.
	if strings.Contains(text, "titulo") {
		t.Error(".stem still references 'titulo'")
	}
	if !strings.Contains(text, "title") {
		t.Error(".stem missing 'title' field")
	}
	// "estado" should still be present.
	if !strings.Contains(text, "estado") {
		t.Error(".stem missing 'estado' — other fields should be preserved")
	}
}

func TestRenameFileWithoutField(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
  estado:
    type: string
`
	files := map[string]string{
		"with.md":    "---\ntitulo: Has it\nestado: Pending\n---\n# Yes\n",
		"without.md": "---\nestado: Pending\n---\n# No titulo here\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &RenameOperation{
		OldField: "titulo",
		NewField: "title",
		RootPath: dir,
		DryRun:   false,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Summary.FilesUpdated != 1 {
		t.Errorf("expected 1 file updated, got %d", result.Summary.FilesUpdated)
	}

	// The file without the field should be untouched.
	content, err := os.ReadFile(filepath.Join(dir, "without.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if strings.Contains(text, "title:") {
		t.Error("file without old field should not gain new field")
	}
	if !strings.Contains(text, "estado:") {
		t.Error("file should still have its existing fields")
	}
}

func TestRenamePreservesBody(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
`
	files := map[string]string{
		"doc.md": "---\ntitulo: Test\n---\n# My Heading\n\nSome body content.\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &RenameOperation{
		OldField: "titulo",
		NewField: "title",
		RootPath: dir,
		DryRun:   false,
	}

	_, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	content, err := os.ReadFile(filepath.Join(dir, "doc.md"))
	if err != nil {
		t.Fatal(err)
	}
	text := string(content)
	if !strings.Contains(text, "# My Heading") {
		t.Error("body heading was lost")
	}
	if !strings.Contains(text, "Some body content.") {
		t.Error("body content was lost")
	}
}

func TestRenameFrontmatterField(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		oldField string
		newField string
		want     string
		changed  bool
	}{
		{
			name:     "simple rename",
			input:    "---\ntitulo: Hello\n---\n# Body\n",
			oldField: "titulo",
			newField: "title",
			want:     "---\ntitle: Hello\n---\n# Body\n",
			changed:  true,
		},
		{
			name:     "field not present",
			input:    "---\nestado: Pending\n---\n# Body\n",
			oldField: "titulo",
			newField: "title",
			want:     "---\nestado: Pending\n---\n# Body\n",
			changed:  false,
		},
		{
			name:     "no frontmatter",
			input:    "# Just a heading\n",
			oldField: "titulo",
			newField: "title",
			want:     "# Just a heading\n",
			changed:  false,
		},
		{
			name:     "preserves other fields",
			input:    "---\nestado: Pending\ntitulo: Hi\ntipo: task\n---\n",
			oldField: "titulo",
			newField: "title",
			want:     "---\nestado: Pending\ntitle: Hi\ntipo: task\n---\n",
			changed:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := renameFrontmatterField(tt.input, tt.oldField, tt.newField)
			if changed != tt.changed {
				t.Errorf("changed = %v, want %v", changed, tt.changed)
			}
			if got != tt.want {
				t.Errorf("got:\n%s\nwant:\n%s", got, tt.want)
			}
		})
	}
}

func TestRenameSubdirectory(t *testing.T) {
	stem := `version: 1
schema:
  titulo:
    type: string
    required: true
`
	files := map[string]string{
		"sub/doc1.md": "---\ntitulo: Sub Doc\n---\n# Sub\n",
		"sub/doc2.md": "---\ntitulo: Sub Doc 2\n---\n# Sub 2\n",
	}

	dir := setupTestDir(t, stem, files)

	op := &RenameOperation{
		OldField: "titulo",
		NewField: "title",
		RootPath: dir,
		DryRun:   false,
	}

	result, err := op.Execute()
	if err != nil {
		t.Fatalf("Execute failed: %v", err)
	}

	if result.Summary.FilesUpdated != 2 {
		t.Errorf("expected 2 files updated, got %d", result.Summary.FilesUpdated)
	}
}
