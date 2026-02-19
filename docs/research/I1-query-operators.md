# I1: Query Operators — Investigation Result

**Status**: Done
**Date**: 2026-02-17
**Method**: Operators derived from real consumer code analysis, not theoretical adoption.

---

## 1. Methodology

Instead of adopting an existing operator set (OData, SQL, Dataview) and trimming it,
this investigation analyzed the **18 actual consumers** in the homeserver automation
codebase that query markdown documents. Every operator in the spec maps to at least
one real code pattern. No operator was included without evidence of need.

Source: Exploration of `.claude/skills/`, `.claude/hooks/`, `.claude/settings.json`
performed 2026-02-17.

---

## 2. Consumer Inventory

### Query patterns by consumer

| Consumer | Fields queried | Operations used | Pattern |
|----------|---------------|-----------------|---------|
| `/roadmap pending` | estado, tipo, filename, path, heading | eq, in, glob, regex extract | Python inline (split on `:`) |
| `/roadmap view` | estado | eq (Completado vs other), count | LLM reads frontmatter |
| `/service` | tipo, estado | eq (substring grep) | `grep -q "servicio-docker"` |
| `/module` | tipo, estado | eq (substring grep) | `grep -q "modulo-sistema"` |
| `/operation` | tipo, estado | eq (substring grep) | `grep -q "operacion-sistema"` |
| `/instance` | tipo, estado | in (3 values), eq | `grep -qE "lxc\|vm\|modulo-infraestructura"` |
| `/instance` (tipo detect) | `Tipo:` line | extract value | `grep -o "Tipo:.*"` |
| `/host-task` | tipo | eq (substring) | `grep -q "host-script"` |
| `/instance-task` | tipo | eq (substring) | `grep -q "instance-script"` |
| `/drift` | estado | exists + eq | prose: "verify estado: Completado" |
| `/prd` | filename | sequence (max+1) | `ls PRD-*.md \| sort -V \| tail -1` |
| `/rca` | body content | contains | `grep -l "keyword" docs/prd/` |
| `/epic-guide` | body content | contains | `grep -rl "keyword" docs/prd/` |
| `/prep-ralph` | filename, body | contains + and | `grep -l X \| xargs grep -l Y` |
| Write hook | estado, tipo | enum validation | LLM checks allowlist |
| SubagentStop hook | PRD section | section exists | LLM reads "Criterios de Aceptacion" |

### 4 distinct regex patterns for Task file discovery

1. `T[0-9][0-9][0-9]-*.md` — glob, `/roadmap pending`
2. `T${ID}-*.md` — shell glob by ID, `/service` `/module` `/operation` `/instance`
3. `T*-$ARGUMENTS*.md` — fuzzy name match, `/host-task` `/instance-task` `/drift`
4. `T` + digits in filename — natural language, Write hook

Rootline eliminates all four. File discovery becomes `rootline query`.

---

## 3. Operator Spec

### Comparison operators

| Operator | Semantics | JSON syntax | Real consumer |
|----------|-----------|-------------|---------------|
| `eq` | Field equals value | `{"eq": ["field", "value"]}` | 6 skills (tipo check), `/roadmap pending` (estado) |
| `ne` | Field not equals value | `{"ne": ["field", "value"]}` | Implicit in "everything except Completado" |
| `in` | Field is one of values | `{"in": ["field", ["v1", "v2"]]}` | `/instance` (3 tipos), `/roadmap pending` (2 estados) |
| `contains` | Field contains substring | `{"contains": ["field", "substr"]}` | `/rca`, `/epic-guide`, `/prep-ralph`, fuzzy find |
| `exists` | Field is present and non-null | `{"exists": "field"}` | `/drift` (verify campo has value) |

### Logical operator

| Operator | Semantics | JSON syntax | Real consumer |
|----------|-----------|-------------|---------------|
| `and` | All conditions must match | `{"and": [cond1, cond2]}` | `/prep-ralph` (two-stage filter), every skill (tipo AND estado) |

### Query functions

| Function | Semantics | JSON syntax | Real consumer |
|----------|-----------|-------------|---------------|
| `limit` | Return at most N results | `"limit": N` | 6 skills (`head -1`) |
| `count` | Return count instead of rows | `"count": true` | `/roadmap view` (completado vs total) |

---

## 4. JSON Query Contract

### Request

```json
{
  "version": 1,
  "from": "docs/epics/",
  "select": ["path", "frontmatter.estado", "frontmatter.tipo"],
  "where": {
    "and": [
      {"eq": ["frontmatter.tipo", "servicio-docker"]},
      {"in": ["frontmatter.estado", ["Pending", "Especificado"]]}
    ]
  },
  "limit": 50,
  "cursor": null,
}
```

### Implicit `and`

When `where` contains multiple conditions at the top level without an explicit
`and` wrapper, they are combined with `and` semantics:

```json
{
  "where": {
    "eq": ["frontmatter.tipo", "servicio-docker"],
    "in": ["frontmatter.estado", ["Pending", "Especificado"]]
  }
}
```

