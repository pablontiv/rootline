package derive

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestDeriveRecordSlugify(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Hello World!"},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"slug": `slugify(titulo)`,
		},
	}

	derived, err := DeriveRecord(record, stem, nil)
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived["slug"] != "hello-world" {
		t.Errorf("slug = %v, want %q", derived["slug"], "hello-world")
	}
	// Check that record.Derived is populated.
	if record.Derived["slug"] != "hello-world" {
		t.Errorf("record.Derived[slug] = %v, want %q", record.Derived["slug"], "hello-world")
	}
}

func TestDeriveRecordWithChildren(t *testing.T) {
	record := &extract.Record{
		Path:        "parent.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Parent"},
	}
	children := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Completado"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "c.md", Frontmatter: map[string]any{"estado": "Completado"}},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"total": "len(children)",
		},
	}

	derived, err := DeriveRecord(record, stem, children)
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived["total"] != 3 {
		t.Errorf("total = %v, want 3", derived["total"])
	}
}

func TestDeriveRecordCountChildren(t *testing.T) {
	record := &extract.Record{
		Path:        "parent.md",
		Type:        "markdown",
		Frontmatter: map[string]any{},
	}
	children := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Completado"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "c.md", Frontmatter: map[string]any{"estado": "Completado"}},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"done": `count(children, {.estado == "Completado"})`,
		},
	}

	derived, err := DeriveRecord(record, stem, children)
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived["done"] != 2 {
		t.Errorf("done = %v, want 2", derived["done"])
	}
}

func TestDeriveRecordNoDerive(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{"titulo": "Hello"},
	}
	stem := &rules.StemFile{} // no derive

	derived, err := DeriveRecord(record, stem, nil)
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if len(derived) != 0 {
		t.Errorf("expected empty map, got %v", derived)
	}
}

func TestDeriveRecordNilStem(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{},
	}

	derived, err := DeriveRecord(record, nil, nil)
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if len(derived) != 0 {
		t.Errorf("expected empty map, got %v", derived)
	}
}

func TestDeriveRecordInvalidExpression(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{"titulo": "Hello"},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"bad": "invalid !!!",
		},
	}

	derived, err := DeriveRecord(record, stem, nil)
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
	// Bad field should not be in derived.
	if _, ok := derived["bad"]; ok {
		t.Error("bad field should not be in derived")
	}
}

func TestDeriveRecordNonStringExpression(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Frontmatter: map[string]any{},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"bad": 42, // not a string
		},
	}

	_, err := DeriveRecord(record, stem, nil)
	if err == nil {
		t.Fatal("expected error for non-string expression")
	}
}

func TestDeriveRecordMultipleFields(t *testing.T) {
	record := &extract.Record{
		Path:        "test.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"titulo": "Hello World", "estado": "Pending"},
	}
	stem := &rules.StemFile{
		Derive: map[string]any{
			"slug":      `slugify(titulo)`,
			"is_active": `estado == "Pending"`,
		},
	}

	derived, err := DeriveRecord(record, stem, nil)
	if err != nil {
		t.Fatalf("DeriveRecord: %v", err)
	}
	if derived["slug"] != "hello-world" {
		t.Errorf("slug = %v, want %q", derived["slug"], "hello-world")
	}
	if derived["is_active"] != true {
		t.Errorf("is_active = %v, want true", derived["is_active"])
	}
}

func TestChildToMap(t *testing.T) {
	c := &extract.Record{
		Path:        "child.md",
		Type:        "markdown",
		Frontmatter: map[string]any{"estado": "Pending", "titulo": "Child"},
	}
	m := childToMap(c)
	if m["path"] != "child.md" {
		t.Errorf("path = %v, want %q", m["path"], "child.md")
	}
	if m["type"] != "markdown" {
		t.Errorf("type = %v, want %q", m["type"], "markdown")
	}
	if m["estado"] != "Pending" {
		t.Errorf("estado = %v, want %q", m["estado"], "Pending")
	}
}
