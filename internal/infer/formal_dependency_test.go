package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestDetectFormalDependencies_WikiLink(t *testing.T) {
	rec := &extract.Record{
		Path: "task.md",
		Links: []extract.Link{
			{Target: "T002", Type: "blocks", Line: 1},
			{Target: "other.md", Type: "reference", Line: 2},
		},
	}

	inferences := DetectFormalDependencies([]*extract.Record{rec})

	if len(inferences) != 1 {
		t.Fatalf("expected 1 inference, got %d: %v", len(inferences), inferences)
	}
	if inferences[0].Type != "formal_dependency" || inferences[0].Field != "blocks" || inferences[0].Value != "T002" {
		t.Errorf("unexpected inference: %+v", inferences[0])
	}
}

func TestDetectFormalDependencies_DependenciasSection(t *testing.T) {
	rec := makeRecord("## Dependencias\n\n- Requiere F01 completado\n- Requiere AST parser\n")
	rec.Path = "task.md"

	inferences := DetectFormalDependencies([]*extract.Record{rec})

	var candidates []string
	for _, inf := range inferences {
		if inf.Type == "informal_dependency_candidate" {
			candidates = append(candidates, inf.Value)
		}
	}

	if len(candidates) != 2 {
		t.Fatalf("expected 2 informal candidates, got %d: %v", len(candidates), candidates)
	}
	if candidates[0] != "Requiere F01 completado" {
		t.Errorf("unexpected first candidate: %s", candidates[0])
	}
}

func TestDetectFormalDependencies_NoDependencies(t *testing.T) {
	rec := &extract.Record{
		Path: "task.md",
		Links: []extract.Link{
			{Target: "other.md", Type: "reference", Line: 1},
		},
	}

	inferences := DetectFormalDependencies([]*extract.Record{rec})
	if len(inferences) != 0 {
		t.Errorf("expected no inferences, got %v", inferences)
	}
}

func TestDetectFormalDependencies_EmptyRecords(t *testing.T) {
	inferences := DetectFormalDependencies(nil)
	if inferences != nil {
		t.Errorf("expected nil, got %v", inferences)
	}
}

func TestDetectFormalDependencies_EmptyListItem(t *testing.T) {
	// Direct record with body content that has empty list items.
	// Use raw body string with section content containing empty items.
	rec := makeRecord("## Dependencias\n\n- \n- Valid dependency\n- \n")
	rec.Path = "task.md"

	inferences := DetectFormalDependencies([]*extract.Record{rec})

	var candidates []string
	for _, inf := range inferences {
		if inf.Type == "informal_dependency_candidate" {
			candidates = append(candidates, inf.Value)
		}
	}

	// Only "Valid dependency" should be extracted; empty items skipped.
	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d: %v", len(candidates), candidates)
	}
	if candidates[0] != "Valid dependency" {
		t.Errorf("unexpected candidate: %s", candidates[0])
	}
}

func TestDetectFormalDependencies_EmptyTextAfterDash(t *testing.T) {
	// Record with empty dash items ("- " with only spaces) and a valid dep.
	// The goldmark AST section content preserves these lines.
	rec := makeRecord("## Dependencias\n\n-   \n- Real dep\n\n## Contexto\n\nSome context.\n")
	rec.Path = "task.md"

	inferences := DetectFormalDependencies([]*extract.Record{rec})

	var candidates []string
	for _, inf := range inferences {
		if inf.Type == "informal_dependency_candidate" {
			candidates = append(candidates, inf.Value)
		}
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate, got %d: %v", len(candidates), candidates)
	}
}

func TestDetectFormalDependencies_AsteriskListItems(t *testing.T) {
	rec := makeRecord("## Dependencias\n\n* Dep with asterisk\n")
	rec.Path = "task.md"

	inferences := DetectFormalDependencies([]*extract.Record{rec})

	var candidates []string
	for _, inf := range inferences {
		if inf.Type == "informal_dependency_candidate" {
			candidates = append(candidates, inf.Value)
		}
	}

	if len(candidates) != 1 {
		t.Fatalf("expected 1 candidate for asterisk list, got %d", len(candidates))
	}
}

func TestDetectFormalDependencies_AllPrefixes(t *testing.T) {
	rec := &extract.Record{
		Path: "task.md",
		Links: []extract.Link{
			{Target: "A", Type: "blocks", Line: 1},
			{Target: "B", Type: "relates", Line: 2},
			{Target: "C", Type: "extends", Line: 3},
		},
	}

	inferences := DetectFormalDependencies([]*extract.Record{rec})
	if len(inferences) != 3 {
		t.Fatalf("expected 3 inferences for 3 semantic prefixes, got %d", len(inferences))
	}
}
