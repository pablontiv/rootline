# Spec: Rootline as a DDL and Governance Layer

## Context
Rootline is the **Data Definition Language (DDL)** for the filesystem-based database ecosystem. Just as SQL defines tables and columns, Rootline defines directories as tables and `.stem` files as schemas.

## Objectives
1. **Metaphor Alignment**: Update documentation to reflect the DDL/Database role.
2. **Schema Autonomy**: Rootline is the sole source of truth for `.stem` structure.
3. **Semantic Tagging**: Allow `.stem` to include metadata "tags" that consumer tools (like Kedral) can use to identify fields without knowing their localized names.

## Changes

### 1.1 Documentation (README.md)
Update the core pitch to use the database metaphor:
- **Directory** = Table
- **Markdown File** = Record (Row)
- **Frontmatter** = Columns
- **.stem** = DDL Schema (The "Source of Truth")

### 1.2 System Interoperability Tags
Add a `system_tags` property to fields in the `.stem` spec. This allows tools to find "The Status Field" even if it's named "Estado" or "Situação".
```yaml
fields:
  mi_estado_personalizado:
    type: enum
    system_tags: ["lifecycle_state"]
    values: [abierto, cerrado]
```

### 1.3 Enhanced `rootline init`
Refine the inference engine to detect existing language patterns and generate a `.stem` that reflects the user's naming while maintaining structural integrity.
