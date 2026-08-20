package rules

import (
	"reflect"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestExplainTask9OneSortedLogicalFieldSet(t *testing.T) {
	record := &extract.Record{
		Path:        "doc.md",
		Body:        "# Doc\n\n## Notes\n\nbody value\n",
		Frontmatter: map[string]any{"notes": "frontmatter value", "plain": "frontmatter plain"},
		Derived:     map[string]any{"notes": "frontmatter value", "slug": "doc"},
	}
	effective := &StemFile{
		Schema: map[string]SchemaField{
			"notes":       {Type: "string", Extract: `body.section["## Notes"]`, Source: "docs/.stem"},
			"plain":       {Type: "string", Source: "root/.stem"},
			"schema_only": {Type: "string", Default: "defaulted", Source: "root/.stem"},
		},
		Derive: map[string]any{"slug": "slugify(plain)"},
	}

	result, err := NewExplainResult("doc.md", nil, effective, record, nil)
	if err != nil {
		t.Fatalf("NewExplainResult error: %v", err)
	}
	var names []string
	byName := map[string]ExplainField{}
	for _, field := range result.Fields {
		names = append(names, field.Name)
		if _, exists := byName[field.Name]; exists {
			t.Fatalf("duplicate explain field row for %q in %#v", field.Name, result.Fields)
		}
		byName[field.Name] = field
	}
	wantNames := []string{"notes", "plain", "schema_only", "slug"}
	if !reflect.DeepEqual(names, wantNames) {
		t.Fatalf("field names = %v, want sorted %v", names, wantNames)
	}
	if got := byName["notes"]; got.Value != "frontmatter value" || got.Origin != "frontmatter" || got.Source != `body.section["## Notes"]` || got.DefinedIn != "docs/.stem" {
		t.Fatalf("notes row = %#v, want one frontmatter override row with source/defined_in", got)
	}
	if got := byName["plain"]; got.Value != "frontmatter plain" || got.Source != "" || got.DefinedIn != "root/.stem" {
		t.Fatalf("plain row = %#v, want non-source schema field without logical source", got)
	}
	if got := byName["schema_only"]; got.Value != "defaulted" || got.Origin != "schema" || got.DefinedIn != "root/.stem" {
		t.Fatalf("schema_only row = %#v, want schema default", got)
	}
	if got := byName["slug"]; got.Value != "doc" || got.Origin != "derived" || got.Expression != "slugify(plain)" || got.Source != "" {
		t.Fatalf("slug row = %#v, want derived expression without logical source", got)
	}
}

func TestExplainTask9AmbiguousSourceReturnsError(t *testing.T) {
	record := &extract.Record{
		Path:        "dup.md",
		Body:        "# Dup\n\n## Notes\n\nfirst\n\n## Notes\n\nsecond\n",
		Frontmatter: map[string]any{},
		Derived:     map[string]any{},
	}
	effective := &StemFile{Schema: map[string]SchemaField{
		"notes": {Type: "string", Extract: `body.section["## Notes"]`, Source: "docs/.stem"},
	}}

	result, err := NewExplainResult("dup.md", nil, effective, record, nil)
	if err == nil || result != nil {
		t.Fatalf("NewExplainResult ambiguity result=%#v err=%v, want nil result with error", result, err)
	}
}
