# I5 — Describe Contract: Effective Schema Resolution

**Fecha**: 2026-02-17
**Estado**: Done
**Pregunta**: What is the exact format of the effective schema, and how does `.stem` merge work?
**Motivación**: Problem-driven — addresses "0 unified schemas" and "format fragility"

---

## 1. Merge Algorithm

### 1.1 Two-Phase Resolution

Rootline uses **walk-up discovery + top-down application**, the same model as
Apache `.htaccess` (25+ years proven):

```
Phase 1: DISCOVERY (walk-up)
  Start at target path, walk up collecting .stem files until .git boundary.

Phase 2: APPLICATION (top-down)
  Merge collected .stem files from root to leaf, producing effective schema.
```

Example: `rootline describe docs/epics/E01/F01/S001/`

```
Discovery (walk-up):
  docs/epics/E01/F01/S001/.stem  ← found
  docs/epics/E01/F01/.stem       ← not found (skip)
  docs/epics/E01/.stem            ← not found (skip)
  docs/epics/.stem                ← found
  docs/.stem                      ← found
  .git                            ← STOP

Application (top-down):
  docs/.stem                      (base)
    + docs/epics/.stem            (refine)
      + docs/epics/E01/F01/S001/.stem  (refine)
        = effective schema
```

### 1.2 Stop Signal

The walk-up stops at the **nearest `.git` directory** ancestor. No `root: true`
directive is needed — the repository boundary is the natural scope limit.

If no `.git` is found, the filesystem root is the stop. This handles non-git usage.

### 1.3 Type-Driven Merge Strategy

The merge behavior is determined by the **YAML data type** of each value,
not by field names. This means no hardcoded knowledge of section names is needed.

| YAML Type | Merge Behavior | Rationale |
|-----------|---------------|-----------|
| **map** | Key-level merge (recursive) | Child adds/overrides keys |
| **array** | Replace | Child redefines entirely |
| **scalar** | Replace | Child overrides |
| **null** | Remove inherited key | Controlled escape hatch |

This rule is **universal** — it applies to every section of `.stem` equally.
If new sections are added in the future, their merge behavior is already defined.

#### Why This Strategy

Investigated 12 systems (see §6 State of the Art). Key findings:

- **tsconfig.json** (10+ years): `compilerOptions` (map) merges, `include` (array) replaces.
  Most successful merge model in the ecosystem.
- **ESLint legacy**: Per-directory cascade abandoned in v9 due to debug complexity.
  Lesson: `explain` command is essential to make cascade debuggable.
- **Puppet Hiera**: Offers configurable merge per key (`first`, `unique`, `hash`, `deep`).
  Powerful but complex. Rootline's type-driven approach is simpler.
- **EditorConfig**: `unset` keyword to remove inherited properties. Rootline uses `null`.

#### Merge Examples

Parent value + Child value → Result:

```yaml
# Maps: key-level merge
parent: { a: 1, b: 2 }
child:  { b: 3, c: 4 }
result: { a: 1, b: 3, c: 4 }    # a inherited, b overridden, c added

# Arrays: replace
parent: [draft, review, published]
child:  [Pending, In Progress, Completado]
result: [Pending, In Progress, Completado]   # parent values gone

# Scalars: replace
parent: true
child:  false
result: false

# Null: remove
parent: { a: 1, b: 2 }
child:  { b: null }
result: { a: 1 }                 # b removed
```

---

## 2. Describe JSON Contract

### 2.1 Input

```bash
rootline describe <path>
rootline describe <path> --field <dot-path>
```

`<path>` is a directory. Rootline resolves the effective schema for that directory.

### 2.2 Output Shape

```json
{
  "version": 1,
  "kind": "rootline/describe",
  "path": "docs/prd/",
  "applies": [
    "docs/.stem",
    "docs/prd/.stem"
  ],
  "scope": {
    "match": "*.md"
  },
  "schema": {
    "Fecha": {
      "type": "string",
      "required": true,
      "source": "docs/.stem"
    },
    "Estado": {
      "type": "enum",
      "values": ["📋 Especificado", "✅ Completado", "❌ Obsoleto"],
      "required": true,
      "source": "docs/prd/.stem"
    },
    "Tipo": {
      "type": "enum",
      "values": [
        "servicio-docker",
        "modulo-sistema",
        "modulo-infraestructura",
        "operacion-sistema",
        "operacion-documentacion",
        "servicio-k8s"
      ],
      "required": true,
      "source": "docs/prd/.stem"
    },
    "Prioridad": {
      "type": "enum",
      "values": ["Alta", "Media", "Baja", "CRÍTICA"],
      "source": "docs/prd/.stem"
    },
    "Prerequisitos": {
      "type": "string",
      "source": "docs/prd/.stem"
    },
    "Dependencias": {
      "type": "string",
      "source": "docs/prd/.stem"
    }
  },
  "validate": [
    {
      "rule": "requires",
      "if": { "Estado": "✅ Completado" },
      "then": { "fields": ["Fecha"] },
      "source": "docs/prd/.stem"
    }
  ],
  "derive": {},
  "state": {},
  "links": {}
}
```

