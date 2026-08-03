# Advanced Rootline Operations Reference

## graph

Use `graph` for repository-level link topology, cycle checks, and broken-link checks. Edges come from `[[wiki-links]]` and, when the effective `.stem` declares `links.styles: [markdown]`, from markdown links `[text](target)`; without that declaration only wikilinks appear (backcompat default). Markdown targets resolve with the same semantics as `validate`'s link checks (v1.13.0+): `%20` decoding, case-sensitive path walk, directory targets → their `README.md`, root-anchored `/x.md` against the scan root; unresolvable targets surface as broken links in `--check` (no basename fallback for markdown, unlike wikilinks).

### Usage

```bash
rootline graph <dir> -o json
rootline graph <dir> --check
rootline graph <dir> -o table --format dot
rootline graph <dir> -o table --format mermaid
rootline graph <dir> --where "tipo == 'feature'" -o json
```

Default global output is JSON, so `--format dot|mermaid` only produces diagram text when paired with `-o table`. Mermaid output renders natively on GitHub and in most editors.

### Flags

| Flag | Use |
|---|---|
| `--format dot|mermaid` | diagram syntax when output is table |
| `--check` | report cycles and broken links, exit non-zero on problems |
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

Output is deterministic: `nodes` is sorted lexically by path and `edges` by
`(source, target, line, type)`, so JSON, DOT and Mermaid are byte-stable across runs and safe to
commit or diff. `tree` likewise picks its status column by sorting the enum-typed schema field
names and taking the first, so a corpus declaring more than one enum field no longer alternates
columns between runs.

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

Emits JSON: version 1, kind `"rootline/analyze"`, with `categories[]` (each `id`, `name`, `inference_count`, `inferences[]`) and a `summary` (`total_inferences`, `agent_required`, `engine_resolved`). Feeds `schema apply`. Full reference: `docs/analyze.md`.

Use `--field summary` only after confirming the analyze JSON contains that path.

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

`repair apply` post-validates each file it writes, on its own, and restores the
pre-write bytes when validation rejects the result. Reverted files come back in
`rolled_back` (`{path, errors}`) and are withdrawn from `changed`, so a document is
never left in a state its own schema refuses. An unrelated failure elsewhere in the
run — including a path that could not be read — does not disable the check for the
rest.

Frontmatter is rewritten through a `yaml.Node` round-trip, so key order and YAML
comments survive and a one-field change produces a one-field diff. Inter-token
whitespace and nested indentation are normalized by the encoder — do not expect a
byte-identical block for untouched fields.

**Legacy mixed apply** (preserved for backward compatibility):
```bash
rootline apply report.json
```

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
rootline completion powershell
```
