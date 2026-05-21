package e2e

import (
	"context"
	"encoding/json"
	"testing"

	"github.com/pablontiv/rootline/internal/derive"
	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/infer"
	"github.com/pablontiv/rootline/internal/rules"
)

// runAnalyze replicates the analyze command pipeline for testing.
func runAnalyze(t *testing.T, root string) *infer.AnalyzeReport {
	t.Helper()
	ctx := context.Background()

	reg := extract.NewRegistry()
	resolver := func(dir string) *rules.StemFile {
		entries, err := rules.WalkUp(dir)
		if err != nil || len(entries) == 0 {
			return nil
		}
		return rules.MergeStemFiles(entries)
	}

	records, err := index.Scan(ctx, root, reg, index.WithScopeResolver(resolver))
	if err != nil {
		t.Fatalf("scan: %v", err)
	}

	derive.DeriveAllSimple(ctx, records, root)
	derive.AggregateAllSimple(ctx, records, root)

	g := graph.Build(ctx, records)
	agentTypes := map[string]bool{
		"informal_dependency_candidate": true,
		"unverified_traceability":       true,
		"missing_domain":                true,
		"implicit_schema":               true,
		"naming_inconsistency":          true,
		"enum_without_values":           true,
		"required_understatement":       true,
	}

	report := infer.NewAnalyzeReport(root)

	// Run detectors.
	schema := infer.Analyze(records)
	report.AddCategory("field_types", "Field Type Inference", fieldStatsInferences(schema), agentTypes)
	report.AddCategory("constant_fields", "Constant Field Detection", infer.DetectConstantFields(records), agentTypes)
	report.AddCategory("section_patterns", "Body Section Patterns", infer.DetectSectionPatterns(records, 0.80), agentTypes)
	report.AddCategory("invariants", "Invariant Extraction", infer.DetectInvariants(records), agentTypes)
	report.AddCategory("formal_deps", "Formal Dependencies", infer.DetectFormalDependencies(records), agentTypes)
	report.AddCategory("traceability", "Traceability Links", infer.DetectTraceabilityLinks(records), agentTypes)
	report.AddCategory("link_types", "Link Types", infer.DetectLinkTypes(records, rules.LinkSchema{}), agentTypes)
	report.AddCategory("back_refs", "Back References", infer.DetectMissingBackReferences(g), agentTypes)
	report.AddCategory("cross_refs", "Cross References", infer.DetectCrossReferences(records, root), agentTypes)

	// Governance detectors.
	var stem *rules.StemFile
	stemEntries, walkErr := rules.WalkUp(root)
	if walkErr == nil && len(stemEntries) > 0 {
		stem = rules.MergeStemFiles(stemEntries)
	}

	report.AddCategory("schema_coverage", "Schema Coverage", infer.DetectMissingSchemata(root), agentTypes)

	// Collect prior inferences for deduplication.
	var priorInfs []infer.Inference
	for _, cat := range report.Categories {
		for _, ri := range cat.Inferences {
			priorInfs = append(priorInfs, infer.Inference{Type: ri.Type, Field: ri.Field, Value: ri.Value})
		}
	}
	report.AddCategory("validation_gaps", "Validation Gaps", infer.DetectValidationGaps(stem, records, priorInfs), agentTypes)

	report.Finalize()
	return report
}

func fieldStatsInferences(schema *infer.InferredSchema) []infer.Inference {
	var inferences []infer.Inference
	for name, stats := range schema.Fields {
		inferences = append(inferences, infer.Inference{
			Type:  "field_type",
			Field: name,
			Value: stats.InferredType,
		})
	}
	return inferences
}

func TestAnalyze_JSONReport(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: enum\n    values: [Pending, Done]\n  tipo:\n    type: string\n",
		"doc1.md": "---\nestado: Pending\ntipo: task\n---\n## Contexto\n\nSome content.\n",
		"doc2.md": "---\nestado: Pending\ntipo: task\n---\n## Contexto\n\nOther content.\n",
		"doc3.md": "---\nestado: Pending\ntipo: task\n---\n## Contexto\n\nMore content.\n",
	})

	report := runAnalyze(t, root)

	// Verify JSON roundtrip.
	data, err := json.Marshal(report)
	if err != nil {
		t.Fatalf("marshal: %v", err)
	}

	var decoded infer.AnalyzeReport
	if err := json.Unmarshal(data, &decoded); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}

	if decoded.Version != 1 {
		t.Errorf("expected version 1, got %d", decoded.Version)
	}
	if decoded.Kind != "analyze" {
		t.Errorf("expected kind 'analyze', got %s", decoded.Kind)
	}
}

func TestAnalyze_EmptyDirectory(t *testing.T) {
	// Use a domain-annotated field so governance detectors produce no inferences.
	// The test intent is: no documents → no data inferences.
	root := setupProject(t, map[string]string{
		".stem": "version: 2\nschema:\n  estado:\n    type: string\n    domain: lifecycle\n",
	})

	report := runAnalyze(t, root)

	if report.Version != 1 {
		t.Errorf("expected version 1, got %d", report.Version)
	}
	if report.Summary.TotalInferences != 0 {
		t.Errorf("expected 0 inferences for empty dir, got %d", report.Summary.TotalInferences)
	}
}

func TestAnalyze_ConstantField(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   "version: 2\nschema:\n  estado:\n    type: string\n  tipo:\n    type: string\n",
		"doc1.md": "---\nestado: Pending\ntipo: task\n---\nBody 1\n",
		"doc2.md": "---\nestado: Pending\ntipo: task\n---\nBody 2\n",
		"doc3.md": "---\nestado: Pending\ntipo: task\n---\nBody 3\n",
	})

	report := runAnalyze(t, root)

	// "estado" and "tipo" are constant across all 3 records.
	foundConstant := false
	for _, cat := range report.Categories {
		if cat.ID != "constant_fields" {
			continue
		}
		for _, inf := range cat.Inferences {
			if inf.Type == "constant_field" && inf.Field == "estado" {
				foundConstant = true
			}
		}
	}
	if !foundConstant {
		t.Error("expected constant_field inference for 'estado'")
	}
}
