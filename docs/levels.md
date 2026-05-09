---
estado: Completed
---
# Hierarchical Schema (Match-Based)

Rootline uses `match:` annotations on individual schema fields to scope them to specific directory levels. This provides a flat, composable approach to defining hierarchical schemas.

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

## Hierarchical Constraints (Monotonic Narrowing)

When multiple `.stem` files exist in a hierarchy (parent and child), they form a **layered constraint system**. A child `.stem` can **narrow** a parent constraint (make it stricter), but cannot **widen** it (make it looser). This ensures schema consistency from root to leaf:

- **Type narrowing** — child can narrow `string` to `enum`, but not widen `enum` to `string`
- **Required narrowing** — child cannot relax `required: true` to `required: false`
- **Enum narrowing** — child's enum values must be a subset of parent's values
- **Severity narrowing** — child cannot reduce severity level

For example:
- Parent: `estado: { type: string }`
- Child: `estado: { type: enum, values: [draft, active] }` ✓ Valid narrowing
- Child: `estado: { type: string, required: false }` ✗ Invalid (widening if parent required)

When a child `.stem` violates monotonic constraints, `rootline validate --all` detects this in the **monotonic-violations** stemhealth check.

## Benefits

- **Single file**: One `.stem` defines all levels — no redundant child `.stem` files.
- **Composable**: Each field independently declares its scope.
- **Debuggable**: `rootline describe` shows which fields apply at each path, with their source.
- **Safe evolution**: Monotonic narrowing prevents accidental schema loosening.
