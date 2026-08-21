package rules

import (
	"path/filepath"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

// Typed-rule filtering was read from the invocation root's stem chain only, so
// a rule declared in a subdirectory was ignored when the command ran from the
// parent. It now resolves per record, matching FilterLinksByStyles.
func TestFilterLinksByTypedRules_ResolvesPerRecord(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".stem"), "version: 2\nroot: true\n")
	writeFile(t, filepath.Join(root, "sub", ".stem"), "version: 2\nlinks:\n  reference:\n    target: \".*\"\n")
	writeFile(t, filepath.Join(root, "sub", "x.md"), "body")
	writeFile(t, filepath.Join(root, "top.md"), "body")

	sub := &extract.Record{Path: filepath.Join("sub", "x.md"), Links: []extract.Link{
		{Target: "y", Type: "reference", Line: 1},
		{Target: "z", Type: "blocks", Line: 2},
	}}
	top := &extract.Record{Path: "top.md", Links: []extract.Link{
		{Target: "y", Type: "blocks", Line: 1},
	}}

	if err := FilterLinksByTypedRules([]*extract.Record{sub, top}, root); err != nil {
		t.Fatalf("FilterLinksByTypedRules: %v", err)
	}

	if len(sub.Links) != 1 || sub.Links[0].Type != "reference" {
		t.Errorf("sub/x.md links = %+v, want only the declared reference rule", sub.Links)
	}
	// top.md's schema declares no typed rules, so nothing is filtered there.
	if len(top.Links) != 1 {
		t.Errorf("top.md links = %+v, want untouched (no typed rules in its schema)", top.Links)
	}
}

// links.checks.cycles declared in a subdirectory used to evaporate when the
// command ran from the repository root — the most likely CI invocation
// (issue #62 sub-defect 6).
func TestCycleFailureScope_ResolvesPerRecord(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".stem"), "version: 2\nroot: true\n")
	writeFile(t, filepath.Join(root, "sub", ".stem"), "version: 2\nlinks:\n  checks:\n    cycles: true\n")
	writeFile(t, filepath.Join(root, "sub", "n1.md"), "body")
	writeFile(t, filepath.Join(root, "top.md"), "body")

	recs := []*extract.Record{
		{Path: filepath.Join("sub", "n1.md")},
		{Path: "top.md"},
	}
	scope, err := CycleFailureScope(recs, root)
	if err != nil {
		t.Fatalf("CycleFailureScope: %v", err)
	}

	if !scope[filepath.Join("sub", "n1.md")] {
		t.Errorf("sub/n1.md should opt into cycle failure via its own .stem")
	}
	if scope["top.md"] {
		t.Errorf("top.md declares no cycle hardening and must not opt in")
	}
}

// A schema carrying only styles, checks or allowed must not suppress links:
// typed filtering applies solely where typed rules are declared, or a
// styles-only repository ends up with an empty graph.
func TestFilterLinksByTypedRules_NonRuleSchemasDoNotSuppress(t *testing.T) {
	for name, stem := range map[string]string{
		"styles only":  "version: 2\nroot: true\nlinks:\n  styles: [wikilink]\n",
		"checks only":  "version: 2\nroot: true\nlinks:\n  checks:\n    anchors: true\n",
		"allowed only": "version: 2\nroot: true\nlinks:\n  allowed: [reference, blocks]\n",
		"no links":     "version: 2\nroot: true\n",
	} {
		t.Run(name, func(t *testing.T) {
			root := t.TempDir()
			writeFile(t, filepath.Join(root, ".stem"), stem)
			writeFile(t, filepath.Join(root, "a.md"), "body")
			rec := &extract.Record{Path: "a.md", Links: []extract.Link{
				{Target: "x", Type: "reference", Line: 1},
				{Target: "y", Type: "blocks", Line: 2},
			}}

			if err := FilterLinksByTypedRules([]*extract.Record{rec}, root); err != nil {
				t.Fatalf("FilterLinksByTypedRules: %v", err)
			}

			if len(rec.Links) != 2 {
				t.Errorf("links = %+v, want both kept", rec.Links)
			}
		})
	}
}

func TestLinkSchemaScopePropagatesInvalidRecordSchema(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, ".stem"), `version: 2
root: true
schema:
  id:
    type: sequence
    match:
      "bad*": {prefix: BAD, digits: 2.0}
links:
  checks:
    cycles: true
`)
	rec := &extract.Record{Path: "bad.md", Links: []extract.Link{{Target: "x", Type: "reference"}}}

	err := FilterLinksByTypedRules([]*extract.Record{rec}, root)
	if err == nil || !strings.Contains(err.Error(), "bad.md") || !strings.Contains(err.Error(), "digits") {
		t.Fatalf("FilterLinksByTypedRules error = %v, want schema digits cause", err)
	}

	_, err = CycleFailureScope([]*extract.Record{{Path: "bad.md"}}, root)
	if err == nil || !strings.Contains(err.Error(), "bad.md") || !strings.Contains(err.Error(), "digits") {
		t.Fatalf("CycleFailureScope error = %v, want schema digits cause", err)
	}
}
