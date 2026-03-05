package infer

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestDetectCrossReferencesValid(t *testing.T) {
	// Create directory structure: rootDir/E02-something/F04-something/
	rootDir := t.TempDir()
	e02 := filepath.Join(rootDir, "E02-something")
	f04 := filepath.Join(e02, "F04-something")
	if err := os.MkdirAll(f04, 0o755); err != nil {
		t.Fatal(err)
	}

	records := []*extract.Record{
		{
			Path:        "doc.md",
			Frontmatter: map[string]any{},
			Body:        "See [E02/F04](../E02/F04/) for details.",
		},
	}

	got := DetectCrossReferences(records, rootDir)
	if len(got) != 1 {
		t.Fatalf("expected 1 cross_reference, got %d", len(got))
	}
	if got[0].Type != "cross_reference" {
		t.Errorf("expected type cross_reference, got %s", got[0].Type)
	}
	if got[0].Value != "E02/F04" {
		t.Errorf("expected value E02/F04, got %s", got[0].Value)
	}
}

func TestDetectCrossReferencesBroken(t *testing.T) {
	rootDir := t.TempDir()
	// No directories created — reference will be broken

	records := []*extract.Record{
		{
			Path:        "doc.md",
			Frontmatter: map[string]any{},
			Body:        "See E99/F01 for details.",
		},
	}

	got := DetectCrossReferences(records, rootDir)
	if len(got) != 1 {
		t.Fatalf("expected 1 broken_cross_reference, got %d", len(got))
	}
	if got[0].Type != "broken_cross_reference" {
		t.Errorf("expected type broken_cross_reference, got %s", got[0].Type)
	}
	if got[0].Value != "E99/F01" {
		t.Errorf("expected value E99/F01, got %s", got[0].Value)
	}
}

func TestDetectCrossReferencesEmptyBody(t *testing.T) {
	records := []*extract.Record{
		{Path: "doc.md", Frontmatter: map[string]any{}, Body: ""},
	}

	got := DetectCrossReferences(records, t.TempDir())
	if len(got) != 0 {
		t.Errorf("expected no inferences for empty body, got %d", len(got))
	}
}

func TestDetectCrossReferencesDedup(t *testing.T) {
	rootDir := t.TempDir()

	records := []*extract.Record{
		{
			Path:        "doc.md",
			Frontmatter: map[string]any{},
			Body:        "See E99/F01 and again E99/F01 referenced twice.",
		},
	}

	got := DetectCrossReferences(records, rootDir)
	if len(got) != 1 {
		t.Errorf("expected 1 deduped inference, got %d", len(got))
	}
}

func TestDetectCrossReferencesDeepPath(t *testing.T) {
	rootDir := t.TempDir()
	s001 := filepath.Join(rootDir, "E13-engine", "F02-cats", "S001-det")
	if err := os.MkdirAll(s001, 0o755); err != nil {
		t.Fatal(err)
	}

	records := []*extract.Record{
		{
			Path:        "doc.md",
			Frontmatter: map[string]any{},
			Body:        "Reference to E13/F02/S001 here.",
		},
	}

	got := DetectCrossReferences(records, rootDir)
	if len(got) != 1 {
		t.Fatalf("expected 1 inference, got %d", len(got))
	}
	if got[0].Type != "cross_reference" {
		t.Errorf("expected type cross_reference, got %s", got[0].Type)
	}
	if got[0].Value != "E13/F02/S001" {
		t.Errorf("expected value E13/F02/S001, got %s", got[0].Value)
	}
}
