package main

import (
	"context"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestFilterRecords_MatchSubset(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Completed"}},
		{Path: "c.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	filtered, err := filterRecords(context.Background(), records, []string{"estado == 'Pending'"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("got %d records, want 2", len(filtered))
	}
	for _, r := range filtered {
		if r.Frontmatter["estado"] != "Pending" {
			t.Errorf("unexpected estado %v for %s", r.Frontmatter["estado"], r.Path)
		}
	}
}

func TestFilterRecords_NoMatch(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	filtered, err := filterRecords(context.Background(), records, []string{"estado == 'Completed'"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 0 {
		t.Errorf("got %d records, want 0", len(filtered))
	}
}

func TestFilterRecords_InvalidExpr(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	_, err := filterRecords(context.Background(), records, []string{"== bad syntax"})
	if err == nil {
		t.Fatal("expected error for invalid expression")
	}
}

func TestFilterRecords_EmptyWheres(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Completed"}},
	}

	filtered, err := filterRecords(context.Background(), records, nil)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("got %d records, want 2 (passthrough)", len(filtered))
	}

	// Also test with empty slice.
	filtered, err = filterRecords(context.Background(), records, []string{})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("got %d records, want 2 (passthrough)", len(filtered))
	}

	// Also test with empty string entries.
	filtered, err = filterRecords(context.Background(), records, []string{"", ""})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 2 {
		t.Errorf("got %d records, want 2 (passthrough)", len(filtered))
	}
}

func TestFilterRecords_MultipleWheres(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending", "tipo": "test"}},
		{Path: "b.md", Frontmatter: map[string]any{"estado": "Pending", "tipo": "prod"}},
		{Path: "c.md", Frontmatter: map[string]any{"estado": "Completed", "tipo": "test"}},
	}

	filtered, err := filterRecords(context.Background(), records, []string{"estado == 'Pending'", "tipo == 'test'"})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(filtered) != 1 {
		t.Errorf("got %d records, want 1", len(filtered))
	}
	if len(filtered) > 0 && filtered[0].Path != "a.md" {
		t.Errorf("got %s, want a.md", filtered[0].Path)
	}
}
