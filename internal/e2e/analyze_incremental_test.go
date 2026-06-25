package e2e

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestAnalyzeIncremental_FiltersCoveredInferences(t *testing.T) {
	root := setupProject(t, map[string]string{
		// Stem covers estado as required enum, tipo as string.
		".stem":   "version: 2\nschema:\n  estado:\n    type: enum\n    required: true\n    values: [Pending, Done]\n  tipo:\n    type: string\n",
		"doc1.md": "---\nestado: Pending\ntipo: task\n---\nBody 1\n",
		"doc2.md": "---\nestado: Pending\ntipo: task\n---\nBody 2\n",
		"doc3.md": "---\nestado: Pending\ntipo: task\n---\nBody 3\n",
	})

	// Full analysis.
	fullReport := runAnalyze(t, root)
	fullTotal := fullReport.Summary.TotalInferences

	// Load records and resolver for filtering.
	records := []*extract.Record{
		{Path: "doc1.md", Frontmatter: map[string]any{"estado": "Pending", "tipo": "task"}},
		{Path: "doc2.md", Frontmatter: map[string]any{"estado": "Pending", "tipo": "task"}},
		{Path: "doc3.md", Frontmatter: map[string]any{"estado": "Pending", "tipo": "task"}},
	}
	gapsResolver := infer.DefaultStemResolver()

	// Collect all inferences from full report.
	var allInferences []infer.Inference
	for _, cat := range fullReport.Categories {
		for _, inf := range cat.Inferences {
			allInferences = append(allInferences, infer.Inference{
				Type:    inf.Type,
				Source:  inf.Source,
				Field:   inf.Field,
				Value:   inf.Value,
				Message: inf.Message,
			})
		}
	}

	deltas := infer.FilterCoveredInferences(allInferences, records, root, gapsResolver)

	if len(deltas) >= fullTotal {
		t.Errorf("incremental should have fewer inferences than full: deltas=%d, full=%d", len(deltas), fullTotal)
	}
}

func TestAnalyzeIncremental_FullCoverage_ZeroDeltas(t *testing.T) {
	// Stem covers everything that the detectors would infer.
	inferences := []infer.Inference{
		{Type: "required_field", Field: "estado"},
		{Type: "field_type", Field: "estado", Value: "enum"},
		{Type: "enum_values", Field: "estado", Value: "[Pending Done]"},
		{Type: "constant_field", Field: "estado", Value: "Pending"},
	}

	stem := &rules.StemFile{
		Schema: map[string]rules.SchemaField{
			"estado": {Type: "enum", Required: true, Values: []string{"Pending", "Done"}},
		},
	}

	singleScope := func(dir string) (*rules.StemFile, string) { return stem, "/.stem" }
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	deltas := infer.FilterCoveredInferences(inferences, records, ".", singleScope)
	if len(deltas) != 0 {
		t.Errorf("expected 0 deltas with full coverage, got %d: %v", len(deltas), deltas)
	}
}

func TestAnalyzeIncremental_NoStem_SameAsFull(t *testing.T) {
	inferences := []infer.Inference{
		{Type: "required_field", Field: "estado"},
		{Type: "field_type", Field: "estado", Value: "enum"},
	}

	// nil stem resolver = no filtering.
	nilResolver := func(dir string) (*rules.StemFile, string) { return nil, "" }
	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "draft"}},
	}

	deltas := infer.FilterCoveredInferences(inferences, records, ".", nilResolver)
	if len(deltas) != len(inferences) {
		t.Errorf("expected same count without stem: deltas=%d, full=%d", len(deltas), len(inferences))
	}
}
