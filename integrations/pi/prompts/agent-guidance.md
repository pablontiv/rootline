# Rootline Project Context

This project uses **Rootline** for schema-governed documentation. Rootline treats the filesystem as a database: directories are tables, files are records, metadata comes from YAML frontmatter, and structure is inherited via `.stem` schema files.

## When to Use Rootline Tools

- **Query records by metadata**: Use `rootline-query` tool (not grep, Read, or file scanning)
  - Example: Find all records with `estado: pendiente` or `tipo: feature`
  - Supports field filtering, expression queries, and compact CSV/JSON output

- **Validate records against schema**: Use `rootline-validate` tool
  - Checks required fields, enum constraints, type validity
  - Supports `--all` to validate entire directories
  - Provides structured error reports with actionable fixes

- **Inspect schema and fields**: Use `rootline-describe` tool
  - See what `.stem` files govern a directory
  - Review field types, validation rules, and aggregation logic
  - Understand scope and inheritance chains

- **Explore structure**: Use `rootline-tree` tool
  - Browse directory structure with metadata at a glance
  - Faster than manual file reads for understanding layout

## Quick Start

If the project has no `.stem` files yet:
```bash
rootline init --directory docs/epics/
```

This creates initial `.stem` schema files from existing records.

## Don't Use

- `grep` or file reads for finding records (use `rootline-query` instead)
- Manual YAML parsing (Rootline handles this consistently)
- `ls -la` for understanding schema (use `rootline-describe` instead)

---

**Note**: This guidance is injected before each agent run and auto-detected based on project state. If you see this message in a non-Rootline project, it's safe to ignore.
