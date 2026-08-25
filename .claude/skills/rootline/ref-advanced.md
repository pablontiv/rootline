# Advanced Rootline Operations Reference

## graph

Use `graph` for repository-level link topology, cycle checks, and broken-link checks. Edges come from the effective `.stem` `links.styles` replacement set: omitted means `[wikilink]`; use `[wikilink, markdown]` to include both wikilinks and markdown links. Markdown targets resolve literally with `%20` decoding, case-sensitive path walk, directory targets → their `README.md`, and root-anchored `/x.md` against the scan root. Path-less basename fallback is opt-in via `links.basename_fallback: true` and uses the graph/query index; unresolvable targets surface as broken links in `--check`.

### Usage

```bash
rootline graph <dir> -o json
rootline graph <dir> --check
rootline graph <dir> -o table --format dot
rootline graph <dir> -o table --format mermaid
rootline graph <dir> --where "tipo == 'feature'" -o json
```

Default global output is JSON, so `--format dot|mermaid` only produces diagram text when paired with `-o table` — and now *only* with `-o table`; `-o jsonl`/`-o csv` are rejected instead of falling through to a diagram. `graph --check` rejects an explicit `--output` outright. Mermaid output renders natively on GitHub and in most editors.

### Flags

| Flag | Use |
|---|---|
| `--format dot|mermaid` | diagram syntax when output is table |
| `--check` | report cycles and broken links; broken links fail, cycles are informational unless hardened |
| `--fail-cycles` | treat cycles as check failures for this run |
| `--quiet-cycles` | suppress per-cycle enumeration when cycles are informational |
| `--where "expr"` | filter records before graph construction |

### JSON Shape

```json
{
  "version": 1,
  "kind": "rootline/graph",
  "nodes": [],
  "edges": [],
  "cycles": [],
  "broken_links": []
}
```

Output is deterministic: `nodes` is sorted lexically by path, `edges` by
`(source, target, line, type)`, and `cycles` by canonical rotation — each cycle starts at its
lexicographically smallest member and the list is sorted element-wise — so JSON, DOT, Mermaid and
the numbered `--check` enumeration are byte-stable across runs and safe to commit or diff. Cycle
detection reports back edges over a canonical spanning forest, not every elementary circuit, so
`len(cycles)` is a stable count of detected back edges rather than of distinct circuits. `tree`
likewise picks its status column by sorting the enum-typed schema field names and taking the
first, so a corpus declaring more than one enum field no longer alternates columns between runs.

## migrate

Use `migrate` for schema diff reports and explicit schema/data migration operations.

### Usage

```bash
rootline migrate <path> -o json
rootline migrate <path> --from old.stem -o json
rootline migrate <path> --rename old_field=new_field --dry-run
rootline migrate <path> --rename old_field=new_field
rootline migrate <path> --split --dry-run
rootline migrate <path> --split
rootline migrate <path> --scaffold --dry-run
rootline migrate <path> --scaffold
```

Plain `migrate` reports differences. Writes require `--rename`, `--split`, or `--scaffold` without `--dry-run`.

### Flags

| Flag | Use |
|---|---|
| `--dry-run` | report planned changes without writing for migrate modes |
| `--from <file>` | compare the target `.stem` against a specified file |
| `--rename old=new` | rename a field across documents and `.stem` files |
| `--split` | split a flat `.stem` into per-level `.stem` files |
| `--scaffold` | add missing required section headings to documents |

## init

Use `init` only when a directory has Markdown documents and needs a `.stem` schema inferred or fetched from a template.

### Usage

```bash
rootline init <dir> --dry-run
rootline init <dir>
rootline init <dir> --force
rootline init <dir> --template owner/repo
rootline init <dir> --template owner/repo@tag
```

`init --dry-run` prints YAML, not Rootline JSON.

### Required Bootstrap Loop

```bash
rootline init <dir> --dry-run
rootline init <dir>
rootline describe <dir> -o table
rootline validate --all <dir> -o json
rootline fix --all <dir> --dry-run -o json
rootline fix --all <dir>
rootline validate --all <dir> -o json
git diff -- <dir>
```

Apply this loop only when the user asks to create or infer schema. Do not run it for simple validation or querying.

## analyze

Use `analyze` to produce inference reports from existing documents.

```bash
rootline analyze <dir> -o json
rootline analyze <dir> --incremental -o json
rootline analyze <dir> --threshold 0.80 -o json
```

Flags:

| Flag | Use |
|---|---|
| `--incremental` | include only inferences not covered by the target `.stem` |
| `--threshold <0..1>` | section pattern detection threshold |

