# Graph Edges from Markdown Links Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Make `rootline graph` build edges from markdown-style links with the same target resolution `validate` uses, so markdown-styles repos (dsprima wiki) get real edges and consistent broken-link reports.

**Architecture:** Two fixes plus one wiring step. (1) `filterLinksBySchema` stops treating a styles/checks-only `.stem` as a typed-rule filter. (2) A new `rules.ResolveMarkdownTargets` rewrites markdown link targets to graph node keys using the existing `resolveCaseSensitive` (decode `%20`, case-sensitive walk, directory → `README.md`); `graph.Build` uses markdown targets as-is (no re-join, no basename fallback) so resolved targets match exactly and unresolved ones surface verbatim as broken links. (3) `runGraph` calls the resolver between `FilterLinksByStyles` and `Build`.

**Tech Stack:** Go 1.25+, cobra CLI, standard `testing` package only.

**Spec:** `docs/superpowers/specs/2026-07-10-graph-markdown-edges-design.md`

## Global Constraints

- Go 1.25+; module `github.com/pablontiv/rootline`.
- `just check` (gofmt + golangci-lint + build) and `just test` (`go test ./... -race`) must pass after every task.
- Per-package coverage floor is 85% (`.coverage-floors.toml`); the pre-push hook runs `just coverage-check` when `.go` files are pushed. Never `git push --no-verify`.
- Conventional commits only. Never add "Co-Authored-By" or any AI attribution.
- JSON output contracts are versioned (`"version": 1`). Do NOT change the exported JSON shape of `graph.Edge`, `graph.BrokenLink`, or `GraphResult` — the new `Edge` field must be unexported.
- Tests live in the same package as the code (`package rules`, `package graph`, `package e2e`, `package main` for cmd) — no `_test` package suffix, no external test frameworks.
- All code, comments, and test names in English.

## Discovered constraints (why Tasks 3's Build change exists)

The spec says "rewrite `link.Target` to the repo-relative path so it matches graph node keys" and "`graph.Build` stays pure". Two facts from the code make a small `Build` change necessary to honor that:

1. `resolveTarget` (`internal/graph/graph.go:300-306`) re-joins any target containing `/` or `..` against the source's directory. A repo-relative rewrite like `docs/sub/page.md` from source `docs/overview.md` would become `docs/docs/sub/page.md`. So `Build` must use markdown targets as-is.
2. `resolveByBasename` (`internal/graph/graph.go:72-96`) rescues any unmatched target with a unique basename match. An unresolved markdown target `missing.md` would silently gain an edge to `other/missing.md` while `validate` flags it broken — violating the spec's parity and canary requirements. So markdown edges must skip the basename fallback (which per spec remains a wikilink affordance).

"Stays pure" is preserved: `Build` still does zero filesystem access.

---

### Task 1: Gate fix — styles/checks-only schema no longer suppresses links

**Files:**
- Modify: `cmd/rootline/graph.go:200-215` (`filterLinksBySchema`)
- Test: `cmd/rootline/graph_test.go` (extend; existing `TestFilterLinksBySchema_*` live at lines 175-218)

**Interfaces:**
- Consumes: `rules.LinkSchema{Styles, Checks, Allowed, Rules}` (`internal/rules/rules.go:54-59`), `extract.Record`, `extract.Link`.
- Produces: `filterLinksBySchema` filters only when `len(schema.Rules) > 0`. No signature change.

- [ ] **Step 1: Write the failing tests**

Add to `cmd/rootline/graph_test.go` (imports `extract` and `rules` already present):

