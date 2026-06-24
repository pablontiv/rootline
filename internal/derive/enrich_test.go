package derive

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestEnrichBuiltins_IndexDetection(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{}},
		{Path: "dir/T001.md", Type: "markdown", Frontmatter: map[string]any{}},
		{Path: "dir/T002.md", Type: "markdown", Frontmatter: map[string]any{}},
	}

	resolver := func(dir, recordPath string) *rules.StemFile { return &rules.StemFile{} }
	EnrichBuiltins(context.Background(), records, "/root", resolver)

	if records[0].Derived["isIndex"] != true {
		t.Errorf("README.isIndex = %v, want true", records[0].Derived["isIndex"])
	}
	if records[1].Derived["isIndex"] != false {
		t.Errorf("T001.isIndex = %v, want false", records[1].Derived["isIndex"])
	}
	if records[2].Derived["isIndex"] != false {
		t.Errorf("T002.isIndex = %v, want false", records[2].Derived["isIndex"])
	}
}

func TestEnrichBuiltins_CustomRequireIndex(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/index.md", Type: "markdown", Frontmatter: map[string]any{}},
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{}},
	}

	stem := &rules.StemFile{
		Structural: rules.StructuralRules{
			Subdirs: rules.SubdirRules{RequireIndex: "index.md"},
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }
	EnrichBuiltins(context.Background(), records, "/root", resolver)

	if records[0].Derived["isIndex"] != true {
		t.Errorf("index.md.isIndex = %v, want true", records[0].Derived["isIndex"])
	}
	if records[1].Derived["isIndex"] != false {
		t.Errorf("README.md.isIndex = %v, want false (custom require_index)", records[1].Derived["isIndex"])
	}
}

func TestEnrichBuiltins_NilResolver(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{}},
	}

	EnrichBuiltins(context.Background(), records, "/root", nil)

	if records[0].Derived != nil {
		t.Errorf("Derived = %v, want nil (nil resolver)", records[0].Derived)
	}
}

func TestEnrichBuiltins_SurvivesDeriveAll(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/README.md", Type: "markdown", Frontmatter: map[string]any{"titulo": "Test"}},
		{Path: "dir/T001.md", Type: "markdown", Frontmatter: map[string]any{"titulo": "Task"}},
	}

	stem := &rules.StemFile{
		Derive: map[string]any{
			"slug": "lower(titulo)",
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	// Pipeline: DeriveAll → EnrichBuiltins (real order)
	DeriveAll(context.Background(), records, "/root", resolver)
	EnrichBuiltins(context.Background(), records, "/root", resolver)

	// isIndex should be set after enrichment.
	if records[0].Derived["isIndex"] != true {
		t.Errorf("README.isIndex = %v, want true", records[0].Derived["isIndex"])
	}
	// Derive results should still be present.
	if records[0].Derived["slug"] != "test" {
		t.Errorf("README.slug = %v, want 'test'", records[0].Derived["slug"])
	}
}

func TestEnrichBuiltinsSimple(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"titulo": "Hello"}},
	}
	// No .stem in temp dir — should not panic.
	EnrichBuiltinsSimple(context.Background(), records, t.TempDir())
}

