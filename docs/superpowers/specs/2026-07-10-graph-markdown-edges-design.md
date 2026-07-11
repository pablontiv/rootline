# Design: graph edges from markdown links

**Date**: 2026-07-10
**Status**: Implemented
**Bug**: `rootline graph` builds zero edges on repos whose `.stem` declares `links.styles: [markdown]`, while `rootline validate` sees and checks those same links correctly. Observed on the DS Prima wiki (`/Users/Shared/dsprima/docs`): 70 nodes, 0 edges, vacuously green `graph --check`.

## Root cause

The graph pipeline runs two link filters:

1. `rules.FilterLinksByStyles` (`f79816e`) — correct; markdown links survive it.
2. `filterLinksBySchema` (`cmd/rootline/graph.go`) — keeps a link only if `schema.Rules[link.Type]` exists, guarded by `schema.IsEmpty()`.

Commit `c38b7d3` added `styles` and `checks` to `LinkSchema` and to `IsEmpty()`. A `.stem` declaring only `styles`/`checks` is now "non-empty" with an empty `Rules` map, so every link fails the `Rules[link.Type]` lookup and is removed. This also affects wikilink repos that declare `styles` or `checks` without typed rules.

A secondary gap blocks full consistency with `validate`: validate resolves targets via `resolveCaseSensitive` (`internal/rules/link_checks.go`) — decodes `%20`, walks path components case-sensitively, resolves directory targets to their `README.md` — while graph's `resolveTarget` (`internal/graph/graph.go`) does a plain `filepath.Join` plus basename fallback. Without parity, `graph --check` would flag encoded and directory links that validate accepts.

## Changes

### 1. Gate fix

In `cmd/rootline/graph.go`, `filterLinksBySchema` changes its guard from `schema.IsEmpty()` to `len(schema.Rules) == 0`. Typed-rule filtering applies only when typed rules are declared; `styles`/`checks` alone no longer suppress links. Repos with typed rules keep current behavior exactly.

### 2. Resolution parity

New exported helper in `internal/rules` — `ResolveMarkdownTargets(records, root)` — called in `runGraph` after `FilterLinksByStyles` and before `graph.Build`:

- For each link with `Style == markdown`: resolve the target with `resolveCaseSensitive` against the record's directory (decode `%20`, case-sensitive walk, directory → `README.md`).
- On success: rewrite `link.Target` to the repo-relative slash path so it matches graph node keys.
- On failure: leave the target verbatim so `Graph.BrokenLinks()` reports it, with fuzzy suggestions as validate gives.
- Wikilinks are untouched; the existing basename fallback continues to handle them.
- `graph.Build` stays pure — all filesystem access happens in this one step.

### 3. Error handling

Unresolvable markdown targets are not dropped; they surface as broken links in `graph --check` (canary requirement). External schemes, `mailto:`, and fragment-only destinations never reach this step — extraction already excludes them.

## Testing

Unit (`internal/rules`):

- Schema with only `styles`/`checks` → links survive `filterLinksBySchema`.
- Schema with typed `Rules` → typed filtering unchanged.
- Resolver rewrites `%20`-encoded targets and directory targets (→ `README.md`); leaves case-mismatched targets unresolved.

e2e (extends the multi-style link pipeline suite from `d44d2a6`):

- Markdown-styles repo → `graph -o json` yields edges > 0.
- Mixed-style repo → edges from both styles when both are declared.
- Wikilink-only repo (no `.stem`, or styles undeclared) → behavior unchanged (backcompat: default `styles: [wikilink]`).
- Broken-link canary (`[x](no-existe.md)`) → `graph --check` exits non-zero and names the link.

## Acceptance criteria

1. On the dsprima wiki: `rootline graph -o json` yields edges > 0 (dozens expected), and `graph --check` reports broken links consistently with `validate`.
2. Canary: an untracked `canary.md` containing `[x](no-existe.md)` makes `graph --check` fail with that broken link.
3. Wikilink-only repos keep current behavior (backcompat: default `styles: [wikilink]`).
4. e2e coverage: markdown-styles repo → graph edges present; mixed-style repo → both styles when declared.

## Definition of Done

- Backlog note: The standard path `/opt/factory/docs/backlog/` does not exist on this machine; outcome recorded in spec status (Implemented).
- `just check` and `just test` green.
- CLI reinstalled and verified: `which rootline && rootline --version`.
- Acceptance: dsprima graph yields 200 edges; graph --check cycles consistent with validate (no new broken links introduced by markdown resolution); canary.md with broken markdown link correctly flagged by graph --check (exit 1, names target).
