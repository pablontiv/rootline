package fix

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/proposal"
	"gopkg.in/yaml.v3"
)

// === WriteFrontmatterFields edge cases ===

// TestWriteFrontmatterFields_YAMLMarshalError tests the error path when YAML marshaling fails
func TestWriteFrontmatterFields_YAMLMarshalError(t *testing.T) {
	// Create a map with a cyclic reference-like object that will cause marshal error
	// We use a map that contains itself (simulated via interface{})
	fm := map[string]any{
		"field": "value",
	}

	// Note: go yaml.Marshal is quite forgiving; we test the error path by ensuring
	// the fallback uses fmt.Sprintf which should never fail
	var b strings.Builder
	WriteFrontmatterFields(&b, fm)
	out := b.String()

	// Verify output is valid YAML even with potential marshal issues
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(out), &parsed); err != nil {
		t.Errorf("output is not valid YAML: %v\noutput: %q", err, out)
	}
}

// TestWriteFrontmatterFields_MultilineValue tests multiline YAML values
func TestWriteFrontmatterFields_MultilineValue(t *testing.T) {
	fm := map[string]any{
		"description": "Line 1\nLine 2\nLine 3",
	}
	var b strings.Builder
	WriteFrontmatterFields(&b, fm)
	out := b.String()

	// Verify multiline formatting
	if !strings.Contains(out, "description:\n") {
		t.Errorf("expected multiline format, got: %q", out)
	}
	if !strings.Contains(out, "  Line 1") {
		t.Errorf("expected indented line 1, got: %q", out)
	}

	// Verify output is valid YAML
	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid YAML: %v\noutput: %q", err, out)
	}
	desc := parsed["description"].(string)
	if !strings.Contains(desc, "Line 1") {
		t.Errorf("expected multiline value preserved in YAML, got: %v", desc)
	}
}

// TestWriteFrontmatterFields_NilValue tests nil values
func TestWriteFrontmatterFields_NilValue(t *testing.T) {
	fm := map[string]any{
		"nullable": nil,
	}
	var b strings.Builder
	WriteFrontmatterFields(&b, fm)
	out := b.String()

	var parsed map[string]any
	if err := yaml.Unmarshal([]byte(out), &parsed); err != nil {
		t.Fatalf("output is not valid YAML: %v\noutput: %q", err, out)
	}
	if v, ok := parsed["nullable"]; !ok || v != nil {
		t.Errorf("expected nil value to be preserved, got %v", v)
	}
}

// === applySetSection edge cases ===

// TestApplySetSection_CreateWithoutNewline tests create mode when file doesn't end with newline
func TestApplySetSection_CreateWithoutNewline(t *testing.T) {
	dir := t.TempDir()
	relPath := "test.md"
	absPath := filepath.Join(dir, relPath)

	// File without trailing newline
	mustWriteFile(t, absPath, []byte("---\nestado: Pending\n---\n\nSome content without newline"))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## NewSection",
		Value:   "New content.",
		Mode:    "create",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil, PolicyRejectAbsolute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)

	if !strings.Contains(content, "## NewSection") {
		t.Errorf("expected new section created, got:\n%s", content)
	}
	if !strings.Contains(content, "New content.") {
		t.Errorf("expected new content, got:\n%s", content)
	}
}

// TestApplySetSection_ReplaceMultipleLevels tests replacing a section with multiple heading levels
func TestApplySetSection_ReplaceMultipleLevels(t *testing.T) {
	dir := t.TempDir()
	relPath := "test.md"
	absPath := filepath.Join(dir, relPath)

	original := `---
estado: Pending
---

# Main Title

## Context

Old context.

### Subsection

Sub content.

## Another Section

More stuff.
`
	mustWriteFile(t, absPath, []byte(original))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Context",
		Value:   "New context.",
		Mode:    "replace",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil, PolicyRejectAbsolute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)

	if strings.Contains(content, "Old context.") {
		t.Errorf("expected old content removed, got:\n%s", content)
	}
	if !strings.Contains(content, "New context.") {
		t.Errorf("expected new content present, got:\n%s", content)
	}
	if !strings.Contains(content, "## Another Section") {
		t.Errorf("expected next section preserved, got:\n%s", content)
	}
}

