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
      "source": "docs/.stem"
    },
    "status": {
      "type": "enum",
      "values": ["draft", "review", "published"],
      "default": "draft",
      "excludes": { "match": "*/README.md" },
      "source": "docs/api/.stem"
    }
  },
  "levels": {
    "task": {
      "match": "T*",
      "children": [],
      "schema": { "tipo": { "type": "string" } }
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
  "structural": {
    "require_index": true,
    "min_children": 1
  }
}
```

Every field includes `source` — the `.stem` file that defined it.
This makes the merge cascade transparent and debuggable.

The `layers` array lists all `.stem` files in resolution order (root-to-leaf).
The `provenance` map shows which `.stem` file defined each field, enabling
field-level traceability of the schema inheritance chain.

### Sections

| Section | Purpose |
|---------|---------|
| `schema` | Field definitions with types, enums, defaults, and sequence auto-numbering |
| `validate` | Explicit validation rules (non_empty, exists, requires, enum) |
| `derive` | Per-record expressions evaluated via expr-lang (e.g., `slugify(title)`) |
| `aggregate` | Bottom-up expressions for index files (e.g., `len(descendants)`) |
| `links` | Wiki-link schema: allowed types and target validation patterns |
| `structural` | Directory constraints: require_index, min/max_children |

## Field Extraction

The `--field` flag extracts values by dot-path:

```bash
rootline describe docs/api/ --field schema.status.values
# ["draft", "review", "published"]
```

This allows tools, editors, and AI assistants to guide authoring
without inspecting `.stem` files directly.