```go
func TestFilterLinksBySchema_StylesOnlySchema(t *testing.T) {
	rec := &extract.Record{
		Path:  "a.md",
		Links: []extract.Link{{Target: "b.md", Type: "reference", Style: extract.StyleMarkdown}},
	}
	schema := rules.LinkSchema{Styles: []string{"markdown"}}
	filterLinksBySchema([]*extract.Record{rec}, schema)
	if len(rec.Links) != 1 {
		t.Fatalf("links = %d, want 1 (styles-only schema must not drop links)", len(rec.Links))
	}
}

func TestFilterLinksBySchema_ChecksOnlySchema(t *testing.T) {
	rec := &extract.Record{
		Path:  "a.md",
		Links: []extract.Link{{Target: "b.md", Style: extract.StyleWikilink}},
	}
	schema := rules.LinkSchema{Checks: &rules.LinkChecks{Resolve: true}}
	filterLinksBySchema([]*extract.Record{rec}, schema)
	if len(rec.Links) != 1 {
		t.Fatalf("links = %d, want 1 (checks-only schema must not drop links)", len(rec.Links))
	}
}

func TestFilterLinksBySchema_AllowedOnlySchema(t *testing.T) {
	rec := &extract.Record{
		Path:  "a.md",
		Links: []extract.Link{{Target: "b.md", Type: "reference", Style: extract.StyleWikilink}},
	}
	schema := rules.LinkSchema{Allowed: []string{"reference"}}
	filterLinksBySchema([]*extract.Record{rec}, schema)
	if len(rec.Links) != 1 {
		t.Fatalf("links = %d, want 1 (typed filtering applies only when rules are declared)", len(rec.Links))
	}
}
```