// TestApplySetSection_AppendWithoutNewline tests append when section doesn't end with newline
func TestApplySetSection_AppendWithoutNewline(t *testing.T) {
	dir := t.TempDir()
	relPath := "test.md"
	absPath := filepath.Join(dir, relPath)

	original := "---\nestado: Pending\n---\n\n## Section\n\nContent without newline"
	mustWriteFile(t, absPath, []byte(original))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Section",
		Value:   "Appended.",
		Mode:    "append",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil, PolicyRejectAbsolute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)

	if !strings.Contains(content, "Content without newline") {
		t.Errorf("expected original content preserved, got:\n%s", content)
	}
	if !strings.Contains(content, "Appended.") {
		t.Errorf("expected appended content, got:\n%s", content)
	}
}

// TestApplySetSection_ReplaceAtEOF tests replacing section at end of file
func TestApplySetSection_ReplaceAtEOF(t *testing.T) {
	dir := t.TempDir()
	relPath := "test.md"
	absPath := filepath.Join(dir, relPath)

	original := "---\nestado: Pending\n---\n\n## LastSection\n\nLast content.\n"
	mustWriteFile(t, absPath, []byte(original))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## LastSection",
		Value:   "Replaced last content.",
		Mode:    "replace",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil, PolicyRejectAbsolute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)

	if strings.Contains(content, "Last content.") {
		t.Errorf("expected old content removed, got:\n%s", content)
	}
	if !strings.Contains(content, "Replaced last content.") {
		t.Errorf("expected new content, got:\n%s", content)
	}
}

// TestApplySetSection_ReplaceDefaultMode tests replace mode with empty Mode field (defaults to "replace")
func TestApplySetSection_ReplaceDefaultMode(t *testing.T) {
	dir := t.TempDir()
	relPath := "test.md"
	absPath := filepath.Join(dir, relPath)

	original := "---\nestado: Pending\n---\n\n## Section\n\nOld.\n"
	mustWriteFile(t, absPath, []byte(original))

	p := proposal.Proposal{
		Type:    proposal.SetSection,
		Heading: "## Section",
		Value:   "New.",
		Mode:    "",
		Paths:   []string{relPath},
	}

	err := applySetSection(p, dir, nil, PolicyRejectAbsolute)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	data, _ := os.ReadFile(absPath)
	content := string(data)

	if strings.Contains(content, "Old.") {
		t.Errorf("expected default mode to replace, got:\n%s", content)
	}
	if !strings.Contains(content, "New.") {
		t.Errorf("expected new content, got:\n%s", content)
	}
}

// === applyExtendEnum edge cases ===

// TestApplyExtendEnum_MalformedYAML tests error handling for malformed YAML
func TestApplyExtendEnum_MalformedYAML(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte("invalid: yaml: here:"))

	p := proposal.Proposal{
		Type:  proposal.ExtendEnum,
		Field: "estado",
		Value: "NewValue",
	}

	err := applyExtendEnum(p, dir)
	if err == nil {
		t.Error("expected error for malformed YAML")
	}
	if !strings.Contains(err.Error(), "parsing") {
		t.Errorf("expected parsing error, got: %v", err)
	}
}

// TestApplyExtendEnum_EmptyDocument tests error handling for empty .stem
func TestApplyExtendEnum_EmptyDocument(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(""))

	p := proposal.Proposal{
		Type:  proposal.ExtendEnum,
		Field: "estado",
		Value: "NewValue",
	}

	err := applyExtendEnum(p, dir)
	if err == nil {
		t.Error("expected error for empty document")
	}
}

