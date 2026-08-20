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

	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return &rules.StemFile{}, nil }
	if err := EnrichBuiltins(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("EnrichBuiltins: %v", err)
	}

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
	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return stem, nil }
	if err := EnrichBuiltins(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("EnrichBuiltins: %v", err)
	}

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

	if err := EnrichBuiltins(context.Background(), records, "/root", nil); err != nil {
		t.Fatalf("EnrichBuiltins: %v", err)
	}

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
	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return stem, nil }

	// Pipeline: DeriveAll → EnrichBuiltins (real order)
	if err := DeriveAll(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("DeriveAll: %v", err)
	}
	if err := EnrichBuiltins(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("EnrichBuiltins: %v", err)
	}

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
	if err := EnrichBuiltinsSimple(context.Background(), records, t.TempDir()); err != nil {
		t.Fatalf("EnrichBuiltinsSimple: %v", err)
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
	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return stem, nil }

	if err := EnrichBuiltins(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("EnrichBuiltins: %v", err)
	}

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
	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return stem, nil }

	if err := EnrichBuiltins(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("EnrichBuiltins: %v", err)
	}

	if records[0].Derived["contexto"] != "This is the context" {
		t.Errorf("contexto = %v, want 'This is the context'", records[0].Derived["contexto"])
	}
}

func TestEnrichBuiltins_SourceExtraction_FrontmatterPresenceOverridesBody(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "docs/T001.md",
			Type:        "markdown",
			Body:        "# Body Title\n",
			Frontmatter: map[string]any{"titulo": ""},
			Derived:     map[string]any{"titulo": "stale-derived"},
		},
	}
	stem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"titulo": {Type: "string", Extract: "body.h1"},
	}}
	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return stem, nil }

	if err := EnrichBuiltins(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("EnrichBuiltins: %v", err)
	}

	if got, ok := records[0].Derived["titulo"]; !ok || got != "" {
		t.Fatalf("derived titulo = %#v, %v; want present empty frontmatter value", got, ok)
	}
}

func TestEnrichBuiltins_SourceExtraction_AbsentSourcePreservesExistingDerivedValue(t *testing.T) {
	records := []*extract.Record{
		{
			Path:        "docs/T001.md",
			Type:        "markdown",
			Body:        "## Not a Title\n\nNo h1 body source here",
			Frontmatter: map[string]any{},
			Derived:     map[string]any{"titulo": "derived-default"},
		},
	}
	stem := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"titulo": {Type: "string", Extract: "body.h1"},
	}}
	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return stem, nil }

	if err := EnrichBuiltins(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("EnrichBuiltins: %v", err)
	}

	if got, ok := records[0].Derived["titulo"]; !ok || got != "derived-default" {
		t.Fatalf("derived titulo = %#v, %v; want existing derived value preserved", got, ok)
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
	resolver := func(dir, recordPath string) (*rules.StemFile, error) { return stem, nil }

	if err := EnrichBuiltins(context.Background(), records, "/root", resolver); err != nil {
		t.Fatalf("EnrichBuiltins: %v", err)
	}

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

	if err := EnrichBuiltins(context.Background(), []*extract.Record{fMatch, tNoMatch}, dir, DefaultResolver()); err != nil {
		t.Fatalf("EnrichBuiltins: %v", err)
	}

	if _, ok := fMatch.Derived["resumen"]; !ok {
		t.Errorf("matching record F01 should have source-derived 'resumen'")
	}
	if v, ok := tNoMatch.Derived["resumen"]; ok {
		t.Errorf("non-matching record T01 must NOT get 'resumen' (match-scope leak), got %v", v)
	}
}
