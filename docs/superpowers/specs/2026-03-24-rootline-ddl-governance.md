# Spec: Rootline as a DDL and Governance Layer

## Context
Rootline is the **Data Definition Language (DDL)** for the filesystem-based database ecosystem. Just as SQL defines tables and columns, Rootline defines directories as tables and `.stem` files as schemas.

## Objectives
1. **Metaphor Alignment**: Update documentation to reflect the DDL/Database role.
2. **Schema Autonomy**: Rootline is the sole source of truth for `.stem` structure.
3. ~~**Semantic Tagging**: Allow `.stem` to include metadata "tags" that consumer tools (like Kedral) can use to identify fields without knowing their localized names.~~ → **Implemented** as `domain` types (see below).

## Status

| Objective | Status | Notes |
|-----------|--------|-------|
| 1.1 Documentation | **Pending** | README partially uses the DDL metaphor but no formal update |
| 1.2 System Interoperability Tags | **Implemented** | Superseded by domain types — see `2026-03-26-stem-domain-types-design.md` |
| 1.3 Enhanced `rootline init` | **Pending** | Init does not infer domains automatically yet |

## Changes

### 1.1 Documentation (README.md)
Update the core pitch to use the database metaphor:
- **Directory** = Table
- **Markdown File** = Record (Row)
- **Frontmatter** = Columns
- **.stem** = DDL Schema (The "Source of Truth")

### ~~1.2 System Interoperability Tags~~ → Implemented as Domain Types

This objective was implemented in `feat(rules): add domain semantic types for .stem schema fields` (commit `6593c1c`). The design evolved from passive `system_tags` annotations to `domain` — a semantic type layer modeled after SQL DOMAINs and JSON Schema `format`.

What was proposed:
```yaml
fields:
  mi_estado_personalizado:
    type: enum
    system_tags: ["lifecycle_state"]
    values: [abierto, cerrado]
```

What was implemented (more powerful):
```yaml
schema:
  mi_estado_personalizado:
    domain: lifecycle_state        # semantic type — implies type: enum
    values: [abierto, cerrado]
```

Key differences from the original proposal:
- **`domain` instead of `system_tags`**: One semantic type per field (not an array of tags). Follows SQL DOMAIN / JSON Schema `format` pattern.
- **Type inference**: `domain: lifecycle_state` implies `type: enum` — no need to declare both.
- **Virtual alias resolution**: Queries like `lifecycle_state == "activo"` resolve to the actual field name automatically.
- **Scope-aware**: Fields with the same domain at different hierarchy levels (via `match:` patterns) resolve correctly per record.
- **12 core domains**: `lifecycle_state`, `record_type`, `identifier`, `title`, `created_date`, `due_date`, `owner`, `parent_ref`, `priority`, `description`, `confidence`, `source`.
- **Custom extension**: `domain: acme/sprint_velocity` for user-defined domains.

Full design: `docs/superpowers/specs/2026-03-26-stem-domain-types-design.md`

### 1.3 Enhanced `rootline init`
Refine the inference engine to detect existing language patterns and generate a `.stem` that reflects the user's naming while maintaining structural integrity.
