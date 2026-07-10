# Multi-Style Links Implementation Plan

> **For agentic workers:** REQUIRED SUB-SKILL: Use superpowers:subagent-driven-development (recommended) or superpowers:executing-plans to implement this plan task-by-task. Steps use checkbox (`- [ ]`) syntax for tracking.

**Goal:** Extract markdown links (`[text](target)`) alongside `[[wiki-links]]`, govern which styles are validated/graphed via `.stem`, and add ADO code-wiki checks (resolve, anchors, encoding) — so `rootline validate` can validate `/Users/Shared/dsprima/docs`.

**Architecture:** Extraction always parses both styles and tags each `Link` with `Style` (+ separate `Anchor` field). `.stem` gains `links.styles` (default `[wikilink]` = full backcompat) and `links.checks` (`resolve`/`anchors`/`encoding`). Validation and graph filter links by effective styles per record. New filesystem-backed checks live in a new `rules.CheckLinks` entry point (called from the CLI, which knows absolute paths) so `rules.Validate` stays pure.

**Tech Stack:** Go 1.25+, yaml.v3, goldmark (existing dep), picokit/fuzzy (existing dep), standard `testing`.

**Spec:** `docs/superpowers/specs/2026-07-10-link-styles-design.md`

## Global Constraints

- All commits conventional (`feat(scope): ...`), no AI attribution trailers.
- `just check` and `just test` must pass after every task; per-package coverage floor is 85% (`.coverage-floors.toml`).
- Backward compat is a hard requirement: with no `links.styles` in any `.stem`, validation and graph behavior must be byte-identical to today (markdown links appear ONLY as new data in `Record.Links` JSON).
- Code, comments, identifiers, test names: English.
- Deviation from spec (approved rationale): markdown links get `Type: "reference"`, not `""` — untyped wikilinks already default to `"reference"` (`internal/extract/links.go:49`), and reusing it makes existing `links.allowed`/`links.rules` apply uniformly.

---

### Task 1: Extraction — `Style`/`Anchor` fields + markdown link parsing

**Files:**
- Modify: `internal/extract/links.go`
- Modify: `internal/extract/links_ast.go`
- Test: `internal/extract/links_test.go` (append), `internal/extract/links_ast_test.go` (append)

**Interfaces:**
- Consumes: existing `Link` struct, `wikilinkRe`, `removeInlineCode` (links.go).
- Produces (later tasks rely on these exact names):
  - `extract.StyleWikilink = "wikilink"`, `extract.StyleMarkdown = "markdown"` (exported consts)
  - `Link.Style string` (json `style,omitempty`), `Link.Anchor string` (json `anchor,omitempty`)
  - Every wikilink (body, AST, frontmatter) gets `Style: StyleWikilink`; markdown links get `Style: StyleMarkdown`, `Type: "reference"`, `Source: "body"`, path in `Target`, fragment in `Anchor`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/extract/links_test.go`:

```go
func TestParseLinks_MarkdownRelative(t *testing.T) {
	links := ParseLinks("Ver [la guia](../guides/setup.md) para mas.")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d: %+v", len(links), links)
	}
	l := links[0]
	if l.Style != StyleMarkdown {
		t.Errorf("Style = %q, want %q", l.Style, StyleMarkdown)
	}
	if l.Target != "../guides/setup.md" {
		t.Errorf("Target = %q, want %q", l.Target, "../guides/setup.md")
	}
	if l.Type != "reference" {
		t.Errorf("Type = %q, want %q", l.Type, "reference")
	}
	if l.Line != 1 {
		t.Errorf("Line = %d, want 1", l.Line)
	}
	if l.Source != "body" {
		t.Errorf("Source = %q, want %q", l.Source, "body")
	}
}

func TestParseLinks_MarkdownAnchorSplit(t *testing.T) {
	links := ParseLinks("[zonas](../project-master.md#6-zonas-inciertas-black-boxes)")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d", len(links))
	}
	if links[0].Target != "../project-master.md" {
		t.Errorf("Target = %q, want path without fragment", links[0].Target)
	}
	if links[0].Anchor != "6-zonas-inciertas-black-boxes" {
		t.Errorf("Anchor = %q", links[0].Anchor)
	}
}

func TestParseLinks_MarkdownSkipsNonPathTargets(t *testing.T) {
	body := "![img](diagram.png) [ext](https://example.com) [mail](mailto:a@b.c) [frag](#local) [dir](./)"
	links := ParseLinks(body)
	// Only [dir](./) survives: images, external schemes, and pure anchors are skipped.
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d: %+v", len(links), links)
	}
	if links[0].Target != "./" {
		t.Errorf("Target = %q, want %q", links[0].Target, "./")
	}
}

func TestParseLinks_MarkdownTitleAndAngleBrackets(t *testing.T) {
	links := ParseLinks(`[a](foo.md "Some Title") [b](<bar.md>)`)
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d: %+v", len(links), links)
	}
	if links[0].Target != "foo.md" {
		t.Errorf("Target = %q, want %q", links[0].Target, "foo.md")
	}
	if links[1].Target != "bar.md" {
		t.Errorf("Target = %q, want %q", links[1].Target, "bar.md")
	}
}

func TestParseLinks_MarkdownRawSpacePreserved(t *testing.T) {
	// A raw space with no quoted title is NOT a title separator — the target
	// must survive intact so the encoding check can flag it later.
	links := ParseLinks("[bad](my file.md)")
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d: %+v", len(links), links)
	}
	if links[0].Target != "my file.md" {
		t.Errorf("Target = %q, want %q", links[0].Target, "my file.md")
	}
}

func TestParseLinks_MarkdownInsideCodeIgnored(t *testing.T) {
	body := "```\n[x](a.md)\n```\ny `[y](b.md)` z"
	if links := ParseLinks(body); len(links) != 0 {
		t.Fatalf("expected 0 links, got %d: %+v", len(links), links)
	}
}

func TestParseLinks_WikilinkStyleTagged(t *testing.T) {
	links := ParseLinks("[[T001-foo]] and [[blocks:T002-bar]]")
	if len(links) != 2 {
		t.Fatalf("expected 2 links, got %d", len(links))
	}
	for _, l := range links {
		if l.Style != StyleWikilink {
			t.Errorf("Style = %q, want %q", l.Style, StyleWikilink)
		}
	}
}

