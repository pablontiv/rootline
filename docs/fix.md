---
estado: Completed
---
# Fix & Proposal Engine

The `rootline fix` command goes beyond simple mechanical repairs. It analyzes validation errors and proposes intelligent solutions based on data patterns and heuristics.

## Link findings

`fix --all` reports link problems it will not repair, under `link_findings` in JSON and as a
"Link findings" block in table output:

```console
$ rootline fix --all docs/
Link findings: 1 (reported, not repaired)
  a.md: link target "guied.md" does not resolve to an existing file (case-sensitive) (link_resolve) — did you mean "guide.md"?
```

`fix` does not rewrite link bodies. Correcting a link on a fuzzy guess is a destructive edit
outside its data-repair contract — the suggestion is offered so a human can make the call. Before
this, a record `validate` failed with `link_resolve` came back from `fix` as a clean run, which
read as a tool bug.

## Type representation repairs

`fix --all` can repair a narrow class of YAML representation mismatches where the schema expects
`type: string` but the frontmatter scalar was parsed by YAML as a native value. The repair preserves
the exact scalar text and changes only the representation by letting the YAML writer quote the value.

| Expected | Actual YAML representation | Result |
|---|---|---|
| string | timestamp | exact lexeme quoted automatically |
| string | boolean | exact lexeme quoted automatically |
| string | integer | exact lexeme quoted automatically |
| string | mapping, sequence, null, number | `type_findings`, no mutation |
| boolean/integer/other | string or another representation | `type_findings`, no coercion |

Contract details:

- `from` and `to` can be textually equal because the representation changes.
- `from_representation` is set only for type repairs and omitted for historical proposal shapes.
- `repair apply` requires exact lexeme and representation matches; unknown markers fail closed. A
  typed stored report re-applied after the field has already been quoted is rejected, not skipped,
  because the current representation is `string` and no longer matches `from_representation`.
- Findings are reported in JSON/table but do not change exit `0`; callers run `validate` for a
  validity verdict.
- The YAML writer adds quoting; reports never embed quote characters in `to`.


## Proposal Types

Rootline categorizes issues into specific proposal types to preserve semantic meaning.

| Type | Logic | Example |
|------|-------|---------|
| **extend_enum** | If N files share an invalid enum value, propose adding it to the `.stem` | Adding "Obsolete" to status |
| **correct_value** | Suggests the closest valid enum value for a typo | "Completd" → "Completed" |
| **migrate_value** | Extracts structured data from free-text invalid values | "Pending (blocked by T001)" → `[[blocks:T001]]` |
| **extract_body** | Finds `**Key**: Value` patterns in the document body | Extracting status from a legacy README |
| **add_field** | Adds a missing required field with a default or inferred value | Adding `estado: Pending` to a file without it |
| **infer_from_siblings** | Uses statistical majority of a directory to fill missing values | Setting `tipo: software` because 90% of files are software |
| **correct_outlier** | Identifies values that differ from a strong consensus in a folder | Flagging a task as "manual" in a "deploy" folder |
| **infer_from_children** | Historical type retained in the JSON counters, but no detector currently emits it; configure parent rollups with `.stem` `aggregate:` expressions instead | Summary counter remains `0` |
| **correct_link** | Historical proposal type; current `fix` reports link problems under `link_findings` and does not rewrite link bodies | Human reviews the `link_resolve` suggestion |
| **add_aggregate** | Schema-surface suggestion to add an aggregate expression to a `.stem`; current generation uses a conservative quoted default, not semantic `len(filter(...))` logic | Adding `estado: "Completed"` to `.stem` for review |
| **remove_stem_field** | Removes invalid fields from `.stem` detected by stem health checks | Removing a field that references a non-existent type |
| **propagate_aggregate** | Detects stale aggregate values in index files and corrects them | Updating `completed` count after child status changes |
| **set_field** | Sets a frontmatter field to a specific value via the `rootline set` pipeline | Setting `estado: Completed` explicitly |
| **set_section** | Sets, appends to, or creates a section body via the `rootline set` pipeline | Replacing `## Summary` with inferred content |
| **schema_evolution** | Explicit migration marker for destructive schema evolution | Recording a reviewed incompatible schema change |
| **remove_field** | Explicit field removal from schema during migration | Removing a deprecated schema field |
| **loosen_required** | Explicit required→optional change during migration | Making `owner` optional |
| **change_type** | Explicit incompatible type change during migration | Changing `priority` from string to integer |
| **replace_enum_values** | Explicit replacement of an enum's value set during migration | Replacing legacy lifecycle values |
| **loosen_severity** | Explicit validation-severity reduction during migration | Lowering a rule from error to warning |

