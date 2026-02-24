package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

// --- ParseBlockingInfo edge cases ---

func TestParseBlockingInfo_EmptyParens(t *testing.T) {
	// "blocked by " followed by ")" doesn't match the regex (.+ requires content),
	// so the full string is returned as base.
	base, targets, notes := ParseBlockingInfo("Pending (blocked by )")
	if base != "Pending (blocked by )" {
		t.Errorf("base = %q, want full string (regex no match)", base)
	}
	if targets != nil {
		t.Errorf("targets = %v, want nil", targets)
	}
	if notes != nil {
		t.Errorf("notes = %v, want nil", notes)
	}
}

func TestParseBlockingInfo_OnlySpaces(t *testing.T) {
	base, targets, notes := ParseBlockingInfo("Pending (blocked by   +   )")
	if base != "Pending" {
		t.Errorf("base = %q, want Pending", base)
	}
	if len(targets) != 0 {
		t.Errorf("targets = %v, want empty (only whitespace parts)", targets)
	}
	if len(notes) != 0 {
		t.Errorf("notes = %v, want empty", notes)
	}
}

func TestParseBlockingInfo_MultipleTargets(t *testing.T) {
	base, targets, notes := ParseBlockingInfo("Blocked (blocked by T001 + T002 + T003)")
	if base != "Blocked" {
		t.Errorf("base = %q, want Blocked", base)
	}
	if len(targets) != 3 {
		t.Fatalf("targets = %v, want 3 items", targets)
	}
	if targets[0] != "T001" || targets[1] != "T002" || targets[2] != "T003" {
		t.Errorf("targets = %v, want [T001, T002, T003]", targets)
	}
	if len(notes) != 0 {
		t.Errorf("notes = %v, want []", notes)
	}
}

func TestParseBlockingInfo_NotBlockedByPattern(t *testing.T) {
	// Parenthesized but doesn't match "blocked by" pattern.
	base, targets, notes := ParseBlockingInfo("Pending (waiting for review)")
	if base != "Pending (waiting for review)" {
		t.Errorf("base = %q, want full string (no match)", base)
	}
	if targets != nil {
		t.Errorf("targets = %v, want nil", targets)
	}
	if notes != nil {
		t.Errorf("notes = %v, want nil", notes)
	}
}

func TestParseBlockingInfo_EmptyString(t *testing.T) {
	base, targets, notes := ParseBlockingInfo("")
	if base != "" {
		t.Errorf("base = %q, want empty", base)
	}
	if targets != nil || notes != nil {
		t.Error("expected nil targets and notes for empty input")
	}
}

func TestParseBlockingInfo_BaseWithSpaces(t *testing.T) {
	base, targets, _ := ParseBlockingInfo("In Progress (blocked by T001)")
	if base != "In Progress" {
		t.Errorf("base = %q, want 'In Progress'", base)
	}
	if len(targets) != 1 || targets[0] != "T001" {
		t.Errorf("targets = %v, want [T001]", targets)
	}
}

func TestParseBlockingInfo_AllNotes(t *testing.T) {
	base, targets, notes := ParseBlockingInfo("Pending (blocked by human + review + approval)")
	if base != "Pending" {
		t.Errorf("base = %q, want Pending", base)
	}
	if len(targets) != 0 {
		t.Errorf("targets = %v, want [] (all are notes)", targets)
	}
	if len(notes) != 3 {
		t.Fatalf("notes = %v, want 3 items", notes)
	}
}

// --- extractLinkTargetAndPattern edge cases ---

func TestExtractLinkTargetAndPattern_NoQuotes(t *testing.T) {
	target, pattern := extractLinkTargetAndPattern("some error without quotes")
	if target != "" || pattern != "" {
		t.Errorf("got (%q, %q), want empty for no quotes", target, pattern)
	}
}

func TestExtractLinkTargetAndPattern_OneQuotedString(t *testing.T) {
	target, pattern := extractLinkTargetAndPattern(`link target "E04" has issues`)
	if target != "E04" {
		t.Errorf("target = %q, want E04", target)
	}
	if pattern != "" {
		t.Errorf("pattern = %q, want empty (only one quoted string)", pattern)
	}
}

func TestExtractLinkTargetAndPattern_UnmatchedQuote(t *testing.T) {
	target, pattern := extractLinkTargetAndPattern(`link target "E04 has issues`)
	if target != "" || pattern != "" {
		t.Errorf("got (%q, %q), want empty for unmatched quote", target, pattern)
	}
}

// --- tryRetypeLink edge cases ---

func TestTryRetypeLink_NoOtherAllowed(t *testing.T) {
	schema := rules.LinkSchema{
		Allowed: []string{"blocks"},
		Rules: map[string]rules.LinkRule{
			"blocks": {Target: `^T\d{3}-`},
		},
	}
	got := tryRetypeLink("E04", "blocks", "", schema)
	if got != "" {
		t.Errorf("got %q, want empty (no other type available)", got)
	}
}

func TestTryRetypeLink_MatchesOtherPattern(t *testing.T) {
	schema := rules.LinkSchema{
		Allowed: []string{"blocks", "depends"},
		Rules: map[string]rules.LinkRule{
			"blocks":  {Target: `^T\d{3}-`},
			"depends": {Target: `^E\d{2}`},
		},
	}
	got := tryRetypeLink("E04", "blocks", "", schema)
	if got != "depends" {
		t.Errorf("got %q, want depends", got)
	}
}

func TestTryRetypeLink_NoMatchAnywhere(t *testing.T) {
	schema := rules.LinkSchema{
		Allowed: []string{"blocks", "depends"},
		Rules: map[string]rules.LinkRule{
			"blocks":  {Target: `^T\d{3}-`},
			"depends": {Target: `^F\d{2}`},
		},
	}
	got := tryRetypeLink("UNKNOWN", "blocks", "", schema)
	if got != "" {
		t.Errorf("got %q, want empty", got)
	}
}

// --- tryExpandTarget edge cases ---

func TestTryExpandTarget_MultipleMatches(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/T001-alpha.md"},
		{Path: "dir/T001-beta.md"},
	}
	got := tryExpandTarget("T001", "dir/T002.md", records)
	if got != "" {
		t.Errorf("got %q, want empty (ambiguous: multiple matches)", got)
	}
}

func TestTryExpandTarget_NoMatch(t *testing.T) {
	records := []*extract.Record{
		{Path: "dir/T002-something.md"},
	}
	got := tryExpandTarget("T001", "dir/T003.md", records)
	if got != "" {
		t.Errorf("got %q, want empty (no match)", got)
	}
}

func TestTryExpandTarget_DifferentDirectory(t *testing.T) {
	records := []*extract.Record{
		{Path: "other/T001-alpha.md"},
	}
	got := tryExpandTarget("T001", "dir/T002.md", records)
	if got != "" {
		t.Errorf("got %q, want empty (different directory)", got)
	}
}
