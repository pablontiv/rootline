# Design: Warn on unknown `links.checks` keys in stem health

**Date**: 2026-07-12
**Status**: Approved
**Breaking**: No (`feat`, minor bump)

## Problem

`links.checks` decodes via `val.Decode(&checks)` in `LinkSchema.UnmarshalYAML`
(`internal/rules/rules.go`), and yaml.v3 silently drops unknown keys. A typo
like `cicles: true` is accepted with no effect — the user believes cycle
gating is armed while `graph --check` treats cycles as informational. That is
the fail-open direction: the gate silently does not gate. The v2.0.0 spec
(`2026-07-12-graph-check-cycle-severity-design.md`) recorded this as a
deferred fix.

## Decision

Capture unknown keys at parse time and surface them as a stem health warning
— the existing diagnostic surface for `.stem` quality issues, run as the
`validate --all` pre-phase.

A parse-time hard error was rejected: `graph.go` swallows `WalkUp` errors
(`if err == nil`), so a parse error would silently unload the entire stem —
strictly worse than one inert key.

## Changes

### 1. Parse-time capture — `internal/rules/rules.go`

`LinkSchema` gains a diagnostic field, excluded from the JSON contract:

```go
type LinkSchema struct {
	Allowed []string            `json:"allowed,omitempty"`
	Styles  []string            `json:"styles,omitempty"`
	Checks  *LinkChecks         `json:"checks,omitempty"`
	Rules   map[string]LinkRule `json:"rules,omitempty"`

	// UnknownCheckKeys lists keys found under links.checks that no check
	// consumes. Diagnostic only — surfaced by stem health, never serialized.
	UnknownCheckKeys []string `json:"-"`
}
```

In `UnmarshalYAML`'s `checks` branch, after the existing `Decode`, iterate the
mapping's key nodes and append any key not in
{`resolve`, `anchors`, `encoding`, `cycles`} to `UnknownCheckKeys`, preserving
document order. Known-key set lives next to `LinkChecks` so adding a future
check updates both in one place.

No changes to `IsEmpty()` or merge semantics: the field is per-file
diagnostic metadata, and stem health parses each `.stem` file individually
(it never sees merged schemas).

### 2. Stem health check #11 — `internal/rules/stemhealth.go`

New check `unknown-check-keys`, appended to `ValidateStemHealth` over
`parsedStems`. For each key in `stem.Links.UnknownCheckKeys`:

- `Name: "unknown-check-keys"`, `Status: "warn"`
- `Path`: stem file path relative to root; `Field`: the unknown key
- `Message`: `unknown key "cicles" in links.checks` plus
  ` (did you mean "cycles"?)` when the shared `fuzzy` package returns a match
  against the known-key set.

Warn, not fail: an inert key degrades a gate but breaks nothing; `validate
--all` output makes it visible where stem quality is already reviewed.

### 3. Tests

Unit — `internal/rules/rules_test.go`:
- Unknown keys captured in order; known keys never captured.
- All-known `checks` block → `UnknownCheckKeys` empty.

Unit — `internal/rules/stemhealth_test.go`:
- Stem with `cicles: true` → one `unknown-check-keys` warn carrying the
  fuzzy suggestion `cycles`.
- Clean stem → no `unknown-check-keys` entries.

### 4. Docs

CLAUDE.md `internal/rules/` bullet: "10 checks" → "11 checks", adding
`unknown-check-keys` to the enumerated list.

## Acceptance criteria

1. `.stem` with `links.checks.cicles: true`: `rootline validate --all` emits a
   `unknown-check-keys` warn naming the key and suggesting `cycles`; exit code
   unchanged by the warn (stem health `warn` maps to severity `warn`, which
   fails only under the existing `--strict` flag — desirable and free).
2. `.stem` with only known check keys: no `unknown-check-keys` output.
3. JSON outputs (`describe`, `graph`, `query`) unchanged — the field never
   serializes.

## Out of scope

- Validating unknown keys elsewhere in `links:` (any other key is by design a
  `LinkRule` name).
- Surfacing the warning in `graph --check` output.

## Commit

`feat(rules): warn on unknown links.checks keys in stem health` → v2.1.0.
