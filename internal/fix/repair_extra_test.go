package fix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/proposal"
)

func TestApplyRepair_SetField(t *testing.T) {
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
			Type:        proposal.SetField,
			Field:       "tipo",
			Description: "set tipo",
			Paths:       []string{"test.md"},
			Value:       "Epic",
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed, got %d", len(result.Changed))
	}

	updated, _ := os.ReadFile(testPath)
	if !contains(string(updated), "tipo: Epic") {
		t.Errorf("file not updated: %s", string(updated))
	}
}

func TestApplyRepair_SetField_DryRun(t *testing.T) {
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
			Field: "tipo",
			Paths: []string{"test.md"},
			Value: "Epic",
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
		t.Errorf("file modified in dry-run")
	}
}

func TestApplyRepair_SetSection_Create(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")
	content := `---
estado: Pending
---
# Test

Some content.
`
	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:    proposal.SetSection,
			Field:   "notes",
			Paths:   []string{"test.md"},
			Heading: "## Notes",
			Value:   "Important note.",
			Mode:    "create",
		},
	}

	if _, err := ApplyRepair(proposals, false, tmpDir, false); err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	updated, _ := os.ReadFile(testPath)
	if !contains(string(updated), "## Notes") || !contains(string(updated), "Important note.") {
		t.Errorf("section not created: %s", string(updated))
	}
}

func TestApplyRepair_SetSection_DryRun(t *testing.T) {
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
			Type:    proposal.SetSection,
			Paths:   []string{"test.md"},
			Heading: "## Notes",
			Value:   "Note body",
			Mode:    "create",
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
		t.Errorf("file modified in dry-run mode")
	}
}

func TestApplyRepair_CorrectLink(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")
	content := `---
estado: Pending
---
# Test

See [[oldname]] for details.
`
	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.CorrectLink,
			Paths: []string{"test.md"},
			From:  "[[oldname]]",
			To:    "[[newname]]",
		},
	}

	result, err := ApplyRepair(proposals, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed, got %d", len(result.Changed))
	}

	updated, _ := os.ReadFile(testPath)
	if !contains(string(updated), "[[newname]]") || contains(string(updated), "[[oldname]]") {
		t.Errorf("link not corrected: %s", string(updated))
	}
}

func TestApplyRepair_CorrectLink_DryRun(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")
	original := "See [[oldname]].\n"
	if err := os.WriteFile(testPath, []byte(original), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	proposals := []proposal.Proposal{
		{
			Type:  proposal.CorrectLink,
			Paths: []string{"test.md"},
			From:  "[[oldname]]",
			To:    "[[newname]]",
		},
	}

	result, err := ApplyRepair(proposals, true, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if !result.DryRun {
		t.Errorf("expected DryRun=true")
	}
	got, _ := os.ReadFile(testPath)
	if string(got) != original {
		t.Errorf("file modified in dry-run")
	}
}

func TestApplyRepair_UnknownTypeRejected(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")
	if err := os.WriteFile(testPath, []byte("---\nestado: x\n---\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	// Unknown repair type falls to default branch in applyRepair switch.
	proposals := []proposal.Proposal{
		{
			Type:  proposal.Type("unknown_repair_type"),
			Field: "estado",
			Paths: []string{"test.md"},
		},
	}

	result, err := ApplyRepair(proposals, true, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if len(result.Rejected) == 0 {
		t.Errorf("expected rejected entry for unknown repair type")
	}
}

func TestReplaceOnce(t *testing.T) {
	cases := []struct {
		s, old, new, want string
	}{
		{"foo bar foo", "foo", "baz", "baz bar foo"},
		{"abc", "x", "y", "abc"},
		{"abcabc", "abc", "X", "Xabc"},
		{"", "x", "y", ""},
	}
	for _, c := range cases {
		if got := replaceOnce(c.s, c.old, c.new); got != c.want {
			t.Errorf("replaceOnce(%q,%q,%q) = %q, want %q", c.s, c.old, c.new, got, c.want)
		}
	}
}

func TestPostValidateWrittenTargetRollsBackWhenSchemaResolutionFails(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(`version: 2
root: true
schema:
  id:
    type: sequence
    match:
      "BAD*": {prefix: BAD, digits: 2.0}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	path := filepath.Join(root, "BAD001.md")
	original := []byte("---\nestado: Pending\n---\n# Original\n")
	written := []byte("---\nestado: Done\n---\n# Written\n")
	if err := os.WriteFile(path, written, 0o644); err != nil {
		t.Fatal(err)
	}
	result := &RepairResult{}
	result.recordChange("BAD001.md", "changed BAD001.md")
	postValidateWrittenTargets(map[string]*repairTarget{
		"BAD001.md": {abs: path, record: &extract.Record{Path: "BAD001.md"}, original: original, written: true},
	}, result)
	if len(result.RolledBack) != 1 || !strings.Contains(strings.Join(result.RolledBack[0].Errors, " "), "resolving schema after write") || !strings.Contains(strings.Join(result.RolledBack[0].Errors, " "), "digits") {
		t.Fatalf("RolledBack = %#v, want schema-resolution rollback message", result.RolledBack)
	}
	if len(result.Changed) != 0 {
		t.Fatalf("Changed = %#v, want rollback to retract change", result.Changed)
	}
	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	if string(got) != string(original) {
		t.Fatalf("bytes after rollback = %q, want %q", got, original)
	}
}

func TestApplyRepairMixedInvalidGovernanceClassifiesTargetAndContinues(t *testing.T) {
	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(`version: 2
root: true
schema:
  id:
    type: sequence
    match:
      "BAD*": {prefix: BAD, digits: 2.0}
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "A.md"), []byte("---\nestado: Pendng\n---\n# A\n"), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "BAD001.md"), []byte("---\n---\n# Bad\n"), 0o644); err != nil {
		t.Fatal(err)
	}

	result, err := ApplyRepair([]proposal.Proposal{{
		Type:        proposal.CorrectValue,
		Field:       "estado",
		Description: "correct status",
		Paths:       []string{"A.md", "BAD001.md"},
		From:        "Pendng",
		To:          "Pending",
	}}, false, root, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if result.Complete {
		t.Fatalf("Complete = true, want false for classified invalid target: %#v", result)
	}
	if len(result.Errors) != 1 || !strings.Contains(result.Errors[0], "BAD001.md") || !strings.Contains(result.Errors[0], "digits") {
		t.Fatalf("Errors = %#v, want BAD001 digits classification", result.Errors)
	}
	if len(result.Changed) != 1 || !strings.Contains(result.Changed[0], "A.md") {
		t.Fatalf("Changed = %#v, want only valid target", result.Changed)
	}
	validBytes, err := os.ReadFile(filepath.Join(root, "A.md"))
	if err != nil {
		t.Fatal(err)
	}
	valid := string(validBytes)
	if !strings.Contains(valid, "estado: Pending") {
		t.Fatalf("valid target was not repaired:\n%s", valid)
	}
	invalidBytes, err := os.ReadFile(filepath.Join(root, "BAD001.md"))
	if err != nil {
		t.Fatal(err)
	}
	invalid := string(invalidBytes)
	if strings.Contains(invalid, "estado:") {
		t.Fatalf("invalid target was mutated:\n%s", invalid)
	}
}
