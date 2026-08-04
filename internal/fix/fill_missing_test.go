package fix

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/proposal"
	"github.com/pablontiv/rootline/internal/rules"
)

// Issue #60 defect 1: a required field with no declared default: must be
// REPORTED, not invented. Engine-chosen values stay out of documents unless the
// caller opts in.

func requiredErr(field string) []rules.ValidationError {
	return []rules.ValidationError{{
		Rule:    "required",
		Field:   field,
		Message: `required field "` + field + `" is missing`,
	}}
}

// --- ApplyFixes: the single-file fix path, which consumes no proposals ---

func TestApplyFixes_DoesNotInventValueWithoutOptIn(t *testing.T) {
	record := &extract.Record{Path: "a.md", Frontmatter: map[string]any{}}
	effective := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"tipo": {Type: "enum", Values: []string{"report", "note"}},
	}}

	added, _, skipped := ApplyFixes(context.Background(), record, effective, requiredErr("tipo"), false)

	if len(added) != 0 {
		t.Errorf("expected no field added without opt-in, got %v", added)
	}
	if _, present := record.Frontmatter["tipo"]; present {
		t.Errorf("expected frontmatter untouched, got %v", record.Frontmatter)
	}
	if len(skipped) != 1 || !strings.Contains(skipped[0], "tipo") {
		t.Errorf("expected the skip to be reported, got %v", skipped)
	}
}

func TestApplyFixes_InventsValueWithOptIn(t *testing.T) {
	record := &extract.Record{Path: "a.md", Frontmatter: map[string]any{}}
	effective := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"tipo": {Type: "enum", Values: []string{"report", "note"}},
	}}

	added, _, _ := ApplyFixes(context.Background(), record, effective, requiredErr("tipo"), true)

	if len(added) != 1 {
		t.Fatalf("expected the field added under opt-in, got %v", added)
	}
	if record.Frontmatter["tipo"] != "report" {
		t.Errorf("expected values[0], got %v", record.Frontmatter["tipo"])
	}
}

func TestApplyFixes_SchemaDefaultAppliesWithoutOptIn(t *testing.T) {
	record := &extract.Record{Path: "a.md", Frontmatter: map[string]any{}}
	effective := &rules.StemFile{Schema: map[string]rules.SchemaField{
		"tipo": {Type: "enum", Values: []string{"report", "note"}, Default: "note"},
	}}

	added, _, skipped := ApplyFixes(context.Background(), record, effective, requiredErr("tipo"), false)

	if len(added) != 1 {
		t.Fatalf("a declared default is schema intent and must still apply, got added=%v skipped=%v", added, skipped)
	}
	if record.Frontmatter["tipo"] != "note" {
		t.Errorf("expected the declared default, got %v", record.Frontmatter["tipo"])
	}
}

// --- ApplyProposals: the fix --all path ---

func TestApplyProposals_SkipsEngineChosenWithoutOptIn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.md")
	original := "---\ntitle: A\n---\n\n# A\n"
	if err := os.WriteFile(path, []byte(original), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	record := &extract.Record{Path: "a.md", Frontmatter: map[string]any{"title": "A"}}
	report := &proposal.Report{Proposals: []proposal.Proposal{{
		Type:        proposal.AddField,
		Field:       "tipo",
		Paths:       []string{"a.md"},
		Value:       "report",
		ValueSource: proposal.ValueSourceEnumFirst,
	}}}

	if _, err := ApplyProposals(context.Background(), report, dir, []*extract.Record{record}, false); err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(got), "tipo") {
		t.Errorf("expected engine-chosen value not written, got:\n%s", got)
	}
	if len(report.Proposals) != 0 {
		t.Errorf("expected the skipped proposal excluded from applied, got %+v", report.Proposals)
	}
}

