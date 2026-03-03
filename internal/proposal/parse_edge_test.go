package proposal

import (
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/rules"
)

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
