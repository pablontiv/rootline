---
estado: Completed
---
# Hierarchical Schema (Match-Based)

Rootline v2 uses `match:` annotations on individual schema fields to scope them to specific directory levels. This replaces the v1 `levels:` keyword with a flat, composable approach.

> **Migration**: v1 `.stem` files with `levels:` can be converted via `rootline migrate --from-levels`.

## .stem Configuration

```yaml
version: 2
scope:
  match: "*.md"
schema:
  estado:
    type: enum
    required: true
    values: [Completed, Pending, Specified]
  id:
    type: sequence
    match:
      "E*": { prefix: E, digits: 2 }
      "F*": { prefix: F, digits: 2 }
      "S*": { prefix: S, digits: 3 }
      "T*": { prefix: T, digits: 3 }
  ejecutable_en:
    type: string
    required:
      match: "T*"
    match: "T*"
```

Fields without `match:` apply to all records. Fields with `match:` apply only to records whose path matches the pattern.

## Match Forms

The `match:` annotation supports three forms:

**String** — single glob pattern:
```yaml
ejecutable_en:
  type: string
  match: "T*"
```

**Array** — multiple patterns:
```yaml
tipo:
  type: enum
  match: ["F*", "S*"]
```

**Map** — per-pattern configuration (used for sequence fields):
```yaml
id:
  type: sequence
  match:
    "E*": { prefix: E, digits: 2 }
    "T*": { prefix: T, digits: 3 }
```

## Conditional Required

The `required` field also supports match scoping. This makes a field required only for records matching the pattern:

```yaml
ejecutable_en:
  type: string
  required:
    match: "T*"
  match: "T*"
```

Here `ejecutable_en` only exists for `T*` records and is required only for them.

## How Resolution Works

When validating or querying a record, `ResolveForRecord` filters schema fields by matching the record's path against each field's `match:` pattern. Fields without `match:` always apply. This means a single root `.stem` can define the schema for an entire hierarchy without child `.stem` files.

## Benefits

- **Single file**: One `.stem` defines all levels — no redundant child `.stem` files.
- **Composable**: Each field independently declares its scope.
- **Debuggable**: `rootline describe` shows which fields apply at each path, with their source.
