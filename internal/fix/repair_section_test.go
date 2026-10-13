package fix

import (
	"bytes"
	"os"
	"path/filepath"
	"reflect"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/proposal"
)

func repairSectionFixture(t *testing.T, names ...string) (string, []byte) {
	t.Helper()
	root := t.TempDir()
	stem := []byte(`version: 2
root: true
schema:
  estado:
    type: enum
    values: [Pending, Done]
  notes:
    type: enum
    source: 'body.section["## Notes"]'
    values: [Allowed]
    required: true
`)
	if err := os.WriteFile(filepath.Join(root, ".stem"), stem, 0o644); err != nil {
		t.Fatal(err)
	}
	original := []byte("---\n# preserve this comment and spacing\nestado:   Pending\n---\n# Test\n\n## Notes\n\nAllowed\n")
	for _, name := range names {
		if err := os.WriteFile(filepath.Join(root, name), original, 0o600); err != nil {
			t.Fatal(err)
		}
	}
	return root, original
}

func assertRepairSectionBytes(t *testing.T, root, path string, want []byte) {
	t.Helper()
	got, err := os.ReadFile(filepath.Join(root, path))
	if err != nil {
		t.Fatal(err)
	}
	if !bytes.Equal(got, want) {
		t.Fatalf("%s bytes = %q, want %q", path, got, want)
	}
}