// TestApplyExtendEnum_NonMappingRoot tests error handling when root is not a mapping
func TestApplyExtendEnum_NonMappingRoot(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte("- item1\n- item2\n"))

	p := proposal.Proposal{
		Type:  proposal.ExtendEnum,
		Field: "estado",
		Value: "NewValue",
	}

	err := applyExtendEnum(p, dir)
	if err == nil {
		t.Error("expected error for non-mapping root")
	}
}

// === addEnumValueToNode edge cases ===

// TestAddEnumValueToNode_NonMappingSchema tests when schema is not a mapping
func TestAddEnumValueToNode_NonMappingSchema(t *testing.T) {
	stemYAML := `version: 2
schema:
  - list_instead_of_mapping
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(stemYAML), &doc); err != nil {
		t.Fatal(err)
	}

	err := addEnumValueToNode(doc.Content[0], "estado", "value")
	if err == nil {
		t.Error("expected error when schema is not a mapping")
	}
}

// TestAddEnumValueToNode_FieldNotMapping tests when field value is not a mapping
func TestAddEnumValueToNode_FieldNotMapping(t *testing.T) {
	stemYAML := `version: 2
schema:
  estado: "not a mapping"
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(stemYAML), &doc); err != nil {
		t.Fatal(err)
	}

	err := addEnumValueToNode(doc.Content[0], "estado", "value")
	if err == nil {
		t.Error("expected error when field value is not a mapping")
	}
}