This is equivalent to the explicit `and` form. Reduces verbosity for the common case.

### Count request

```json
{
  "version": 1,
  "from": "docs/epics/",
  "where": {
    "eq": ["frontmatter.estado", "Completado"]
  },
  "count": true
}
```

### Count response

```json
{
  "version": 1,
  "kind": "rootline/count",
  "meta": {},
  "count": 12
}
```

### Response (standard query)

No changes from existing README contract. The `rows` array shape is unchanged.

---

## 5. CLI Flag Mapping

```bash
# eq
rootline query --where 'estado eq Completado'

# ne
rootline query --where 'estado ne Completado'

# in
rootline query --where 'tipo in servicio-docker,modulo-sistema'

# contains (field)
rootline query --where 'title contains backup'

# contains (body)
rootline query --where 'body contains infraestructura'

# exists
rootline query --where 'estado exists'

# and (multiple --where flags = implicit and)
rootline query --where 'tipo eq servicio-docker' --where 'estado in Pending,Especificado'

# count
rootline query --where 'estado eq Completado' --count

# limit
rootline query --where 'estado eq Pending' --limit 10
```

Multiple `--where` flags are combined with `and`. This matches the implicit `and`
semantics of the JSON contract and covers 100% of real consumer patterns without
requiring explicit `--and` syntax.

---

## 6. Field Extraction (`--field`)

All commands that produce JSON support `--field` for dot-path extraction:

```bash
# Extract a single value
rootline describe docs/prd/ --field schema.id.next
# output: 302

# Extract nested value
rootline query --where 'id eq T005' --field frontmatter.estado
# output: Pending

# Without --field (default: full JSON)
rootline describe docs/prd/
# output: { "version": 1, "schema": { ... } }
```

### Rationale

Eliminates `jq` as a dependency for consumers. Real consumers are:
- Claude Code skills (LLM reads output directly)
- CI pipelines (GitHub Actions `fromJSON()`)
- Shell scripts (need single values, not full JSON)

Follows established patterns:
- `kubectl get pod -o jsonpath='{.status.phase}'`
- `docker inspect --format '{{.State.Status}}'`

---

## 7. Excluded Operators

| Operator | Why excluded |
|----------|-------------|
| `gt`, `lt`, `ge`, `le` | No consumer compares numerically or by date |
| `or` | No consumer does "this OR that" across different fields. `in` covers multiple values on the same field. |
| `not` | `ne` covers simple negation. No consumer needs `not(contains(...))` |
| `startswith`, `endswith` | `contains` + scope patterns cover all real cases |
| `order_by` | Only `/prd` used sorting (`sort -V`), and that's a sequence concern — belongs in `describe` contract (I5), not query operators |

### Future expansion

If a real consumer emerges that needs `gt`/`lt` (e.g., "tasks created after date X"),
operators can be added in a backward-compatible way. The JSON contract is versioned.

---

## 8. Sequence Type (cross-reference I5)

The `/prd` skill's `ls docs/prd/PRD-*.md | sort -V | tail -1` pattern is not a query.
It derives the **next valid ID** in a collection. This is a property of the directory
schema, exposed via `rootline describe`:

```yaml
# .stem for docs/prd/
schema:
  id:
    type: sequence
    pattern: "PRD-{n}"
    extract_from: filename
```

```bash
rootline describe docs/prd/ --field schema.id.next
# output: 302
```

Full spec deferred to I5 (Describe contract).

---

## 9. Edge Cases

### Null handling

- `eq` against a field that doesn't exist in a document: **no match** (not an error)
- `ne` against a missing field: **matches** (absent != value)
- `exists` is the explicit way to check for field presence

### Nested field access

Fields use dot notation: `frontmatter.estado`, `state.visibility`, `derived.slug`.
Top-level shortcuts are supported for common fields:

| Shortcut | Expands to |
|----------|-----------|
| `estado` | `frontmatter.estado` |
| `tipo` | `frontmatter.tipo` |
| `path` | `path` (already top-level) |
| `body` | document body text (not frontmatter) |

Shortcuts are defined per `.stem` scope. Unknown shortcuts are treated as literal
field paths.

### Body search

`contains` on `body` searches the markdown content excluding frontmatter.
This replaces `grep -l "keyword" docs/prd/` patterns.

### Array fields

If a frontmatter field contains an array (e.g., `tags: [infra, docker]`):
- `eq` matches if the array contains the value
- `contains` matches if any element contains the substring
- `in` matches if the array intersects with the provided values

---

## 10. Summary

| Aspect | Result |
|--------|--------|
| Operators | 5 comparison (`eq`, `ne`, `in`, `contains`, `exists`) + 1 logical (`and`) |
| Query functions | `count`, `limit` |
| CLI pattern | `--where 'field op value'` (multiple = implicit and) |
| Field extraction | `--field dot.path` on all commands |
| Excluded | 8 operators (no real consumer needs them) |
| Coverage | 100% of 18 real consumer patterns |
| Contract version | `1` (integer in JSON, backward-compatible expansion possible) |