func TestApplyProposals_AppliesEngineChosenWithOptIn(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.md")
	if err := os.WriteFile(path, []byte("---\ntitle: A\n---\n\n# A\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	record := &extract.Record{Path: "a.md", Frontmatter: map[string]any{"title": "A"}}
	report := &proposal.Report{Proposals: []proposal.Proposal{{
		Type:        proposal.AddField,
		Field:       "tipo",
		Paths:       []string{"a.md"},
		Value:       "report",
		ValueSource: proposal.ValueSourceEnumFirst,
	}}}

	if _, err := ApplyProposals(context.Background(), report, dir, []*extract.Record{record}, true); err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(got), "tipo: report") {
		t.Errorf("expected engine-chosen value written under opt-in, got:\n%s", got)
	}
}

func TestApplyProposals_LegacyProposalWithoutMarkerStillApplies(t *testing.T) {
	dir := t.TempDir()
	path := filepath.Join(dir, "a.md")
	if err := os.WriteFile(path, []byte("---\ntitle: A\n---\n\n# A\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	record := &extract.Record{Path: "a.md", Frontmatter: map[string]any{"title": "A"}}
	// No ValueSource: a report file written by an earlier rootline.
	report := &proposal.Report{Proposals: []proposal.Proposal{{
		Type:  proposal.AddField,
		Field: "tipo",
		Paths: []string{"a.md"},
		Value: "report",
	}}}

	if _, err := ApplyProposals(context.Background(), report, dir, []*extract.Record{record}, false); err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}

	got, err := os.ReadFile(path)
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(got), "tipo: report") {
		t.Errorf("an unmarked legacy proposal must keep applying, got:\n%s", got)
	}
}

// --- ApplyRepair: the report-driven path ---

func TestApplyRepair_SkipsEngineChosenWithoutOptIn(t *testing.T) {
	dir := newRollbackFixture(t, map[string]string{
		"a.md": "---\nestado: Pending\n---\n\n# A\n",
	})

	proposals := []proposal.Proposal{{
		Type:        proposal.AddField,
		Field:       "otro",
		Paths:       []string{"a.md"},
		Value:       "",
		ValueSource: proposal.ValueSourceEmpty,
	}}

	result, err := ApplyRepair(proposals, false, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if strings.Contains(string(got), "otro") {
		t.Errorf("expected engine-chosen value not written, got:\n%s", got)
	}
	if len(result.Skipped) != 1 {
		t.Errorf("expected the skip reported, got %v", result.Skipped)
	}
}

func TestApplyRepair_AppliesEngineChosenWithOptIn(t *testing.T) {
	dir := newRollbackFixture(t, map[string]string{
		"a.md": "---\nestado: Pending\n---\n\n# A\n",
	})

	proposals := []proposal.Proposal{{
		Type:        proposal.AddField,
		Field:       "otro",
		Paths:       []string{"a.md"},
		Value:       "x",
		ValueSource: proposal.ValueSourceEnumFirst,
	}}

	if _, err := ApplyRepair(proposals, false, dir, true); err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}

	got, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatalf("read: %v", err)
	}
	if !strings.Contains(string(got), "otro: x") {
		t.Errorf("expected engine-chosen value written under opt-in, got:\n%s", got)
	}
}

func TestApplyProposals_ReportsWhatItSkipped(t *testing.T) {
	// A skip that leaves no trace reads as "nothing needed doing".
	dir := t.TempDir()
	path := filepath.Join(dir, "a.md")
	if err := os.WriteFile(path, []byte("---\ntitle: A\n---\n\n# A\n"), 0644); err != nil {
		t.Fatalf("write: %v", err)
	}

	record := &extract.Record{Path: "a.md", Frontmatter: map[string]any{"title": "A"}}
	report := &proposal.Report{Proposals: []proposal.Proposal{{
		Type:        proposal.AddField,
		Field:       "tipo",
		Paths:       []string{"a.md"},
		Value:       "report",
		ValueSource: proposal.ValueSourceEnumFirst,
	}}}

	skipped, err := ApplyProposals(context.Background(), report, dir, []*extract.Record{record}, false)
	if err != nil {
		t.Fatalf("ApplyProposals: %v", err)
	}

	if len(skipped) != 1 {
		t.Fatalf("expected the declined proposal returned, got %+v", skipped)
	}
	if skipped[0].Field != "tipo" {
		t.Errorf("expected the skipped field named, got %q", skipped[0].Field)
	}
	if skipped[0].ValueSource != proposal.ValueSourceEnumFirst {
		t.Errorf("expected the provenance carried through, got %q", skipped[0].ValueSource)
	}
}