func TestApplyRepair_SetSectionInvalidRollback(t *testing.T) {
	root, original := repairSectionFixture(t, "invalid.md")
	stemBefore, err := os.ReadFile(filepath.Join(root, ".stem"))
	if err != nil {
		t.Fatal(err)
	}
	proposals := []proposal.Proposal{
		{Type: proposal.SetSection, Paths: []string{"invalid.md"}, Heading: "## Notes", Value: "Invalid"},
		{Type: proposal.SetSection, Paths: []string{"invalid.md"}, Heading: "## Notes", Value: "Still invalid"},
	}
	result, err := ApplyRepair(proposals, false, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || len(result.Errors) != 0 || len(result.RolledBack) != 1 {
		t.Fatalf("result = %+v, want incomplete run with exactly one successful rollback", result)
	}
	if result.RolledBack[0].Path != "invalid.md" || !strings.Contains(strings.Join(result.RolledBack[0].Errors, " "), "notes") {
		t.Fatalf("RolledBack = %+v, want notes validation failure for invalid.md", result.RolledBack)
	}
	if len(result.Changed) != 0 || len(result.changedPaths) != 0 {
		t.Fatalf("rollback left ledger entries: %v / %v", result.Changed, result.changedPaths)
	}
	assertRepairSectionBytes(t, root, "invalid.md", original)
	assertRepairSectionBytes(t, root, ".stem", stemBefore)
	info, err := os.Stat(filepath.Join(root, "invalid.md"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("rollback mode = %o, want 600", info.Mode().Perm())
	}
}

func TestApplyRepair_SetSectionMixedLedgerRetraction(t *testing.T) {
	root, original := repairSectionFixture(t, "section.md", "invalid.md", "field.md")
	proposals := []proposal.Proposal{
		{Type: proposal.SetSection, Paths: []string{"section.md"}, Heading: "## Extra", Mode: "create", Value: "Kept section"},
		{Type: proposal.SetField, Paths: []string{"invalid.md"}, Field: "estado", Value: "Done"},
		{Type: proposal.SetSection, Paths: []string{"invalid.md"}, Heading: "## Notes", Value: "Invalid"},
		{Type: proposal.CorrectValue, Paths: []string{"field.md"}, Field: "estado", From: "Pending", To: "Done"},
	}
	result, err := ApplyRepair(proposals, false, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || len(result.Errors) != 0 || len(result.RolledBack) != 1 || result.RolledBack[0].Path != "invalid.md" {
		t.Fatalf("result = %+v, want only invalid.md rolled back", result)
	}
	wantPaths := []string{"section.md", "field.md"}
	wantChanges := []string{"create section ## Extra in section.md", "correct estado: \"Pending\"->\"Done\" in field.md"}
	if !reflect.DeepEqual(result.changedPaths, wantPaths) || !reflect.DeepEqual(result.Changed, wantChanges) {
		t.Fatalf("remaining ledger = %v / %v, want %v / %v", result.Changed, result.changedPaths, wantChanges, wantPaths)
	}
	assertRepairSectionBytes(t, root, "invalid.md", original)
	assertRepairSectionBytes(t, root, "section.md", append(append([]byte(nil), original...), []byte("\n## Extra\n\nKept section\n")...))
	fieldContent, err := os.ReadFile(filepath.Join(root, "field.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(fieldContent), "estado: Done") || !strings.Contains(string(fieldContent), "Allowed") {
		t.Fatalf("unrelated frontmatter repair did not survive: %s", fieldContent)
	}
}

func TestApplyRepair_SetSectionDryRunLedgerAlignment(t *testing.T) {
	root, original := repairSectionFixture(t, "test.md")
	proposals := []proposal.Proposal{
		{Type: proposal.CorrectValue, Paths: []string{"test.md"}, Field: "estado", From: "Pending", To: "Done"},
		{Type: proposal.AddField, Paths: []string{"test.md"}, Field: "owner", Value: "Ada"},
		{Type: proposal.SetField, Paths: []string{"test.md"}, Field: "label", Value: "Kept"},
		{Type: proposal.SetSection, Paths: []string{"test.md"}, Heading: "## Extra", Mode: "create", Value: "First [[old]]"},
		{Type: proposal.CorrectLink, Paths: []string{"test.md"}, From: "[[old]]", To: "[[new]]"},
		{Type: proposal.SetSection, Paths: []string{"test.md"}, Heading: "## Extra", Mode: "append", Value: "Second"},
		{Type: proposal.SetField, Paths: []string{"test.md"}, Field: "last", Value: "Also kept"},
	}
	dry, err := ApplyRepair(proposals, true, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !dry.DryRun || !dry.Complete || len(dry.Errors) != 0 || len(dry.RolledBack) != 0 || len(dry.Changed) != len(proposals) {
		t.Fatalf("dry-run result = %+v, want all proposals previewed without rollback", dry)
	}
	wantPaths := make([]string, len(proposals))
	for i := range wantPaths {
		wantPaths[i] = "test.md"
	}
	if !reflect.DeepEqual(dry.changedPaths, wantPaths) {
		t.Fatalf("dry-run ledger = %v / %v, want all entries paired with test.md", dry.Changed, dry.changedPaths)
	}
	assertRepairSectionBytes(t, root, "test.md", original)
	real, err := ApplyRepair(proposals, false, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if !real.Complete || len(real.Errors) != 0 || len(real.RolledBack) != 0 || !reflect.DeepEqual(real.Changed, dry.Changed) || !reflect.DeepEqual(real.changedPaths, dry.changedPaths) {
		t.Fatalf("real result = %+v, want same ledger as dry-run %+v", real, dry)
	}
	content, err := os.ReadFile(filepath.Join(root, "test.md"))
	if err != nil {
		t.Fatal(err)
	}
	for _, want := range []string{"estado: Done", "owner: Ada", "label: Kept", "last: Also kept", "Allowed", "First [[new]]\n\nSecond"} {
		if !strings.Contains(string(content), want) {
			t.Errorf("same-file repairs clobbered %q: %s", want, content)
		}
	}
	if strings.Contains(string(content), "[[old]]") {
		t.Errorf("section append lost preceding link repair: %s", content)
	}
}

func TestApplyRepair_SetSectionPartialFailureKeepsWrittenLedger(t *testing.T) {
	root, original := repairSectionFixture(t, "first.md", "second.md")
	first := append(append([]byte(nil), original...), []byte("\n## Extra\n\nFirst\n")...)
	if err := os.WriteFile(filepath.Join(root, "first.md"), first, 0o600); err != nil {
		t.Fatal(err)
	}
	p := proposal.Proposal{Type: proposal.SetSection, Paths: []string{"first.md", "second.md"}, Heading: "## Extra", Value: "Updated"}
	result, err := ApplyRepair([]proposal.Proposal{p}, false, root, false)
	if err != nil {
		t.Fatal(err)
	}
	if result.Complete || len(result.Errors) != 1 || len(result.RolledBack) != 0 || !reflect.DeepEqual(result.changedPaths, []string{"first.md"}) || len(result.Changed) != 1 {
		t.Fatalf("result = %+v, want earlier successful target retained despite missing later heading", result)
	}
	assertRepairSectionBytes(t, root, "first.md", append(append([]byte(nil), original...), []byte("\n## Extra\n\nUpdated\n")...))
	assertRepairSectionBytes(t, root, "second.md", original)
}

func TestRepairTargetCurrentContentLazySnapshot(t *testing.T) {
	target := &repairTarget{original: []byte("original")}
	if target.contentSet || target.content != "" {
		t.Fatal("candidate content must stay unloaded before a transform")
	}
	if got := target.currentContent(); got != "original" || !target.contentSet {
		t.Fatalf("currentContent() = %q, loaded = %v, want original snapshot loaded", got, target.contentSet)
	}
	target.setContent("")
	if got := target.currentContent(); got != "" {
		t.Fatalf("currentContent() = %q, want explicitly empty candidate, not original", got)
	}
	if string(target.original) != "original" {
		t.Fatalf("candidate mutation changed rollback snapshot: %q", target.original)
	}
}
