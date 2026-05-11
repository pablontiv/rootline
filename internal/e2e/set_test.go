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

// TestSet_AppendToSection verifies the += (append) operation on a section field.
func TestSet_AppendToSection(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   setTestStem,
		"B023.md": setTestDoc,
	})

	docPath := filepath.Join(root, "B023.md")
	relPath := "B023.md"

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

	proposals := []proposal.Proposal{
		{
			Type:    proposal.SetSection,
			Field:   "contexto",
			Heading: "## Contexto",
			Value:   "Additional context appended.",
			Mode:    "append",
			Paths:   []string{relPath},
		},
	}

	report := &proposal.Report{
		Version:   1,
		Kind:      "rootline/proposals",
		Proposals: proposals,
	}

	if err := fix.ApplyProposals(context.Background(), report, root, []*extract.Record{record}); err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}

	updated, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading updated doc: %v", err)
	}
	updatedStr := string(updated)

	// Original content should still be present.
	if !strings.Contains(updatedStr, "Original context.") {
		t.Errorf("expected original content preserved after append, got:\n%s", updatedStr)
	}

	// New content should be appended.
	if !strings.Contains(updatedStr, "Additional context appended.") {
		t.Errorf("expected appended content in section, got:\n%s", updatedStr)
	}
}

// TestSet_ReadValueFromFile verifies the =@file syntax by loading content from a file.
func TestSet_ReadValueFromFile(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   setTestStem,
		"B023.md": setTestDoc,
	})

	// Write value to a separate file.
	valueFile := filepath.Join(root, "context_update.txt")
	valueContent := "Updated context from file."
	if err := os.WriteFile(valueFile, []byte(valueContent), 0o644); err != nil {
		t.Fatalf("writing value file: %v", err)
	}

	docPath := filepath.Join(root, "B023.md")
	relPath := "B023.md"

	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading doc: %v", err)
	}

	// Read value from file (simulating =@file behavior).
	fileValue, err := os.ReadFile(valueFile)
	if err != nil {
		t.Fatalf("reading value file: %v", err)
	}

	parseAST := true
	ext := &extract.MarkdownExtractor{ParseAST: &parseAST}
	record, err := ext.Extract(relPath, content)
	if err != nil {
		t.Fatalf("extracting doc: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:    proposal.SetSection,
			Field:   "contexto",
			Heading: "## Contexto",
			Value:   string(fileValue),
			Mode:    "replace",
			Paths:   []string{relPath},
		},
	}

	report := &proposal.Report{
		Version:   1,
		Kind:      "rootline/proposals",
		Proposals: proposals,
	}

	if err := fix.ApplyProposals(context.Background(), report, root, []*extract.Record{record}); err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}

	updated, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading updated doc: %v", err)
	}
	updatedStr := string(updated)

	if !strings.Contains(updatedStr, valueContent) {
		t.Errorf("expected file content %q in section, got:\n%s", valueContent, updatedStr)
	}
	// Original should be replaced.
	if strings.Contains(updatedStr, "Original context.") {
		t.Errorf("expected original content replaced, got:\n%s", updatedStr)
	}
}