func TestExtractBodyH1(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		expected string
	}{
		{
			name:     "simple h1",
			body:     "# Hello World\n\nSome content",
			expected: "Hello World",
		},
		{
			name:     "h1 with extra spaces",
			body:     "#  Spaced Title  \n\nContent",
			expected: "Spaced Title",
		},
		{
			name:     "no h1",
			body:     "## Section\n\nContent without H1",
			expected: "",
		},
		{
			name:     "empty body",
			body:     "",
			expected: "",
		},
		{
			name:     "h1 at end of body",
			body:     "Some content\n\n# Final Title",
			expected: "Final Title",
		},
		{
			name:     "h1 only",
			body:     "# Only H1",
			expected: "Only H1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBodyH1(tt.body)
			if result != tt.expected {
				t.Errorf("extractBodyH1() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestExtractBodySection(t *testing.T) {
	tests := []struct {
		name     string
		body     string
		heading  string
		expected string
	}{
		{
			name:     "simple section",
			body:     "# Title\n\n## Introduction\n\nIntro content\n\n## Next Section\n\nOther",
			heading:  "## Introduction",
			expected: "Intro content",
		},
		{
			name:     "section with multiple lines",
			body:     "## Details\n\nLine 1\nLine 2\nLine 3\n\n## End\n\nAfter",
			heading:  "## Details",
			expected: "Line 1\nLine 2\nLine 3",
		},
		{
			name:     "nonexistent section",
			body:     "## First\n\nContent",
			heading:  "## Missing",
			expected: "",
		},
		{
			name:     "empty body",
			body:     "",
			heading:  "## Section",
			expected: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := extractBodySection(tt.body, tt.heading)
			if result != tt.expected {
				t.Errorf("extractBodySection() = %q, want %q", result, tt.expected)
			}
		})
	}
}

func TestEnrichBuiltins_SourceExtraction_BodyH1(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "docs/T001.md",
			Type:        "markdown",
			Body:        "# Extract This Title\n\nSome body content",
			Frontmatter: map[string]any{},
		},
	}

	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"titulo": {
				Type:    "string",
				Extract: "body.h1",
			},
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)

	if records[0].Derived["titulo"] != "Extract This Title" {
		t.Errorf("titulo = %v, want 'Extract This Title'", records[0].Derived["titulo"])
	}
}

func TestEnrichBuiltins_SourceExtraction_BodySection(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "docs/T001.md",
			Type:        "markdown",
			Body:        "# Title\n\n## Context\n\nThis is the context\n\n## Details\n\nOther",
			Frontmatter: map[string]any{},
		},
	}

	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"contexto": {
				Type:    "string",
				Extract: `body.section["## Context"]`,
			},
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)

	if records[0].Derived["contexto"] != "This is the context" {
		t.Errorf("contexto = %v, want 'This is the context'", records[0].Derived["contexto"])
	}
}

func TestEnrichBuiltins_SourceExtraction_NoExtract(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "docs/T001.md",
			Type:        "markdown",
			Body:        "# Title\n\nContent",
			Frontmatter: map[string]any{},
		},
	}

	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"campo": {
				Type: "string",
			},
		},
	}
	resolver := func(dir, recordPath string) *rules.StemFile { return stem }

	EnrichBuiltins(context.Background(), records, "/root", resolver)

	// No Extract directive, so campo should not be in Derived
	if _, ok := records[0].Derived["campo"]; ok {
		t.Errorf("campo should not be derived without Extract directive")
	}
}

func TestEnrichBuiltins_SourceFieldRespectsMatchScope(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}
	// .stem: "resumen" is source-extracted from a body section, scoped to F* records only.
	stem := `version: 2
scope:
  match: "*.md"
schema:
  resumen:
    type: string
    source: body.section["## Resumen"]
    match: ["F*"]
`
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}

	fMatch := &extract.Record{
		Path:        "F01.md",
		Type:        "markdown",
		Body:        "# Title\n\n## Resumen\n\nfeature summary",
		Frontmatter: map[string]any{},
		Derived:     map[string]any{},
	}
	tNoMatch := &extract.Record{
		Path:        "T01.md",
		Type:        "markdown",
		Body:        "# Title\n\n## Resumen\n\ntask summary",
		Frontmatter: map[string]any{},
		Derived:     map[string]any{},
	}

	EnrichBuiltins(context.Background(), []*extract.Record{fMatch, tNoMatch}, dir, DefaultResolver())

	if _, ok := fMatch.Derived["resumen"]; !ok {
		t.Errorf("matching record F01 should have source-derived 'resumen'")
	}
	if v, ok := tNoMatch.Derived["resumen"]; ok {
		t.Errorf("non-matching record T01 must NOT get 'resumen' (match-scope leak), got %v", v)
	}
}