func TestParseFrontmatterLinks_StyleTagged(t *testing.T) {
	links := ParseFrontmatterLinks(map[string]any{"dep": "[[T001-foo]]"})
	if len(links) != 1 || links[0].Style != StyleWikilink {
		t.Fatalf("expected 1 wikilink-style link, got %+v", links)
	}
}
```

Append to `internal/extract/links_ast_test.go` (mirror the parity contract; reuse the file's existing helper for parsing markdown into an AST — check its top for a `parseAST`-style helper and use the same one):

```go
func TestParseLinksAST_MarkdownLink(t *testing.T) {
	source := []byte("Ver [la guia](../guides/setup.md#intro).")
	node := parseTestAST(t, source) // use the existing AST-construction helper in this file
	links := ParseLinksAST(node, source)
	if len(links) != 1 {
		t.Fatalf("expected 1 link, got %d: %+v", len(links), links)
	}
	l := links[0]
	if l.Style != StyleMarkdown || l.Target != "../guides/setup.md" || l.Anchor != "intro" {
		t.Errorf("got %+v", l)
	}
}
```

If `links_ast_test.go` has no reusable AST helper, add one at the top of the test file:

```go
func parseTestAST(t *testing.T, source []byte) ast.Node {
	t.Helper()
	return goldmark.DefaultParser().Parse(gmtext.NewReader(source))
}
```

with imports `"github.com/yuin/goldmark"`, `"github.com/yuin/goldmark/ast"`, `gmtext "github.com/yuin/goldmark/text"`.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/extract/ -run 'TestParseLinks_Markdown|TestParseLinks_Wikilink|TestParseFrontmatterLinks_Style|TestParseLinksAST_Markdown' -v`
Expected: compile error `undefined: StyleMarkdown` (that counts as the failing state).

- [ ] **Step 3: Implement**

In `internal/extract/links.go`:

1. Extend the struct and add constants + regex:

```go
// Link represents a link extracted from document text.
type Link struct {
	Target string `json:"target"`
	Type   string `json:"type"`
	Line   int    `json:"line"`
	Source string `json:"source,omitempty"` // "body" or "frontmatter:<fieldname>"
	Style  string `json:"style,omitempty"`  // StyleWikilink or StyleMarkdown
	Anchor string `json:"anchor,omitempty"` // fragment part of markdown targets, without '#'
}

// Link styles produced by extraction.
const (
	StyleWikilink = "wikilink"
	StyleMarkdown = "markdown"
)

var wikilinkRe = regexp.MustCompile(`\[\[([^\]]+)\]\]`)

// markdownLinkRe captures an optional image marker and the destination of
// inline markdown links. Wikilinks don't match: their brackets are doubled
// and have no (...) destination.
var markdownLinkRe = regexp.MustCompile(`(!?)\[[^\]]*\]\(([^)]+)\)`)
```

2. Add the destination parser:

```go
// parseMarkdownDestination converts an inline-link destination into a Link.
// Returns false for destinations that aren't local paths: external schemes,
// mailto, and pure fragments. Quoted titles (`foo.md "Title"`) and angle
// brackets (`<foo.md>`) are stripped; a raw space with no quoted title after
// it is kept so the encoding check can flag it.
func parseMarkdownDestination(dest string) (Link, bool) {
	dest = strings.TrimSpace(dest)
	if i := strings.IndexAny(dest, " \t"); i >= 0 {
		rest := strings.TrimSpace(dest[i+1:])
		if strings.HasPrefix(rest, `"`) || strings.HasPrefix(rest, "'") || strings.HasPrefix(rest, "(") {
			dest = dest[:i]
		}
	}
	dest = strings.TrimPrefix(dest, "<")
	dest = strings.TrimSuffix(dest, ">")
	if dest == "" || strings.HasPrefix(dest, "#") {
		return Link{}, false
	}
	if strings.Contains(dest, "://") || strings.HasPrefix(dest, "mailto:") {
		return Link{}, false
	}
	link := Link{Type: "reference", Style: StyleMarkdown, Target: dest}
	if i := strings.Index(dest, "#"); i >= 0 {
		link.Target = dest[:i]
		link.Anchor = dest[i+1:]
		if link.Target == "" {
			return Link{}, false
		}
	}
	return link, true
}
```

3. In `ParseLinks`, tag wikilinks and add markdown matching inside the per-line loop (after the existing wikilink loop, still on `cleaned`):

```go
		for _, match := range wikilinkRe.FindAllStringSubmatch(cleaned, -1) {
			inner := match[1]
			link := Link{Line: lineNum, Source: "body", Style: StyleWikilink}
			// ... existing type/target split unchanged ...
		}

		for _, match := range markdownLinkRe.FindAllStringSubmatch(cleaned, -1) {
			if match[1] == "!" {
				continue // image
			}
			link, ok := parseMarkdownDestination(match[2])
			if !ok {
				continue
			}
			link.Line = lineNum
			link.Source = "body"
			links = append(links, link)
		}
```

4. In `parseWikilinksFromString`, set `Style: StyleWikilink` on the constructed `Link` (frontmatter links stay wikilink-only).

In `internal/extract/links_ast.go`, `extractLinksFromLines`: set `Style: StyleWikilink` on wikilinks and append the same markdown-matching loop (with `link.Line = lineNum`, `link.Source = "body"`) after the wikilink loop.

- [ ] **Step 4: Run tests to verify they pass, then the full package**

Run: `go test ./internal/extract/ -v`
Expected: PASS (existing tests construct `Link{...}` literals without `Style` — they must still pass; if any assert full-struct equality on parsed output, update those expectations to include `Style: StyleWikilink`).

- [ ] **Step 5: Run the whole suite to catch downstream consumers**

Run: `just test`
Expected: 0 failures. Watch specifically `internal/infer` (traceability/formal-dependency detectors iterate `Record.Links`) and `internal/graph`. If an infer test breaks because markdown links now appear in fixtures, the fix is in the detector's filter (only consider `Style == extract.StyleWikilink` where the detector is wikilink-semantic), NOT in extraction.

- [ ] **Step 6: Commit**

```bash
git add internal/extract/
git commit -m "feat(extract): parse markdown links alongside wikilinks with style tagging"
```

---

### Task 2: `.stem` schema — `links.styles` + `links.checks`

**Files:**
- Modify: `internal/rules/rules.go` (LinkSchema at :53, UnmarshalYAML at :67, IsEmpty at :99)
- Modify: `internal/rules/merge.go` (mergeLinkSchema at :172)
- Test: `internal/rules/rules_test.go`, `internal/rules/merge_test.go` (append)

**Interfaces:**
- Consumes: `extract.StyleWikilink` (Task 1).
- Produces:
  - `LinkSchema.Styles []string` (json `styles,omitempty`), `LinkSchema.Checks *LinkChecks` (json `checks,omitempty`)
  - `type LinkChecks struct { Resolve, Anchors, Encoding bool }` (yaml/json tags `resolve`, `anchors`, `encoding`)
  - `func (ls LinkSchema) EffectiveStyles() []string` — returns `Styles`, or `[]string{extract.StyleWikilink}` when empty.

- [ ] **Step 1: Write the failing tests**

Append to `internal/rules/rules_test.go`:

```go
func TestLinkSchema_UnmarshalStylesAndChecks(t *testing.T) {
	src := `
links:
  styles: [markdown]
  checks:
    resolve: true
    anchors: true
    encoding: true
  rules:
    reference:
      target: '.*\.md$'
`
	// Reuse the file's existing stem-parsing helper if present; otherwise:
	var stem StemFile
	if err := yaml.Unmarshal([]byte(src), &stem); err != nil {
		t.Fatalf("unmarshal: %v", err)
	}
	ls := stem.Links
	if len(ls.Styles) != 1 || ls.Styles[0] != "markdown" {
		t.Errorf("Styles = %v, want [markdown]", ls.Styles)
	}
	if ls.Checks == nil || !ls.Checks.Resolve || !ls.Checks.Anchors || !ls.Checks.Encoding {
		t.Errorf("Checks = %+v, want all true", ls.Checks)
	}
	if ls.IsEmpty() {
		t.Error("IsEmpty() = true with styles+checks set")
	}
}