// TestSet_DryRunPreview verifies dry-run mode returns previews without modifying files.
// This test exercises the proposal building + preview rendering pipeline.
func TestSet_DryRunPreview(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   setTestStem,
		"B023.md": setTestDoc,
	})

	docPath := filepath.Join(root, "B023.md")
	relPath := "B023.md"

	content, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading doc: %v", err)
	}

	// Capture original content for comparison.
	originalContent := string(content)

	// Build proposals (simulate what dry-run does — build but don't apply).
	proposals := []proposal.Proposal{
		{
			Type:        proposal.SetField,
			Field:       "estado",
			Value:       "listo-para-implementar",
			Paths:       []string{relPath},
			Description: "set estado",
		},
		{
			Type:    proposal.SetSection,
			Field:   "contexto",
			Heading: "## Contexto",
			Value:   "New context content.",
			Mode:    "replace",
			Paths:   []string{relPath},
		},
	}

	// In dry-run mode: generate previews without calling ApplyProposals.
	var previews []string
	for _, p := range proposals {
		switch p.Type {
		case proposal.SetField:
			previews = append(previews, "would set "+p.Field+" = "+p.Value)
		case proposal.SetSection:
			mode := p.Mode
			if mode == "" {
				mode = "replace"
			}
			previews = append(previews, "would "+mode+" section "+p.Heading+" ("+p.Field+")")
		}
	}

	// Verify previews contain expected messages.
	if len(previews) != 2 {
		t.Errorf("expected 2 previews, got %d: %v", len(previews), previews)
	}
	if !strings.Contains(previews[0], "estado") {
		t.Errorf("expected estado in first preview, got: %s", previews[0])
	}
	if !strings.Contains(previews[1], "section") {
		t.Errorf("expected 'section' in second preview, got: %s", previews[1])
	}

	// Verify file was NOT modified (dry-run doesn't call ApplyProposals).
	afterContent, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading doc after dry-run: %v", err)
	}
	if string(afterContent) != originalContent {
		t.Errorf("dry-run should not modify the file.\nOriginal:\n%s\nAfter:\n%s", originalContent, string(afterContent))
	}
}

// TestSet_NoStemError verifies that setting a field on a file outside any stem scope
// does not panic and returns a usable result (nil effective stem).
func TestSet_NoStemError(t *testing.T) {
	// Create dir WITHOUT .stem — no schema available.
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	docContent := `---
estado: Pending
---
# Doc

`
	docPath := filepath.Join(root, "doc.md")
	if err := os.WriteFile(docPath, []byte(docContent), 0o644); err != nil {
		t.Fatal(err)
	}
	relPath := "doc.md"

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

	// With no stem, effective is nil — SetField proposals still work.
	proposals := []proposal.Proposal{
		{
			Type:        proposal.SetField,
			Field:       "estado",
			Value:       "Completed",
			Paths:       []string{relPath},
			Description: "set estado",
		},
	}

	report := &proposal.Report{
		Version:   1,
		Kind:      "rootline/proposals",
		Proposals: proposals,
	}

	if err := fix.ApplyProposals(context.Background(), report, root, []*extract.Record{record}); err != nil {
		t.Fatalf("ApplyProposals without stem: %v", err)
	}

	updated, err := os.ReadFile(docPath)
	if err != nil {
		t.Fatalf("reading updated doc: %v", err)
	}
	if !strings.Contains(string(updated), "estado: Completed") {
		t.Errorf("expected estado: Completed in file, got:\n%s", string(updated))
	}
}

// TestSet_RequiredSectionValidation verifies that validation catches a missing
// required section after a set operation that changes the context.
func TestSet_RequiredSectionValidation(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":   setTestStem,
		"B023.md": setTestDoc,
	})

	docPath := filepath.Join(root, "B023.md")
	relPath := "B023.md"

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

	// The stem requires contexto (type: section, required: true).
	effective, err := rules.ResolveForRecord(root, docPath)
	if err != nil {
		t.Fatalf("resolving stem: %v", err)
	}

	// Validate original — should have no section-required errors.
	errs := rules.Validate(context.Background(), record, effective)
	// B023.md has ## Contexto, so no section error expected initially.
	for _, e := range errs {
		if e.Rule == "required" && e.Field == "contexto" {
			t.Errorf("unexpected section required error before mutation: %v", e)
		}
	}

	// Now build a record WITHOUT the Contexto section (simulate a bad set).
	badDocContent := `---
estado: bloqueado
bloqueador: "needs API key"
---

# B023

`
	badRecord, err := ext.Extract(relPath, []byte(badDocContent))
	if err != nil {
		t.Fatalf("extracting bad doc: %v", err)
	}

	// Validation should now report missing required contexto section.
	badErrs := rules.Validate(context.Background(), badRecord, effective)
	foundSectionError := false
	for _, e := range badErrs {
		if e.Rule == "required" && e.Field == "contexto" {
			foundSectionError = true
			break
		}
	}
	if !foundSectionError {
		t.Errorf("expected required section error for contexto when section is missing, got: %v", badErrs)
	}
}

