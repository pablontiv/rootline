package fix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

func TestApplyRepair_CorrectValueRepair(t *testing.T) {
	// Setup: Create a temporary directory with a test file.
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	content := `---
estado: Inprogres
tipo: Epic
---
# Test Document
`

	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create a correct_value proposal.
	proposals := []proposal.Proposal{
		{
			Type:        proposal.CorrectValue,
			Field:       "estado",
			Description: "correct typo",
			Paths:       []string{"test.md"},
			From:        "Inprogres",
			To:          "In Progress",
		},
	}

	// Apply repair (dry-run false).
	result, err := ApplyRepair(proposals, false, tmpDir)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	// Verify result.
	if result.Version != 1 {
		t.Errorf("expected version 1, got %d", result.Version)
	}
	if result.Kind != "rootline/repair" {
		t.Errorf("expected kind rootline/repair, got %s", result.Kind)
	}
	if result.DryRun {
		t.Errorf("expected DryRun=false, got true")
	}

	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}

	// Verify file was updated.
	updatedContent, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	if !contains(string(updatedContent), "estado: In Progress") {
		t.Errorf("file not updated correctly; content: %s", string(updatedContent))
	}
}

func TestApplyRepair_SchemaProposalRejected(t *testing.T) {
	// Setup: Create a temporary directory with a test file.
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, ".stem")

	stemContent := `schema:
  estado:
    type: enum
    values:
      - Pending
      - In Progress
`

	if err := os.WriteFile(testPath, []byte(stemContent), 0644); err != nil {
		t.Fatalf("failed to write .stem file: %v", err)
	}

	// Create an extend_enum proposal (which is SurfaceSchema).
	proposals := []proposal.Proposal{
		{
			Type:        proposal.ExtendEnum,
			Field:       "estado",
			Description: "extend enum",
			Paths:       []string{".stem"},
			Value:       "Blocked",
		},
	}

	// Apply repair.
	result, err := ApplyRepair(proposals, false, tmpDir)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	// Verify the proposal was rejected (not applied).
	if len(result.Rejected) != 1 {
		t.Errorf("expected 1 rejected proposal, got %d", len(result.Rejected))
	}

	// Verify .stem was not modified.
	stemAfter, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("failed to read .stem after: %v", err)
	}

	if string(stemAfter) != stemContent {
		t.Errorf(".stem was modified; expected unchanged")
	}
}

func TestApplyRepair_DryRun(t *testing.T) {
	// Setup: Create a temporary directory with a test file.
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	originalContent := `---
estado: Inprogres
---
# Test
`

	if err := os.WriteFile(testPath, []byte(originalContent), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create a correct_value proposal.
	proposals := []proposal.Proposal{
		{
			Type:        proposal.CorrectValue,
			Field:       "estado",
			Description: "correct typo",
			Paths:       []string{"test.md"},
			From:        "Inprogres",
			To:          "In Progress",
		},
	}

	// Apply repair with dry-run = true.
	result, err := ApplyRepair(proposals, true, tmpDir)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	// Verify DryRun flag.
	if !result.DryRun {
		t.Errorf("expected DryRun=true, got false")
	}

	// Verify file was NOT modified.
	fileContent, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("failed to read file: %v", err)
	}

	if string(fileContent) != originalContent {
		t.Errorf("file was modified in dry-run mode; expected unchanged")
	}

	// Verify changed list was recorded.
	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry in dry-run, got %d", len(result.Changed))
	}
}

func TestApplyRepair_AddField(t *testing.T) {
	// Setup: Create a temporary directory with a test file.
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "test.md")

	content := `---
estado: Pending
---
# Test Document
`

	if err := os.WriteFile(testPath, []byte(content), 0644); err != nil {
		t.Fatalf("failed to write test file: %v", err)
	}

	// Create an add_field proposal.
	proposals := []proposal.Proposal{
		{
			Type:        proposal.AddField,
			Field:       "tipo",
			Description: "add missing required field",
			Paths:       []string{"test.md"},
			Value:       "Epic",
		},
	}

	// Apply repair.
	result, err := ApplyRepair(proposals, false, tmpDir)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	// Verify file was updated.
	updatedContent, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatalf("failed to read updated file: %v", err)
	}

	if !contains(string(updatedContent), "tipo: Epic") {
		t.Errorf("field not added correctly; content: %s", string(updatedContent))
	}

	// Verify changed list.
	if len(result.Changed) != 1 {
		t.Errorf("expected 1 changed entry, got %d", len(result.Changed))
	}
}

