package extract

import (
	"testing"
)

func TestNewRegistry_IncludesMarkdown(t *testing.T) {
	r := NewRegistry()
	e := r.ForFile("doc.md", "")
	if e == nil {
		t.Fatal("ForFile(doc.md) returned nil, want MarkdownExtractor")
	}
	if e.Name() != "markdown" {
		t.Errorf("name = %q, want markdown", e.Name())
	}
}

func TestForFile_MarkdownExtension(t *testing.T) {
	r := NewRegistry()
	for _, path := range []string{"doc.md", "README.markdown", "dir/file.md"} {
		e := r.ForFile(path, "")
		if e == nil {
			t.Errorf("ForFile(%q) = nil, want MarkdownExtractor", path)
		}
	}
}

func TestForFile_UnknownExtension(t *testing.T) {
	r := NewRegistry()
	for _, path := range []string{"data.json", "config.yaml", "readme.txt", "noext"} {
		e := r.ForFile(path, "")
		if e != nil {
			t.Errorf("ForFile(%q) = %q, want nil", path, e.Name())
		}
	}
}

func TestForFile_StemOverrideByName(t *testing.T) {
	r := NewRegistry()
	// Override: use "markdown" extractor for a .json file.
	e := r.ForFile("data.json", "markdown")
	if e == nil {
		t.Fatal("ForFile with stemExtractor=markdown returned nil")
	}
	if e.Name() != "markdown" {
		t.Errorf("name = %q, want markdown", e.Name())
	}
}

func TestForFile_StemOverrideUnknownName(t *testing.T) {
	r := NewRegistry()
	e := r.ForFile("doc.md", "nonexistent")
	if e != nil {
		t.Errorf("ForFile with unknown stemExtractor = %q, want nil", e.Name())
	}
}

func TestRegister_DuplicateNamePanics(t *testing.T) {
	r := NewRegistry()
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate name registration")
		}
	}()
	r.Register(&MarkdownExtractor{}) // already registered by NewRegistry
}

func TestRegister_DuplicateExtensionPanics(t *testing.T) {
	r := &Registry{
		byName:      make(map[string]Extractor),
		byExtension: make(map[string]Extractor),
	}
	r.Register(&MarkdownExtractor{})
	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for duplicate extension registration")
		}
	}()
	// A fake extractor with different name but overlapping extension.
	r.Register(&fakeExtractor{name: "other", exts: []string{".md"}})
}

// fakeExtractor for testing duplicate extension detection.
type fakeExtractor struct {
	name string
	exts []string
}

func (f *fakeExtractor) Name() string                            { return f.name }
func (f *fakeExtractor) Extensions() []string                    { return f.exts }
func (f *fakeExtractor) Extract(string, []byte) (*Record, error) { return nil, nil }

func TestNewASTRegistry_IncludesMarkdown(t *testing.T) {
	r := NewASTRegistry()
	e := r.ForFile("doc.md", "")
	if e == nil {
		t.Fatal("NewASTRegistry ForFile(doc.md) returned nil")
	}
	if e.Name() != "markdown" {
		t.Errorf("name = %q, want markdown", e.Name())
	}
	me, ok := e.(*MarkdownExtractor)
	if !ok {
		t.Fatal("expected *MarkdownExtractor")
	}
	if me.ParseAST == nil || !*me.ParseAST {
		t.Error("expected ParseAST to be set to true")
	}
}
