---
estado: Completed
---
# Hierarchical Levels

Rootline allows declaring schemas for an entire directory tree in a single root `.stem` file using the `levels:` keyword.

This eliminates the need for redundant child `.stem` files and enables structural validation of the hierarchy.

## .stem Configuration

```yaml
levels:
  epic:
    match: "E*"           # Glob pattern for directory name
    children: [feature]   # Allowed child levels
    schema:
      id:
        type: sequence
        prefix: E
        digits: 2
  feature:
    match: "F*"
    children: [story]
    schema:
      tipo:
        type: enum
        values: [software, infra, docs]
  story:
    match: "S*"
    children: [task]
  task:
    match: "T*"
    children: []          # Leaf level: no subdirectories allowed
    schema:
      ejecutable_en:
        type: string
        required: true
```

## How it Works

1. **Expansion**: When a record at `E01/F01/S001/T001.md` is processed, Rootline matches each path component against the `levels` definitions.
2. **Virtual Merge**: It generates "virtual" rules for each level and merges them top-down.
3. **Nesting Validation**: `rootline validate` checks if the actual directory structure follows the `children` constraints.

### Nesting Errors

If a task file is placed directly under an epic directory, and the epic level only allows `feature` children, Rootline will report a **nesting violation**:

```bash
rootline validate docs/epics/E01/T001.md
# [nesting] level 'task' is not an allowed child of 'epic' (allowed: [feature])
```

## Benefits

- **Centralization**: Change the schema for all tasks in one place.
- **Structural Integrity**: Ensure the project follows the Epic -> Feature -> Story -> Task architecture.
- **Clarity**: The `.stem` file becomes a formal contract of the project's structure.