Emits JSON: version 1, kind `"rootline/analyze"`, with `categories[]` (each `id`, `name`, `inference_count`, `inferences[]`) and a `summary` (`total_inferences`, `agent_required`, `engine_resolved`). Category order follows the command's detector sequence; each `inferences[]` array is deterministically ordered by its serialized identity fields, so identical inputs produce byte-identical JSON. Feeds `schema apply`. Full reference: `docs/analyze.md`.

Use `--field summary` only after confirming the analyze JSON contains that path.

## Canonical schema transport

`init`, `analyze`, `schema apply`, and `migrate --split` preserve a section as a real type plus `source: body.section["## Heading"]`. Inference preserves exact headings, makes partial-frequency candidates optional, and fails logical-name collisions. `new` and `migrate --scaffold` materialize missing required sections in lexical heading order with a non-empty default or `<!-- TODO -->`; frontmatter overrides are never written as empty shadow keys.

## apply (Removed)

The legacy `apply` command was removed — it fails with `unknown command "apply"`. Use the specialized commands instead:

**For schema proposals** (add fields to `.stem`, extend enums, etc.):
```bash
rootline schema propose <dir> -o json > proposals.json
rootline schema apply --report proposals.json --dry-run
rootline schema apply --report proposals.json
```

**For data-only repairs** (fix document frontmatter values):
```bash
rootline fix --all <dir> --dry-run -o json > repairs.json
rootline repair apply --report repairs.json --dry-run
rootline repair apply --report repairs.json
```

**Atomicity contract (apply commands).** Two guarantees, one deliberate non-guarantee:
1. *Per file, atomic* — writes stage to a sibling temp file and rename over the target, so a
   file is only ever its old self or its new self, never truncated. A failed write leaves the
   target untouched and removes the staging file.
2. *Per file, validated and reverted* — a written file is re-read and validated on its own; if
   it fails, the original bytes are restored and it moves to `rolled_back[]`.
3. *Per run, NOT all-or-nothing* — best-effort with honest reporting. A run that fails partway
   leaves earlier writes in place. Buffering the whole run was rejected: it would discard 99
   good repairs because of 1 unreadable path, and would still not survive a kill.

Read the envelope to know where a run got to: `complete` (true iff it carried through everything
it accepted — same condition as exit 0), `changed[]` (on disk), `rolled_back[]` (restored),
`rejected[]`/`skipped[]` (never attempted), `errors[]` (attempted and failed). Recover from a
partial run by re-running the report; it is declarative and skips what is already correct.

**Report root (both apply commands).** The paths in a report resolve against the directory
that was SCANNED, not the directory the report file sits in. `fix --all` and `schema propose`
both record it (`path` + absolute `root`). One shared precedence chain: `--root <dir>` >
report `root` > report `path` > the report file's own directory (the pre-`root` behaviour,
kept so old reports still apply). The resolved root comes back in the output envelope as
`root`, so a run that changed nothing tells you where it looked. This is why storing a report
in `reports/` or an artifacts directory now works instead of silently doing nothing.

**Exit status (both apply commands).** Non-zero exactly when `errors[]` or `rolled_back[]`
is non-empty; `rejected[]` and `skipped[]` alone exit `0`. `rolled_back[]` is a separate
condition — a successful revert leaves `errors[]` empty, so testing `errors[]` alone reads a
reverted run as a success. The JSON payload is always emitted on stdout before the non-zero
exit, and the short failure line goes to stderr, so `repair apply ... && deploy` now stops on
failure while still giving you a parseable result. `--dry-run` follows the same rule: a preview
that cannot resolve a path exits `1`, which makes it usable as a CI precondition check.

`repair apply` post-validates each file it writes, on its own, and restores the
pre-write bytes when validation rejects the result. Reverted files come back in
`rolled_back` (`{path, errors}`) and are withdrawn from `changed`, so a document is
never left in a state its own schema refuses. An unrelated failure elsewhere in the
run — including a path that could not be read — does not disable the check for the
rest. `add_field` proposals whose `value_source` is `enum_first` or `empty` are
skipped unless `--fill-missing` is passed; see `ref-validate.md`.

Frontmatter is rewritten through a `yaml.Node` round-trip, so key order and YAML
comments survive and a one-field change produces a one-field diff. Inter-token
whitespace and nested indentation are normalized by the encoder — do not expect a
byte-identical block for untouched fields.

Always inspect proposals before applying and validate after:
```bash
rootline validate --all <dir> -o json
git diff -- <dir>
```

## hooks and completion

Use only when the user asks for shell or repository integration.

```bash
rootline hooks status
rootline hooks install
rootline hooks uninstall
rootline completion bash
rootline completion zsh
rootline completion fish
```
