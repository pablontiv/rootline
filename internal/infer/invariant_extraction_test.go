package infer

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestDetectInvariants_InvariantesSection(t *testing.T) {
	rec := makeRecord("## Invariantes\n\n- INV1: tests pass\n- INV2: coverage above 85%\n")
	inferences := DetectInvariants([]*extract.Record{rec})

	if len(inferences) != 2 {
		t.Fatalf("expected 2 inferences, got %d: %v", len(inferences), inferences)
	}

	if inferences[0].Type != "invariant_declaration" || inferences[0].Field != "INV1" || inferences[0].Value != "tests pass" {
		t.Errorf("unexpected first inference: %+v", inferences[0])
	}
	if inferences[1].Field != "INV2" || inferences[1].Value != "coverage above 85%" {
		t.Errorf("unexpected second inference: %+v", inferences[1])
	}
}

func TestDetectInvariants_PreservaSection(t *testing.T) {
	rec := makeRecord("## Preserva\n\n- INV3: no regression\n")
	inferences := DetectInvariants([]*extract.Record{rec})

	if len(inferences) != 1 {
		t.Fatalf("expected 1 inference, got %d", len(inferences))
	}
	if inferences[0].Field != "INV3" {
		t.Errorf("expected INV3, got %s", inferences[0].Field)
	}
}

func TestDetectInvariants_CodeBlockIgnored(t *testing.T) {
	// INV1 inside a code block should not be extracted.
	// ExtractSections returns content between headings, but code blocks
	// are rendered as text in the content. However, the AST-based approach
	// means we need a section named "Invariantes" to even look for matches.
	// A code block inside "## Other" won't be scanned.
	rec := makeRecord("## Other\n\n```\nINV1: should not match\n```\n")
	inferences := DetectInvariants([]*extract.Record{rec})

	if len(inferences) != 0 {
		t.Errorf("expected no inferences from non-target section, got %v", inferences)
	}
}

func TestDetectInvariants_NoInvariantSection(t *testing.T) {
	rec := makeRecord("## Contexto\n\nSome context.\n")
	inferences := DetectInvariants([]*extract.Record{rec})

	if len(inferences) != 0 {
		t.Errorf("expected no inferences, got %v", inferences)
	}
}

func TestDetectInvariants_EmptyBody(t *testing.T) {
	rec := &extract.Record{Path: "empty.md", Body: "", AST: nil}
	inferences := DetectInvariants([]*extract.Record{rec})

	if inferences != nil {
		t.Errorf("expected nil for empty body, got %v", inferences)
	}
}