func TestLinkSchema_EffectiveStylesDefault(t *testing.T) {
	var ls LinkSchema
	got := ls.EffectiveStyles()
	if len(got) != 1 || got[0] != extract.StyleWikilink {
		t.Errorf("EffectiveStyles() = %v, want [wikilink]", got)
	}
	ls.Styles = []string{"markdown", "wikilink"}
	if got := ls.EffectiveStyles(); len(got) != 2 {
		t.Errorf("EffectiveStyles() = %v, want declared list", got)
	}
}
```

NOTE: the `rules` key nesting above must match how `LinkSchema.UnmarshalYAML` reads keys today — existing stems put rule types directly under `links:` (e.g. `links: {blocked_by: {target: ...}}`), NOT under a `rules:` sub-key. Verify against an existing test in `rules_test.go` and write the YAML accordingly (i.e. `reference: {target: ...}` directly under `links:`). The `styles`/`checks`/`allowed` keys are the reserved non-rule keys.

Append to `internal/rules/merge_test.go`:

```go
func TestMergeLinkSchema_StylesAndChecksChildReplace(t *testing.T) {
	parent := LinkSchema{Styles: []string{"wikilink"}, Checks: &LinkChecks{Resolve: true}}
	child := LinkSchema{Styles: []string{"markdown"}}
	got := mergeLinkSchema(parent, child)
	if len(got.Styles) != 1 || got.Styles[0] != "markdown" {
		t.Errorf("Styles = %v, want child's [markdown]", got.Styles)
	}
	if got.Checks == nil || !got.Checks.Resolve {
		t.Errorf("Checks = %+v, want inherited from parent", got.Checks)
	}
	child2 := LinkSchema{Checks: &LinkChecks{Encoding: true}}
	got2 := mergeLinkSchema(parent, child2)
	if got2.Checks == nil || got2.Checks.Resolve || !got2.Checks.Encoding {
		t.Errorf("Checks = %+v, want child replacement", got2.Checks)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/rules/ -run 'TestLinkSchema_Unmarshal|TestLinkSchema_Effective|TestMergeLinkSchema_Styles' -v`
Expected: compile error `undefined: LinkChecks` / `ls.Styles undefined`.

- [ ] **Step 3: Implement**

In `internal/rules/rules.go`:

```go
type LinkSchema struct {
	Allowed []string            `json:"allowed,omitempty"`
	Styles  []string            `json:"styles,omitempty"`
	Checks  *LinkChecks         `json:"checks,omitempty"`
	Rules   map[string]LinkRule `json:"rules,omitempty"`
}

// LinkChecks enables filesystem-backed link checks (ADO code-wiki conventions).
type LinkChecks struct {
	Resolve  bool `yaml:"resolve" json:"resolve,omitempty"`
	Anchors  bool `yaml:"anchors" json:"anchors,omitempty"`
	Encoding bool `yaml:"encoding" json:"encoding,omitempty"`
}

// EffectiveStyles returns the link styles governed by this schema.
// An empty declaration defaults to wikilink-only for backward compatibility.
func (ls LinkSchema) EffectiveStyles() []string {
	if len(ls.Styles) > 0 {
		return ls.Styles
	}
	return []string{extract.StyleWikilink}
}
```

In `UnmarshalYAML`, add branches BEFORE the fallthrough-to-LinkRule decode (without them, `styles:`/`checks:` would decode as a `LinkRule` and be silently misread):

```go
		if key == "styles" {
			var styles []string
			if err := val.Decode(&styles); err != nil {
				return fmt.Errorf("links.styles: %w", err)
			}
			ls.Styles = styles
			continue
		}
		if key == "checks" {
			var checks LinkChecks
			if err := val.Decode(&checks); err != nil {
				return fmt.Errorf("links.checks: %w", err)
			}
			ls.Checks = &checks
			continue
		}
```

Update `IsEmpty`:

```go
func (ls LinkSchema) IsEmpty() bool {
	return len(ls.Allowed) == 0 && len(ls.Rules) == 0 && len(ls.Styles) == 0 && ls.Checks == nil
}
```

In `internal/rules/merge.go`, extend `mergeLinkSchema` (after the Allowed block):

```go
	// Styles: child replaces (array semantics).
	if child.Styles != nil {
		result.Styles = child.Styles
	} else {
		result.Styles = parent.Styles
	}

	// Checks: child replaces when declared (struct semantics).
	if child.Checks != nil {
		result.Checks = child.Checks
	} else {
		result.Checks = parent.Checks
	}
```

`rules.go` already imports `extract` (used at validate.go; confirm the import exists in rules.go itself — add `"github.com/pablontiv/rootline/internal/extract"` if missing).

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/rules/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/rules/rules.go internal/rules/merge.go internal/rules/rules_test.go internal/rules/merge_test.go
git commit -m "feat(rules): add links.styles and links.checks to .stem schema"
```

---

### Task 3: Style filtering in `validateLinks`

**Files:**
- Modify: `internal/rules/validate.go` (validateLinks at :347)
- Test: `internal/rules/validate_test.go` (append)

**Interfaces:**
- Consumes: `LinkSchema.EffectiveStyles()` (Task 2), `extract.StyleMarkdown`/`StyleWikilink` (Task 1).
- Produces: `linkStyle(l extract.Link) string` helper (unexported; empty `Style` → wikilink, so hand-built `extract.Link{}` literals in old tests keep behaving).

- [ ] **Step 1: Write the failing tests**

Append to `internal/rules/validate_test.go`:

```go
func TestValidateLinks_MarkdownIgnoredByDefault(t *testing.T) {
	schema := LinkSchema{Allowed: []string{"blocks"}}
	links := []extract.Link{{Target: "../foo.md", Type: "reference", Style: extract.StyleMarkdown}}
	if errs := validateLinks(links, schema, "test.stem"); len(errs) != 0 {
		t.Fatalf("markdown link validated under default styles: %+v", errs)
	}
}

func TestValidateLinks_MarkdownValidatedWhenDeclared(t *testing.T) {
	schema := LinkSchema{
		Styles: []string{extract.StyleMarkdown},
		Rules:  map[string]LinkRule{"reference": {Target: `.*\.md$`}},
	}
	bad := []extract.Link{{Target: "../foo.txt", Type: "reference", Style: extract.StyleMarkdown}}
	if errs := validateLinks(bad, schema, "test.stem"); len(errs) != 1 {
		t.Fatalf("expected 1 link_target error, got %+v", errs)
	}
	// And wikilinks are now excluded (styles replaced the default).
	wiki := []extract.Link{{Target: "nope", Type: "reference", Style: extract.StyleWikilink}}
	if errs := validateLinks(wiki, schema, "test.stem"); len(errs) != 0 {
		t.Fatalf("wikilink validated despite styles=[markdown]: %+v", errs)
	}
}

func TestValidateLinks_EmptyStyleDefaultsToWikilink(t *testing.T) {
	schema := LinkSchema{Allowed: []string{"blocks"}}
	links := []extract.Link{{Target: "x", Type: "reference"}} // no Style set (legacy shape)
	if errs := validateLinks(links, schema, "test.stem"); len(errs) != 1 {
		t.Fatalf("legacy style-less wikilink not validated: %+v", errs)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/rules/ -run TestValidateLinks_Markdown -v`
Expected: FAIL — `TestValidateLinks_MarkdownIgnoredByDefault` gets a `link_type` error because filtering doesn't exist yet.

- [ ] **Step 3: Implement**

In `internal/rules/validate.go`, top of `validateLinks` after the `IsEmpty` guard:

```go
	styles := make(map[string]bool)
	for _, s := range schema.EffectiveStyles() {
		styles[s] = true
	}

	var errs []ValidationError
	for _, link := range links {
		if !styles[linkStyle(link)] {
			continue
		}
		// ... existing allowed/target checks unchanged ...
	}
```

Add next to `stringSliceContains`:

```go
// linkStyle returns the link's style, defaulting legacy style-less links to wikilink.
func linkStyle(l extract.Link) string {
	if l.Style == "" {
		return extract.StyleWikilink
	}
	return l.Style
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/rules/ -v`
Expected: PASS (all pre-existing `TestValidateLinks_*` still green — their links have empty `Style` → wikilink → still validated).

- [ ] **Step 5: Commit**

```bash
git add internal/rules/validate.go internal/rules/validate_test.go
git commit -m "feat(rules): filter link validation by effective link styles"
```

---

### Task 4: `CheckLinks` — encoding + case-sensitive resolve

**Files:**
- Create: `internal/rules/link_checks.go`
- Test: create `internal/rules/link_checks_test.go`

**Interfaces:**
- Consumes: `LinkSchema.Checks`, `EffectiveStyles()`, `linkStyle()`, `extract.Link`, `fuzzy.MatchN` (import `github.com/pablontiv/picokit/fuzzy`, same as graph).
- Produces (Task 5/6 rely on):
  - `func CheckLinks(links []extract.Link, schema LinkSchema, sourceAbsPath string, cache *HeadingCache) []ValidationError` — runs checks gated by `schema.Checks`; `cache` may be nil (anchors check then parses without caching; Task 5 defines it, this task declares the parameter but only threads it through).
  - Error rules emitted: `link_encoding`, `link_resolve` (this task); `link_anchor` (Task 5).
  - Targets starting with `/` are skipped by resolve/anchors (root-relative ADO form — out of scope, documented).

To keep this task compilable before Task 5, define the placeholder in `link_checks.go` now: `type HeadingCache struct{ slugs map[string][]string }` with `func NewHeadingCache() *HeadingCache { return &HeadingCache{slugs: make(map[string][]string)} }` — Task 5 fills in its behavior.

- [ ] **Step 1: Write the failing tests**

Create `internal/rules/link_checks_test.go`:

```go
package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	if err := os.MkdirAll(filepath.Dir(path), 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(path, []byte(content), 0o644); err != nil {
		t.Fatal(err)
	}
}

func mdChecksSchema(c LinkChecks) LinkSchema {
	return LinkSchema{Styles: []string{extract.StyleMarkdown}, Checks: &c}
}

func mdLink(target string) extract.Link {
	return extract.Link{Target: target, Type: "reference", Style: extract.StyleMarkdown, Line: 1}
}

func TestCheckLinks_NilChecksIsNoop(t *testing.T) {
	schema := LinkSchema{Styles: []string{extract.StyleMarkdown}}
	errs := CheckLinks([]extract.Link{mdLink("missing.md")}, schema, "/nonexistent/src.md", nil)
	if len(errs) != 0 {
		t.Fatalf("expected no errors without checks, got %+v", errs)
	}
}

func TestCheckLinks_EncodingRejectsRawSpaces(t *testing.T) {
	schema := mdChecksSchema(LinkChecks{Encoding: true})
	errs := CheckLinks([]extract.Link{mdLink("my file.md")}, schema, "/tmp/src.md", nil)
	if len(errs) != 1 || errs[0].Rule != "link_encoding" {
		t.Fatalf("expected 1 link_encoding error, got %+v", errs)
	}
}

func TestCheckLinks_ResolveExisting(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	errs := CheckLinks([]extract.Link{mdLink("guides/setup.md")}, schema, filepath.Join(dir, "src.md"), nil)
	if len(errs) != 0 {
		t.Fatalf("expected resolve to pass, got %+v", errs)
	}
}

func TestCheckLinks_ResolveBrokenWithSuggestion(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	errs := CheckLinks([]extract.Link{mdLink("guides/setpu.md")}, schema, filepath.Join(dir, "src.md"), nil)
	if len(errs) != 1 || errs[0].Rule != "link_resolve" {
		t.Fatalf("expected 1 link_resolve error, got %+v", errs)
	}
	if errs[0].Suggestion != "setup.md" {
		t.Errorf("Suggestion = %q, want %q", errs[0].Suggestion, "setup.md")
	}
}

func TestCheckLinks_ResolveCaseMismatchIsBroken(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "Setup.md"), "# Setup\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	// APFS would resolve this case-insensitively; ADO/git will not.
	errs := CheckLinks([]extract.Link{mdLink("guides/setup.md")}, schema, filepath.Join(dir, "src.md"), nil)
	if len(errs) != 1 || errs[0].Rule != "link_resolve" {
		t.Fatalf("expected case mismatch to be broken, got %+v", errs)
	}
}

func TestCheckLinks_ResolveDirectoryTargetNeedsReadme(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "guides", "README.md"), "# Guides\n")
	writeFile(t, filepath.Join(dir, "empty", ".keep"), "")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	src := filepath.Join(dir, "src.md")
	if errs := CheckLinks([]extract.Link{mdLink("guides/")}, schema, src, nil); len(errs) != 0 {
		t.Fatalf("dir with README should resolve, got %+v", errs)
	}
	if errs := CheckLinks([]extract.Link{mdLink("empty/")}, schema, src, nil); len(errs) != 1 {
		t.Fatalf("dir without README should be broken, got %+v", errs)
	}
}

func TestCheckLinks_ResolveDecodesPercent20(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "my page.md"), "# P\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true})
	errs := CheckLinks([]extract.Link{mdLink("my%20page.md")}, schema, filepath.Join(dir, "src.md"), nil)
	if len(errs) != 0 {
		t.Fatalf("%%20 target should resolve, got %+v", errs)
	}
}

func TestCheckLinks_SkipsAbsoluteAndFilteredStyles(t *testing.T) {
	schema := mdChecksSchema(LinkChecks{Resolve: true, Encoding: true})
	links := []extract.Link{
		mdLink("/Root/Page.md"), // absolute: skipped
		{Target: "no such file", Type: "reference", Style: extract.StyleWikilink, Line: 1}, // filtered style
	}
	if errs := CheckLinks(links, schema, "/tmp/src.md", nil); len(errs) != 0 {
		t.Fatalf("expected no errors, got %+v", errs)
	}
}
```

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/rules/ -run TestCheckLinks -v`
Expected: compile error `undefined: CheckLinks`.

- [ ] **Step 3: Implement**

Create `internal/rules/link_checks.go`:

```go
package rules

import (
	"fmt"
	"net/url"
	"os"
	"path/filepath"
	"strings"

	"github.com/pablontiv/picokit/fuzzy"
	"github.com/pablontiv/rootline/internal/extract"
)

// HeadingCache caches heading slugs per target file within a validation run.
// Behavior is implemented alongside the anchors check.
type HeadingCache struct {
	slugs map[string][]string
}

// NewHeadingCache creates an empty heading cache.
func NewHeadingCache() *HeadingCache {
	return &HeadingCache{slugs: make(map[string][]string)}
}

// CheckLinks runs filesystem-backed link checks (links.checks in .stem)
// against a record's links. sourceAbsPath is the absolute path of the record
// file; relative targets resolve against its directory. Links whose style is
// not in the schema's effective styles are skipped, as are absolute targets
// (root-relative ADO form is out of scope).
func CheckLinks(links []extract.Link, schema LinkSchema, sourceAbsPath string, cache *HeadingCache) []ValidationError {
	if schema.Checks == nil {
		return nil
	}

	styles := make(map[string]bool)
	for _, s := range schema.EffectiveStyles() {
		styles[s] = true
	}

	var errs []ValidationError
	for _, link := range links {
		if !styles[linkStyle(link)] {
			continue
		}

		if schema.Checks.Encoding && strings.Contains(link.Target, " ") {
			errs = append(errs, ValidationError{
				Rule:     "link_encoding",
				Field:    "links",
				Message:  fmt.Sprintf("link target %q contains unencoded spaces (use %%20)", link.Target),
				Source:   "links.checks",
				Severity: "error",
			})
		}

		if strings.HasPrefix(link.Target, "/") {
			continue
		}

		if schema.Checks.Resolve || schema.Checks.Anchors {
			resolved, suggestion, ok := resolveCaseSensitive(filepath.Dir(sourceAbsPath), link.Target)
			if !ok {
				if schema.Checks.Resolve {
					msg := fmt.Sprintf("link target %q does not resolve to an existing file (case-sensitive)", link.Target)
					if suggestion != "" {
						msg += fmt.Sprintf(" (did you mean %q?)", suggestion)
					}
					errs = append(errs, ValidationError{
						Rule:       "link_resolve",
						Field:      "links",
						Message:    msg,
						Source:     "links.checks",
						Severity:   "error",
						Suggestion: suggestion,
					})
				}
				continue
			}
			if schema.Checks.Anchors && link.Anchor != "" {
				errs = append(errs, checkAnchor(link, resolved, cache)...)
			}
		}
	}
	return errs
}

// checkAnchor is implemented with the anchors check (see link_checks anchors task).
func checkAnchor(_ extract.Link, _ string, _ *HeadingCache) []ValidationError {
	return nil
}

// resolveCaseSensitive resolves a relative link target against baseDir,
// requiring exact-case matches for every target path component (APFS is
// case-insensitive; ADO and git are not). Directory targets resolve to their
// README.md. Returns the resolved absolute path, a fuzzy suggestion for the
// first unmatched component (may be empty), and whether resolution succeeded.
func resolveCaseSensitive(baseDir, target string) (string, string, bool) {
	decoded, err := url.PathUnescape(target)
	if err != nil {
		decoded = target
	}

	cur := baseDir
	for _, comp := range strings.Split(filepath.ToSlash(filepath.Clean(decoded)), "/") {
		switch comp {
		case "", ".":
			continue
		case "..":
			cur = filepath.Dir(cur)
			continue
		}
		entry, suggestion, ok := findEntry(cur, comp)
		if !ok {
			return "", suggestion, false
		}
		cur = filepath.Join(cur, entry)
	}

	info, err := os.Stat(cur)
	if err != nil {
		return "", "", false
	}
	if info.IsDir() {
		entry, suggestion, ok := findEntry(cur, "README.md")
		if !ok {
			return "", suggestion, false
		}
		cur = filepath.Join(cur, entry)
	}
	return cur, "", true
}

// findEntry looks for an exact-case directory entry, returning a fuzzy
// suggestion from the directory's entries when absent.
func findEntry(dir, name string) (string, string, bool) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return "", "", false
	}
	names := make([]string, 0, len(entries))
	for _, e := range entries {
		if e.Name() == name {
			return name, "", true
		}
		names = append(names, e.Name())
	}
	suggestions := fuzzy.MatchN(name, names, 1)
	if len(suggestions) > 0 {
		return "", suggestions[0], false
	}
	return "", "", false
}
```

NOTE: check `fuzzy.MatchN`'s exact signature in `internal/graph/graph.go:171` usage before writing — it is `fuzzy.MatchN(query, candidates, n)` returning `[]string`. If picokit's fuzzy exposes `Match` (single best), prefer `fuzzy.Match(name, names)` returning `string` (used in validate.go:63) and drop the slice handling.

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/rules/ -run TestCheckLinks -v`
Expected: PASS. Then `go test ./internal/rules/` for the full package.

