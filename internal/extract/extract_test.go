package extract

import (
	"testing"
)

var ext = &MarkdownExtractor{}

func TestName(t *testing.T) {
	if ext.Name() != "markdown" {
		t.Errorf("Name() = %q, want markdown", ext.Name())
	}
}

func TestExtensions(t *testing.T) {
	exts := ext.Extensions()
	if len(exts) != 2 || exts[0] != ".md" || exts[1] != ".markdown" {
		t.Errorf("Extensions() = %v, want [.md .markdown]", exts)
	}
}

// EC-1 (I7): File has no frontmatter.
func TestExtract_NoFrontmatter(t *testing.T) {
	content := []byte("# Hello World\n\nSome content here.")
	rec, err := ext.Extract("test.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.Frontmatter) != 0 {
		t.Errorf("frontmatter len = %d, want 0", len(rec.Frontmatter))
	}
	if rec.Body != "# Hello World\n\nSome content here." {
		t.Errorf("body = %q", rec.Body)
	}
	if len(rec.Errors) != 0 {
		t.Errorf("errors = %v, want none", rec.Errors)
	}
}

// Valid frontmatter + body.
func TestExtract_ValidFrontmatterAndBody(t *testing.T) {
	content := []byte("---\ntitle: Hello\nstatus: draft\n---\n# Content\n\nBody text.")
	rec, err := ext.Extract("doc.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Frontmatter["title"] != "Hello" {
		t.Errorf("title = %v, want Hello", rec.Frontmatter["title"])
	}
	if rec.Frontmatter["status"] != "draft" {
		t.Errorf("status = %v, want draft", rec.Frontmatter["status"])
	}
	if rec.Body != "# Content\n\nBody text." {
		t.Errorf("body = %q", rec.Body)
	}
	if rec.Path != "doc.md" {
		t.Errorf("path = %q, want doc.md", rec.Path)
	}
	if rec.Type != "markdown" {
		t.Errorf("type = %q, want markdown", rec.Type)
	}
	if len(rec.Errors) != 0 {
		t.Errorf("errors = %v, want none", rec.Errors)
	}
}

// EC-2 (I7): Malformed YAML frontmatter → fallback parser.
func TestExtract_MalformedYAML(t *testing.T) {
	content := []byte("---\ntitle: Hello\nstatus: [broken\ninvalid yaml here\n---\n# Body")
	rec, err := ext.Extract("bad.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.Errors) == 0 {
		t.Fatal("expected extraction errors for malformed YAML")
	}
	// Fallback should extract at least "title".
	if rec.Frontmatter["title"] != "Hello" {
		t.Errorf("fallback title = %v, want Hello", rec.Frontmatter["title"])
	}
	if rec.Body != "# Body" {
		t.Errorf("body = %q, want # Body", rec.Body)
	}
}

// EC-3 (I7): Empty file.
func TestExtract_EmptyFile(t *testing.T) {
	rec, err := ext.Extract("empty.md", []byte{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.Frontmatter) != 0 {
		t.Errorf("frontmatter len = %d, want 0", len(rec.Frontmatter))
	}
	if rec.Body != "" {
		t.Errorf("body = %q, want empty", rec.Body)
	}
	if len(rec.Errors) != 0 {
		t.Errorf("errors = %v, want none", rec.Errors)
	}
}

// EC-6 (I7): Duplicate YAML keys — last wins.
func TestExtract_DuplicateKeys(t *testing.T) {
	content := []byte("---\nstatus: draft\nstatus: published\n---\n")
	rec, err := ext.Extract("dup.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	// yaml.v3 last-wins behavior.
	if rec.Frontmatter["status"] != "published" {
		t.Errorf("status = %v, want published (last wins)", rec.Frontmatter["status"])
	}
}

// EC-7 (I7): BOM at start.
func TestExtract_BOM(t *testing.T) {
	content := []byte("\xef\xbb\xbf---\ntitle: BOM Test\n---\n# Body")
	rec, err := ext.Extract("bom.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Frontmatter["title"] != "BOM Test" {
		t.Errorf("title = %v, want BOM Test", rec.Frontmatter["title"])
	}
	if rec.Body != "# Body" {
		t.Errorf("body = %q, want # Body", rec.Body)
	}
}

// EC-8 (I7): Frontmatter-only file (no body).
func TestExtract_FrontmatterOnly(t *testing.T) {
	content := []byte("---\ntitle: Metadata Only\nstatus: draft\n---\n")
	rec, err := ext.Extract("meta.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Frontmatter["title"] != "Metadata Only" {
		t.Errorf("title = %v, want Metadata Only", rec.Frontmatter["title"])
	}
	if rec.Body != "" {
		t.Errorf("body = %q, want empty", rec.Body)
	}
}

// Unclosed delimiter.
func TestExtract_UnclosedDelimiter(t *testing.T) {
	content := []byte("---\ntitle: No Close\nstatus: draft\n")
	rec, err := ext.Extract("noclose.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(rec.Errors) == 0 {
		t.Fatal("expected error for unclosed delimiter")
	}
	if rec.Errors[0].Message != "unclosed frontmatter delimiter" {
		t.Errorf("error message = %q", rec.Errors[0].Message)
	}
	// Body should be the full text.
	if rec.Body == "" {
		t.Error("body should contain the full text when delimiter unclosed")
	}
}

// Windows-style line endings (\r\n).
func TestExtract_WindowsLineEndings(t *testing.T) {
	content := []byte("---\r\ntitle: Windows\r\nstatus: ok\r\n---\r\n# Body\r\n")
	rec, err := ext.Extract("win.md", content)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if rec.Frontmatter["title"] != "Windows" {
		t.Errorf("title = %v, want Windows", rec.Frontmatter["title"])
	}
	if len(rec.Errors) != 0 {
		t.Errorf("errors = %v, want none", rec.Errors)
	}
}

// Fallback parser skips comments and empty lines.
func TestFallbackParser(t *testing.T) {
	content := "title: Hello\n# comment\n\nstatus: draft\nbadline\n"
	result := fallbackParseFrontmatter(content)
	if result["title"] != "Hello" {
		t.Errorf("title = %v, want Hello", result["title"])
	}
	if result["status"] != "draft" {
		t.Errorf("status = %v, want draft", result["status"])
	}
	if _, exists := result["badline"]; exists {
		t.Error("badline should not be in result")
	}
}
