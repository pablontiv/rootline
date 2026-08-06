package e2e

import (
	"context"
	"fmt"
	"path/filepath"
	"sort"
	"strings"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

// Issue #62 was that `validate` and `graph` ran two disjoint link engines and
// never agreed. These tests are the regression guard: each one runs BOTH
// engines over the same corpus and asserts they reach the same verdict, so a
// future change cannot quietly reintroduce the divergence.
//
// Rows that intentionally differ get their own test pinning the DECLARED
// difference, because the defect was never "they differ" — it was that they
// differed silently.

// scanFresh returns a fresh record set. Both engines mutate the links they are
// given, so each needs its own copy of the corpus.
func scanFresh(t *testing.T, root string) []*extract.Record {
	t.Helper()
	records, err := index.Scan(context.Background(), root, extract.NewRegistry())
	if err != nil {
		t.Fatalf("scan: %v", err)
	}
	return records
}

// brokenByValidate returns "<record> -> <target>" for every link validate
// reports as an error, which is the verdict a CI job running validate sees.
func brokenByValidate(t *testing.T, root string) []string {
	t.Helper()
	var out []string
	cache := rules.NewHeadingCache()
	for _, rec := range scanFresh(t, root) {
		abs := filepath.Join(root, rec.Path)
		effective, err := rules.ResolveForRecord(filepath.Dir(abs), rec.Path)
		if err != nil {
			continue
		}
		for _, e := range rules.CheckLinks(rec.Links, effective.Links, abs, root, cache) {
			if e.Severity != "error" {
				continue // warnings are not a broken-link verdict
			}
			out = append(out, fmt.Sprintf("%s -> %s", rec.Path, targetOf(e.Message)))
		}
	}
	sort.Strings(out)
	return out
}

// brokenByGraph returns the same shape for the verdict `graph --check` sees.
func brokenByGraph(t *testing.T, root string) []string {
	t.Helper()
	records := scanFresh(t, root)
	rules.FilterLinksByStyles(records, root)
	rules.FilterLinksByTypedRules(records, root)
	rules.PrepareLinks(records, root)
	g := graph.Build(context.Background(), records)

	var out []string
	for _, b := range g.BrokenLinks() {
		out = append(out, fmt.Sprintf("%s -> %s", b.Source, b.Target))
	}
	sort.Strings(out)
	return out
}

// targetOf pulls the quoted target out of a link error message so the two
// engines can be compared on the same key.
func targetOf(msg string) string {
	first := strings.Index(msg, "\"")
	if first < 0 {
		return msg
	}
	rest := msg[first+1:]
	end := strings.Index(rest, "\"")
	if end < 0 {
		return msg
	}
	return rest[:end]
}

// assertEngines checks both that the two engines agree AND that they reached
// the verdict the row expects.
//
// Agreement alone is a false guard: a row where both engines resolve every
// link compares empty against empty and passes no matter what, which is
// exactly how the original defect would slip back in — validate going silent
// looks like agreement. wantBroken pins the expected count so a row that is
// supposed to find something must actually find it.
func assertEngines(t *testing.T, root string, wantBroken int) {
	t.Helper()
	v, g := brokenByValidate(t, root), brokenByGraph(t, root)
	if strings.Join(v, "|") != strings.Join(g, "|") {
		t.Errorf("engines disagree on broken links:\n  validate: %v\n  graph:    %v", v, g)
	}
	if len(v) != wantBroken {
		t.Errorf("validate broken = %d, want %d: %v", len(v), wantBroken, v)
	}
	if len(g) != wantBroken {
		t.Errorf("graph broken = %d, want %d: %v", len(g), wantBroken, g)
	}
}

// One corpus per parity-matrix row. Each row is a link shape the two engines
// used to answer differently.
func TestE2E_Parity_EnginesAgreePerMatrixRow(t *testing.T) {
	stem := "version: 2\nroot: true\nscope:\n  match: \"*.md\"\n"
	mdStem := stem + "links:\n  styles: [markdown]\n"

	type row struct {
		files      map[string]string
		wantBroken int
	}
	rows := map[string]row{
		// Sub-defect 1: the headline. validate reported every valid
		// extension-less wikilink broken while graph reported clean.
		"extension-less wikilink resolves": {map[string]string{
			".stem": stem, "b.md": "# B\n", "t.md": "See [[b]].\n",
		}, 0},
		// Sub-defect 10: worked for [[b]] but not once a directory prefix
		// was added, an inconsistency inside graph itself.
		"path-qualified wikilink resolves": {map[string]string{
			".stem": stem, "sub/README.md": "# Sub\n", "t.md": "See [[sub/README]].\n",
		}, 0},
		// Sub-defect 9: the anchor was folded into the filename.
		"wikilink anchor resolves": {map[string]string{
			".stem": stem, "b.md": "# Heading One\n", "t.md": "See [[b#heading-one]].\n",
		}, 0},
		// Sub-defect 3: validate skipped root-anchored targets outright.
		"root-anchored target resolves": {map[string]string{
			".stem": mdStem, "docs/Page.md": "# P\n", "deep/t.md": "See [x](/docs/Page.md).\n",
		}, 0},
		"root-anchored missing target is broken": {map[string]string{
			".stem": mdStem, "deep/t.md": "See [x](/docs/Nope.md).\n",
		}, 1},
		// Sub-defect 2: no links.checks block. validate was silent, graph failed.
		"broken target with no checks block": {map[string]string{
			".stem": stem, "b.md": "# B\n", "t.md": "See [[nope]].\n",
		}, 1},
		// Markdown stays literal: no .md inference for that style.
		"markdown target resolves literally": {map[string]string{
			".stem": mdStem, "b.md": "# B\n", "t.md": "See [x](b.md).\n",
		}, 0},
		"markdown target without extension is broken": {map[string]string{
			".stem": mdStem, "b.md": "# B\n", "t.md": "See [x](b).\n",
		}, 1},
		"case mismatch is broken": {map[string]string{
			".stem": mdStem, "Setup.md": "# S\n", "t.md": "See [x](setup.md).\n",
		}, 1},
		"directory target resolves to README": {map[string]string{
			".stem": mdStem, "guides/README.md": "# G\n", "t.md": "See [x](guides/).\n",
		}, 0},
		"percent-encoded target resolves": {map[string]string{
			".stem": mdStem, "my page.md": "# P\n", "t.md": "See [x](my%20page.md).\n",
		}, 0},
	}

	for name, r := range rows {
		t.Run(name, func(t *testing.T) {
			assertEngines(t, setupProject(t, r.files), r.wantBroken)
		})
	}
}

// A target that exists but is not a governed record resolved fine: the schema
// declares what is governed, not what exists. Reporting it broken is what made
// the two commands disagree, so neither may report it.
func TestE2E_Parity_UngovernedButExistingTargetIsNotBroken(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":    "version: 2\nroot: true\nscope:\n  match: \"T*.md\"\n",
		"T001.md":  "See [[other]].\n",
		"other.md": "# Other\n",
	})
	assertEngines(t, root, 0)
	if g := brokenByGraph(t, root); len(g) != 0 {
		t.Errorf("existing-but-ungoverned target reported broken by graph: %v", g)
	}
}