- [ ] **Step 5: Commit**

```bash
git add internal/rules/link_checks.go internal/rules/link_checks_test.go
git commit -m "feat(rules): add CheckLinks with encoding and case-sensitive resolve checks"
```

---

### Task 5: Anchors check — heading slugs + cache

**Files:**
- Modify: `internal/rules/link_checks.go` (fill in `checkAnchor`, `HeadingCache`)
- Test: `internal/rules/link_checks_test.go` (append)

**Interfaces:**
- Consumes: `extract.MarkdownExtractor` + `extract.ExtractSections` (`internal/extract/body.go:21`, returns `[]Section{Heading, Level, ...}`); `Record.AST`/`Record.Body`.
- Produces: `slugifyHeading(s string) string` (unexported); `link_anchor` validation errors; `(*HeadingCache).headingSlugs(absPath string) ([]string, error)`.

- [ ] **Step 1: Write the failing tests**

Append to `internal/rules/link_checks_test.go`:

```go
func TestSlugifyHeading(t *testing.T) {
	cases := map[string]string{
		"6. Zonas inciertas (black boxes)": "6-zonas-inciertas-black-boxes",
		"Glosario de siglas":               "glosario-de-siglas",
		"  Setup & Config  ":               "setup--config",
		"Ya_valido":                        "ya_valido",
	}
	for in, want := range cases {
		if got := slugifyHeading(in); got != want {
			t.Errorf("slugifyHeading(%q) = %q, want %q", in, got, want)
		}
	}
}

func TestCheckLinks_AnchorValidAndInvalid(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "master.md"), "# Title\n\n## 6. Zonas inciertas (black boxes)\n\ntext\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true, Anchors: true})
	src := filepath.Join(dir, "src.md")
	cache := NewHeadingCache()

	good := extract.Link{Target: "master.md", Anchor: "6-zonas-inciertas-black-boxes", Type: "reference", Style: extract.StyleMarkdown, Line: 1}
	if errs := CheckLinks([]extract.Link{good}, schema, src, cache); len(errs) != 0 {
		t.Fatalf("valid anchor rejected: %+v", errs)
	}

	bad := good
	bad.Anchor = "7-no-existe"
	errs := CheckLinks([]extract.Link{bad}, schema, src, cache)
	if len(errs) != 1 || errs[0].Rule != "link_anchor" {
		t.Fatalf("expected 1 link_anchor error, got %+v", errs)
	}
}

func TestCheckLinks_AnchorNilCacheStillWorks(t *testing.T) {
	dir := t.TempDir()
	writeFile(t, filepath.Join(dir, "master.md"), "## Intro\n")
	writeFile(t, filepath.Join(dir, "src.md"), "body")
	schema := mdChecksSchema(LinkChecks{Resolve: true, Anchors: true})
	link := extract.Link{Target: "master.md", Anchor: "intro", Type: "reference", Style: extract.StyleMarkdown, Line: 1}
	if errs := CheckLinks([]extract.Link{link}, schema, filepath.Join(dir, "src.md"), nil); len(errs) != 0 {
		t.Fatalf("nil cache should parse on the fly: %+v", errs)
	}
}
```

