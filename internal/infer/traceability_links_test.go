package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestDetectTraceabilityLinks_ContribuyeA(t *testing.T) {
	rec := &extract.Record{
		Path: "task.md",
		Body: "**Contribuye a**: tests pass\n",
	}

	inferences := DetectTraceabilityLinks([]*extract.Record{rec})

	if len(inferences) != 1 {
		t.Fatalf("expected 1 inference, got %d: %v", len(inferences), inferences)
	}
	inf := inferences[0]
	if inf.Type != "unverified_traceability" || inf.Field != "contribuye_a" || inf.Value != "tests pass" {
		t.Errorf("unexpected inference: %+v", inf)
	}
}

func TestDetectTraceabilityLinks_SatisfaceMultiple(t *testing.T) {
	rec := &extract.Record{
		Path: "feature.md",
		Body: "**Satisface**: P1, P3\n",
	}

	inferences := DetectTraceabilityLinks([]*extract.Record{rec})

	if len(inferences) != 2 {
		t.Fatalf("expected 2 inferences for comma-separated targets, got %d: %v", len(inferences), inferences)
	}
	if inferences[0].Value != "P1" || inferences[1].Value != "P3" {
		t.Errorf("unexpected values: %s, %s", inferences[0].Value, inferences[1].Value)
	}
}

func TestDetectTraceabilityLinks_WikiLinkVerified(t *testing.T) {
	rec := &extract.Record{
		Path: "task.md",
		Body: "**Contribuye a**: [[E13/F01]]\n",
	}

	inferences := DetectTraceabilityLinks([]*extract.Record{rec})

	if len(inferences) != 1 {
		t.Fatalf("expected 1 inference, got %d: %v", len(inferences), inferences)
	}
	if inferences[0].Type != "verified_traceability" || inferences[0].Value != "E13/F01" {
		t.Errorf("unexpected inference: %+v", inferences[0])
	}
}

func TestDetectTraceabilityLinks_FreeTextUnverified(t *testing.T) {
	rec := &extract.Record{
		Path: "task.md",
		Body: "**Cubre**: Milestone de F02\n",
	}

	inferences := DetectTraceabilityLinks([]*extract.Record{rec})

	if len(inferences) != 1 {
		t.Fatalf("expected 1 inference, got %d", len(inferences))
	}
	if inferences[0].Type != "unverified_traceability" {
		t.Errorf("expected unverified_traceability, got %s", inferences[0].Type)
	}
}

func TestDetectTraceabilityLinks_NoPatterns(t *testing.T) {
	rec := &extract.Record{
		Path: "task.md",
		Body: "## Contexto\n\nJust regular content.\n",
	}

	inferences := DetectTraceabilityLinks([]*extract.Record{rec})
	if len(inferences) != 0 {
		t.Errorf("expected no inferences, got %v", inferences)
	}
}

func TestDetectTraceabilityLinks_EmptyBody(t *testing.T) {
	rec := &extract.Record{Path: "empty.md", Body: ""}
	inferences := DetectTraceabilityLinks([]*extract.Record{rec})
	if inferences != nil {
		t.Errorf("expected nil, got %v", inferences)
	}
}
