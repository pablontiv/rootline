// Package e2e end-to-end integration tests for rootline set.
package e2e

import (
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/fix"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
)

const setTestStem = `version: 2
schema:
    estado:
        type: enum
        values: [bloqueado, listo-para-implementar, descartado]
        required: true
    bloqueador:
        type: string
        required: true
    contexto:
        type: section
        heading: "## Contexto"
        required: true
    investigacion:
        type: section
        heading: "## Investigación"
        required: false
`

const setTestDoc = `---
estado: bloqueado
bloqueador: "needs API key"
---

# B023

## Contexto

Original context.
`

// TestSet_FullPipeline exercises the complete set pipeline end-to-end:
// set frontmatter fields + create a section in one invocation, then
// verify the result with schema validation.
func TestSet_FullPipeline(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   setTestStem,
		"B023.md": setTestDoc,
	})

	docPath := filepath.Join(root, "B023.md")
	relPath := "B023.md"

	// Read and extract the document.
	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading doc: %v", err)
	}

	parseAST := true
	ext := &extract.MarkdownExtractor{ParseAST: &parseAST}
	record, err := ext.Extract(relPath, content)
	if err != nil {
		t.Fatalf("extracting doc: %v", err)
	}

	// Load effective schema.
	effective, err := rules.ResolveForRecord(root, docPath)
	if err != nil {
		t.Fatalf("resolving stem: %v", err)
	}

	// Build proposals:
	//   1. Set estado = listo-para-implementar (enum frontmatter)
	//   2. Clear bloqueador = "" (string frontmatter)
	//   3. Create investigacion section with --create mode
	proposals := []proposal.Proposal{
		{
			Type:        proposal.SetField,
			Field:       "estado",
			Value:       "listo-para-implementar",
			Paths:       []string{relPath},
			Description: "set estado to listo-para-implementar",
		},
		{
			Type:        proposal.SetField,
			Field:       "bloqueador",
			Value:       "",
			Paths:       []string{relPath},
			Description: "clear bloqueador",
		},
		{
			Type:    proposal.SetSection,
			Field:   "investigacion",
			Heading: "## Investigación",
			Value:   "Hallazgos iniciales.",
			Mode:    "create",
			Paths:   []string{relPath},
		},
	}

	report := &proposal.Report{
		Version:   1,
		Kind:      "rootline/proposals",
		Proposals: proposals,
	}

	recordMap := []*extract.Record{record}
	if err := fix.ApplyProposals(context.Background(), report, root, recordMap); err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}

	// Read back the modified file.
	updated, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading updated doc: %v", err)
	}
	updatedStr := string(updated)

	// Verify frontmatter updated correctly.
	if !strings.Contains(updatedStr, "estado: listo-para-implementar") {
		t.Errorf("expected 'estado: listo-para-implementar' in file, got:\n%s", updatedStr)
	}

	// Verify bloqueador cleared (empty string).
	if !strings.Contains(updatedStr, "bloqueador:") {
		t.Errorf("expected 'bloqueador:' field to still be present in file, got:\n%s", updatedStr)
	}
	// The old value should no longer appear.
	if strings.Contains(updatedStr, "needs API key") {
		t.Errorf("expected old bloqueador value to be cleared, got:\n%s", updatedStr)
	}

	// Verify original Contexto section preserved.
	if !strings.Contains(updatedStr, "## Contexto") {
		t.Errorf("expected '## Contexto' section preserved, got:\n%s", updatedStr)
	}
	if !strings.Contains(updatedStr, "Original context.") {
		t.Errorf("expected original context content preserved, got:\n%s", updatedStr)
	}

	// Verify new Investigación section created.
	if !strings.Contains(updatedStr, "## Investigación") {
		t.Errorf("expected '## Investigación' section created, got:\n%s", updatedStr)
	}
	if !strings.Contains(updatedStr, "Hallazgos iniciales.") {
		t.Errorf("expected section content present, got:\n%s", updatedStr)
	}

	// Validate the result against the schema — should pass with no errors.
	newContent, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("re-reading doc for validation: %v", err)
	}
	newRecord, err := ext.Extract(relPath, newContent)
	if err != nil {
		t.Fatalf("re-extracting doc for validation: %v", err)
	}

	errs := rules.Validate(context.Background(), newRecord, effective)
	if len(errs) > 0 {
		t.Errorf("expected 0 validation errors after set, got %d: %v", len(errs), errs)
	}
}

// TestSet_RollbackOnInvalidValue verifies that setting an enum field to an
// invalid value causes the command to fail AND leaves the file unchanged.
// This test exercises the pre-validation guard in the set command by running
// the compiled binary so the full cobra execution path is covered.
func TestSet_RollbackOnInvalidValue(t *testing.T) {
	// Locate the rootline binary (built by the task steps).
	binPath, err := exec.LookPath("rootline")
	if err != nil {
		// Try the local build output as fallback.
		binPath = "/usr/local/bin/rootline"
		if _, statErr := os.Stat(binPath); statErr != nil {
			t.Skipf("rootline binary not found (%v); build with 'go build ./cmd/rootline/'", err)
		}
	}

	root := setupProject(t, map[string]string{
		".stem":   setTestStem,
		"B023.md": setTestDoc,
	})

	docPath := filepath.Join(root, "B023.md")

	// Capture the original file content.
	original, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading original doc: %v", err)
	}

	// Run: rootline set B023.md estado=valor-invalido
	// The enum only allows: bloqueado, listo-para-implementar, descartado.
	cmd := exec.Command(binPath, "set", docPath, "estado=valor-invalido") //nolint:gosec
	out, cmdErr := cmd.CombinedOutput()

	// The command must fail.
	if cmdErr == nil {
		t.Fatalf("expected command to fail for invalid enum value, but it succeeded.\nOutput: %s", out)
	}

	// The error message must mention the invalid value or allowed values.
	outStr := string(out)
	if !strings.Contains(outStr, "not in allowed values") && !strings.Contains(outStr, "valor-invalido") {
		t.Errorf("expected error about invalid enum value, got: %s", outStr)
	}

	// The file must be unchanged (no rollback needed since pre-validation
	// prevents any write, but this confirms the guarantee).
	after, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading doc after failed set: %v", err)
	}
	if string(after) != string(original) {
		t.Errorf("file was modified despite failed set command.\nOriginal:\n%s\nAfter:\n%s", original, after)
	}
}
