package infer

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/yuin/goldmark"
	"github.com/yuin/goldmark/text"
)

func loadSemanticFixture(t *testing.T, name string) *extract.Record {
	t.Helper()
	path := filepath.Join("testdata", "fixtures", "semantic", name)
	data, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read %s: %v", path, err)
	}

	ext := &extract.MarkdownExtractor{}
	rec, err := ext.Extract(name, data)
	if err != nil {
		t.Fatalf("extract %s: %v", name, err)
	}

	if rec.Body != "" {
		source := []byte(rec.Body)
		reader := text.NewReader(source)
		rec.AST = goldmark.DefaultParser().Parse(reader)
	}

	return rec
}

func TestIntegration_FormalDependencyFixture(t *testing.T) {
	rec := loadSemanticFixture(t, "formal-deps.md")
	// Re-parse links from body (Extract already does this, but let's be explicit).
	rec.Links = extract.ParseLinks(rec.Body)

	inferences := DetectFormalDependencies([]*extract.Record{rec})

	var formal, informal []Inference
	for _, inf := range inferences {
		switch inf.Type {
		case "formal_dependency":
			formal = append(formal, inf)
		case "informal_dependency_candidate":
			informal = append(informal, inf)
		}
	}

	// Should have 1 formal dependency (blocks:T002-other-task).
	if len(formal) != 1 {
		t.Fatalf("expected 1 formal dependency, got %d: %v", len(formal), formal)
	}
	if formal[0].Value != "T002-other-task" {
		t.Errorf("expected target T002-other-task, got %s", formal[0].Value)
	}

	// Should have 2 informal dependency candidates.
	if len(informal) != 2 {
		t.Fatalf("expected 2 informal candidates, got %d: %v", len(informal), informal)
	}
	// Informal messages contain "requires_agent: true".
	for _, inf := range informal {
		if !strings.Contains(inf.Message, "requires_agent: true") {
			t.Errorf("informal dependency should flag requires_agent: true, got: %s", inf.Message)
		}
	}
}

func TestIntegration_TraceabilityFixture(t *testing.T) {
	rec := loadSemanticFixture(t, "traceability.md")

	inferences := DetectTraceabilityLinks([]*extract.Record{rec})

	var verified, unverified []Inference
	for _, inf := range inferences {
		switch inf.Type {
		case "verified_traceability":
			verified = append(verified, inf)
		case "unverified_traceability":
			unverified = append(unverified, inf)
		}
	}

	// Wiki-link target [[E13/F02]] → verified.
	if len(verified) != 1 {
		t.Fatalf("expected 1 verified traceability, got %d: %v", len(verified), verified)
	}
	if verified[0].Value != "E13/F02" {
		t.Errorf("expected E13/F02, got %s", verified[0].Value)
	}

	// Free-text targets: "P1", "P3", "Milestone de F02" → unverified.
	if len(unverified) != 3 {
		t.Fatalf("expected 3 unverified traceability, got %d: %v", len(unverified), unverified)
	}
	for _, inf := range unverified {
		if !strings.Contains(inf.Message, "requires_agent: true") {
			t.Errorf("unverified traceability should flag requires_agent: true, got: %s", inf.Message)
		}
	}
}

func TestIntegration_RequiresAgentConsistency(t *testing.T) {
	// Verify that formal dependency inferences and verified traceability
	// are NOT flagged as requires_agent, while informal/unverified ARE.
	agentTypes := map[string]bool{
		"informal_dependency_candidate": true,
		"unverified_traceability":       true,
	}

	// Build inferences from formal-deps fixture.
	formalRec := loadSemanticFixture(t, "formal-deps.md")
	formalRec.Links = extract.ParseLinks(formalRec.Body)
	formalInfs := DetectFormalDependencies([]*extract.Record{formalRec})

	report := NewAnalyzeReport("/test")
	report.AddCategory("formal_deps", "Formal Dependencies", formalInfs, agentTypes)

	// Build inferences from traceability fixture.
	traceRec := loadSemanticFixture(t, "traceability.md")
	traceInfs := DetectTraceabilityLinks([]*extract.Record{traceRec})

	report.AddCategory("traceability", "Traceability Links", traceInfs, agentTypes)
	report.Finalize()

	// formal_dependency should have requires_agent: false.
	for _, cat := range report.Categories {
		for _, inf := range cat.Inferences {
			switch inf.Type {
			case "formal_dependency":
				if inf.RequiresAgent {
					t.Errorf("formal_dependency should not require agent: %+v", inf)
				}
			case "informal_dependency_candidate":
				if !inf.RequiresAgent {
					t.Errorf("informal_dependency_candidate should require agent: %+v", inf)
				}
			case "verified_traceability":
				if inf.RequiresAgent {
					t.Errorf("verified_traceability should not require agent: %+v", inf)
				}
			case "unverified_traceability":
				if !inf.RequiresAgent {
					t.Errorf("unverified_traceability should require agent: %+v", inf)
				}
			}
		}
	}

	// Summary should reflect the mix.
	if report.Summary.TotalInferences == 0 {
		t.Error("expected non-zero total inferences")
	}
	if report.Summary.AgentRequired == 0 {
		t.Error("expected non-zero agent-required inferences")
	}
	if report.Summary.EngineResolved == 0 {
		t.Error("expected non-zero engine-resolved inferences")
	}
}

func TestIntegration_EmptyDependenciesSection(t *testing.T) {
	rec := loadSemanticFixture(t, "edge-empty-deps.md")

	inferences := DetectFormalDependencies([]*extract.Record{rec})

	// No links, empty Dependencias section → no inferences.
	if len(inferences) != 0 {
		t.Errorf("expected no inferences from empty deps section, got %v", inferences)
	}
}

func TestIntegration_NoTraceabilityPatterns(t *testing.T) {
	rec := loadSemanticFixture(t, "edge-empty-deps.md")

	inferences := DetectTraceabilityLinks([]*extract.Record{rec})

	if len(inferences) != 0 {
		t.Errorf("expected no traceability inferences, got %v", inferences)
	}
}
