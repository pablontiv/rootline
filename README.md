# Rootline

> **Status**: Core engine functional — validation, query, describe, and CLI complete.
> Features marked with **(planned)** are designed but not yet implemented.

Rootline is a **file-based database and constraint engine** for structured documentation.

It treats the filesystem itself as the model:

- **directories are tables**
- **files are records**
- **metadata is extracted, validated, and derived**
- **structure is inherited along the directory tree**
- **relationships are first-class data**

Rootline is designed for projects that already use Markdown,
but need consistency, queryability, explainability, and machine-readable structure
— without moving content into a platform or proprietary system.

---

## Core idea

Documentation already has structure.

Rootline makes that structure **explicit**, **inherited**, and **queryable**.

The directory tree defines hierarchy.
Rules flow from parent to child.
State is derived, not manually maintained.
Links define relationships, not navigation.

Rootline does not render documentation.
It **models** it.

---

## The `.stem` file

Rootline uses a single dotfile called **`.stem`**.

A `.stem` file may appear in **any directory**.

Rootline resolves configuration using **walk-up discovery + top-down merge**:

1. From the target path, walk **up** collecting `.stem` files until the repository root (`.git`)
2. Merge them **top-down** (parent → child), producing an effective schema

A parent directory defines the *stem*.
Children grow from it.

This model is inspired by `.htaccess`, but designed for structured content.

### Merge rules

Merge behavior is determined by **YAML data type**, not by field names:

| YAML type | Behavior | Example |
|-----------|----------|---------|
| **map** | Key-level merge | Child adds or overrides keys |
| **array** | Replace | Child redefines entirely |
| **scalar** | Replace | Child overrides value |
| **null** | Remove | Child removes inherited key |

No section-specific logic. The same rules apply everywhere.

---

## What `.stem` defines

A `.stem` file is written in YAML and may declare:

- **Schemas** for document metadata
- **Validation rules** for consistency
- **Derived fields** computed from existing data **(planned)**
- **Derived state** inferred from structure and relationships **(planned)**
- **Scope rules** that define which files it applies to
- **Link semantics and constraints** **(planned)**

All definitions are inherited unless explicitly overridden.

---

## Example tree

```
docs/
├── .stem
├── design/
│   ├── .stem
│   ├── overview.md
│   └── decisions.md
└── api/
    ├── endpoints.md
    └── errors.md
```

- `docs/.stem` defines the baseline constraints
- `design/.stem` refines them
- `api/` inherits unchanged

---

## Example `.stem`

```yaml
version: 1

scope:
  match: "*.md"

schema:
  title:
    type: string
    required: true

  status:
    type: enum
    values: [draft, review, published]
    default: draft

validate:
  - field: title
    rule: non_empty

  - rule: requires
    if:
      status: published
    then:
      fields: [owner]

derive:  # (planned — pipeline slot reserved, not yet functional)
  slug:
    from: title
    using: slugify

state:  # (planned — pipeline slot reserved, not yet functional)
  visibility:
    derive:
      when:
        status: published
      then: public

```

> Derivation functions (`slugify`, `when/then`, etc.) are planned. The pipeline slot and data structures are reserved.

> Link constraints (`links.allowed`) are planned.

---

## Markdown as records

Rootline indexes **Markdown files**.

- YAML frontmatter is treated as structured metadata
- The document body remains free-form
- Derived fields and state are **never written back** to files

Markdown remains human-first.
Rootline adds a machine-readable layer on top.

---

## Links as relationships **(planned)**

> Link extraction and relationship modeling are planned.

Rootline will treat links as **data**, not navigation.

Wiki-style links in Markdown:

```
[[auth-model]]
[[decision:rate-limiting]]
```

will be extracted as **explicit relationships** between records.

Planned capabilities:
- queried as first-class data
- validated against constraints

Future exploration:
- state derivation from link graphs
- causal tracing across relationships

Rootline does not render backlinks or graphs.
It indexes and reasons about relationships.

---

## Authoring and introspection

Rootline is not only a validator.

Because `.stem` files fully describe structure and constraints,
Rootline can explain what a valid document looks like *before it exists*.

```bash
rootline describe docs/api/
```

