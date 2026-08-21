---
estado: Completed
---
# Describe

Rootline is not only a validator.

Because `.stem` files fully describe structure and constraints,
Rootline can explain what a valid document looks like *before it exists*.

```bash
rootline describe docs/api/
```

## Result Shape

```json
{
  "version": 1,
  "kind": "rootline/describe",
  "path": "docs/api/",
  "applies": ["docs/.stem", "docs/api/.stem"],
  "scope": { "match": "*.md" },
  "schema": {
    "title": {
      "type": "string",
      "required": true,
      "source": "body.section[\"## Title\"]",
      "defined_in": "docs/.stem"
    },
    "status": {
      "type": "enum",
      "values": ["draft", "review", "published"],
      "default": "draft",
      "excludes": { "match": "*/README.md" },
      "defined_in": "docs/api/.stem"
    }
  },
  "validate": [
    {
      "rule": "requires",
      "if": { "status": "published" },
      "then": { "fields": ["owner"] },
      "source": "docs/.stem"
    }
  ],
  "derive": {
    "slug": "slugify(title)"
  },
  "aggregate": {
    "total": "len(descendants)"
  },
  "links": {
    "allowed": ["blocks", "depends"]
  },
  "layers": ["docs/.stem", "docs/api/.stem"],
  "provenance": {
    "title": "docs/.stem",
    "status": "docs/api/.stem"
  }
}
```

These are every key `describe` can emit. `layers`, `provenance` and `hints` are
omitted when empty, so a run on a governed directory looks exactly like the
above; `hints` only appears when there is advice to give:

```json
{
  "hints": ["No .stem schema found. Run 'rootline init <path>' to infer schema from existing files."]
}
```

For source-backed fields, `source` is the logical extraction directive (for example `body.section["## Title"]`); `defined_in` is the physical `.stem` declaration path. This separation keeps source identity and schema provenance unambiguous.

The `layers` array lists all `.stem` files in resolution order (root-to-leaf).
The `provenance` map shows which `.stem` file defined each field, enabling
field-level traceability of the schema inheritance chain. Paths are printed
absolute; they are shortened here for readability.

### Sections

| Section | Purpose |
|---------|---------|
| `schema` | Field definitions with types, enums, defaults, and sequence auto-numbering |
| `validate` | Explicit validation rules (non_empty, exists, requires, enum) |
| `derive` | Per-record expressions evaluated via expr-lang (e.g., `slugify(title)`) |
| `aggregate` | Bottom-up expressions for index files (e.g., `len(descendants)`) |
| `links` | Wiki-link schema: allowed types and target validation patterns |
| `layers` | The `.stem` chain in resolution order, root-most first (omitted when empty) |
| `provenance` | Field name → the `.stem` file that defined it (omitted when empty) |
| `source` / `defined_in` | Logical extraction directive / physical `.stem` declaration for a field |
| `hints` | Actionable advice, e.g. run `rootline init` when no schema was found (omitted when empty) |

## Field Extraction

The `--field` flag extracts values by dot-path. Most commands support both simple object paths and array projection:

```bash
# Simple dot-path
rootline describe docs/api/ --field schema.status.values
# ["draft", "review", "published"]

# Array projection: extract a field from each array element
rootline query --from docs/ --field 'rows[].path'
# ["docs/api/auth.md", "docs/api/endpoints.md"]

rootline query --from docs/ --field 'rows[].frontmatter.estado'
# ["Pending", "In Progress", "Completed"]

# Nested array projections
rootline graph docs/ --field 'edges[].source'
# ["docs/api/auth.md", "docs/api/validate.md"]

rootline graph docs/ --field 'broken_links'
# [{"link": "[[NonExistent]]", "source": "docs/api/auth.md"}, ...]
```

Array projection syntax `field[].subfield` works for:
- Query results: `rows[].path`, `rows[].frontmatter.*`, `rows[].derived.*`
- Graph results: `edges[]`, `broken_links[]`

### Repeating `--field`

`--field` is repeatable. One path emits the bare value it resolves to; several emit a JSON array in **flag order**:

```bash
rootline query docs/ --count --field kind
# "rootline/count"

rootline query docs/ --count --field kind --field meta.count
# ["rootline/count",6]

rootline query docs/ --count --field meta.count --field kind
# [6,"rootline/count"]
```

An array is the only shape that keeps N results distinguishable without inventing keys the envelope never had. All paths resolve against the same document, so a failure on the last one fails the whole run rather than emitting a partial array.

### `--field` requires `--output json`

Extraction reads the JSON envelope, so a non-JSON format has nothing to read:

```console
$ rootline stats docs/ -o table --field kind
Error: --field requires --output json: extraction reads the JSON envelope, and "table" has none
```

Previously the flag was applied only inside the JSON writer, so it worked under the default `-o json` and stopped working the moment a caller added `-o table` — silently. Commands that emit no envelope at all (`init`, `new`, `set`, `hooks`, `completion`) reject `--field` for the same reason. See [Output Formats](output.md).

This allows tools, editors, and AI assistants to guide authoring
and extract specific fields without Python postprocessing.

## Multi-pattern Sequences

**Multi-pattern sequences** — get the next value for each pattern:

```bash
rootline describe docs/roadmap/ --field schema.id.next_by_pattern
# → {"O*": "O14", "T*": "T014"}
```

`describe` supports `-o json` and `-o table` only — see [Output Formats](output.md).
