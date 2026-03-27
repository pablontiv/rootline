package e2e

import (
	"context"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/query"
	"github.com/pablontiv/rootline/internal/rules"
)

// TestFuzzySuggestions_EnumValidation exercises the full pipeline:
// .stem defines enum, document has a typo, validation error includes suggestion.
func TestFuzzySuggestions_EnumValidation(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
schema:
  estado:
    type: enum
    values: [Pending, Completed, "In Progress"]
    required: true
  titulo:
    type: string
    required: true
`,
		"doc.md": `---
estado: Peding
titulo: Test Document
---
# Test
`,
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)

	found := false
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, err := rules.ResolveForRecord(dir, rec.Path)
		if err != nil {
			continue
		}
		errs := rules.Validate(ctx, rec, effective)
		for _, e := range errs {
			if e.Rule == "enum" && e.Field == "estado" {
				found = true
				if !strings.Contains(e.Message, `did you mean "Pending"`) {
					t.Errorf("enum error message missing suggestion: %s", e.Message)
				}
				if e.Suggestion != "Pending" {
					t.Errorf("Suggestion = %q, want %q", e.Suggestion, "Pending")
				}
			}
		}
	}
	if !found {
		t.Error("expected enum validation error for estado=Peding, got none")
	}
}

// TestFuzzySuggestions_RequiredFieldTypo exercises the pipeline when
// a required field is missing but a similarly-named field exists.
func TestFuzzySuggestions_RequiredFieldTypo(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
schema:
  estado:
    type: enum
    values: [Pending, Completed]
    required: true
`,
		"doc.md": `---
estdo: Pending
---
# Test
`,
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)

	found := false
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		dir := filepath.Dir(absPath)
		effective, err := rules.ResolveForRecord(dir, rec.Path)
		if err != nil {
			continue
		}
		errs := rules.Validate(ctx, rec, effective)
		for _, e := range errs {
			if e.Rule == "required" && e.Field == "estado" {
				found = true
				if !strings.Contains(e.Message, `found similar: "estdo"`) {
					t.Errorf("required error message missing similar hint: %s", e.Message)
				}
				if e.Suggestion != "estdo" {
					t.Errorf("Suggestion = %q, want %q", e.Suggestion, "estdo")
				}
			}
		}
	}
	if !found {
		t.Error("expected required validation error for estado with similar estdo, got none")
	}
}

// TestFuzzySuggestions_BrokenLinks exercises the graph pipeline:
// a wiki-link target is close to an existing node, suggestions are provided.
func TestFuzzySuggestions_BrokenLinks(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem": `version: 2
schema:
  titulo:
    type: string
`,
		"source.md": `---
titulo: Source
---
# Source

See [[blocks:trget.md]]
`,
		"target.md": `---
titulo: Target
---
# Target
`,
	})

	ctx := context.Background()
	records := scanAndAggregate(t, ctx, root)

	g := graph.Build(ctx, records)
	broken := g.BrokenLinks()

	found := false
	for _, b := range broken {
		if b.Target == "trget.md" {
			found = true
			if len(b.Suggestions) == 0 {
				t.Error("expected suggestions for broken link trget.md, got none")
			} else {
				hasSuggestion := false
				for _, s := range b.Suggestions {
					if s == "target.md" {
						hasSuggestion = true
					}
				}
				if !hasSuggestion {
					t.Errorf("suggestions %v don't contain target.md", b.Suggestions)
				}
			}
		}
	}
	if !found {
		t.Error("expected broken link for trget.md, got none")
	}
}

// TestFuzzySuggestions_QueryFieldWarnings exercises the field check:
// an unknown field in a where expression triggers a warning with suggestion.
func TestFuzzySuggestions_QueryFieldWarnings(t *testing.T) {
	knownFields := []string{"estado", "tipo", "titulo", "path", "body", "type"}
	warnings := query.CheckFieldNames(`estdo == "Pending"`, knownFields)

	if len(warnings) == 0 {
		t.Fatal("expected field warning for estdo, got none")
	}

	w := warnings[0]
	if w.Field != "estdo" {
		t.Errorf("warning field = %q, want %q", w.Field, "estdo")
	}
	if w.Suggestion != "estado" {
		t.Errorf("suggestion = %q, want %q", w.Suggestion, "estado")
	}
	if !strings.Contains(w.Message, "did you mean") {
		t.Errorf("message missing 'did you mean': %s", w.Message)
	}
}
