package fix

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

// TestApplyRepair_Issue178_RerunClobbersHumanEdit tests that a re-run of the same
// repair report does not clobber a human's manual edit between runs.
//
// This is the key test for issue #178: declarativity.
// 1. First run: apply a proposal that sets `owner: ""`
// 2. Human edits: manually set `owner: "Alice"`
// 3. Second run: apply the SAME proposal
// 4. Expected: Human's value "Alice" survives, proposal moves to skipped
// 5. Actual (before fix): Human's value is clobbered to ""; proposal moves to changed
func TestApplyRepair_Issue178_RerunClobbersHumanEdit(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "a.md")

	// Initial state: missing required field
	initialContent := `---
titulo: Alpha
---

## Notes

Body.
`

	if err := os.WriteFile(testPath, []byte(initialContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// First run: apply repair to add missing `owner` field with empty value
	proposal1 := proposal.Proposal{
		Type:        proposal.AddField,
		Field:       "owner",
		Description: "required field missing",
		Paths:       []string{"a.md"},
		Value:       "",
		ValueSource: "empty",
	}

	result1, err := ApplyRepair([]proposal.Proposal{proposal1}, false, tmpDir, true)
	if err != nil {
		t.Fatalf("First ApplyRepair failed: %v", err)
	}
	if !result1.Complete {
		t.Fatalf("First run should complete: errors=%v", result1.Errors)
	}
	if len(result1.Changed) != 1 {
		t.Fatalf("First run should change one thing: changed=%v", result1.Changed)
	}

	// Verify first run wrote owner: ""
	after1, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after1), "owner: \"\"") {
		t.Fatalf("First run should have written owner:\n%s", after1)
	}

	// Human edits the file to set owner: Alice
	humanEditedContent := `---
titulo: Alpha
owner: Alice
---

## Notes

Body.
`
	if err := os.WriteFile(testPath, []byte(humanEditedContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Second run: apply the SAME proposal again
	// The field exists with a different value (Alice vs ""), so it's rejected (not applied).
	// The key is that the rejection prevents a write, so the human's value survives.
	result2, err := ApplyRepair([]proposal.Proposal{proposal1}, false, tmpDir, true)
	if err != nil {
		t.Fatalf("Second ApplyRepair failed: %v", err)
	}

	// Second run should complete (rejection is not an error, it's a deliberate refusal)
	if !result2.Complete {
		t.Fatalf("Second run should complete: errors=%v", result2.Errors)
	}

	// Changed should be empty because the field already exists with a conflicting value,
	// so the proposal is rejected (not applied, no write)
	if len(result2.Changed) != 0 {
		t.Fatalf("Second run should not change anything: changed=%v", result2.Changed)
	}

	// The proposal should be rejected (field exists with different value)
	if len(result2.Rejected) != 1 {
		t.Fatalf("Second run should reject because field exists with conflicting value: rejected=%v", result2.Rejected)
	}

	// Verify human's value survived (because the rejected proposal was not written)
	after2, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after2), "owner: Alice") {
		t.Fatalf("Human's edit should survive second run (via rejection): %s", after2)
	}
	if strings.Contains(string(after2), "owner: \"\"") {
		t.Fatalf("Empty owner should NOT be written on second run: %s", after2)
	}
}

// TestApplyRepair_Issue178_MigrateValueWikiLinksIdempotent tests that
// a migrate_value proposal with wiki_links does not duplicate them on re-run.
//
// This is the third issue described in #178: wiki-link duplication.
func TestApplyRepair_Issue178_MigrateValueWikiLinksIdempotent(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "b.md")

	initialContent := `---
titulo: Beta
estado: "Pending (blocked by E04/F01)"
owner: x
---

## Notes

Body.
`

	if err := os.WriteFile(testPath, []byte(initialContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Proposal to migrate estado and add wiki-links
	prop := proposal.Proposal{
		Type:        proposal.MigrateValue,
		Field:       "estado",
		Description: "migrate",
		Paths:       []string{"b.md"},
		From:        "Pending (blocked by E04/F01)",
		To:          "Pending",
		WikiLinks:   []string{"[[blocks:E04/F01]]"},
	}

	// First run
	result1, err := ApplyRepair([]proposal.Proposal{prop}, false, tmpDir, false)
	if err != nil {
		t.Fatalf("First ApplyRepair failed: %v", err)
	}
	if !result1.Complete {
		t.Fatalf("First run should complete: errors=%v", result1.Errors)
	}

	after1, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatal(err)
	}

	// Count occurrences of wiki-link in first run result
	count1 := strings.Count(string(after1), "[[blocks:E04/F01]]")
	if count1 != 1 {
		t.Fatalf("First run should insert wiki-link once, got %d times:\n%s", count1, after1)
	}

	// Second run: apply the same proposal again (should be idempotent)
	result2, err := ApplyRepair([]proposal.Proposal{prop}, false, tmpDir, false)
	if err != nil {
		t.Fatalf("Second ApplyRepair failed: %v", err)
	}

	after2, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatal(err)
	}

	// Count occurrences of wiki-link in second run result
	count2 := strings.Count(string(after2), "[[blocks:E04/F01]]")
	if count2 != 1 {
		t.Fatalf("Second run should NOT duplicate wiki-link, got %d times:\n%s", count2, after2)
	}

	// The file content should not change on second run
	if string(after1) != string(after2) {
		t.Fatalf("Second run should not modify file (idempotent):\nFirst:\n%s\nSecond:\n%s", after1, after2)
	}

	// Second run should skip (proposal already applied)
	if len(result2.Changed) != 0 {
		t.Fatalf("Second run should not have changed: changed=%v", result2.Changed)
	}
	if len(result2.Skipped) == 0 {
		t.Fatalf("Second run should have skipped: skipped=%v", result2.Skipped)
	}
}