Note on `"Setup & Config"` → `"setup--config"`: the `&` is dropped but both surrounding spaces produce hyphens (GitHub/ADO behavior). The implementation below must NOT collapse consecutive hyphens.

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/rules/ -run 'TestSlugifyHeading|TestCheckLinks_Anchor' -v`
Expected: FAIL (`undefined: slugifyHeading`, and the bad-anchor case returns 0 errors from the stub).

- [ ] **Step 3: Implement**

In `internal/rules/link_checks.go`, replace the `checkAnchor` stub and add helpers (new imports: `"unicode"`, `"github.com/yuin/goldmark"`, `gmtext "github.com/yuin/goldmark/text"` — or reuse the extractor as below, which avoids direct goldmark imports):

```go
// checkAnchor verifies the link's anchor matches a heading slug in the
// resolved target file.
func checkAnchor(link extract.Link, resolvedPath string, cache *HeadingCache) []ValidationError {
	if cache == nil {
		cache = NewHeadingCache()
	}
	slugs, err := cache.headingSlugs(resolvedPath)
	if err != nil {
		return nil // unreadable/non-markdown target: resolve check already covers existence
	}
	want, err := url.PathUnescape(link.Anchor)
	if err != nil {
		want = link.Anchor
	}
	want = strings.ToLower(want)
	for _, s := range slugs {
		if s == want {
			return nil
		}
	}
	return []ValidationError{{
		Rule:     "link_anchor",
		Field:    "links",
		Message:  fmt.Sprintf("anchor %q not found in %q", link.Anchor, filepath.Base(resolvedPath)),
		Source:   "links.checks",
		Severity: "error",
	}}
}