### Describe result

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
      "source": "docs/.stem"
    },
    "status": {
      "type": "enum",
      "values": ["draft", "review", "published"],
      "default": "draft",
      "source": "docs/api/.stem"
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
  "derive": {},
  "state": {},
  "links": {}
}
```

Every field includes `source` — the `.stem` file that defined it.
This makes the merge cascade transparent and debuggable.

The `--field` flag extracts values by dot-path:

```bash
rootline describe docs/api/ --field schema.status.values
# ["draft", "review", "published"]
```

This allows tools, editors, and AI assistants to guide authoring
without inspecting `.stem` files directly.

---

## Querying data

Rootline exposes data through a **declarative query model**.

Queries return **records**, not rendered documents.

### Query result shape

```json
{
  "version": 1,
  "kind": "rootline/query",
  "meta": {
    "count": 1
  },
  "rows": [
    {
      "path": "docs/api/endpoints.md",
      "type": "markdown",
      "frontmatter": {
        "title": "Endpoints",
        "status": "published"
      }
    }
  ]
}
```

### Count result shape

```json
{
  "version": 1,
  "kind": "rootline/count",
  "meta": {},
  "count": 12
}
```

---

## Query request contract

```json
{
  "version": 1,
  "from": "docs/",
  "where": {
    "and": [
      {"eq": ["frontmatter.status", "published"]},
      {"exists": "frontmatter.owner"}
    ]
  },
  "limit": 50
}
```

When `where` contains multiple conditions without an explicit `and` wrapper,
they are combined with `and` semantics (implicit `and`).

### Operators

| Operator | Semantics | Example |
|----------|-----------|---------|
| `eq` | Equals | `{"eq": ["status", "published"]}` |
| `ne` | Not equals | `{"ne": ["status", "draft"]}` |
| `in` | One of | `{"in": ["tipo", ["lxc", "vm"]]}` |
| `contains` | Substring match | `{"contains": ["body", "migration"]}` |
| `exists` | Field is present | `{"exists": "owner"}` |
| `and` | All conditions match | `{"and": [cond1, cond2]}` |

Query functions: `limit`, `count`.

- Queries are **purely declarative**
- No embedded code or scripts
- Stable semantics for automation and AI
- Operator set derived from real consumer analysis

---

## JSON-RPC protocol **(planned)**

Rootline will use **JSON-RPC 2.0** as its interaction protocol via the MCP server.

All core capabilities will be exposed as methods:

- `query`
- `describe`
- `explain`
- `validate`
- `tree`
- `stats`

The query contract above maps directly to JSON-RPC `params`:

### JSON-RPC query example

```json
{
  "jsonrpc": "2.0",
  "id": "q1",
  "method": "query",
  "params": {
    "from": "docs/",
    "where": {
      "eq": ["state.visibility", "public"]
    }
  }
}
```

### Response

```json
{
  "jsonrpc": "2.0",
  "id": "q1",
  "result": {
    "version": 1,
    "kind": "rootline/query",
    "meta": { "count": 1 },
    "rows": []
  }
}
```

---

## Explainability **(planned)**

Rootline will explain **why** a document has a given state,
validation error, or derived value.

```bash
rootline explain docs/api/endpoints.md
```

Explain output will trace:
- which `.stem` files applied
- which rules fired
- which conditions matched

Explainability is a first-class design goal. Currently, `describe` shows the full schema cascade with `source` tracking per field, and `validate` errors include source `.stem` paths.

---

## Extensibility

Rootline is built around **extractors**.

Markdown is the built-in extractor.

The architecture is designed so that other extractors can be added
without changing the core model. Future extractors may include:

- YAML / JSON / TOML files
- MDX
- API specifications (OpenAPI, AsyncAPI)

All extractors feed the same pipeline:
rules by directory, inheritance, validation, derivation, querying.

> LSP integration has been considered but carries very high complexity.
> It is not in scope.

---

## CLI

Rootline ships as a **single static Go binary** with no dependencies.

```bash
# Core
rootline validate [file|--all|--staged]
rootline query --where 'field op value'
rootline describe <path>
rootline tree [path]
rootline stats [path]

# Document lifecycle
rootline init [path]          # Infer .stem from existing documents
rootline new <file>           # Scaffold document from effective schema
rootline fix <file|--all>     # Repair files (add missing fields, fix values)

# Tooling
rootline doctor               # Check .stem configuration health
rootline hooks install|uninstall|status
rootline completion bash|zsh|fish|powershell

# Planned
rootline explain <file>       # Trace derivation chain (stub)
rootline serve                # MCP server (stub)
```

All commands support `--field` for dot-path extraction:

```bash
rootline describe docs/prd/ --field schema.id.next
# 302

rootline query --where 'estado eq Pending' --field path
# docs/epics/E01/F01/S001/T005-deploy-grafana.md
```

All commands produce **stable JSON output**, suitable for:

- CI
- automation
- editors
- tooling
- AI assistants

---

## Versioning

JSON output contracts carry their own version (`"version": 1`), independent of any future release version.
Breaking changes to output schemas require a version bump.

---

## AI-native **(planned)**

Rootline will embed a **Model Context Protocol (MCP)** server.

AI assistants will query Rootline using the same contracts as the CLI,
making documentation a structured, explainable knowledge source.

---

## Philosophy

Rootline does not replace your editor or your documentation.

It adds:

- structure without lock-in
- rules without ceremony
- constraints without platforms
- intelligence without rewriting content

Grow structure from the root.
Follow the line.

---

## Visual identity

See [identity](docs/identity.md).