## Proposal Struct

Each proposal in the output has the following fields:

| Field | Type | Description |
|-------|------|-------------|
| `type` | string | Proposal type (see table above) |
| `field` | string | Field name the proposal targets |
| `description` | string | Human-readable explanation |
| `paths` | string[] | Affected file paths |
| `from` | string | Exact scalar text expected before a `correct_value` repair applies |
| `to` | string | Replacement scalar text; for representation-only repairs this matches `from` textually |
| `value_source` | string | Optional provenance for `add_field` values (`schema_default`, `enum_first`, or `empty`) |
| `from_representation` | string | Optional evidence for type representation repairs (`timestamp`, `boolean`, or `integer`) |
| `heading` | string | For `set_section` proposals: the Markdown heading to mutate |
| `mode` | string | For `set_section` proposals: `replace` (default), `append`, or `create` |

`heading` and `mode` are only populated on `set_section` proposals. `set_field` reuses the existing `applySetField` pipeline from `rootline set`.

## Three Repair Paths

Rootline provides **three separate commands** for different repair scenarios. Choose the right one for your workflow:

### 1. `rootline fix` — Built-in Repair (Legacy)

**Deprecated.** Use `rootline repair apply` instead (see below).

```bash
rootline fix --all --dry-run   # Preview data repairs and withheld schema suggestions
rootline fix --all             # Apply data-only repairs to documents
```

`fix` combines proposal generation and application in one step. It writes document frontmatter only. Schema-surface proposals such as `extend_enum`, `add_aggregate`, and `remove_stem_field` are withheld for review: in dry-run JSON they appear as a `schema_suggestions` array, while apply JSON reports a `schema_suggestions` integer count.

### 2. `rootline repair apply` — Data-Only Bulk Repair (Current)

Use this for applying the versioned `rootline/proposals` report produced by
`rootline fix --all --dry-run`. Analyze reports are schema/diagnostic inference
reports and are accepted by `schema apply`, not `repair apply`.

**Workflow:**

```bash
cd docs/

# 1. Generate a proposals report
rootline fix --all . --dry-run -o json > fix-proposals.json

# 2. Review the proposals (schema_suggestions are separated)
# 3. Apply repair proposals (frontmatter only, never .stem)
rootline repair apply --report fix-proposals.json --dry-run
rootline repair apply --report fix-proposals.json
```

**Where the paths in a report resolve.** `fix --all` records the directory it scanned in the
report (`path` as you spelled it, `root` as an absolute path), and both `repair apply` and
`schema apply` resolve against that — not against wherever the report file happens to sit. So the
normal CI shape works:

```bash
rootline fix --all docs/ --dry-run -o json > artifacts/repairs.json
rootline repair apply --report artifacts/repairs.json    # resolves to docs/, not artifacts/
```

The rule is one shared precedence chain, identical in both commands:

1. `--root <dir>` — an explicit override, the last word
2. `root` recorded in the report — the absolute scan root
3. `path` recorded in the report — the scan root as it was spelled
4. the directory holding the report file — the pre-`root` behaviour

Rung 4 keeps reports written before rungs 2 and 3 existed working exactly as they did, so an old
report stored beside its documents still applies. It was also the whole problem: on its own it
made a report in a `reports/` directory resolve to paths that do not exist, and the run silently
changed nothing.

