# Multi-style links: markdown link extraction + ADO code-wiki validation

**Date**: 2026-07-10
**Status**: Approved design, pending implementation plan

## Problem

Rootline only extracts `[[wiki-links]]` (`internal/extract/links.go:16`). The dsprima docs repo (`/Users/Shared/dsprima/docs`, 54 markdown files, 141+ links) uses ~95% relative markdown links with `.md` extension (`[text](../../foo.md)`), 6 anchor links (`x.md#kebab-anchor`), 1 directory link (`./`), and zero wiki-links. None of those links are visible to rootline validation or the dependency graph.

dsprima publishes to Azure DevOps as a **code wiki** (repo published as wiki), where relative `.md` links work as-is. The goal is to validate both:

1. **Resolution** — links point to files that exist (broken-link detection).
2. **ADO code-wiki conventions** — anchors resolve to real headings, no unencoded spaces, case-sensitive path matching.

## Key finding

Everything downstream of extraction is already format-agnostic: `LinkSchema` validation (`internal/rules/rules.go:53`, `validate.go:345`), graph resolution and broken-link detection with fuzzy suggestions (`internal/graph/graph.go:154,297`). Only the extraction layer is wikilink-hardcoded.

## Approach (chosen: B — extract all, govern via `.stem`)

Extraction always parses both styles and tags each link; the `.stem` decides which styles participate in validation and graph. Extraction stays config-free — no `.stem` plumbing into the extract layer. Consistent with rootline philosophy: data is data, governance lives in `.stem`.

Rejected alternatives:

- **A — always-on markdown parsing, no config**: silent behavior change for every existing rootline repo; cannot express anchor/encoding checks as target regexes.
- **C — full `LinkParser` plugin registry with convention profiles**: maximum extensibility, but there is no third style today (YAGNI). The chosen design admits a future `ado-provisioned` style (`/Page-Path`) without a registry.

## Design

### Link styles

Exactly two styles:

- `wikilink` — `[[target]]` and `[[type:target]]` (existing behavior).
- `markdown` — `[text](target)` relative/absolute path targets.

ADO code-wiki links ARE relative markdown links; ADO specifics are conventions (checks), not a separate syntax.

### 1. Extraction (`internal/extract/`)

- `Link` struct gains two additive fields (contract-safe under `version: 1`):
  - `Style string` — `"wikilink"` | `"markdown"`. Existing wikilinks get explicit `"wikilink"`.
  - `Anchor string` — the `#fragment`, stored separately; `Target` keeps the path only.
- `ParseLinks` (regex path) and `ParseLinksAST` (goldmark path — `ast.Link` nodes are first-class) additionally emit markdown links:
  - Extract only path-like targets. **Skip**: external schemes (`http://`, `https://`, `mailto:`), images (`![...](...)`), pure-anchor links (`](#foo)`).
  - `target.md#anchor` → `Target: "target.md"`, `Anchor: "anchor"`.
  - `Type` stays `""` for markdown links; `Line` and `Source: "body"` as today.
- `ParseFrontmatterLinks` unchanged — wikilink-only.

### 2. `.stem` schema (`internal/rules/`)

```yaml
links:
  styles: [markdown]        # which styles are governed; DEFAULT [wikilink] = full backcompat
  checks:
    resolve: true           # target file exists (relative to source, case-sensitive)
    anchors: true           # #anchor matches a heading slug in the target file
    encoding: true          # no raw spaces in targets (must be %20 or absent)
  rules:                    # existing per-type rules keep working
    reference:
      target: '.*\.md(#.*)?$'
```

- `LinkSchema.Styles []string` — filter applied by consumers (validate, graph). Empty/absent → `["wikilink"]`.
- `LinkSchema.Checks LinkChecks{Resolve, Anchors, Encoding bool}` — new struct.
- Existing `Allowed` + per-type `target` regex rules apply only to links whose style is in `Styles`.
- `.stem` merge needs no special-casing: maps merge, arrays replace (existing semantics).

### 3. Validation checks (`internal/rules/`)

New checks in `validateLinks`, gated by `Checks`, applied to style-filtered links:

- **resolve**: join source dir + target; check the file exists with case-sensitive comparison against actual directory entries (APFS is case-insensitive, ADO/git are not). Directory targets (`./`, `../foo/`) resolve to `README.md` (rootline's index convention). Broken → validation error with fuzzy suggestions (`internal/fuzzy`).
- **anchors**: when `Anchor != ""` and the target resolves, parse the target's headings with goldmark (existing dependency) and slugify ADO-style (lowercase, spaces→`-`, strip/percent-encode specials); error if no heading matches. Cache parsed headings per target file within a validation run.
- **encoding**: reject raw spaces in `Target`.

### 4. Graph (`internal/graph/`)

- `Build` filters `Record.Links` by the effective `links.styles` resolved **per record** via `rules.ResolveForRecord` — the per-scope pattern established in the June 2026 governance work (not merge-only `DefaultResolver`).
- `resolveTarget` already handles relative paths; add README resolution for directory targets. Anchors live in a separate field, so no target stripping is needed.
- JSON output gains `style`/`anchor` fields on links (additive; no CLI flag changes).

### 5. Backward compatibility

- Default `styles: [wikilink]` → existing repos see zero behavior change in validation and graph.
- `Link.Style`/`Link.Anchor` are additive JSON fields.
- Field type `"link"` (`ContainsWikilink`) unchanged.

## Files to modify (rootline)

- `internal/extract/links.go`, `links_ast.go`, `extract.go` — markdown link parsing, `Style`/`Anchor` fields
- `internal/rules/rules.go` — `LinkSchema.Styles`, `LinkChecks`
- `internal/rules/validate.go` (+ new `link_checks.go` if it grows) — style filter, resolve/anchors/encoding checks
- `internal/graph/graph.go` — per-record style filtering, directory→README resolution
- `internal/e2e/` — fixture mimicking dsprima: relative `.md` links, anchors, one broken link, one bad anchor, one unencoded space
- Docs: `CLAUDE.md` package notes

## dsprima onboarding (final phase)

1. Write a minimal `.stem` at `/Users/Shared/dsprima/docs/.stem`:
   `links: {styles: [markdown], checks: {resolve: true, anchors: true, encoding: true}}`
   (schematizing frontmatter fields `estado`/`validado`/`dominio` is out of scope).
2. Run `rootline validate --all /Users/Shared/dsprima/docs` and `rootline graph`; report real findings.

## Testing & verification

- Strict TDD: test-first per unit — parser cases (code-fence skipping, anchors, image/external exclusion), each check's behavior, graph filtering, `.stem` parsing/merge.
- E2E fixture asserts: broken link reported with suggestion, bad anchor reported, unencoded space reported, and a wikilink-only repo is unaffected (backcompat regression).
- Definition of Done: `just check`, `just test` (85% coverage floors), conventional commits + push, `rootline --version` shows the new release, backlog updated in `/opt/factory/docs/backlog/`.
- End-to-end proof: run the installed CLI against dsprima/docs and show output.
