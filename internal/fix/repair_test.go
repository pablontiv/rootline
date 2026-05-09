package fix

import (
	"os"
	"path/filepath"
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
