# Output Formats

`--output` / `-o` is a global flag. It takes one of four values:

| Value | Shape |
|-------|-------|
| `json` | One JSON envelope carrying `version` and `kind` (default) |
| `jsonl` | JSON Lines: one object per line |
| `csv` | CSV with a header row |
| `table` | Human-readable text — a summary line, an ASCII tree, or a diagram, depending on the command |

Anything else is rejected before the command runs:

```console
$ rootline query docs/ -o sdlkfj
Error: unknown output format "sdlkfj" (use json, jsonl, csv or table)
```

The check is case-sensitive and rejects the empty string. `-o JSON` is an error, not a
synonym for `-o json`.

## Not every command supports all four

`jsonl` and `csv` are row formats. They need a flat, column-shaped result, which only
`query --select` produces. Every other command emits a nested envelope with no
defensible flattening, so it declares `json` and `table` only, and an unsupported
combination is rejected rather than quietly downgraded:

```console
$ rootline stats docs/ -o csv
Error: rootline stats does not support output format "csv" (use json or table)
```

| Command | Supported |
|---------|-----------|
| `query` | `json`, `jsonl`, `csv`, `table` |
| `analyze`, `describe`, `explain`, `fix`, `graph`, `migrate`, `repair`, `schema propose`, `schema apply`, `stats`, `tree`, `validate` | `json`, `table` |
| `completion`, `hooks`, `init`, `new`, `set` | none — these emit human text or write files and never consult `--output` |

The matrix is declared in one table (`cmd/rootline/output.go`) and a test walks the
whole command tree asserting every command has an entry, so a new command cannot ship
without deciding what it supports.

## What `table` means per command

`table` is "the human-readable rendering", which is not the same rendering everywhere:

- `stats` — a summary line
- `tree` — the ASCII tree
- `graph` — the diagram selected by `--format dot|mermaid`
- `query`, `validate`, `describe`, `explain` — a text table or report

`tree` and `graph` previously produced their diagram for any value that was not `json`,
so `-o jsonl` returned box-drawing characters or Graphviz DOT at exit 0. The diagram is
now bound to `-o table` and to nothing else.

## `--field`

`--field` extracts a dot-path from the JSON envelope, so it applies to `-o json` only. Combining it with any other format — or with a command that emits no envelope — is an error, not a silent no-op:

```console
$ rootline stats docs/ -o table --field kind
Error: --field requires --output json: extraction reads the JSON envelope, and "table" has none
```

The flag is repeatable, as its help has always claimed. One path emits the bare value; several emit a JSON array in flag order. Full syntax in [Describe](describe.md#field-extraction).

## `graph --check`

`--check` is a text-plus-exit-code validator; it emits no envelope. Passing `--output`
explicitly alongside it is an error rather than a silent no-op:

```console
$ rootline graph docs/ --check -o json
Error: --check does not support --output: it emits a text report and an exit code; drop
--output, or drop --check to get the rootline/graph envelope, which carries the same
cycles and broken_links
```

The default `-o json` is not an explicit request, so `rootline graph docs/ --check`
keeps working unchanged.