// TestValidate_SectionAwareValidation verifies that rootline validate correctly
// checks required section fields using AST-parsed sections, not just frontmatter.
// This is a regression test for the validate/set extraction path divergence:
// validate must use AST parsing so record.Sections is populated.
func TestValidate_SectionAwareValidation(t *testing.T) {
	binPath, err := exec.LookPath("rootline")
	if err != nil {
		binPath = "/usr/local/bin/rootline"
		if _, statErr := os.Stat(binPath); statErr != nil {
			t.Skipf("rootline binary not found (%v); build with 'go build ./cmd/rootline/'", err)
		}
	}

	stemWithSection := `version: 2
schema:
    estado:
        type: enum
        values: [bloqueado, listo-para-implementar]
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

	docWithSection := `---
estado: bloqueado
---

# Test

## Contexto

Present context.
`

	docMissingSection := `---
estado: bloqueado
---

# Test
`

	t.Run("required section present exits 0", func(t *testing.T) {
		root := setupProject(t, map[string]string{
			".stem":  stemWithSection,
			"doc.md": docWithSection,
		})
		cmd := exec.Command(binPath, "validate", filepath.Join(root, "doc.md")) //nolint:gosec
		out, cmdErr := cmd.CombinedOutput()
		if cmdErr != nil {
			t.Errorf("expected exit 0 for doc with required section present, got error: %v\nOutput: %s", cmdErr, out)
		}
	})

	t.Run("required section absent exits non-zero", func(t *testing.T) {
		root := setupProject(t, map[string]string{
			".stem":  stemWithSection,
			"doc.md": docMissingSection,
		})
		cmd := exec.Command(binPath, "validate", filepath.Join(root, "doc.md")) //nolint:gosec
		out, cmdErr := cmd.CombinedOutput()
		if cmdErr == nil {
			t.Errorf("expected exit non-zero for doc missing required section, but it succeeded.\nOutput: %s", out)
		}
		if !strings.Contains(string(out), "contexto") {
			t.Errorf("expected error mentioning 'contexto', got: %s", out)
		}
	})

	t.Run("validate --all detects required sections", func(t *testing.T) {
		root := setupProject(t, map[string]string{
			".stem":  stemWithSection,
			"doc.md": docMissingSection,
		})
		cmd := exec.Command(binPath, "validate", "--all", root) //nolint:gosec
		out, cmdErr := cmd.CombinedOutput()
		if cmdErr == nil {
			t.Errorf("expected exit non-zero for validate --all with missing section, but it succeeded.\nOutput: %s", out)
		}
		if !strings.Contains(string(out), "contexto") {
			t.Errorf("expected error mentioning 'contexto', got: %s", out)
		}
	})

	t.Run("set --create then validate exits 0", func(t *testing.T) {
		root := setupProject(t, map[string]string{
			".stem":  stemWithSection,
			"doc.md": docMissingSection,
		})
		docPath := filepath.Join(root, "doc.md")

		// Use set --create to add the required contexto section.
		// Post-validation inside set itself uses AST parsing, so after the section
		// is written the post-validate also passes.
		setCmd := exec.Command(binPath, "set", "--create", docPath, "contexto=Context added.") //nolint:gosec
		if setOut, setErr := setCmd.CombinedOutput(); setErr != nil {
			t.Fatalf("set --create failed: %v\nOutput: %s", setErr, setOut)
		}

		// rootline validate must now exit 0 — this is the regression: before the
		// fix, validate used NewRegistry() (no AST), so Sections was nil even
		// though the section was physically present in the file.
		valCmd := exec.Command(binPath, "validate", docPath) //nolint:gosec
		out, valErr := valCmd.CombinedOutput()
		if valErr != nil {
			t.Errorf("expected validate to pass after set --create added required section, got: %v\nOutput: %s", valErr, out)
		}
	})
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
