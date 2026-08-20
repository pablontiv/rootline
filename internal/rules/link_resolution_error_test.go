package rules

import (
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestLinkHelpersReturnInvalidGovernanceErrorsWithoutMutatingRecords(t *testing.T) {
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

	for name, run := range map[string]func([]*extract.Record, string) error{
		"filter styles": FilterLinksByStyles,
		"prepare":       PrepareLinks,
		"filter typed":  FilterLinksByTypedRules,
	} {
		t.Run(name, func(t *testing.T) {
			record := &extract.Record{Path: "BAD001.md", Links: []extract.Link{{Target: "target.md", Style: extract.StyleMarkdown}}}
			before := append([]extract.Link(nil), record.Links...)
			err := run([]*extract.Record{record}, root)
			if err == nil || !strings.Contains(err.Error(), "BAD001.md") || !strings.Contains(err.Error(), "digits") {
				t.Fatalf("error = %v, want strict BAD001 resolution cause", err)
			}
			if len(record.Links) != len(before) || record.Links[0] != before[0] {
				t.Fatalf("links mutated after failed resolution: %#v, want %#v", record.Links, before)
			}
		})
	}

	record := &extract.Record{Path: "BAD001.md"}
	if _, err := CycleFailureScope([]*extract.Record{record}, root); err == nil || !strings.Contains(err.Error(), "digits") {
		t.Fatalf("CycleFailureScope error = %v, want strict resolution cause", err)
	}
}
