package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

// loadBodyAwareFixtures loads fixtures from body-aware dir, re-parsing AST for body content.
func loadBodyAwareFixtures(t *testing.T, pattern string) []*extract.Record {
	t.Helper()
	records := loadFixtures(t, "testdata/fixtures/body-aware")

	// Filter by pattern prefix (e.g., "section-" or "invariant-").
	var filtered []*extract.Record
	for _, rec := range records {
		matched, _ := matchPrefix(rec.Path, pattern)
		if !matched {
			continue
		}
		// Re-parse AST from body (loadFixtures uses file-based AST which includes frontmatter offsets).
		if rec.Body != "" {
			source := []byte(rec.Body)
			reader := text.NewReader(source)
			rec.AST = goldmark.DefaultParser().Parse(reader)
		}
		filtered = append(filtered, rec)
	}
	return filtered
}

// matchPrefix checks if a filename starts with the given prefix.
func matchPrefix(path, prefix string) (bool, error) {
	// Extract filename from path.
	for i := len(path) - 1; i >= 0; i-- {
		if path[i] == '/' || path[i] == '\\' {
			path = path[i+1:]
			break
		}
	}
	if len(path) >= len(prefix) && path[:len(prefix)] == prefix {
		return true, nil
	}
	return false, nil
}

func TestIntegration_BodySectionPatterns(t *testing.T) {
	records := loadBodyAwareFixtures(t, "section-")
	if len(records) != 5 {
		t.Fatalf("expected 5 section fixtures, got %d", len(records))
	}

	inferences := DetectSectionPatterns(records, 0.80)

	// Contexto, Alcance, Criterios de Aceptacion appear in 5/5 → required
	requiredSections := map[string]bool{
		"Contexto":                false,
		"Alcance":                 false,
		"Criterios de Aceptacion": false,
	}

	for _, inf := range inferences {
		if inf.Type == "required_section" {
			if _, expected := requiredSections[inf.Field]; expected {
				requiredSections[inf.Field] = true
			}
		}
	}

	for section, found := range requiredSections {
		if !found {
			t.Errorf("expected required_section for %q", section)
		}
	}

	// "Notas" appears in 1/5 = 20% → NOT optional (threshold is <20%)
	for _, inf := range inferences {
		if inf.Type == "optional_section" && inf.Field == "Notas" {
			t.Error("Notas at 1/5=20% should not be optional_section")
		}
	}
}

func TestIntegration_InvariantExtraction(t *testing.T) {
	records := loadBodyAwareFixtures(t, "invariant-doc1")
	if len(records) != 1 {
		t.Fatalf("expected 1 invariant fixture, got %d", len(records))
	}

	inferences := DetectInvariants(records)

	expectedIDs := map[string]bool{"INV1": false, "INV2": false, "INV3": false}
	for _, inf := range inferences {
		if _, ok := expectedIDs[inf.Field]; ok {
			expectedIDs[inf.Field] = true
		}
	}

	for id, found := range expectedIDs {
		if !found {
			t.Errorf("expected invariant %s to be extracted", id)
		}
	}
}

func TestIntegration_InvariantEdgeCases(t *testing.T) {
	records := loadBodyAwareFixtures(t, "invariant-edge")
	if len(records) != 1 {
		t.Fatalf("expected 1 edge case fixture, got %d", len(records))
	}

	inferences := DetectInvariants(records)

	// "INV:" without number should not match
	// "INV1:" in Contexto section should not match
	if len(inferences) != 0 {
		t.Errorf("expected no inferences from edge case fixture, got %v", inferences)
	}
}

func TestIntegration_SubSchemaDetection(t *testing.T) {
	records := loadFixtures(t, "testdata/fixtures/body-aware")

	// Filter to subschema- prefixed fixtures.
	var subschemaRecords []*extract.Record
	for _, rec := range records {
		matched, _ := matchPrefix(rec.Path, "subschema-")
		if matched {
			subschemaRecords = append(subschemaRecords, rec)
		}
	}
	if len(subschemaRecords) != 5 {
		t.Fatalf("expected 5 subschema fixtures, got %d", len(subschemaRecords))
	}

	inferences := DetectSubSchemas(subschemaRecords, "tipo")

	// "ejecutable_en" is specific to tipo=test (3 records), not in implementacion.
	// "componente" is specific to tipo=implementacion (2 records), not in test.
	foundEjec := false
	foundComp := false
	for _, inf := range inferences {
		if inf.Type == "type_specific_field" && inf.Field == "ejecutable_en" && inf.Value == "test" {
			foundEjec = true
		}
		if inf.Type == "type_specific_field" && inf.Field == "componente" && inf.Value == "implementacion" {
			foundComp = true
		}
	}
	if !foundEjec {
		t.Error("expected type_specific_field for ejecutable_en in test group")
	}
	if !foundComp {
		t.Error("expected type_specific_field for componente in implementacion group")
	}
}

func TestIntegration_BodyNoHeadings(t *testing.T) {
	records := loadBodyAwareFixtures(t, "no-headings")
	if len(records) != 1 {
		t.Fatalf("expected 1 no-headings fixture, got %d", len(records))
	}

	// Should not panic
	inferences := DetectSectionPatterns(records, 0.80)

	for _, inf := range inferences {
		if inf.Type == "required_section" || inf.Type == "optional_section" {
			t.Errorf("unexpected section inference from headingless doc: %+v", inf)
		}
	}
}
