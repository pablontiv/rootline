package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func makeRecord(body string) *extract.Record {
	source := []byte(body)
	reader := text.NewReader(source)
	node := goldmark.DefaultParser().Parse(reader)
	return &extract.Record{
		Path: "test.md",
		Body: body,
		AST:  node,
	}
}

func TestDetectSectionPatterns_RequiredSection(t *testing.T) {
	// 5/5 records have "## Contexto" → required_section
	records := make([]*extract.Record, 5)
	for i := range records {
		records[i] = makeRecord("## Contexto\n\nSome context here.\n")
	}

	inferences := DetectSectionPatterns(records)
	if len(inferences) == 0 {
		t.Fatal("expected at least one inference, got none")
	}

	found := false
	for _, inf := range inferences {
		if inf.Type == "required_section" && inf.Field == "Contexto" {
			found = true
			if inf.Value != "1.00" {
				t.Errorf("expected frequency 1.00, got %s", inf.Value)
			}
		}
	}
	if !found {
		t.Errorf("expected required_section for 'Contexto', got %v", inferences)
	}
}

func TestDetectSectionPatterns_OptionalSection(t *testing.T) {
	// 1/5 records has "## Notas" → optional_section
	records := make([]*extract.Record, 5)
	for i := range records {
		body := "## Contexto\n\nSome text.\n"
		if i == 0 {
			body += "\n## Notas\n\nA note.\n"
		}
		records[i] = makeRecord(body)
	}

	inferences := DetectSectionPatterns(records)

	foundOptional := false
	for _, inf := range inferences {
		if inf.Type == "optional_section" && inf.Field == "Notas" {
			foundOptional = true
			if inf.Value != "0.20" {
				t.Errorf("expected frequency 0.20, got %s", inf.Value)
			}
		}
	}
	// 1/5 = 0.20 which is not < 0.20, so should NOT be optional
	// Actually 0.20 is not < 0.20. Let me reconsider the threshold.
	// The task says <20% for optional. 1/5 = 20% exactly, so it should NOT be optional.
	if foundOptional {
		t.Error("1/5 = 20% should not be optional_section (threshold is <20%)")
	}
}

func TestDetectSectionPatterns_StrictOptional(t *testing.T) {
	// 1/6 records has "## Notas" → ~16.7% → optional_section
	records := make([]*extract.Record, 6)
	for i := range records {
		body := "## Contexto\n\nSome text.\n"
		if i == 0 {
			body += "\n## Notas\n\nA note.\n"
		}
		records[i] = makeRecord(body)
	}

	inferences := DetectSectionPatterns(records)

	foundOptional := false
	for _, inf := range inferences {
		if inf.Type == "optional_section" && inf.Field == "Notas" {
			foundOptional = true
		}
	}
	if !foundOptional {
		t.Error("expected optional_section for 'Notas' at 1/6 = 16.7%")
	}
}

func TestDetectSectionPatterns_EmptyBodyIgnored(t *testing.T) {
	records := []*extract.Record{
		makeRecord("## Contexto\n\nText.\n"),
		{Path: "empty.md", Body: "", AST: nil}, // no AST
	}

	inferences := DetectSectionPatterns(records)

	// Only 1 valid record, Contexto appears in 1/1 = 100% → required
	found := false
	for _, inf := range inferences {
		if inf.Type == "required_section" && inf.Field == "Contexto" {
			found = true
		}
	}
	if !found {
		t.Error("expected required_section for 'Contexto' (empty body record should be excluded)")
	}
}

func TestDetectSectionPatterns_NoRecords(t *testing.T) {
	inferences := DetectSectionPatterns(nil)
	if inferences != nil {
		t.Errorf("expected nil for empty records, got %v", inferences)
	}
}

func TestDetectSectionPatterns_AllEmptyBodies(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Body: "", AST: nil},
		{Path: "b.md", Body: "", AST: nil},
	}

	inferences := DetectSectionPatterns(records)
	if inferences != nil {
		t.Errorf("expected nil for all-empty records, got %v", inferences)
	}
}
