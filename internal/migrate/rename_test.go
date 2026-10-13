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
	stem := `version: 2
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

func TestRenameCollisionPreflight(t *testing.T) {
	const stem = "version: 2\nroot: true\nschema:\n  titulo:\n    type: string\n"
	const collidingStem = stem + "  title:\n    type: string\n"
	tests := []struct {
		name          string
		stem          string
		files         map[string]string
		root          string
		collidingPath string
	}{
		{
			name: "later document",
			stem: stem,
			files: map[string]string{
				"a.md": "---\ntitulo: Rename me\n---\n# First\n",
				"z.md": "---\ntitulo: Keep me\ntitle: null\n---\n# Last\n",
			},
			collidingPath: "z.md",
		},
		{
			name: "root stem",
			stem: collidingStem,
			files: map[string]string{
				"a.md": "---\ntitulo: Rename me\n---\n# First\n",
			},
			collidingPath: ".stem",
		},
		{
			name:          "stem without documents",
			stem:          collidingStem,
			collidingPath: ".stem",
		},
		{
			name: "child stem after valid root",
			stem: stem,
			files: map[string]string{
				"a.md":      "---\ntitulo: Rename me\n---\n# First\n",
				"sub/.stem": collidingStem,
				"sub/z.md":  "---\ntitulo: Rename me too\n---\n# Last\n",
			},
			collidingPath: filepath.Join("sub", ".stem"),
		},
		{
			name: "ancestor stem",
			stem: collidingStem,
			files: map[string]string{
				"sub/a.md": "---\ntitulo: Rename me\n---\n# First\n",
			},
			root:          "sub",
			collidingPath: ".stem",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			dir := setupTestDir(t, tt.stem, tt.files)
			before := map[string]string{".stem": tt.stem}
			for path, content := range tt.files {
				before[path] = content
			}
			var dryRunError string
			for _, dryRun := range []bool{true, false} {
				op := &RenameOperation{
					OldField: "titulo",
					NewField: "title",
					RootPath: filepath.Join(dir, tt.root),
					DryRun:   dryRun,
				}
				result, err := op.Execute()
				if err == nil {
					t.Fatalf("dry-run=%v: expected collision refusal", dryRun)
				}
				if result != nil {
					t.Errorf("dry-run=%v: refused rename returned a result", dryRun)
				}
				if !strings.Contains(err.Error(), tt.collidingPath) || !strings.Contains(err.Error(), `"title"`) {
					t.Errorf("dry-run=%v: error must name colliding path and destination key: %v", dryRun, err)
				}
				if dryRun {
					dryRunError = err.Error()
				} else if err.Error() != dryRunError {
					t.Errorf("real refusal %q differs from dry-run %q", err.Error(), dryRunError)
				}
				for path, want := range before {
					content, readErr := os.ReadFile(filepath.Join(dir, path))
					if readErr != nil {
						t.Fatal(readErr)
					}
					if string(content) != want {
						t.Errorf("dry-run=%v: %s changed on collision refusal", dryRun, path)
					}
				}
			}
		})
	}
}

func TestRenameDestinationWithoutSource(t *testing.T) {
	const stem = "version: 2\nroot: true\nschema:\n    titulo:\n        type: string\n"
	files := map[string]string{
		"a.md":      "---\ntitulo: Rename me\n---\n# First\n",
		"z.md":      "---\ntitle: Keep me\n---\n# Last\n",
		"sub/.stem": "version: 2\nschema:\n  title:\n    type: string\n",
	}
	dir := setupTestDir(t, stem, files)
	op := &RenameOperation{OldField: "titulo", NewField: "title", RootPath: dir}
	result, err := op.Execute()
	if err != nil {
		t.Fatalf("destination without source must not collide: %v", err)
	}
	if result.Summary.FilesUpdated != 1 || result.Summary.StemsUpdated != 1 {
		t.Fatalf("unexpected update counts: %+v", result.Summary)
	}
	for _, path := range []string{"a.md", "z.md", ".stem", "sub/.stem"} {
		want := files[path]
		switch path {
		case "a.md":
			want = strings.Replace(want, "titulo:", "title:", 1)
		case ".stem":
			want = strings.Replace(stem, "titulo:", "title:", 1)
		}
		content, err := os.ReadFile(filepath.Join(dir, path))
		if err != nil {
			t.Fatal(err)
		}
		if string(content) != want {
			t.Errorf("%s = %q, want %q", path, content, want)
		}
	}
}

func TestRenameNonexistentField(t *testing.T) {
	stem := `version: 2
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
	stem := `version: 2
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
	stem := `version: 2
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
	stem := `version: 2
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
	stem := `version: 2
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