Both commands report the root they settled on in their output envelope as `root`, so a run that
touched nothing tells you where it looked:

```bash
rootline repair apply --report artifacts/repairs.json -o json | jq .root
# "/abs/path/to/docs"
```

**Report envelope.** `repair apply` validates the report envelope before touching any file
and rejects anything that is not `version: 1` and `kind: rootline/proposals`:

```bash
rootline repair apply --report analyze.json
# Error: wrong report kind: rootline/analyze (expected rootline/proposals)
```

An `analyze` report is therefore not a valid input here — produce the report with
`rootline fix --all <dir> --dry-run -o json`. For the schema-surface half of an analyze
report, use `rootline schema apply`, which does accept `kind: rootline/analyze`.

**Exit status.** `repair apply` and `schema apply` exit non-zero exactly when the run failed to
carry through something it accepted — that is, when `errors[]` or `rolled_back[]` came back
non-empty. Everything else exits `0`. The payload is always written to stdout first, so a
failing run stays machine-readable; the short `Error: apply failed: ...` line goes to stderr.

| Outcome | Field | Exit | Why |
|---------|-------|------|-----|
| Applied cleanly | `changed[]` | `0` | The command did what it was asked. |
| Refused on policy | `rejected[]` | `0` | A deliberate refusal, already reported — a containment violation, a schema proposal handed to `repair apply`, an existing `.stem` without `--force`. |
| Deferred | `skipped[]` | `0` | Nothing was attempted; for `schema apply` this includes `requires_agent` work, while for `repair apply` it includes engine-chosen missing-field values withheld unless `--fill-missing` is passed. |
| Could not be done | `errors[]` | `1` | An unreadable path, a failed write, an unresolvable target. |
| Written, then reverted | `rolled_back[]` | `1` | Post-validation rejected the result and the pre-write bytes were restored. The caller asked for a change that could not be made, so a tidy revert is still a failure. |

`rolled_back[]` is a separate condition, not a subset of `errors[]`: a successful revert leaves
`errors[]` empty, so a script testing only `errors[]` would read that run as a success.

With `repair apply -o table`, reverted files appear in a separate `Rolled back (N)`
section with each path and all recorded validation reasons. They are not counted as
successful changes, and a rollback-only result does not print `No repairs applied`.

`--dry-run` is not an exception. A preview that cannot resolve a path reports it in `errors[]`
and exits `1`, which makes `repair apply --report r.json --dry-run` usable as a CI precondition
check. A dry run performs no writes, so it can never populate `rolled_back[]`.

```bash
rootline repair apply --report fix-proposals.json && deploy
# the deploy no longer runs when the repair failed
```

### Atomicity contract

An apply run makes two guarantees and deliberately does not make a third. Stated plainly, because
the cost of guessing is a half-written governed document.

**Per file: atomic.** Every write goes to a staging file in the target's own directory and is
renamed over the target. A file is therefore only ever observed as its old self or its new self,
never as a truncated middle. A bare write truncates and then writes, so a process killed in
between leaves a document its own schema would reject. If the write fails, the target is untouched
and the staging file is removed.

**Per file: validated, and reverted if it fails.** After a write, the file is re-read and
validated on its own. If validation rejects the result, the pre-write bytes are restored, the file
moves from `changed[]` to `rolled_back[]`, and the run reports failure. The check is per file: an
error against one path — including a path that could not be read at all — never disables the check
for the rest.

**Per run: best-effort with honest reporting. NOT all-or-nothing.** A run that fails partway leaves
the files it already wrote in place. It does not buffer every rewrite and flush at the end, and it
does not roll the whole run back.

That is a deliberate choice, not an omission. Run-level atomicity would mean holding every
rewritten document in memory until the last proposal succeeded, and it would still not be atomic
against a kill — the flush itself is many writes. Worse, it makes the common case worse: one
unreadable path in a hundred-file report would discard ninety-nine good repairs. Best-effort plus
an exact account of what happened is more useful and more honest than a guarantee that cannot be
kept.