// TestApplyRepair_Issue178_CorrectValueFromMismatchHandling tests that
// when a stale report's `from` value doesn't match the current value on disk,
// the proposal is properly rejected (not silently clobbered).
//
// This is the second issue described in #178.
func TestApplyRepair_Issue178_CorrectValueFromMismatchHandling(t *testing.T) {
	tmpDir := t.TempDir()
	testPath := filepath.Join(tmpDir, "one.md")

	// Document has a different value than the proposal's `from` field
	initialContent := `---
titulo: One
owner: zed
---
Body one.
`

	if err := os.WriteFile(testPath, []byte(initialContent), 0o644); err != nil {
		t.Fatal(err)
	}

	// Proposal claims to correct owner from "alice" to "bob",
	// but the document has "zed" (doesn't match the `from` value)
	prop := proposal.Proposal{
		Type:        proposal.CorrectValue,
		Field:       "owner",
		Description: "correct owner",
		Paths:       []string{"one.md"},
		From:        "alice",
		To:          "bob",
	}

	result, err := ApplyRepair([]proposal.Proposal{prop}, false, tmpDir, false)
	if err != nil {
		t.Fatalf("ApplyRepair failed: %v", err)
	}

	// Should be rejected because current value "zed" doesn't match `from` "alice"
	if len(result.Rejected) != 1 {
		t.Fatalf("Should have 1 rejection (from mismatch): rejected=%v", result.Rejected)
	}

	// Should not be changed
	if len(result.Changed) != 0 {
		t.Fatalf("Should not change (from mismatch): changed=%v", result.Changed)
	}

	// Verify value was not clobbered
	after, err := os.ReadFile(testPath)
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(after), "owner: zed") {
		t.Fatalf("Current value should not be clobbered:\n%s", after)
	}
}

// TestApplyRepair_Issue163_SetSectionDryRunValidation tests that
// set_section proposals validate modes and headings in dry-run mode,
// matching the validation that happens in real-run mode.
//
// This is the key test for issue #163: dry-run/real-run parity.
func TestApplyRepair_Issue163_SetSectionDryRunValidation(t *testing.T) {
	tests := []struct {
		name      string
		mode      string
		heading   string
		wantError bool
	}{
		{"valid mode replace, valid heading", "replace", "## Notes", false},
		{"valid mode upsert is invalid", "upsert", "## Notes", true},
		{"valid mode set is invalid", "set", "## Notes", true},
		{"invalid mode bogusmode", "bogusmode", "## Notes", true},
		{"valid mode but missing heading", "replace", "## Missing", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Test dry-run
			tmpDirDry := t.TempDir()
			testPathDry := filepath.Join(tmpDirDry, "a.md")
			content := `---
estado: Pending
---
# Alpha

## Notes

original body
`
			if err := os.WriteFile(testPathDry, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}

			propDry := proposal.Proposal{
				Type:        proposal.SetSection,
				Heading:     tt.heading,
				Mode:        tt.mode,
				Value:       "NEWBODY",
				Description: "test",
				Paths:       []string{"a.md"},
			}

			dryResult, err := ApplyRepair([]proposal.Proposal{propDry}, true, tmpDirDry, false)
			if err != nil {
				t.Fatalf("Dry-run ApplyRepair failed: %v", err)
			}

			// Test real run
			tmpDirReal := t.TempDir()
			testPathReal := filepath.Join(tmpDirReal, "a.md")
			if err := os.WriteFile(testPathReal, []byte(content), 0o644); err != nil {
				t.Fatal(err)
			}

			propReal := proposal.Proposal{
				Type:        proposal.SetSection,
				Heading:     tt.heading,
				Mode:        tt.mode,
				Value:       "NEWBODY",
				Description: "test",
				Paths:       []string{"a.md"},
			}

			realResult, err := ApplyRepair([]proposal.Proposal{propReal}, false, tmpDirReal, false)
			if err != nil {
				t.Fatalf("Real-run ApplyRepair failed: %v", err)
			}

			// Both should agree on whether there are errors
			dryHasErrors := len(dryResult.Errors) > 0
			realHasErrors := len(realResult.Errors) > 0

			if dryHasErrors != realHasErrors {
				t.Fatalf("Dry-run and real-run disagree on errors:\ndry errors=%v\nreal errors=%v",
					dryResult.Errors, realResult.Errors)
			}

			// Both should agree on whether it succeeded
			if dryResult.Complete != realResult.Complete {
				t.Fatalf("Dry-run complete=%v, real-run complete=%v (should match)",
					dryResult.Complete, realResult.Complete)
			}

			// Verify expectation about errors
			if tt.wantError && !realHasErrors {
				t.Fatalf("Expected error but got none for mode=%s heading=%s", tt.mode, tt.heading)
			}
			if !tt.wantError && realHasErrors {
				t.Fatalf("Expected no error but got one for mode=%s heading=%s: %v",
					tt.mode, tt.heading, realResult.Errors)
			}

			// Dry-run file should never be modified
			dryFileContent, err := os.ReadFile(testPathDry)
			if err != nil {
				t.Fatal(err)
			}
			if string(dryFileContent) != content {
				t.Fatalf("Dry-run file should not be modified:\n%s", dryFileContent)
			}

			// Real-run file should only be modified if no errors
			realFileContent, err := os.ReadFile(testPathReal)
			if err != nil {
				t.Fatal(err)
			}
			if tt.wantError && string(realFileContent) != content {
				t.Fatalf("Real-run should not modify on error: mode=%s heading=%s", tt.mode, tt.heading)
			}
			// If no error, file may or may not be modified (we're testing dry-run/real-run parity, not the specific content)
		})
	}
}