// headingSlugs returns the slugified headings of a markdown file, cached.
func (c *HeadingCache) headingSlugs(absPath string) ([]string, error) {
	if slugs, ok := c.slugs[absPath]; ok {
		return slugs, nil
	}
	content, err := os.ReadFile(absPath)
	if err != nil {
		return nil, err
	}
	parseAST := true
	ext := &extract.MarkdownExtractor{ParseAST: &parseAST}
	rec, err := ext.Extract(absPath, content)
	if err != nil {
		return nil, err
	}
	var slugs []string
	if rec.AST != nil {
		for _, sec := range extract.ExtractSections(rec.AST, []byte(rec.Body)) {
			slugs = append(slugs, slugifyHeading(sec.Heading))
		}
	}
	c.slugs[absPath] = slugs
	return slugs, nil
}

// slugifyHeading converts a heading to its anchor slug: lowercase, spaces and
// hyphens become hyphens (not collapsed), punctuation is dropped, letters,
// digits and underscores are kept (GitHub/ADO code-wiki convention).
func slugifyHeading(h string) string {
	h = strings.ToLower(strings.TrimSpace(h))
	var b strings.Builder
	for _, r := range h {
		switch {
		case unicode.IsLetter(r) || unicode.IsDigit(r) || r == '_':
			b.WriteRune(r)
		case r == ' ' || r == '-':
			b.WriteByte('-')
		}
	}
	return b.String()
}
```

- [ ] **Step 4: Run tests to verify they pass**

Run: `go test ./internal/rules/ -v`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/rules/link_checks.go internal/rules/link_checks_test.go
git commit -m "feat(rules): validate markdown link anchors against target heading slugs"
```

---

### Task 6: CLI wiring — `validate` runs `CheckLinks`

**Files:**
- Modify: `cmd/rootline/validate.go` (`runValidateFiles` ~:104, `runValidateAll` ~:179)
- Test: `internal/e2e/link_checks_e2e_test.go` (create — mirrors how `runValidateAll` composes the pipeline; look at an existing `internal/e2e/*_test.go` first and copy its structure/naming conventions)

**Interfaces:**
- Consumes: `rules.CheckLinks`, `rules.NewHeadingCache` (Tasks 4-5); `index.Scan`, `rules.ResolveForRecord` (existing).
- Produces: `validate` (single-file and `--all`) reports `link_resolve`/`link_anchor`/`link_encoding` errors.

- [ ] **Step 1: Write the failing e2e test**

Create `internal/e2e/link_checks_e2e_test.go` (adapt package name and helpers to match the existing e2e files):

```go
package e2e

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

func writeTree(t *testing.T, root string, files map[string]string) {
	t.Helper()
	for rel, content := range files {
		p := filepath.Join(root, rel)
		if err := os.MkdirAll(filepath.Dir(p), 0o755); err != nil {
			t.Fatal(err)
		}
		if err := os.WriteFile(p, []byte(content), 0o644); err != nil {
			t.Fatal(err)
		}
	}
}

func TestE2E_MarkdownLinkChecks(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".stem": `version: 2
links:
  styles: [markdown]
  checks:
    resolve: true
    anchors: true
    encoding: true
`,
		"master.md":        "# Master\n\n## Glosario de siglas\n\ntext\n",
		"guides/setup.md":  "# Setup\n\nVer [master](../master.md#glosario-de-siglas).\n",
		"broken.md":        "[roto](guides/setpu.md)\n[anchor malo](master.md#no-existe)\n[espacio](my file.md)\n",
	})

	ctx := context.Background()
	reg := extract.NewASTRegistry()
	records, err := index.Scan(ctx, root, reg)
	if err != nil {
		t.Fatal(err)
	}

	cache := rules.NewHeadingCache()
	errsByRule := map[string]int{}
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		effective, err := rules.ResolveForRecord(filepath.Dir(absPath), rec.Path)
		if err != nil {
			t.Fatalf("resolve %s: %v", rec.Path, err)
		}
		all := rules.Validate(ctx, rec, effective)
		all = append(all, rules.CheckLinks(rec.Links, effective.Links, absPath, cache)...)
		for _, e := range all {
			errsByRule[e.Rule]++
		}
	}

	if errsByRule["link_resolve"] != 2 { // setpu.md + "my file.md" (also fails resolve)
		t.Errorf("link_resolve = %d, want 2 (map: %v)", errsByRule["link_resolve"], errsByRule)
	}
	if errsByRule["link_anchor"] != 1 {
		t.Errorf("link_anchor = %d, want 1 (map: %v)", errsByRule["link_anchor"], errsByRule)
	}
	if errsByRule["link_encoding"] != 1 {
		t.Errorf("link_encoding = %d, want 1 (map: %v)", errsByRule["link_encoding"], errsByRule)
	}
}
```