So a run is answerable rather than transactional. Read the envelope to know where it got to:

| Field | Meaning |
|-------|---------|
| `complete` | `true` iff the run carried through everything it accepted. Same condition as exit `0`. |
| `changed[]` | written and validated — these are on disk |
| `rolled_back[]` | written, rejected by validation, restored — these are back to their original bytes |
| `rejected[]` | never attempted, refused on policy |
| `skipped[]` | never attempted, deferred |
| `errors[]` | attempted or resolved and failed |

`complete` is not redundant with the exit status. A report saved as a CI artifact is read long
after `$?` is gone, and a consumer should not have to re-derive the rule — or parse prose — to
learn whether the tree is in the state it asked for:

```bash
rootline repair apply --report r.json -o json > result.json   # exit status observed here
jq -e .complete result.json                                    # ...and still answerable here
```

Recovering from a partial run is a re-run: the report is declarative, and re-applying it skips what
is already correct. Take a `--dry-run` first if you want to see what remains.

For `set_field`, an existing string equal to the requested value is skipped in both dry-run and
apply, and the remaining paths are still processed. Equality is type-strict: a YAML boolean or
integer is not the same as its string spelling, and a missing field is not an empty string.

`repair apply` applies only repair-surface proposals to document frontmatter:
- correct_value
- add_field
- migrate_value
- extract_body
- infer_from_siblings
- correct_outlier
- infer_from_children
- propagate_aggregate
- set_field
- set_section

Schema proposals (extend_enum, add_aggregate, remove_stem_field) are rejected visibly in `rejected[]` — use `rootline schema apply` instead.

**Path containment.** A report is untrusted input, so every path it names is checked against
the scan root before any file is opened. A path that escapes the root — `../../../etc/shadow`
or an absolute path — is refused outright and never read, let alone written. Refusals land in
`rejected[]`, not `errors[]`, keeping a policy decision distinguishable from a failed write:

```bash
rootline repair apply --report fix-proposals.json
# rejected: containment violation: path "../outside.md" escapes root (root "/abs/scan")
```

A proposal is applied whole or not at all, so a single escaping path discards the entire
proposal rather than applying it partially.

`schema apply` classifies the identical event the identical way: a `create_stem` target outside
the scan root lands in `rejected[]` and the run still exits `0`. It previously reported that
refusal in `errors[]`, which made the two commands disagree about the same containment decision
and — once the exit rule above exists — would have turned a deliberate refusal into a build
failure.

With `--dry-run`, the output additionally carries `resolved_targets` so you can see where each
write would have landed and why any were refused before committing to the run:

```json
{
  "resolved_targets": {
    "accepted": { "tasks/T001.md": "/abs/scan/tasks/T001.md" },
    "rejected": { "../outside.md": "escapes root" }
  }
}
```

`resolved_targets` is additive and omitted outside dry-run; the report contract stays `version: 1`.

Flags:
- `--dry-run` — Preview changes without modifying files
- `--fill-missing` — Also apply `add_field` proposals whose value was engine-chosen rather than schema-declared
- `--report <file>` — Path to proposals JSON file (required)
- `--root <dir>` — Absolute scan-root override; takes precedence over report `root`/`path`

### 3. `rootline schema apply` — Schema Mutation

Use this for applying schema changes (extending enums, creating `.stem` fields, updating `.stem` files).

**Workflow:**

```bash
# 1. Generate schema proposals from analyze report
rootline analyze docs/ --incremental > analyze.json

# 2. Review schema_suggestions in analyze.json
# 3. Apply schema proposals to .stem files
rootline schema apply --report analyze.json --dry-run
rootline schema apply --report analyze.json

# Or generate proposals without analysis:
rootline schema propose docs/ > proposals.json
rootline schema apply --report proposals.json --dry-run
rootline schema apply --report proposals.json
```

`schema apply` accepts a report from `rootline analyze` or `rootline schema propose` and applies only schema-surface proposals to `.stem` files:
- create_stem
- extend_enum (analyze path only)

