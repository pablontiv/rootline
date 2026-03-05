package infer

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/rules"
)

// loadFixtures reads all .md files from a testdata directory and returns extract.Records.
func loadFixtures(t *testing.T, dir string) []*extract.Record {
	t.Helper()
	entries, err := os.ReadDir(dir)
	if err != nil {
		t.Fatal(err)
	}
	ext := &extract.MarkdownExtractor{}
	var records []*extract.Record
	for _, e := range entries {
		if e.IsDir() || filepath.Ext(e.Name()) != ".md" {
			continue
		}
		data, err := os.ReadFile(filepath.Join(dir, e.Name()))
		if err != nil {
			t.Fatal(err)
		}
		rec, err := ext.Extract(e.Name(), data)
		if err != nil {
			t.Fatal(err)
		}
		records = append(records, rec)
	}
	return records
}

func TestIntegrationCat5WithFixtures(t *testing.T) {
	records := loadFixtures(t, "testdata/categories")

	// Inference mode (no allowed list): "reference" is the only link type in fixtures.
	got := DetectLinkTypes(records, rules.LinkSchema{})
	// All links are type "reference" (wiki-links default), so 100% consensus.
	found := false
	for _, inf := range got {
		if inf.Type == "link_type_suggestion" {
			found = true
		}
	}
	if !found {
		t.Error("expected link_type_suggestion for unanimous 'reference' links")
	}

	// Validate mode: only "reference" allowed — should produce no violations.
	schema := rules.LinkSchema{Allowed: []string{"reference"}}
	violations := DetectLinkTypes(records, schema)
	for _, v := range violations {
		if v.Type == "link_type_violation" {
			t.Errorf("unexpected violation: %s", v.Message)
		}
	}
}

func TestIntegrationCat7WithFixtures(t *testing.T) {
	records := loadFixtures(t, "testdata/categories")

	g := graph.Build(context.Background(), records)
	got := DetectMissingBackReferences(g)

	// doc1↔doc2 bidirectional, but doc4→doc1 has no back-ref from doc1.
	foundMissing := false
	for _, inf := range got {
		if inf.Type == "missing_back_reference" {
			foundMissing = true
		}
	}
	if !foundMissing {
		t.Error("expected at least one missing_back_reference for doc4→doc1")
	}
}

func TestIntegrationCat8WithFixtures(t *testing.T) {
	records := loadFixtures(t, "testdata/categories")

	got := DetectConstantFields(records)

	// "estado" is "Specified" in all 4 docs → constant.
	// "ejecutable_en" is "1 sesion" in all 4 docs → constant.
	// "tipo" has "software-module" (3) and "test" (1) → NOT constant.
	constants := make(map[string]string)
	for _, inf := range got {
		if inf.Type == "constant_field" {
			constants[inf.Field] = inf.Value
		}
	}
	if v, ok := constants["estado"]; !ok || v != "Specified" {
		t.Errorf("expected estado=Specified as constant, got %q (present=%v)", v, ok)
	}
	if v, ok := constants["ejecutable_en"]; !ok || v != "1 sesion" {
		t.Errorf("expected ejecutable_en='1 sesion' as constant, got %q (present=%v)", v, ok)
	}
	if _, ok := constants["tipo"]; ok {
		t.Error("tipo should NOT be constant (has 2 distinct values)")
	}
}

func TestIntegrationCat10WithFixtures(t *testing.T) {
	// Use the real docs/epics/ directory for path resolution.
	rootDir := filepath.Join("..", "..", "docs", "epics")
	if _, err := os.Stat(rootDir); err != nil {
		t.Skip("docs/epics/ not available, skipping cross-reference integration test")
	}

	records := loadFixtures(t, "testdata/categories")
	got := DetectCrossReferences(records, rootDir)

	// doc1 references E13/F02/S001 → should resolve (exists in docs/epics/)
	// doc2 references E99/F01 → broken
	var validRefs, brokenRefs []string
	for _, inf := range got {
		switch inf.Type {
		case "cross_reference":
			validRefs = append(validRefs, inf.Value)
		case "broken_cross_reference":
			brokenRefs = append(brokenRefs, inf.Value)
		}
	}

	if len(validRefs) == 0 {
		t.Error("expected at least one valid cross_reference (E13/F02/S001)")
	}

	foundBroken := false
	for _, ref := range brokenRefs {
		if ref == "E99/F01" {
			foundBroken = true
		}
	}
	if !foundBroken {
		t.Error("expected broken_cross_reference for E99/F01")
	}
}

func TestIntegrationEmptyRecordsNoPanic(t *testing.T) {
	var empty []*extract.Record

	// None of these should panic.
	got5 := DetectLinkTypes(empty, rules.LinkSchema{})
	got7 := DetectMissingBackReferences(graph.Build(context.Background(), empty))
	got8 := DetectConstantFields(empty)
	got10 := DetectCrossReferences(empty, t.TempDir())

	if got5 != nil {
		t.Errorf("Cat5: expected nil for empty, got %d", len(got5))
	}
	if got7 != nil {
		t.Errorf("Cat7: expected nil for empty, got %d", len(got7))
	}
	if got8 != nil {
		t.Errorf("Cat8: expected nil for empty, got %d", len(got8))
	}
	if got10 != nil {
		t.Errorf("Cat10: expected nil for empty, got %d", len(got10))
	}
}

func TestIntegrationSingleRecordNoPanic(t *testing.T) {
	records := []*extract.Record{
		{Path: "solo.md", Frontmatter: map[string]any{"estado": "Draft"}, Body: "No links here."},
	}

	// Should not panic with 1 record.
	_ = DetectLinkTypes(records, rules.LinkSchema{})
	_ = DetectMissingBackReferences(graph.Build(context.Background(), records))
	_ = DetectConstantFields(records) // <3 records, returns nil
	_ = DetectCrossReferences(records, t.TempDir())
}

func TestAllDetectorsReturnSliceNotNil(t *testing.T) {
	// With enough records producing inferences, verify return type is []Inference.
	records := make([]*extract.Record, 5)
	for i := range records {
		records[i] = &extract.Record{
			Path:        "doc.md",
			Frontmatter: map[string]any{"estado": "Specified"},
			Links:       []extract.Link{{Target: "other.md", Type: "reference", Line: 1}},
		}
	}

	got8 := DetectConstantFields(records)
	if got8 == nil {
		t.Error("Cat8: expected non-nil slice when inferences are produced")
	}

	got5 := DetectLinkTypes(records, rules.LinkSchema{})
	if got5 == nil {
		t.Error("Cat5: expected non-nil slice when inferences are produced")
	}
}