func TestApplyRepair_MultipleProposals(t *testing.T) {
	// Setup: Create a temporary directory with test files.
	tmpDir := t.TempDir()
	testPath1 := filepath.Join(tmpDir, "test1.md")
	testPath2 := filepath.Join(tmpDir, "test2.md")

	content1 := `---
estado: Inprogres
---
# Test 1
`

	content2 := `---
estado: Pendng
---
# Test 2
`

	if err := os.WriteFile(testPath1, []byte(content1), 0644); err != nil {
		t.Fatalf("failed to write test1.md: %v", err)
	}
	if err := os.WriteFile(testPath2, []byte(content2), 0644); err != nil {
		t.Fatalf("failed to write test2.md: %v", err)
	}

	// Create multiple repair proposals.
	proposals := []proposal.Proposal{
		{
			Type:        proposal.CorrectValue,
			Field:       "estado",
			Description: "correct typo",
			Paths:       []string{"test1.md"},
			From:        "Inprogres",
			To:          "In Progress",
		},
		{
			Type:        proposal.CorrectValue,
			Field:       "estado",
			Description: "correct typo",
			Paths:       []string{"test2.md"},
			From:        "Pendng",
			To:          "Pending",
		},
	}

	// Apply repair.
	result, err := ApplyRepair(proposals, false, tmpDir)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	// Verify both files were updated.
	if len(result.Changed) != 2 {
		t.Errorf("expected 2 changed entries, got %d", len(result.Changed))
	}

	content1After, _ := os.ReadFile(testPath1)
	if !contains(string(content1After), "estado: In Progress") {
		t.Errorf("test1.md not updated correctly")
	}

	content2After, _ := os.ReadFile(testPath2)
	if !contains(string(content2After), "estado: Pending") {
		t.Errorf("test2.md not updated correctly")
	}
}

// Helper function.
func contains(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// --- path containment (issue #69) ---

// escapeFixture returns a scan root and a sibling file one level above it, so a
// "../outside.md" proposal path names a real file that must stay untouched.
func escapeFixture(t *testing.T) (root, outside string) {
	t.Helper()

	base := t.TempDir()
	root = filepath.Join(base, "scan")
	if err := os.MkdirAll(root, 0o755); err != nil {
		t.Fatal(err)
	}

	outside = filepath.Join(base, "outside.md")
	body := "---\nestado: Pending\n---\n# Outside\n\n## Status\n\nuntouched\n"
	if err := os.WriteFile(outside, []byte(body), 0644); err != nil {
		t.Fatal(err)
	}

	return root, outside
}

func TestApplyRepair_RejectsEscapingPath(t *testing.T) {
	root, outside := escapeFixture(t)
	before, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}

	result, err := ApplyRepair([]proposal.Proposal{{
		Type:        proposal.CorrectValue,
		Field:       "estado",
		Description: "correct estado",
		Paths:       []string{"../outside.md"},
		From:        "Pending",
		To:          "Completed",
	}}, false, root)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	if len(result.Rejected) != 1 {
		t.Fatalf("Rejected = %v, want exactly one containment violation", result.Rejected)
	}
	if len(result.Changed) != 0 {
		t.Errorf("Changed = %v, want empty", result.Changed)
	}
	// The escaping path must never be opened, so it cannot produce a read error.
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want empty", result.Errors)
	}

	after, err := os.ReadFile(outside)
	if err != nil {
		t.Fatal(err)
	}
	if string(before) != string(after) {
		t.Error("file outside the root was modified")
	}
}

func TestApplyRepair_RejectsAbsolutePath(t *testing.T) {
	root, outside := escapeFixture(t)

	result, err := ApplyRepair([]proposal.Proposal{{
		Type:        proposal.AddField,
		Field:       "tampered",
		Description: "add tampered field",
		Paths:       []string{outside},
		Value:       "true",
	}}, false, root)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	if len(result.Rejected) != 1 {
		t.Fatalf("Rejected = %v, want exactly one containment violation", result.Rejected)
	}
	if len(result.Errors) != 0 {
		t.Errorf("Errors = %v, want empty (a policy refusal is not an I/O error)", result.Errors)
	}
}

func TestApplySetSection_ContainmentPolicy(t *testing.T) {
	// applySetSection is reached from both repair apply and fix --all, so it
	// carries its own guard rather than trusting the caller.
	t.Run("rejects escaping path", func(t *testing.T) {
		root, outside := escapeFixture(t)
		before, err := os.ReadFile(outside)
		if err != nil {
			t.Fatal(err)
		}

		p := proposal.Proposal{
			Type:    proposal.SetSection,
			Heading: "## Status",
			Value:   "injected",
			Mode:    "replace",
			Paths:   []string{"../outside.md"},
		}
		if err := applySetSection(p, root, nil, PolicyRejectAbsolute); err == nil {
			t.Fatal("expected a containment error")
		}

		after, err := os.ReadFile(outside)
		if err != nil {
			t.Fatal(err)
		}
		if string(before) != string(after) {
			t.Error("file outside the root was modified")
		}
	})

	// The fix --all path supplies scan-derived relative paths; the check must
	// pass them through so that pipeline keeps behaving exactly as before.
	t.Run("passes through a contained relative path", func(t *testing.T) {
		root, _ := escapeFixture(t)
		target := filepath.Join(root, "task.md")
		if err := os.WriteFile(target, []byte("# Task\n\n## Status\n\noriginal\n"), 0644); err != nil {
			t.Fatal(err)
		}

		p := proposal.Proposal{
			Type:    proposal.SetSection,
			Heading: "## Status",
			Value:   "rewritten",
			Mode:    "replace",
			Paths:   []string{"task.md"},
		}
		if err := applySetSection(p, root, nil, PolicyAcceptAbsolute); err != nil {
			t.Fatalf("unexpected error: %v", err)
		}

		content, err := os.ReadFile(target)
		if err != nil {
			t.Fatal(err)
		}
		if want := "rewritten"; !strings.Contains(string(content), want) {
			t.Errorf("task.md does not contain %q:\n%s", want, content)
		}
	})
}