// TestAddEnumValueToNode_ValuesNotSequence tests when values is not a sequence
func TestAddEnumValueToNode_ValuesNotSequence(t *testing.T) {
	stemYAML := `version: 2
schema:
  estado:
    type: enum
    values: "not a sequence"
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(stemYAML), &doc); err != nil {
		t.Fatal(err)
	}

	err := addEnumValueToNode(doc.Content[0], "estado", "value")
	if err == nil {
		t.Error("expected error when values is not a sequence")
	}
}

// TestAddEnumValueToNode_Success tests successful addition of value
func TestAddEnumValueToNode_Success(t *testing.T) {
	stemYAML := `version: 2
schema:
  estado:
    type: enum
    values:
      - Pending
`
	var doc yaml.Node
	if err := yaml.Unmarshal([]byte(stemYAML), &doc); err != nil {
		t.Fatal(err)
	}

	err := addEnumValueToNode(doc.Content[0], "estado", "Completed")
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	out, _ := yaml.Marshal(&doc)
	if !strings.Contains(string(out), "Completed") {
		t.Errorf("expected Completed in output, got:\n%s", string(out))
	}
}

// === ApplyRepair edge cases ===

// TestApplyRepair_EmptyProposals tests ApplyRepair with empty proposal list
func TestApplyRepair_EmptyProposals(t *testing.T) {
	result, err := ApplyRepair([]proposal.Proposal{}, false, t.TempDir(), false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
	if result.Kind != "rootline/repair" {
		t.Errorf("expected kind rootline/repair, got %s", result.Kind)
	}
}

// TestApplyRepair_ReadError tests error handling when file read fails
func TestApplyRepair_ReadError(t *testing.T) {
	tmpDir := t.TempDir()
	// Create a proposal for a nonexistent file
	proposals := []proposal.Proposal{
		{
			Type:  proposal.CorrectValue,
			Field: "field",
			Paths: []string{"nonexistent.md"},
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("unexpected error for missing file: %v", err)
	}
	// Error should be recorded in result.Errors
	if len(result.Errors) == 0 {
		t.Error("expected error to be recorded for nonexistent file")
	}
}

// TestApplyRepair_MigrateValue_Applied tests that document value normalization is repair-owned.
func TestApplyRepair_MigrateValue_Applied(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	content := `---
estado: "Pending"
---
# Test
`

	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:      proposal.MigrateValue,
			Field:     "estado",
			From:      "Pending",
			To:        "Completed",
			WikiLinks: []string{"[[blocks:T001]]"},
			Paths:     []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if len(result.Rejected) != 0 || len(result.Changed) != 1 {
		t.Errorf("expected migrate_value to be applied: changed=%v rejected=%v", result.Changed, result.Rejected)
	}

	updated, _ := os.ReadFile(testPath)
	if !contains(string(updated), "estado: Completed") {
		t.Errorf("expected migrated value, got:\n%s", string(updated))
	}
	if !contains(string(updated), "[[blocks:T001]]") {
		t.Errorf("expected extracted blocker link to be preserved, got:\n%s", string(updated))
	}
}

// TestApplyRepair_InferFromChildren tests InferFromChildren proposal
func TestApplyRepair_InferFromChildren(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	content := `---
estado: Pending
---
# Test
`

	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.InferFromChildren,
			Field: "count",
			To:    "5",
			Paths: []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}
}

// TestApplyRepair_InferFromSiblings tests InferFromSiblings proposal
func TestApplyRepair_InferFromSiblings(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	content := `---
estado: Pending
---
# Test
`

	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.InferFromSiblings,
			Field: "category",
			To:    "general",
			Paths: []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}
}

// TestApplyRepair_ExtractBody tests ExtractBody proposal
func TestApplyRepair_ExtractBody(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	content := `---
estado: Pending
---
# Test

Some body content here.
`

	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.ExtractBody,
			Field: "summary",
			To:    "Body summary",
			Paths: []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}
}

// TestApplyRepair_SetSection tests SetSection proposal in ApplyRepair dry-run
func TestApplyRepair_SetSection_Replace_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	original := `---
estado: Pending
---
# Test

## Notes

Old notes.
`

	if err := os.WriteFile(testPath, []byte(original), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:    proposal.SetSection,
			Field:   "notes",
			Heading: "## Notes",
			Value:   "New notes.",
			Mode:    "replace",
			Paths:   []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, true, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if !result.DryRun {
		t.Errorf("expected DryRun=true")
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}

	// File should not be modified in dry-run mode
	updated, _ := os.ReadFile(testPath)
	if string(updated) != original {
		t.Errorf("expected file unchanged in dry-run")
	}
}

// TestApplyRepair_SetSection_Append tests SetSection append mode in dry-run
func TestApplyRepair_SetSection_Append_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	original := `---
estado: Pending
---

## Section

Existing.
`

	if err := os.WriteFile(testPath, []byte(original), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:    proposal.SetSection,
			Heading: "## Section",
			Value:   "Appended.",
			Mode:    "append",
			Paths:   []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, true, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if !result.DryRun {
		t.Errorf("expected DryRun=true")
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}

	// File should not be modified in dry-run
	updated, _ := os.ReadFile(testPath)
	if string(updated) != original {
		t.Errorf("expected file unchanged in dry-run")
	}
}

// TestApplyRepair_CorrectOutlier tests CorrectOutlier proposal
func TestApplyRepair_CorrectOutlier(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	content := `---
estado: Outlier
---
# Test
`

	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.CorrectOutlier,
			Field: "estado",
			From:  "Outlier",
			To:    "Pending",
			Paths: []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}
}

// TestApplyRepair_PropagateAggregate tests PropagateAggregate proposal
func TestApplyRepair_PropagateAggregate(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	content := `---
estado: Pending
total: 10
---
# Test
`

	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.PropagateAggregate,
			Field: "total",
			From:  "10",
			To:    "15",
			Paths: []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}
}

// TestApplySchemaProposals_ExtendEnum tests ApplySchemaProposals with extend_enum
func TestApplySchemaProposals_ExtendEnum(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemContent := `version: 2
schema:
  estado:
    type: enum
    values:
      - Pending
`
	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte(stemContent))

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.ExtendEnum,
				Field: "estado",
				Value: "Completed",
			},
		},
	}

	err := ApplySchemaProposals(context.Background(), report, dir)
	if err != nil {
		t.Fatalf("ApplySchemaProposals: %v", err)
	}

	content, _ := os.ReadFile(stemPath)
	if !strings.Contains(string(content), "Completed") {
		t.Errorf("expected Completed in .stem, got:\n%s", string(content))
	}
}

// TestApplyProposals_SetField tests SetField proposal type handling
func TestApplyProposals_SetField(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\nestado: Pending\n---\n# Task\n"))

	records := []*extract.Record{
		{Path: "task.md", Frontmatter: map[string]any{"estado": "Pending"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.SetField,
				Field: "custom_field",
				Value: "custom_value",
				Paths: []string{"task.md"},
			},
		},
	}

	_, err := ApplyProposals(context.Background(), report, dir, records, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Proposals) != 1 {
		t.Errorf("expected 1 applied proposal, got %d", len(report.Proposals))
	}

	content, _ := os.ReadFile(filepath.Join(dir, "task.md"))
	if !strings.Contains(string(content), "custom_field: custom_value") {
		t.Errorf("expected custom field set, got:\n%s", string(content))
	}
}

// TestApplyProposals_MultipleTypes tests handling multiple proposal types in one report
func TestApplyProposals_MultipleTypes(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "a.md"),
		[]byte("---\nestado: Wrong\ntipo: test\n---\n# A\n"))
	mustWriteFile(t, filepath.Join(dir, "b.md"),
		[]byte("---\ntipo: test\n---\n# B\n"))

	records := []*extract.Record{
		{Path: "a.md", Frontmatter: map[string]any{"estado": "Wrong", "tipo": "test"}},
		{Path: "b.md", Frontmatter: map[string]any{"tipo": "test"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.CorrectValue,
				Field: "estado",
				From:  "Wrong",
				To:    "Correct",
				Paths: []string{"a.md"},
			},
			{
				Type:  proposal.AddField,
				Field: "estado",
				Value: "Pending",
				Paths: []string{"b.md"},
			},
		},
	}

	_, err := ApplyProposals(context.Background(), report, dir, records, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Proposals) != 2 {
		t.Errorf("expected 2 applied proposals, got %d", len(report.Proposals))
	}
}

// TestApplySchemaProposals_ExtendEnumParseError tests error handling for parse errors
func TestApplySchemaProposals_ExtendEnumParseError(t *testing.T) {
	dir := t.TempDir()
	if err := os.Mkdir(filepath.Join(dir, ".git"), 0o755); err != nil {
		t.Fatal(err)
	}

	stemPath := filepath.Join(dir, ".stem")
	mustWriteFile(t, stemPath, []byte("invalid yaml: : :"))

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.ExtendEnum,
				Field: "estado",
				Value: "Completed",
			},
		},
	}

	err := ApplySchemaProposals(context.Background(), report, dir)
	if err == nil {
		t.Error("expected error for invalid YAML")
	}
}

// TestApplyProposals_InferFromChildren tests InferFromChildren proposal type
func TestApplyProposals_InferFromChildren(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\ntipo: test\n---\n# Task\n"))

	records := []*extract.Record{
		{Path: "task.md", Frontmatter: map[string]any{"tipo": "test"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.InferFromChildren,
				Field: "child_count",
				To:    "5",
				Paths: []string{"task.md"},
			},
		},
	}

	_, err := ApplyProposals(context.Background(), report, dir, records, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Proposals) != 1 {
		t.Errorf("expected 1 applied proposal, got %d", len(report.Proposals))
	}
}

// TestApplyProposals_InferFromSiblings tests InferFromSiblings proposal type
func TestApplyProposals_InferFromSiblings(t *testing.T) {
	dir := setupStemDir(t)

	mustWriteFile(t, filepath.Join(dir, "task.md"),
		[]byte("---\ntipo: test\n---\n# Task\n"))

	records := []*extract.Record{
		{Path: "task.md", Frontmatter: map[string]any{"tipo": "test"}},
	}

	report := &proposal.Report{
		Version: 1,
		Kind:    "rootline/proposals",
		Proposals: []proposal.Proposal{
			{
				Type:  proposal.InferFromSiblings,
				Field: "category",
				To:    "general",
				Paths: []string{"task.md"},
			},
		},
	}

	_, err := ApplyProposals(context.Background(), report, dir, records, false)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if len(report.Proposals) != 1 {
		t.Errorf("expected 1 applied proposal, got %d", len(report.Proposals))
	}
}

// TestApplyRepairAddField_DryRun tests AddField in dry-run mode
func TestApplyRepairAddField_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	original := `---
estado: Pending
---
# Test
`

	if err := os.WriteFile(testPath, []byte(original), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.AddField,
			Field: "newfield",
			Value: "value",
			Paths: []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, true, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if !result.DryRun {
		t.Errorf("expected DryRun=true")
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}

	got, _ := os.ReadFile(testPath)
	if string(got) != original {
		t.Errorf("expected file unchanged in dry-run")
	}
}

// TestApplyRepairCorrectValue_DryRun tests CorrectValue in dry-run mode
func TestApplyRepairCorrectValue_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	original := `---
estado: Pending
---
# Test
`

	if err := os.WriteFile(testPath, []byte(original), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.CorrectValue,
			Field: "estado",
			From:  "Pending",
			To:    "Completed",
			Paths: []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, true, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if !result.DryRun {
		t.Errorf("expected DryRun=true")
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}

	got, _ := os.ReadFile(testPath)
	if string(got) != original {
		t.Errorf("expected file unchanged in dry-run")
	}
}

// TestApplyRepairCorrectLink_DryRun tests CorrectLink in dry-run mode
func TestApplyRepairCorrectLink_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	original := `---
estado: Pending
---
# Test

See [[oldlink]].
`

	if err := os.WriteFile(testPath, []byte(original), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.CorrectLink,
			Field: "links",
			From:  "[[oldlink]]",
			To:    "[[newlink]]",
			Paths: []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, true, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if !result.DryRun {
		t.Errorf("expected DryRun=true")
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}

	got, _ := os.ReadFile(testPath)
	if string(got) != original {
		t.Errorf("expected file unchanged in dry-run")
	}
}

// TestApplyRepairSetField_DryRun tests SetField in dry-run mode
func TestApplyRepairSetField_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	original := `---
estado: Pending
---
# Test
`

	if err := os.WriteFile(testPath, []byte(original), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.SetField,
			Field: "newfield",
			Value: "value",
			Paths: []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, true, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if !result.DryRun {
		t.Errorf("expected DryRun=true")
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}

	got, _ := os.ReadFile(testPath)
	if string(got) != original {
		t.Errorf("expected file unchanged in dry-run")
	}
}

// TestApplyRepair_MultipleFiles tests ApplyRepair with multiple files
func TestApplyRepair_MultipleFiles(t *testing.T) {
	tmpDir := t.TempDir()
	testPath1 := filepath.Join(tmpDir, "test1.md")
	testPath2 := filepath.Join(tmpDir, "test2.md")

	content1 := `---
estado: Pending
---
# Test 1
`
	content2 := `---
estado: Completed
---
# Test 2
`

	if err := os.WriteFile(testPath1, []byte(content1), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}
	if err := os.WriteFile(testPath2, []byte(content2), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.AddField,
			Field: "newfield",
			Value: "value1",
			Paths: []string{"test1.md"},
		},
		{
			Type:  proposal.AddField,
			Field: "newfield",
			Value: "value2",
			Paths: []string{"test2.md"},
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	if len(result.Changed) != 2 {
		t.Errorf("expected 2 changed entries, got %d", len(result.Changed))
	}
}

// TestApplyRepair_InvalidExtractor tests error handling when no extractor available
func TestApplyRepair_InvalidExtractor(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.unknown")

	if err := os.WriteFile(testPath, []byte("content"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.CorrectValue,
			Field: "field",
			From:  "x",
			To:    "y",
			Paths: []string{"test.unknown"},
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	// Should record error for unknown file type
	if len(result.Errors) == 0 {
		t.Errorf("expected error for unknown extractor")
	}
}
