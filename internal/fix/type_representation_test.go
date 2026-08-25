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

func writeTypeRepairFixture(t *testing.T, valueLine string) string {
	t.Helper()
	dir := t.TempDir()
	stem := "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nschema:\n  value:\n    type: string\n"
	if err := os.WriteFile(filepath.Join(dir, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	content := "---\nvalue: " + valueLine + "\n---\n# Probe\n"
	if err := os.WriteFile(filepath.Join(dir, "a.md"), []byte(content), 0o600); err != nil {
		t.Fatal(err)
	}
	return dir
}

func typeRepairProposal(from, representation string) proposal.Proposal {
	return proposal.Proposal{
		Type: proposal.CorrectValue, Field: "value", Paths: []string{"a.md"},
		From: from, To: from, FromRepresentation: representation,
	}
}

func assertTypeRepairFileValid(t *testing.T, dir string) {
	t.Helper()
	path := filepath.Join(dir, "a.md")
	content, err := os.ReadFile(path)
	if err != nil {
		t.Fatal(err)
	}
	record, err := (&extract.MarkdownExtractor{}).Extract("a.md", content)
	if err != nil {
		t.Fatalf("Extract after repair: %v", err)
	}
	effective, err := rules.ResolveForRecord(dir, "a.md")
	if err != nil {
		t.Fatalf("ResolveForRecord after repair: %v", err)
	}
	if errs := rules.Validate(context.Background(), record, effective); len(errs) != 0 {
		t.Fatalf("post-repair validation errors = %#v", errs)
	}
}

func TestApplyRepairDuplicateTypedProposalInvalidatesScalarEvidence(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry-run", false: "real"}[dryRun], func(t *testing.T) {
			dir := writeTypeRepairFixture(t, "042")
			p := typeRepairProposal("042", "integer")
			result, err := ApplyRepair([]proposal.Proposal{p, p}, dryRun, dir, false)
			if err != nil {
				t.Fatalf("ApplyRepair: %v", err)
			}
			if len(result.Changed) != 1 || len(result.Rejected) != 1 || len(result.Skipped) != 0 || len(result.RolledBack) != 0 {
				t.Fatalf("result = %#v", result)
			}

			content, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			if dryRun {
				want := "---\nvalue: 042\n---\n# Probe\n"
				if string(content) != want {
					t.Fatalf("dry-run changed bytes\nwant:\n%s\ngot:\n%s", want, content)
				}
				return
			}

			want := "---\nvalue: \"042\"\n---\n# Probe\n"
			if string(content) != want {
				t.Fatalf("real run bytes\nwant:\n%s\ngot:\n%s", want, content)
			}
			assertTypeRepairFileValid(t, dir)
		})
	}
}

func TestApplyRepairSetFieldInvalidatesTypedScalarEvidence(t *testing.T) {
	for _, dryRun := range []bool{true, false} {
		t.Run(map[bool]string{true: "dry-run", false: "real"}[dryRun], func(t *testing.T) {
			dir := writeTypeRepairFixture(t, "042")
			proposals := []proposal.Proposal{
				{Type: proposal.SetField, Field: "value", Paths: []string{"a.md"}, Value: "intermediate"},
				typeRepairProposal("042", "integer"),
			}

			result, err := ApplyRepair(proposals, dryRun, dir, false)
			if err != nil {
				t.Fatalf("ApplyRepair: %v", err)
			}
			if len(result.Changed) != 1 || len(result.Rejected) != 1 || len(result.Skipped) != 0 || len(result.RolledBack) != 0 {
				t.Fatalf("stale typed evidence was accepted: %#v", result)
			}

			content, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			want := "---\nvalue: 042\n---\n# Probe\n"
			if !dryRun {
				want = "---\nvalue: intermediate\n---\n# Probe\n"
			}
			if string(content) != want {
				t.Fatalf("file content\nwant:\n%s\ngot:\n%s", want, content)
			}
		})
	}
}

func TestApplyRepairRejectsMalformedTypedProposalContracts(t *testing.T) {
	cases := []struct {
		name   string
		mutate func(proposal.Proposal) proposal.Proposal
	}{
		{
			name: "migrate value cannot carry representation marker",
			mutate: func(p proposal.Proposal) proposal.Proposal {
				p.Type = proposal.MigrateValue
				return p
			},
		},
		{
			name: "correct value marker cannot change text",
			mutate: func(p proposal.Proposal) proposal.Proposal {
				p.To = "43"
				return p
			},
		},
		{
			name: "unknown marker is not accepted",
			mutate: func(p proposal.Proposal) proposal.Proposal {
				p.FromRepresentation = "number"
				return p
			},
		},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeTypeRepairFixture(t, "42")
			before, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := ApplyRepair([]proposal.Proposal{tc.mutate(typeRepairProposal("42", "integer"))}, false, dir, false)
			if err != nil {
				t.Fatalf("ApplyRepair: %v", err)
			}
			if len(result.Rejected) != 1 || len(result.Changed) != 0 || len(result.Skipped) != 0 || len(result.RolledBack) != 0 {
				t.Fatalf("result = %#v", result)
			}
			after, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Fatalf("rejected malformed typed proposal modified bytes\nbefore:\n%s\nafter:\n%s", before, after)
			}
		})
	}
}