// The one DECLARED divergence. With links.basename_fallback on, graph resolves
// a bare cross-directory target and validate cannot, because it checks one
// record at a time and has no index. validate must say so explicitly — a
// warning, not silence and not a wrong "broken" verdict. This test exists so a
// future change cannot turn the declared difference back into a silent one.
func TestE2E_Parity_BasenameFallbackDivergenceIsDeclared(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":       "version: 2\nroot: true\nscope:\n  match: \"*.md\"\nlinks:\n  basename_fallback: true\n",
		"deep/far.md": "# Far\n",
		"t.md":        "See [[far]].\n",
	})

	if g := brokenByGraph(t, root); len(g) != 0 {
		t.Errorf("graph should resolve the bare target via basename fallback, got broken: %v", g)
	}
	if v := brokenByValidate(t, root); len(v) != 0 {
		t.Errorf("validate must not report a hard error it cannot substantiate: %v", v)
	}

	var sawUnverifiable bool
	cache := rules.NewHeadingCache()
	for _, rec := range scanFresh(t, root) {
		abs := filepath.Join(root, rec.Path)
		effective, err := rules.ResolveForRecord(filepath.Dir(abs), rec.Path)
		if err != nil {
			continue
		}
		for _, e := range rules.CheckLinks(rec.Links, effective.Links, abs, root, cache) {
			if e.Rule == "link_unverifiable" {
				sawUnverifiable = true
				if e.Severity != "warn" {
					t.Errorf("link_unverifiable severity = %q, want warn so it does not fail the run", e.Severity)
				}
			}
		}
	}
	if !sawUnverifiable {
		t.Error("validate went silent on a link it cannot decide — that is the silent divergence issue #62 removed")
	}
}

// Sub-defect 13, fixed earlier: graph refused to build without a .stem. Link
// integrity is a property of document bodies, not of any schema.
func TestE2E_Parity_GraphWorksWithoutSchema(t *testing.T) {
	root := setupProject(t, map[string]string{
		"a.md": "See [[missing]].\n",
	})
	records := scanFresh(t, root)
	rules.FilterLinksByStyles(records, root)
	rules.PrepareLinks(records, root)
	g := graph.Build(context.Background(), records)
	if len(g.Nodes) != 1 {
		t.Fatalf("nodes = %d, want the schemaless record still graphed", len(g.Nodes))
	}
	if len(g.BrokenLinks()) != 1 {
		t.Errorf("broken = %v, want the dangling target reported", g.BrokenLinks())
	}
}