Note: if `rules.LinkChecks` field is named differently than `Resolve`, check `internal/rules/rules.go` — `CheckLinks` references `schema.Checks.Resolve`, `schema.Checks.Anchors`, `schema.Checks.Encoding`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./cmd/rootline/ -run 'TestFilterLinksBySchema_(StylesOnly|ChecksOnly|AllowedOnly)' -v`
Expected: all three FAIL with `links = 0, want 1` (current guard `schema.IsEmpty()` is false, `Rules` is empty, every link is dropped).

- [ ] **Step 3: Change the guard**

In `cmd/rootline/graph.go`, replace `filterLinksBySchema`'s guard:

```go
// filterLinksBySchema removes links from records whose type has no rule in the schema.
// Typed-rule filtering applies only when typed rules are declared; a schema
// carrying only styles/checks/allowed must not suppress links (the graph
// showed zero edges on styles-only repos otherwise).
func filterLinksBySchema(records []*extract.Record, schema rules.LinkSchema) {
	if len(schema.Rules) == 0 {
		return
	}
	for _, rec := range records {
		filtered := rec.Links[:0]
		for _, link := range rec.Links {
			if _, ok := schema.Rules[link.Type]; ok {
				filtered = append(filtered, link)
			}
		}
		rec.Links = filtered
	}
}
```

- [ ] **Step 4: Run tests to verify they pass, plus existing suite**

Run: `go test ./cmd/rootline/ -run 'TestFilterLinksBySchema|TestGraph' -v`
Expected: PASS, including existing `TestFilterLinksBySchema_WithRules`, `TestFilterLinksBySchema_EmptySchema`, `TestGraphCheck_SchemaFiltersReferenceLinks` (typed-rule behavior unchanged).

- [ ] **Step 5: Full check and commit**

```bash
just check && just test
git add cmd/rootline/graph.go cmd/rootline/graph_test.go
git commit -m "fix(graph): styles-only link schema no longer suppresses edges"
```

---

### Task 2: `rules.ResolveMarkdownTargets` — validate-parity resolution

**Files:**
- Create: `internal/rules/link_targets.go`
- Test: `internal/rules/link_targets_test.go`

**Interfaces:**
- Consumes: `resolveCaseSensitive(baseDir, target string) (string, string, bool)` (unexported, `internal/rules/link_checks.go:159-198`; decodes `%20`, walks components case-sensitively, resolves directory targets to `README.md`); `extract.Link.Style`, `extract.StyleMarkdown`, `extract.StyleWikilink` (`internal/extract/links.go`).
- Produces: `func ResolveMarkdownTargets(records []*extract.Record, root string)` — exported, mutates `rec.Links[i].Target` in place. Resolved targets become root-relative paths in `filepath.Rel` form (OS separators — this is what graph node keys `rec.Path` use). Unresolved targets stay verbatim. Wikilinks untouched. Tasks 3 and 4 depend on this exact name and signature.

- [ ] **Step 1: Write the failing tests**

Create `internal/rules/link_targets_test.go`:

```go
package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func writeLinkTargetFixture(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		abs := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(abs), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(abs, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestResolveMarkdownTargets_RewritesResolvableTargets(t *testing.T) {
	root := t.TempDir()
	writeLinkTargetFixture(t, root, map[string]string{
		"README.md":          "# Root\n",
		"docs/a.md":          "# A\n",
		"docs/mi page.md":    "# Encoded\n",
		"docs/sub/README.md": "# Sub index\n",
	})

	rec := &extract.Record{
		Path: filepath.Join("docs", "a.md"),
		Links: []extract.Link{
			{Target: "mi%20page.md", Style: extract.StyleMarkdown},
			{Target: "sub", Style: extract.StyleMarkdown},
			{Target: "../README.md", Style: extract.StyleMarkdown},
			{Target: "/README.md", Style: extract.StyleMarkdown},
		},
	}
	ResolveMarkdownTargets([]*extract.Record{rec}, root)

	want := []string{
		filepath.Join("docs", "mi page.md"),
		filepath.Join("docs", "sub", "README.md"),
		"README.md",
		"README.md",
	}
	for i, w := range want {
		if rec.Links[i].Target != w {
			t.Errorf("link %d target = %q, want %q", i, rec.Links[i].Target, w)
		}
	}
}

func TestResolveMarkdownTargets_LeavesUnresolvableAndWikilinks(t *testing.T) {
	root := t.TempDir()
	writeLinkTargetFixture(t, root, map[string]string{
		"docs/a.md":    "# A\n",
		"docs/Page.md": "# Page\n",
	})

	rec := &extract.Record{
		Path: filepath.Join("docs", "a.md"),
		Links: []extract.Link{
			{Target: "page.md", Style: extract.StyleMarkdown},
			{Target: "no-existe.md", Style: extract.StyleMarkdown},
			{Target: "Page.md", Style: extract.StyleWikilink},
		},
	}
	ResolveMarkdownTargets([]*extract.Record{rec}, root)

	want := []string{"page.md", "no-existe.md", "Page.md"}
	for i, w := range want {
		if rec.Links[i].Target != w {
			t.Errorf("link %d target = %q, want %q", i, rec.Links[i].Target, w)
		}
	}
}
```

The first test covers: `%20` decoding, directory → `README.md`, `../` traversal, and root-anchored (`/`-prefixed) targets resolved against the repo root (ADO wikis use root-anchored paths; `CheckLinks` skips them without error, so the graph must not flag them broken — resolving them keeps parity). The second covers: case-mismatch left verbatim (case-sensitive even on APFS, via `findEntry`), missing file left verbatim, wikilink untouched.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/rules/ -run TestResolveMarkdownTargets -v`
Expected: FAIL to compile — `undefined: ResolveMarkdownTargets`.

- [ ] **Step 3: Implement**

Create `internal/rules/link_targets.go`:

```go
package rules

import (
	"path/filepath"
	"strings"

	"github.com/pablontiv/rootline/internal/extract"
)

// ResolveMarkdownTargets rewrites markdown-style link targets to the
// root-relative paths used as graph node keys, applying the same resolution
// as validate's link checks: %20 decoding, case-sensitive component walk,
// and directory targets resolving to their README.md. Root-anchored targets
// ("/x.md") resolve against root. Targets that fail to resolve are left
// verbatim so the graph reports them as broken links. Wikilinks are never
// touched.
func ResolveMarkdownTargets(records []*extract.Record, root string) {
	for _, rec := range records {
		dir := filepath.Dir(filepath.Join(root, rec.Path))
		for i, link := range rec.Links {
			if link.Style != extract.StyleMarkdown {
				continue
			}
			base, target := dir, link.Target
			if strings.HasPrefix(target, "/") {
				base, target = root, strings.TrimPrefix(target, "/")
			}
			resolved, _, ok := resolveCaseSensitive(base, target)
			if !ok {
				continue
			}
			if rel, err := filepath.Rel(root, resolved); err == nil {
				rec.Links[i].Target = rel
			}
		}
	}
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/rules/ -run TestResolveMarkdownTargets -v`
Expected: PASS.

- [ ] **Step 5: Full check and commit**

```bash
just check && just test
git add internal/rules/link_targets.go internal/rules/link_targets_test.go
git commit -m "feat(rules): resolve markdown link targets with validate parity"
```

---

### Task 3: `graph.Build` honors pre-resolved markdown targets

**Files:**
- Modify: `internal/graph/graph.go` (`Edge` struct at 23-28, `Build` loop at 52-62, `resolveByBasename` at 72-96)
- Test: `internal/graph/graph_test.go` (extend)
- Modify: `internal/e2e/link_styles_e2e_test.go` (markdown-style tests gain the resolve step; add mixed-style test)

**Interfaces:**
- Consumes: `extract.StyleMarkdown`, `rules.ResolveMarkdownTargets(records, root)` from Task 2 (e2e only — `internal/graph` must NOT import `internal/rules`).
- Produces: `Build` contract — edges for links with `Style == extract.StyleMarkdown` use `link.Target` verbatim (no `resolveTarget` re-join) and are excluded from basename fallback. `Edge` gains unexported `noFallback bool` (JSON shape unchanged). Wikilink behavior byte-identical.

- [ ] **Step 1: Write the failing unit tests**

Add to `internal/graph/graph_test.go` (package `graph`; add `"path/filepath"` and `extract` imports if missing):

```go
func TestBuild_MarkdownTargetsUsedAsIs(t *testing.T) {
	records := []*extract.Record{
		{Path: filepath.Join("docs", "a.md"), Links: []extract.Link{
			{Target: filepath.Join("docs", "b.md"), Style: extract.StyleMarkdown, Line: 3},
		}},
		{Path: filepath.Join("docs", "b.md")},
	}
	g := Build(context.Background(), records)
	edges := g.Edges[filepath.Join("docs", "a.md")]
	if len(edges) != 1 || edges[0].Target != filepath.Join("docs", "b.md") {
		t.Fatalf("edges = %+v, want one edge to docs/b.md", edges)
	}
	if broken := g.BrokenLinks(); len(broken) != 0 {
		t.Errorf("broken = %+v, want none", broken)
	}
}

func TestBuild_MarkdownSkipsBasenameFallback(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Links: []extract.Link{
			{Target: "missing.md", Style: extract.StyleMarkdown, Line: 1},
		}},
		{Path: filepath.Join("other", "missing.md")},
	}
	g := Build(context.Background(), records)
	broken := g.BrokenLinks()
	if len(broken) != 1 || broken[0].Target != "missing.md" {
		t.Fatalf("broken = %+v, want unresolved markdown target reported verbatim", broken)
	}
}

func TestBuild_WikilinkBasenameFallbackUnchanged(t *testing.T) {
	records := []*extract.Record{
		{Path: "a.md", Links: []extract.Link{
			{Target: "missing.md", Style: extract.StyleWikilink, Line: 1},
		}},
		{Path: filepath.Join("other", "missing.md")},
	}
	g := Build(context.Background(), records)
	if broken := g.BrokenLinks(); len(broken) != 0 {
		t.Fatalf("broken = %+v, want basename fallback to resolve wikilink", broken)
	}
}
```

The third test is a regression guard (passes before and after).

- [ ] **Step 2: Run tests to verify the first two fail**

Run: `go test ./internal/graph/ -run TestBuild_ -v`
Expected: `TestBuild_MarkdownTargetsUsedAsIs` FAILS (resolveTarget re-joins to `docs/docs/b.md` → broken); `TestBuild_MarkdownSkipsBasenameFallback` FAILS (fallback rescues `missing.md` → `other/missing.md`, broken = 0); `TestBuild_WikilinkBasenameFallbackUnchanged` PASSES.

- [ ] **Step 3: Implement**

In `internal/graph/graph.go`, extend `Edge`:

```go
// Edge represents a directed link from one document to another.
type Edge struct {
	Source string `json:"source"`
	Target string `json:"target"`
	Type   string `json:"type"`
	Line   int    `json:"line"`

	// noFallback excludes the edge from basename fallback resolution.
	// Markdown targets are path-resolved upstream; an unresolved one must
	// stay verbatim so BrokenLinks reports it, matching validate.
	noFallback bool
}
```

Change the `Build` edge loop:

```go
	for _, rec := range records {
		for _, link := range rec.Links {
			target := link.Target
			markdown := link.Style == extract.StyleMarkdown
			if !markdown {
				target = resolveTarget(rec.Path, link.Target)
			}
			g.Edges[rec.Path] = append(g.Edges[rec.Path], Edge{
				Source:     rec.Path,
				Target:     target,
				Type:       link.Type,
				Line:       link.Line,
				noFallback: markdown,
			})
		}
	}
```

In `resolveByBasename`, skip flagged edges:

```go
	for src, edges := range g.Edges {
		for i, edge := range edges {
			if edge.noFallback {
				continue
			}
			if _, exists := g.Nodes[edge.Target]; exists {
				continue // already resolved
			}
			// Try basename lookup.
			if matches, ok := idx[edge.Target]; ok && len(matches) == 1 {
				g.Edges[src][i].Target = matches[0]
			}
		}
	}
```

- [ ] **Step 4: Update the markdown e2e tests to mirror the real pipeline**

`internal/e2e/link_styles_e2e_test.go` — `TestE2E_LinkStyles_GraphIncludesMarkdownWhenDeclared` now fails because `[back](../README.md)` no longer re-joins. Insert the resolve step (the pipeline order `runGraph` will use) and strengthen the assertion:

```go
func TestE2E_LinkStyles_GraphIncludesMarkdownWhenDeclared(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":            "version: 2\nlinks:\n  styles: [markdown]\n",
		"README.md":        "# Root\n\n[overview](docs/overview.md)\n",
		"docs/overview.md": "# Overview\n\n[back](../README.md)\n",
	})
	ctx := context.Background()
	records, err := index.Scan(ctx, root, extract.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	rules.FilterLinksByStyles(records, root)
	rules.ResolveMarkdownTargets(records, root)
	g := graph.Build(ctx, records)

	if len(g.Edges["README.md"]) != 1 {
		t.Fatalf("README edges = %+v, want 1 markdown edge", g.Edges["README.md"])
	}
	if g.Edges["README.md"][0].Target != filepath.Join("docs", "overview.md") {
		t.Errorf("edge target = %q", g.Edges["README.md"][0].Target)
	}
	back := g.Edges[filepath.Join("docs", "overview.md")]
	if len(back) != 1 || back[0].Target != "README.md" {
		t.Errorf("back edges = %+v, want one edge to README.md", back)
	}
	if broken := g.BrokenLinks(); len(broken) != 0 {
		t.Errorf("broken = %+v, want none", broken)
	}
}
```

Apply the same `rules.ResolveMarkdownTargets(records, root)` insertion to any other test in this file that declares markdown styles. Then add the mixed-style test:

```go
func TestE2E_LinkStyles_GraphMixedStyles(t *testing.T) {
	root := setupProject(t, map[string]string{
		".stem":     "version: 2\nlinks:\n  styles: [wikilink, markdown]\n",
		"README.md": "# Root\n\n[[a.md]]\n[b](docs/b.md)\n",
		"a.md":      "# A\n",
		"docs/b.md": "# B\n",
	})
	ctx := context.Background()
	records, err := index.Scan(ctx, root, extract.NewRegistry())
	if err != nil {
		t.Fatal(err)
	}
	rules.FilterLinksByStyles(records, root)
	rules.ResolveMarkdownTargets(records, root)
	g := graph.Build(ctx, records)

	if len(g.Edges["README.md"]) != 2 {
		t.Fatalf("README edges = %+v, want wikilink + markdown edges", g.Edges["README.md"])
	}
	if broken := g.BrokenLinks(); len(broken) != 0 {
		t.Errorf("broken = %+v, want none", broken)
	}
}
```

- [ ] **Step 5: Run tests to verify everything passes**

Run: `go test ./internal/graph/ ./internal/e2e/ -v -run 'TestBuild_|TestE2E_LinkStyles'`
Expected: PASS.

- [ ] **Step 6: Full check and commit**

```bash
just check && just test
git add internal/graph/graph.go internal/graph/graph_test.go internal/e2e/link_styles_e2e_test.go
git commit -m "fix(graph): honor pre-resolved markdown targets without basename fallback"
```

---

### Task 4: Wire `ResolveMarkdownTargets` into `runGraph`

**Files:**
- Modify: `cmd/rootline/graph.go` (`runGraph`, right after the `rules.FilterLinksByStyles` call at ~line 76)
- Test: `cmd/rootline/graph_test.go` (extend)

**Interfaces:**
- Consumes: `rules.ResolveMarkdownTargets(records, absRoot)` (Task 2), `rules.FilterLinksByStyles` (existing).
- Produces: full command behavior — `graph`/`graph --check` on markdown-styles repos.

- [ ] **Step 1: Write the failing command-level tests**

Add to `cmd/rootline/graph_test.go` (`os` is already imported):

```go
func TestGraphCheck_MarkdownStylesClean(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nlinks:\n  styles: [markdown]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "README.md"), []byte("---\n---\n# Root\n[overview](docs/overview.md)\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "docs", "overview.md"), []byte("---\n---\n# Overview\n[back](../README.md)\n"), 0644)

	out, err := runCmd(t, "graph", "--check", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "No cycles or broken links") {
		t.Errorf("expected clean check, got: %s", out)
	}
}

func TestGraphJSON_MarkdownStylesProducesEdges(t *testing.T) {
	dir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(dir, "docs"), 0o755); err != nil {
		t.Fatal(err)
	}
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nlinks:\n  styles: [markdown]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "README.md"), []byte("---\n---\n# Root\n[overview](docs/overview.md)\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "docs", "overview.md"), []byte("---\n---\n# Overview\n[back](../README.md)\n"), 0644)

	out, err := runCmd(t, "graph", dir)
	if err != nil {
		t.Fatalf("unexpected error: %v\noutput: %s", err, out)
	}
	var result map[string]any
	if err := json.Unmarshal([]byte(out), &result); err != nil {
		t.Fatalf("invalid JSON: %v\nraw: %s", err, out)
	}
	edges := result["edges"].([]any)
	if len(edges) != 2 {
		t.Errorf("edges = %d, want 2", len(edges))
	}
}

func TestGraphCheck_MarkdownBrokenLinkCanary(t *testing.T) {
	dir := t.TempDir()
	mustWriteFile(t, filepath.Join(dir, ".stem"), []byte("version: 2\nlinks:\n  styles: [markdown]\n"), 0644)
	mustWriteFile(t, filepath.Join(dir, "a.md"), []byte("---\n---\n# A\n[x](no-existe.md)\n"), 0644)

	out, err := runCmd(t, "graph", "--check", dir)
	if err != ErrValidationFailed {
		t.Fatalf("expected ErrValidationFailed, got: %v\noutput: %s", err, out)
	}
	if !strings.Contains(out, "no-existe.md") {
		t.Errorf("expected broken link named in output, got: %s", out)
	}
}
```

- [ ] **Step 2: Run tests to verify the clean/edges tests fail**

Run: `go test ./cmd/rootline/ -run 'TestGraphCheck_MarkdownStylesClean|TestGraphJSON_MarkdownStylesProducesEdges|TestGraphCheck_MarkdownBrokenLinkCanary' -v`
Expected: `TestGraphCheck_MarkdownStylesClean` FAILS (`[back](../README.md)` stays verbatim → broken link, check exits non-zero). `TestGraphJSON_MarkdownStylesProducesEdges` may pass on edge count but the suite must be red overall via the clean test. `TestGraphCheck_MarkdownBrokenLinkCanary` PASSES already (canary regression guard).

- [ ] **Step 3: Wire the resolver**

In `runGraph` (`cmd/rootline/graph.go`), immediately after `rules.FilterLinksByStyles(records, absRoot)`:

```go
	rules.FilterLinksByStyles(records, absRoot)
	rules.ResolveMarkdownTargets(records, absRoot)
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./cmd/rootline/ -run 'TestGraph|TestFilterLinksBySchema' -v`
Expected: PASS, including all pre-existing graph tests (wikilink repos unaffected — the resolver only touches `Style == markdown` links, and undeclared styles default to `[wikilink]` so markdown links never reach it).

- [ ] **Step 5: Full check and commit**

```bash
just check && just test
git add cmd/rootline/graph.go cmd/rootline/graph_test.go
git commit -m "fix(graph): resolve markdown link targets in the graph pipeline"
```

---

### Task 5: Acceptance run, docs, push, release verification

**Files:**
- Modify: `docs/superpowers/specs/2026-07-10-graph-markdown-edges-design.md` (status + backlog note)
- Modify: `.claude/skills/rootline/` reference docs if they state graph/link behavior that changed (check `rg -l 'graph' .claude/skills/rootline/`)

**Interfaces:**
- Consumes: the built CLI (`go run ./cmd/rootline` — local `dev` builds skip autoupdate network calls).
- Produces: verified acceptance evidence, pushed release.

- [ ] **Step 1: Run acceptance against the dsprima wiki**

```bash
go run ./cmd/rootline graph -o json /Users/Shared/dsprima/docs | jq '.edges | length'
```
Expected: > 0 (dozens).

```bash
go run ./cmd/rootline graph --check /Users/Shared/dsprima/docs; echo "graph exit: $?"
go run ./cmd/rootline validate --all /Users/Shared/dsprima/docs; echo "validate exit: $?"
```
Expected: broken links reported by `graph --check` are consistent with `validate`'s link errors (same targets flagged; graph may additionally report wikilink-style issues only if wikilink styles are declared there).

- [ ] **Step 2: Canary check on dsprima**

```bash
printf '# Canary\n\n[x](no-existe.md)\n' > /Users/Shared/dsprima/docs/canary.md
go run ./cmd/rootline graph --check /Users/Shared/dsprima/docs; echo "exit: $?"
rm /Users/Shared/dsprima/docs/canary.md
```
Expected: non-zero exit and output naming `no-existe.md`. The canary file is temporary — verify it is deleted afterwards (`eza /Users/Shared/dsprima/docs | rg canary` returns nothing).

- [ ] **Step 3: Update spec status and backlog note**

In the spec, set `**Status**: Implemented` and record under Definition of Done that `/opt/factory/docs/backlog/` does not exist on this machine, so the outcome is recorded in this spec (per the spec's own DoD provision). Update skill docs only if they describe the old zero-edge behavior.

```bash
git add docs/superpowers/specs/2026-07-10-graph-markdown-edges-design.md
git commit -m "docs(spec): mark graph markdown edges design implemented"
```

- [ ] **Step 4: Push (triggers coverage gate + auto-release)**

```bash
just check && just test
git push
```
Expected: pre-push hook runs `just coverage-check` (85% floors) and passes; push to master triggers the crossbeam auto-release.

- [ ] **Step 5: Verify the released binary**

```bash
gh run list --limit 3
gh run watch <release-run-id>
which rootline && rootline --version
```
Expected: release workflow green; installed `rootline --version` shows the new version (autoupdate applies the staged update on next invocation — run `rootline --version` twice if the first still shows the old version). Re-run the Step 1 acceptance command with the installed binary:

```bash
rootline graph -o json /Users/Shared/dsprima/docs | jq '.edges | length'
```
Expected: > 0.