### 2.3 Key Design Decisions in the Contract

#### `source` field

Every schema field and validation rule includes `source` — the `.stem` file that
defined or last overrode it. This is critical for debuggability (the ESLint lesson).

Without `source`, answering "why does this directory require field X?" requires
manually inspecting all ancestor `.stem` files.

#### `applies` array

Ordered list of `.stem` files that were merged, from root to leaf.
This is the trace of the merge — the minimal `explain` for `describe`.

#### `--field` extraction

```bash
rootline describe docs/prd/ --field schema.Tipo.values
# → ["servicio-docker", "modulo-sistema", ...]

rootline describe docs/prd/ --field schema.Estado.values
# → ["📋 Especificado", "✅ Completado", "❌ Obsoleto"]
```

This enables external consumers (Claude Code hooks, CI scripts) to query the
effective schema without parsing `.stem` files or understanding merge logic.

**Consumer example** — a governance hook that validates PRD type:

```bash
valid_types=$(rootline describe docs/prd/ --field schema.Tipo.values)
# Instead of hardcoding: ["servicio-docker", "modulo-sistema", ...]
```

### 2.4 Validate Section in Describe Output

The `validate` array is the **concatenation** of all ancestor validation rules,
ordered from root to leaf. Rules reference the effective schema, not hardcoded values.

Initially, validation rules are structural only:

| Rule | Parameters | What it checks |
|------|-----------|----------------|
| `non_empty` | `field` | Field exists and is not empty string |
| `enum` | (none — references schema) | Field value is in `schema.<field>.values` |
| `requires` | `if`, `then.fields` | If condition matches, listed fields must exist |
| `exists` | `field` | Field is present (even if empty) |

Parametric rules (format, max_length, pattern) are deferred.

---

## 3. Real `.stem` Examples (from homeserver analysis)

### 3.1 `docs/.stem` — Root

```yaml
version: 1

scope:
  match: "*.md"

schema:
  Fecha:
    type: string
    required: true
```

Only `Fecha` is universal across all `docs/` subdirectories.

### 3.2 `docs/prd/.stem` — PRD Documents

```yaml
version: 1

schema:
  Estado:
    type: enum
    values:
      - "📋 Especificado"
      - "✅ Completado"
      - "❌ Obsoleto"
    required: true
  Tipo:
    type: enum
    values:
      - servicio-docker
      - modulo-sistema
      - modulo-infraestructura
      - operacion-sistema
      - operacion-documentacion
      - servicio-k8s
    required: true
  Prioridad:
    type: enum
    values: [Alta, Media, Baja, CRÍTICA]
  Prerequisitos:
    type: string
  Dependencias:
    type: string

validate:
  - rule: requires
    if: { Estado: "✅ Completado" }
    then: { fields: [Fecha] }
```

### 3.3 `docs/epics/.stem` — Epic-Level Documents

```yaml
version: 1

schema:
  Estado:
    type: enum
    values: [Activa, Completada, Diferida]
    required: true
  Cliente:
    type: string
    default: "Platform Owner"
```

### 3.4 `docs/epics/E01/F01/S001/.stem` — Task-Level Documents

```yaml
version: 1

schema:
  Estado:
    type: enum
    values: [Pending, In Progress, Completado]
    required: true
  Tipo:
    type: enum
    values:
      - servicio-docker
      - modulo-sistema
      - operacion-sistema
    required: true
  In:
    type: string
    required: true
  Out:
    type: string
    required: true
```

### 3.5 `docs/research/.stem` — Research Documents

```yaml
version: 1

schema:
  Contexto:
    type: string
    required: true
  Esfuerzo:
    type: string
```

### 3.6 Effective Schema Comparison

| Path | Fecha | Estado | Tipo | Source of Estado |
|------|-------|--------|------|------------------|
| `docs/prd/` | required | 3 values (PRD states) | 6 values | `docs/prd/.stem` |
| `docs/epics/E01/F01/S001/` | required | 3 values (Task states) | 3 values | `S001/.stem` (replaces epics) |
| `docs/research/` | required | — (not defined) | — | — |

---

## 4. Edge Cases Resolved

### EC-1: Conflicting Enum Values Between Parent and Child

**Scenario**: Parent defines `Estado: [Activa, Completada]`, child defines
`Estado: [Pending, In Progress, Completado]`.

**Resolution**: `values` is an array → **replace**. The child's values completely
replace the parent's. Validation rules use the effective schema, so `enum` checks
validate against the child's values.

**No conflict possible** — there's only one effective set of values per directory.

### EC-2: Child Wants to Relax a Parent Rule

**Scenario**: Parent requires `owner` when `Estado = published`. Child's documents
don't have an owner field.

**Resolution**: Child redefines `Estado.values` without `published`. The parent's
conditional rule (`if: { Estado: published }`) never activates because `published`
is not a valid value in the child's effective schema.

**No rule disabling mechanism needed** — schema refinement solves relaxation.

### EC-3: links.allowed and Governance Direction

**Scenario**: Parent allows `[decision, reference]`. Child adds `[dependency]`.