func TestApplyRepairQuotesExactScalarLexeme(t *testing.T) {
	dir := writeTypeRepairFixture(t, "042")
	result, err := ApplyRepair([]proposal.Proposal{typeRepairProposal("042", "integer")}, false, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if len(result.Changed) != 1 || len(result.Rejected) != 0 || len(result.RolledBack) != 0 {
		t.Fatalf("result = %#v", result)
	}
	content, err := os.ReadFile(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if !strings.Contains(string(content), `value: "042"`) {
		t.Fatalf("exact lexeme was not quoted once:\n%s", content)
	}
	info, err := os.Stat(filepath.Join(dir, "a.md"))
	if err != nil {
		t.Fatal(err)
	}
	if info.Mode().Perm() != 0o600 {
		t.Fatalf("mode = %o, want 600", info.Mode().Perm())
	}
}

func TestApplyRepairTypeGuardRejectsStaleOrUnknownEvidence(t *testing.T) {
	cases := []struct {
		name           string
		valueLine      string
		from           string
		representation string
	}{
		{name: "changed lexeme", valueLine: "+42", from: "42", representation: "integer"},
		{name: "changed representation", valueLine: "true", from: "true", representation: "integer"},
		{name: "unknown marker", valueLine: "42", from: "42", representation: "number"},
		{name: "quoted integer", valueLine: `"42"`, from: "42", representation: "integer"},
		{name: "quoted signed integer", valueLine: `"+42"`, from: "+42", representation: "integer"},
		{name: "quoted leading-zero integer", valueLine: `"042"`, from: "042", representation: "integer"},
		{name: "quoted boolean", valueLine: `"TRUE"`, from: "TRUE", representation: "boolean"},
		{name: "quoted timestamp", valueLine: `"2026-06-22T00:00:00Z"`, from: "2026-06-22T00:00:00Z", representation: "timestamp"},
	}
	for _, tc := range cases {
		t.Run(tc.name, func(t *testing.T) {
			dir := writeTypeRepairFixture(t, tc.valueLine)
			before, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			result, err := ApplyRepair([]proposal.Proposal{typeRepairProposal(tc.from, tc.representation)}, false, dir, false)
			if err != nil {
				t.Fatalf("ApplyRepair: %v", err)
			}
			if len(result.Rejected) != 1 || len(result.Changed) != 0 {
				t.Fatalf("result = %#v", result)
			}
			after, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			if string(after) != string(before) {
				t.Fatal("rejected proposal modified the file")
			}
		})
	}
}

func TestApplyRepairLegacyCorrectValueRemainsStrict(t *testing.T) {
	dir := writeTypeRepairFixture(t, "zed")
	legacy := proposal.Proposal{
		Type: proposal.CorrectValue, Field: "value", Paths: []string{"a.md"},
		From: "alice", To: "bob",
	}
	result, err := ApplyRepair([]proposal.Proposal{legacy}, false, dir, false)
	if err != nil {
		t.Fatalf("ApplyRepair: %v", err)
	}
	if len(result.Rejected) != 1 || len(result.Changed) != 0 {
		t.Fatalf("legacy strict guard changed: %#v", result)
	}
}

func TestApplyRepairTypeRepresentationReapplyRejectsAlreadyQuoted(t *testing.T) {
	cases := []struct {
		line, from, representation, quoted string
	}{
		{"2026-06-22T00:00:00Z", "2026-06-22T00:00:00Z", "timestamp", `"2026-06-22T00:00:00Z"`},
		{"TRUE", "TRUE", "boolean", `"TRUE"`},
		{"+42", "+42", "integer", `"+42"`},
		{"042", "042", "integer", `"042"`},
	}
	for _, tc := range cases {
		t.Run(tc.representation+"/"+tc.from, func(t *testing.T) {
			dir := writeTypeRepairFixture(t, tc.line)
			p := typeRepairProposal(tc.from, tc.representation)
			first, err := ApplyRepair([]proposal.Proposal{p}, false, dir, false)
			if err != nil || len(first.Changed) != 1 || len(first.Skipped) != 0 || len(first.Rejected) != 0 {
				t.Fatalf("first apply: result=%#v err=%v", first, err)
			}
			firstBytes, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			if !strings.Contains(string(firstBytes), "value: "+tc.quoted) {
				t.Fatalf("exact quote missing:\n%s", firstBytes)
			}
			second, err := ApplyRepair([]proposal.Proposal{p}, false, dir, false)
			if err != nil || len(second.Changed) != 0 || len(second.Skipped) != 0 || len(second.Rejected) != 1 {
				t.Fatalf("second apply: result=%#v err=%v", second, err)
			}
			secondBytes, err := os.ReadFile(filepath.Join(dir, "a.md"))
			if err != nil {
				t.Fatal(err)
			}
			if string(secondBytes) != string(firstBytes) {
				t.Fatal("second apply changed bytes")
			}
		})
	}
}