NOTE: check the `.stem` version header convention first — look at any fixture `.stem` in the repo's tests (`rg -l 'version: 2' internal/`) and copy the exact header the engine accepts (the engine rejects v0/v1 stems).

- [ ] **Step 2: Run test to verify it fails**

Run: `go test ./internal/e2e/ -run TestE2E_MarkdownLinkChecks -v`
Expected: PASS for resolve/anchor/encoding counts is NOT yet expected — the library calls exist since Tasks 4-5, so this test may already pass at the library level. If it passes, that's fine: it locks the pipeline contract. The CLI wiring below is still required (verify with Step 4's manual run).

- [ ] **Step 3: Wire the CLI**

In `cmd/rootline/validate.go`:

`runValidateFiles` — after `errs = append(errs, rules.Validate(ctx, record, effective)...)` (~line 104):

```go
		errs = append(errs, rules.CheckLinks(record.Links, effective.Links, absPath, linkCache)...)
```

with `linkCache := rules.NewHeadingCache()` declared once before the `for _, file := range files` loop.

`runValidateAll` — after `errs := rules.Validate(ctx, rec, effective)` (~line 179):

```go
		errs = append(errs, rules.CheckLinks(rec.Links, effective.Links, absPath, linkCache)...)
```

with `linkCache := rules.NewHeadingCache()` declared once before the `for _, rec := range records` loop (absPath already exists at :173).

Also check `runValidateStaged` (:239) — if it validates records through a similar loop, add the same call there for consistency.

- [ ] **Step 4: Manual verification of the binary**

```bash
go build -o /tmp/rootline-dev ./cmd/rootline
mkdir -p /tmp/lc-demo && cd /tmp/lc-demo
printf 'version: 2\nlinks:\n  styles: [markdown]\n  checks:\n    resolve: true\n    anchors: true\n    encoding: true\n' > .stem
printf '# A\n\n[roto](nope.md)\n' > a.md
/tmp/rootline-dev validate --all . --output json
```

Expected: JSON contains a `link_resolve` error for `a.md` targeting `nope.md`.

- [ ] **Step 5: Run full suite and commit**

Run: `just check && just test`
Expected: both pass.

```bash
git add cmd/rootline/validate.go internal/e2e/link_checks_e2e_test.go
git commit -m "feat(cli): run link checks in validate and validate --all"
```

---

### Task 7: Graph — filter links by effective styles

**Files:**
- Create: `internal/rules/link_filter.go`
- Modify: `cmd/rootline/graph.go` (after `index.Scan` at :71, before `graph.Build` at :93)
- Test: create `internal/rules/link_filter_test.go`

**Interfaces:**
- Consumes: `rules.ResolveForRecord`, `LinkSchema.EffectiveStyles()`, `linkStyle()`.
- Produces: `func FilterLinksByStyles(records []*extract.Record, root string)` — mutates each record's `Links` in place, keeping only links whose style is in the record's effective `links.styles`. With no `.stem` anywhere, the default `[wikilink]` drops markdown links → graph behavior identical to today.

Rationale for placement: `graph` imports only `extract` + `fuzzy` today; putting the stem-aware filter in `rules` keeps that dependency direction clean, and the cmd layer composes them (per-record resolution follows the per-scope pattern — `ResolveForRecord`, not merge-only resolution).

- [ ] **Step 1: Write the failing tests**

Create `internal/rules/link_filter_test.go`:

```go
package rules

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
)

func TestFilterLinksByStyles_DefaultDropsMarkdown(t *testing.T) {
	root := t.TempDir() // no .stem at all
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := &extract.Record{Path: "a.md", Links: []extract.Link{
		{Target: "[[w]]", Style: extract.StyleWikilink},
		{Target: "b.md", Style: extract.StyleMarkdown},
		{Target: "legacy"}, // style-less → wikilink
	}}
	FilterLinksByStyles([]*extract.Record{rec}, root)
	if len(rec.Links) != 2 {
		t.Fatalf("Links = %+v, want wikilink + legacy only", rec.Links)
	}
}

func TestFilterLinksByStyles_StemDeclaresMarkdown(t *testing.T) {
	root := t.TempDir()
	stem := "version: 2\nlinks:\n  styles: [markdown]\n"
	if err := os.WriteFile(filepath.Join(root, ".stem"), []byte(stem), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(root, "a.md"), []byte("x"), 0o644); err != nil {
		t.Fatal(err)
	}
	rec := &extract.Record{Path: "a.md", Links: []extract.Link{
		{Target: "[[w]]", Style: extract.StyleWikilink},
		{Target: "b.md", Style: extract.StyleMarkdown},
	}}
	FilterLinksByStyles([]*extract.Record{rec}, root)
	if len(rec.Links) != 1 || rec.Links[0].Style != extract.StyleMarkdown {
		t.Fatalf("Links = %+v, want markdown only", rec.Links)
	}
}
```