**Resolution**: `allowed` is an array → **replace**. If child needs all three:
`allowed: [decision, reference, dependency]` (explicit). If child needs fewer:
`allowed: [reference]` (also explicit).

**Concatenation was rejected** because it would allow children to widen permissions,
violating top-down governance.

### EC-4: Walk from "Last Child" (pwd at leaf)

**Scenario**: User runs `rootline describe .` from `docs/epics/E01/`.

**Resolution**: Walk-up discovery finds all ancestor `.stem` files automatically.
The user doesn't need to know where the root `.stem` is.

### EC-5: No `.stem` Ancestors

**Scenario**: `rootline describe some/path/` where no `.stem` exists in any ancestor.

**Resolution**: Describe returns an empty effective schema:
```json
{
  "version": 1,
  "kind": "rootline/describe",
  "path": "some/path/",
  "applies": [],
  "scope": {},
  "schema": {},
  "validate": [],
  "derive": {},
  "state": {},
  "links": {}
}
```

No error — an unconstrained directory is valid. It just has no rules.

---

## 5. Decisions

| # | Decision | Result | Rationale |
|---|----------|--------|-----------|
| D12 | Walk direction | Walk-up discovery + top-down merge | Same as .htaccess. "Walk-up vs walk-down" is false dichotomy — all systems do both. |
| D13 | Stop signal | `.git` boundary | No `root: true` needed. Repository = natural scope. |
| D14 | Merge strategy | Type-driven (map→merge, array→replace, scalar→replace, null→remove) | Universal rule, no field-name-specific logic. Future sections auto-inherit behavior. |
| D15 | Validate section | Array, concatenated from ancestors | Structural rules initially (non_empty, enum, requires, exists). No parametric conflicts. |
| D16 | Rule disabling | Not initially | Schema refinement solves relaxation. No real case in homeserver requires disabling. |
| D17 | `source` in describe | Every field includes source `.stem` path | ESLint lesson: cascade without traceability → abandoned. Explainability is essential. |
| D18 | `--field` extraction | Dot-path into describe JSON | Enables hooks and CI to query effective schema without parsing .stem files. |

---

## 6. State of the Art (Summary)

Full research conducted 2026-02-17. Systems analyzed:

| System | Walk Direction | Merge | Stop | Escape |
|--------|---------------|-------|------|--------|
| **EditorConfig** | Walk-up, merge all | Last-write-wins | `root=true` | `unset` |
| **ESLint legacy** | Walk-up, merge all | Cascade + overrides | `root: true` | `"off"` by rule ID |
| **ESLint v9 flat** | Single file | Array cascade | N/A | N/A |
| **Prettier** | Walk-up, first wins | No merge | None | N/A |
| **Ruff** | Walk-up, first wins | Explicit `extend` | None | `ignore` codes |
| **Biome** | Walk-up, nearest | `extends` + `root` | `root: false` | N/A |
| **Apache .htaccess** | Walk-down (applied) | Per-directive override | N/A | `AllowOverride` |
| **.gitattributes** | Walk-down | Pattern-based override | N/A | N/A |
| **Terragrunt** | Explicit include | Configurable (4 strategies) | N/A | N/A |
| **Vale** | Walk-up | Multi-value merge, single-value replace | None | `Rule = NO` |
| **tsconfig.json** | Explicit extends | Maps merge, arrays replace | N/A | N/A |
| **Puppet Hiera** | Hierarchy-based | 4 strategies (first, unique, hash, deep) | N/A | `knockout_prefix` (buggy) |

**Key insight**: ESLint abandoned per-directory cascade in v9 because debugging was
impossible. Rootline's `explain` command directly addresses this gap. The `.stem`
model is viable **only if** `explain` and `describe` make the cascade transparent.

---

## 7. Pain Points Addressed

From analysis of 250+ homeserver sessions:

| Pain Point | Frequency | How Rootline Addresses |
|-----------|-----------|----------------------|
| State inconsistency between levels | 64 mentions | `derive` (I3) — computed from children, not manual |
| Information scattered across docs | 63 mentions | `.stem` = single source of truth for schema |
| Schema/type fragility | 58 mentions | `describe --field schema.Tipo.values` replaces hardcoded lists |
| Silent query failures | 34 mentions | `validate` returns warnings for non-compliant docs |
| Governance hook false positives | 31 mentions | Hooks query `describe` instead of hardcoding valid values |

**Coverage**: `validate` + `describe` cover ~70% of pain points.
Remaining ~30% requires `derive` (I3) for computed state across hierarchy levels.

---

## 8. What This Investigation Does NOT Cover

Explicitly deferred:

| Topic | Investigation | Why Deferred |
|-------|--------------|-------------|
| Derivation functions | I3 | Separate concern — what functions exist and how they compute |
| Plugin architecture | I2 | Future extensibility, no immediate impact |
| Explain tracing depth | I4 | `source` in describe is sufficient initially |
| Cache strategy | I6 | <200 files, performance not a concern |
| Parametric validation rules | Deferred | format, max_length, pattern — may require map-with-IDs for validate |
| Rule disabling mechanism | Deferred | If open-source adoption requires it, add `null` or similar |