func TestRenameTopLevelFrontmatterBytes(t *testing.T) {
	const stem = "version: 2\nroot: true\nschema:\n  titulo:\n    type: string\n"
	tests := []struct {
		name    string
		input   string
		want    string
		changed bool
	}{
		{
			name:  "nested only",
			input: "---\nmetadata:\n  titulo: Nested\n---\n# Body\n",
			want:  "---\nmetadata:\n  titulo: Nested\n---\n# Body\n",
		},
		{
			name:    "nested before top level",
			input:   "---\n# Keep this comment\nmetadata:\n    titulo: Nested # keep indentation\n    items:\n      - titulo: Item\ntitulo: 'Top level'  # keep spacing\n---\n# Body\n\n```yaml\ntitulo: Body\n---\n```\n",
			want:    "---\n# Keep this comment\nmetadata:\n    titulo: Nested # keep indentation\n    items:\n      - titulo: Item\ntitle: 'Top level'  # keep spacing\n---\n# Body\n\n```yaml\ntitulo: Body\n---\n```\n",
			changed: true,
		},
		{
			name:    "CRLF",
			input:   "---\r\nmetadata:\r\n  titulo: Nested\r\ntitulo: Top # comment\r\n---\r\n# Body\r\n\r\n---\r\n",
			want:    "---\r\nmetadata:\r\n  titulo: Nested\r\ntitle: Top # comment\r\n---\r\n# Body\r\n\r\n---\r\n",
			changed: true,
		},
		{
			name:    "BOM with CRLF",
			input:   "\xef\xbb\xbf---\r\nmetadata:\r\n  titulo: Nested\r\ntitulo: Top\r\n---\r\n# Body\r\n",
			want:    "\xef\xbb\xbf---\r\nmetadata:\r\n  titulo: Nested\r\ntitle: Top\r\n---\r\n# Body\r\n",
			changed: true,
		},
		{
			name:    "normal success",
			input:   "---\ntitulo: Top\nmetadata:\n  titulo: Nested\n---\n# Body\n",
			want:    "---\ntitle: Top\nmetadata:\n  titulo: Nested\n---\n# Body\n",
			changed: true,
		},
		{
			name:  "unmatched quoted key is not reported updated",
			input: "---\n\"titulo\": Top\nmetadata:\n  titulo: Nested\n---\n# Body\n",
			want:  "---\n\"titulo\": Top\nmetadata:\n  titulo: Nested\n---\n# Body\n",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got, changed := renameFrontmatterField(tt.input, "titulo", "title")
			if changed != tt.changed || got != tt.want {
				t.Fatalf("renameFrontmatterField = (%q, %v), want (%q, %v)", got, changed, tt.want, tt.changed)
			}

			for _, dryRun := range []bool{false, true} {
				name := "apply"
				if dryRun {
					name = "dry run"
				}
				t.Run(name, func(t *testing.T) {
					dir := setupTestDir(t, stem, map[string]string{"doc.md": tt.input})
					op := &RenameOperation{OldField: "titulo", NewField: "title", RootPath: dir, DryRun: dryRun}
					result, err := op.Execute()
					if err != nil {
						t.Fatalf("Execute failed: %v", err)
					}
					wantUpdated := 0
					if tt.changed {
						wantUpdated = 1
					}
					if result.Summary.FilesScanned != 1 || result.Summary.FilesUpdated != wantUpdated || len(result.FilesUpdated) != wantUpdated {
						t.Fatalf("unexpected update counts: %+v, files_updated: %v", result.Summary, result.FilesUpdated)
					}
					if tt.changed && result.FilesUpdated[0] != "doc.md" {
						t.Errorf("files_updated = %v, want [doc.md]", result.FilesUpdated)
					}
					if len(result.FileDetails) != 1 || result.FileDetails[0].Path != "doc.md" || result.FileDetails[0].Updated != tt.changed {
						t.Errorf("unexpected file details: %+v", result.FileDetails)
					}
					content, err := os.ReadFile(filepath.Join(dir, "doc.md"))
					if err != nil {
						t.Fatal(err)
					}
					want := tt.want
					if dryRun {
						want = tt.input
					}
					if string(content) != want {
						t.Errorf("document bytes = %q, want %q", content, want)
					}
					if result.Summary.StemsUpdated != 1 {
						t.Errorf("stems_updated = %d, want 1", result.Summary.StemsUpdated)
					}
					stemContent, err := os.ReadFile(filepath.Join(dir, ".stem"))
					if err != nil {
						t.Fatal(err)
					}
					if dryRun {
						if string(stemContent) != stem {
							t.Error("dry run changed schema bytes")
						}
					} else if strings.Contains(string(stemContent), "titulo:") || !strings.Contains(string(stemContent), "title:") {
						t.Errorf("schema rename did not complete: %s", stemContent)
					}
				})
			}
		})
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
		{
			name:     "similar key is not a match",
			input:    "---\ntitulo_extra: Keep\n---\ntitulo: Body\n",
			oldField: "titulo",
			newField: "title",
			want:     "---\ntitulo_extra: Keep\n---\ntitulo: Body\n",
			changed:  false,
		},
		{
			name:     "closing delimiter at EOF",
			input:    "---\ntitulo: Top\n---",
			oldField: "titulo",
			newField: "title",
			want:     "---\ntitle: Top\n---",
			changed:  true,
		},
		{
			name:     "unterminated frontmatter",
			input:    "---\ntitulo: Top\n",
			oldField: "titulo",
			newField: "title",
			want:     "---\ntitulo: Top\n",
			changed:  false,
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
	stem := `version: 2
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