(Use the same `.stem` version header found in Task 6's note.)

- [ ] **Step 2: Run tests to verify they fail**

Run: `go test ./internal/rules/ -run TestFilterLinksByStyles -v`
Expected: compile error `undefined: FilterLinksByStyles`.

- [ ] **Step 3: Implement**

Create `internal/rules/link_filter.go`:

```go
package rules

import (
	"path/filepath"

	"github.com/pablontiv/rootline/internal/extract"
)

// FilterLinksByStyles rewrites each record's Links to only those whose style
// is declared in the record's effective links.styles (resolved per record).
// Records that resolve no .stem keep the wikilink default, so markdown links
// never leak into style-unaware consumers like the graph.
func FilterLinksByStyles(records []*extract.Record, root string) {
	for _, rec := range records {
		if len(rec.Links) == 0 {
			continue
		}
		styles := map[string]bool{extract.StyleWikilink: true}
		dir := filepath.Dir(filepath.Join(root, rec.Path))
		if effective, err := ResolveForRecord(dir, rec.Path); err == nil && effective != nil {
			styles = make(map[string]bool)
			for _, s := range effective.Links.EffectiveStyles() {
				styles[s] = true
			}
		}
		filtered := rec.Links[:0]
		for _, l := range rec.Links {
			if styles[linkStyle(l)] {
				filtered = append(filtered, l)
			}
		}
		rec.Links = filtered
	}
}
```

In `cmd/rootline/graph.go`, after `records, err := index.Scan(ctx, absRoot, reg)` and its error check:

```go
	rules.FilterLinksByStyles(records, absRoot)
```

(add the `rules` import if missing).

- [ ] **Step 4: Run tests and full suite**

Run: `go test ./internal/rules/ -run TestFilterLinksByStyles -v && just test`
Expected: PASS.

- [ ] **Step 5: Commit**

```bash
git add internal/rules/link_filter.go internal/rules/link_filter_test.go cmd/rootline/graph.go
git commit -m "feat(graph): filter graph links by effective .stem link styles"
```

---

### Task 8: E2E — dsprima-shaped fixture + backcompat regression

**Files:**
- Test: `internal/e2e/link_styles_e2e_test.go` (create)

**Interfaces:**
- Consumes: everything above; `graph.Build`, `rules.FilterLinksByStyles`.
- Produces: regression coverage locking (a) the full dsprima-shaped pipeline including the graph, (b) wikilink-only repos are untouched.

- [ ] **Step 1: Write the tests**

Create `internal/e2e/link_styles_e2e_test.go` (reuse `writeTree` from Task 6's file):

```go
package e2e

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/pablontiv/rootline/internal/extract"
	"github.com/pablontiv/rootline/internal/graph"
	"github.com/pablontiv/rootline/internal/index"
	"github.com/pablontiv/rootline/internal/rules"
)

func TestE2E_LinkStyles_GraphIncludesMarkdownWhenDeclared(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".stem":     "version: 2\nlinks:\n  styles: [markdown]\n",
		"README.md": "# Root\n\n[overview](docs/overview.md)\n",
		"docs/overview.md": "# Overview\n\n[back](../README.md)\n",
	})
	ctx := context.Background()
	records, err := index.Scan(ctx, root, extract.NewASTRegistry())
	if err != nil {
		t.Fatal(err)
	}
	rules.FilterLinksByStyles(records, root)
	g := graph.Build(ctx, records)

	if len(g.Edges["README.md"]) != 1 {
		t.Fatalf("README edges = %+v, want 1 markdown edge", g.Edges["README.md"])
	}
	if g.Edges["README.md"][0].Target != filepath.Join("docs", "overview.md") {
		t.Errorf("edge target = %q", g.Edges["README.md"][0].Target)
	}
	if broken := g.BrokenLinks(); len(broken) != 0 {
		t.Errorf("broken = %+v, want none", broken)
	}
}

func TestE2E_LinkStyles_WikilinkRepoUnaffected(t *testing.T) {
	root := t.TempDir()
	writeTree(t, root, map[string]string{
		".stem":   "version: 2\nlinks:\n  allowed: [reference]\n",
		"a.md":    "[[b]]\n\nAlso a [markdown link](c.md) that must stay invisible.\n",
		"b.md":    "# B\n",
	})
	ctx := context.Background()
	records, err := index.Scan(ctx, root, extract.NewASTRegistry())
	if err != nil {
		t.Fatal(err)
	}

	// Validation: the markdown link to a NONEXISTENT c.md produces no errors.
	for _, rec := range records {
		absPath := filepath.Join(root, rec.Path)
		effective, err := rules.ResolveForRecord(filepath.Dir(absPath), rec.Path)
		if err != nil {
			t.Fatal(err)
		}
		errs := rules.Validate(ctx, rec, effective)
		errs = append(errs, rules.CheckLinks(rec.Links, effective.Links, absPath, nil)...)
		if len(errs) != 0 {
			t.Errorf("%s: unexpected errors %+v", rec.Path, errs)
		}
	}

	// Graph: only the wikilink edge exists.
	rules.FilterLinksByStyles(records, root)
	g := graph.Build(ctx, records)
	edges := g.Edges["a.md"]
	if len(edges) != 1 || edges[0].Type != "reference" {
		t.Fatalf("edges = %+v, want single wikilink edge to b", edges)
	}
	if edges[0].Target == "c.md" {
		t.Error("markdown link leaked into wikilink-only graph")
	}
}
```

- [ ] **Step 2: Run tests**

Run: `go test ./internal/e2e/ -run TestE2E_LinkStyles -v`
Expected: PASS (this is post-hoc integration locking; if anything fails, fix the offending layer — most likely the `.stem` version header or `graph.Build` target normalization: compare edge target against `resolveTarget` output, which is `filepath.Clean`ed relative to source dir).

- [ ] **Step 3: Full suite + coverage gate**

Run: `just check && just test && just coverage-check`
Expected: all pass, every touched package ≥ 85%.

- [ ] **Step 4: Commit**

```bash
git add internal/e2e/link_styles_e2e_test.go
git commit -m "test(e2e): lock multi-style link pipeline and wikilink backcompat"
```

---

### Task 9: Docs, dsprima onboarding, DoD

**Files:**
- Modify: `CLAUDE.md` (internal/extract and internal/rules bullets in Package Layout)
- Create: `/Users/Shared/dsprima/docs/.stem` (outside this repo — do NOT commit dsprima; report findings and leave committing to the user)
- Modify: `/opt/factory/docs/backlog/` (mark this work done per Definition of Done)

- [ ] **Step 1: Update CLAUDE.md**

In the `internal/extract/` bullet add: "Extracts both `[[wiki-links]]` and markdown links `[text](target)`; each `Link` carries `style` (`wikilink`/`markdown`) and `anchor`. External schemes, images, and pure fragments are skipped."

In the `internal/rules/` bullet add: "`links.styles` selects which link styles are governed (default `[wikilink]`); `links.checks` (`resolve`/`anchors`/`encoding`) enables ADO code-wiki checks via `CheckLinks` (case-sensitive resolution, heading-slug anchors, `%20` encoding). Graph respects styles via `FilterLinksByStyles`."

- [ ] **Step 2: Commit docs**

```bash
git add CLAUDE.md
git commit -m "docs: document link styles and ADO code-wiki checks"
```

- [ ] **Step 3: Push and verify release (Definition of Done)**

```bash
git push origin master
# wait for CI release, then:
which rootline && rootline --version
```

Expected: new version (auto-released via conventional commits). If the installed binary hasn't auto-updated yet, run any rootline command once to stage the update and re-check.

- [ ] **Step 4: Onboard dsprima**

Write `/Users/Shared/dsprima/docs/.stem`:

```yaml
version: 2
links:
  styles: [markdown]
  checks:
    resolve: true
    anchors: true
    encoding: true
```

(match the exact version header used in rootline's own stems). Then:

```bash
rootline validate --all /Users/Shared/dsprima/docs
rootline graph /Users/Shared/dsprima/docs --broken
```

Expected: exploration found no dangling anchors and consistent kebab-case names, so likely zero errors — but report whatever appears to the user verbatim. Leave the `.stem` uncommitted in dsprima for the user to review.

- [ ] **Step 5: Update backlog + memory**

Update `/opt/factory/docs/backlog/` marking this feature delivered, and save an engram observation (topic `architecture/link-styles`) with the final state.

---

## Verification (end-to-end)

1. `just check && just test && just coverage-check` — all green.
2. `/tmp/rootline-dev validate --all <fixture>` shows `link_resolve`/`link_anchor`/`link_encoding` errors on a broken fixture (Task 6 Step 4).
3. A wikilink-only repo (e.g. `rootline validate --all docs/epics/` in this repo) produces IDENTICAL output before/after the change — run it on master and on the branch head, diff the JSON.
4. `rootline validate --all /Users/Shared/dsprima/docs` runs clean or surfaces real link defects.
