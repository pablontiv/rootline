---
estado: Completado
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

## Field Extraction

The `--field` flag extracts values by dot-path:

```bash
rootline describe docs/api/ --field schema.status.values
# ["draft", "review", "published"]
```

This allows tools, editors, and AI assistants to guide authoring
without inspecting `.stem` files directly.