Data-only repairs (correct_value, add_field, etc.) are rejected visibly in `rejected[]` — use `rootline repair apply` instead.

**What propose proposes is what apply writes.** Each `create_stem` proposal carries a `patch`
field holding the complete inferred YAML, and `schema apply` writes those bytes verbatim. It does
not re-derive the schema at apply time, so the artifact a reviewer approved is the artifact that
lands on disk. The neighbouring `patch_preview` is the same YAML truncated to 200 characters for
display; it is not the write source. A proposal with no `patch` — a report generated before this
field existed — is refused with an instruction to re-run `schema propose`, rather than falling
back to a generator that emits untyped `type: string` shells.

**Where the scan root comes from.** `schema propose` records an absolute `root`, so a report
applies from any working directory — the usual CI shape, where proposals are written into an
`artifacts/` directory and applied from elsewhere. Reports without `root` keep the older behaviour
of resolving `path` against the caller's working directory. Post-apply validation runs against
that root, and a scan that fails surfaces in `errors[]` rather than as an all-zero
`validation_summary`, which reads identically to a clean run.

**Path containment.** As with `repair apply`, a report is untrusted input: every `create_stem`
target is checked against the scan root before a `.stem` is written, so a report naming
`<root>/../../outside/.stem` no longer scaffolds outside the tree the command was pointed at.

`schema propose` emits **absolute** targets, so the propose→apply loop depends on absolute paths
staying valid — they are accepted and then confined, not refused outright. That is the one
difference from `repair apply`, which rejects absolute paths as malformed input.

The second difference is where refusals land. `schema apply` reports policy refusals such as overwrite without `--force`, unknown operations, and containment violations in `rejected[]`; operational failures such as scan or write errors remain in `errors[]`:

```bash
rootline schema apply --report proposals.json
# rejected: ".stem already exists in /docs (use --force to overwrite)"
# OR
# rejected: containment violation: path "/tmp/outside/.stem" escapes root (root "/abs/scan")
```

With schema-proposals reports, `--dry-run` carries an additive `resolved_targets` envelope so you can see where each `.stem` would land before writing. Analyze-report dry runs do not emit this key: they plan directly from in-memory inferences and report accepted/skipped actions instead.

```json
{
  "resolved_targets": {
    "accepted": { "/abs/scan/tasks/.stem": "/abs/scan/tasks/.stem" },
    "rejected": { "/tmp/outside/.stem": "escapes root" }
  }
}
```

`resolved_targets` is omitted outside dry-run; the contract stays `version: 1`.

Flags:
- `--dry-run` — Preview changes without modifying files
- `--force` — Overwrite existing `.stem` files when applying `create_stem` proposals
- `--report <file>` — Path to schema proposals report JSON file (required)
- `--root <dir>` — Absolute scan-root override; takes precedence over report `root`/`path`

## When to Use Each

| Scenario | Command | Example |
|----------|---------|---------|
| Add missing required fields to documents | `repair apply` | Missing `estado` field in all tasks |
| Correct misspelled enum values | `repair apply` | `"Complted"` → `"Completed"` |
| Extend `.stem` enum to accept new valid values | `schema apply` | Adding `"Archived"` to allowed statuses |
| Create `.stem` for a new directory | `schema apply` | Initialize schema after inferring patterns |
| Update `.stem` after schema changes | `schema apply` | Add new field definitions after inference |

## YAML AST Preservation

When Rootline modifies a `.stem` or a frontmatter block, it rewrites through YAML nodes so comments and key order are preserved where possible. Inter-token whitespace, inline-comment spacing, and nested indentation are normalized by the YAML encoder; do not expect byte-identical formatting outside the edited value.

## Sibling Inference Logic

To prevent noisy guesses, sibling inference requires:
- A minimum number of siblings (default: 2 for missing, 3 for outliers).
- A strong consensus threshold (default: 60% for missing, 75% for outliers).
