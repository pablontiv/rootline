---
estado: In Progress
tipo: task
---
# T001: Define parameter and result contracts for read-only Rootline tools.

**Outcome**: [O02 Design Pi extension architecture](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:../O01-map-rootline-integration-surface/T003-classify-pi-exposure.md]]

## Preserva

- INV1: The extension treats Rootline CLI JSON as the integration boundary; it does not import Go internal packages.
  - Verificar: Check architecture docs and implementation tasks.

## Contexto

Esta task forma parte de O02 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Define parameter and result contracts for read-only Rootline tools.

## Alcance

**In**:
1. Schemas exist for query, tree, validate, describe, stats, graph/explain if selected.
2. Each schema avoids arbitrary shell command strings.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia ~../O01-map-rootline-integration-surface/T003-classify-pi-exposure.md~ está completada o su salida está disponible.

## Criterios de Aceptación

- Schemas exist for query, tree, validate, describe, stats, graph/explain if selected.
- Each schema avoids arbitrary shell command strings.
- ~rootline validate --all docs/roadmap/~ retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- ~docs/roadmap/O01-map-rootline-integration-surface/~

## Esquemas de Herramientas Pi (Read-Only)

This section defines parameter and result contracts for all read-only Pi tools. Each tool accepts typed parameters (avoiding arbitrary shell strings) and returns JSON responses with a ~version~ (1) and ~kind~ field.

### 1. rootline-query

**Purpose**: Search and filter Rootline records by frontmatter fields using declarative filter expressions.

**Parameters**:
- ~path~ (required, string): Directory or file path to search within.
- ~filter~ (optional, string): Expression filter (expr-lang): eq, ne, in, contains, exists, and. Example: ~estado=="Completed" and tipo=="task"~.
- ~--count~ (optional, flag): Return only record count instead of full records.
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~rootline/query~. Fields: ~meta~ (count), ~rows~ array with path, frontmatter object, derived object.

**Source**: T002 — query section

#####


### 2. rootline-describe

**Purpose**: Introspect schema and rules for a directory or file. Returns field types, enums, required rules, derivations, aggregations.

**Parameters**:
- ~path~ (required, string): Directory or file to describe.
- ~--by-domain~ (optional, string): Filter schema by semantic domain (governance, traceability, etc.).
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~rootline/describe~. Fields: path, applies (stem chain), scope, schema (fieldName -> {type, enum, required, source}), validate array, derive object, aggregate object, links object, hints array.

**Source**: T002 — describe section

#####


### 3. rootline-validate

**Purpose**: Validate a single file against its applicable ~.stem~ schema rules.

**Parameters**:
- ~path~ (required, string): File path to validate.
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~rootline/validate~. Fields: path, valid (boolean), errors array (rule, field, message, source, severity), warnings array.

**Source**: T002 — validate (un archivo) section

#####


### 4. rootline-validate-all

**Purpose**: Validate all records in a directory tree. Includes drift detection between ~.stem~ and documents.

**Parameters**:
- ~path~ (required, string): Root directory to validate recursively.
- ~--where~ (optional, string): Filter records using expr-lang (e.g., ~tipo=="task"~).
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~rootline/validate-batch~. Fields: results array (path, valid, errors, warnings), drift_warnings array (path, message), summary (total, valid, invalid, errors_count, warnings_count, drift_warnings_count).

**Source**: T002 — validate --all section

#####


### 5. rootline-tree

**Purpose**: Display directory hierarchy with metadata and completion metrics for all records.

**Parameters**:
- ~path~ (required, string): Root directory to display.
- ~--where~ (optional, string): Filter records using expr-lang (e.g., ~estado=="In Progress"~).
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~rootline/tree~. Fields: root object (name, path, estado, children array, completed, total).

**Source**: T002 — tree section

#####


### 6. rootline-stats

**Purpose**: Aggregate statistics across records: counts by field value.

**Parameters**:
- ~path~ (required, string): Directory to analyze.
- ~--where~ (optional, string): Filter records before computing stats (e.g., ~tipo=="task"~).
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~rootline/stats~. Fields: by_estado object (field_value -> count), by_tipo object, total (number).

**Source**: T002 — stats section

#####


### 7. rootline-graph

**Purpose**: Extract and analyze dependency graph from wiki-links in document bodies. Detect cycles and broken links.

**Parameters**:
- ~path~ (required, string): Directory or file to extract graph from.
- ~--format~ (optional, enum): json (default), mermaid, or dot.
- ~--open~ (optional, flag): Render interactive HTML diagram (browser-only, not for agent use).
- ~--output~ (optional, enum): json (default) or text.

**Result Contract**: JSON version 1, kind ~rootline/graph~. Fields: nodes array (name, path, tipo), edges array (source, target, type, line), cycles array, broken_links array (source, target, type, line, suggestions).

**Source**: T002 — graph section

#####


### 8. rootline-explain

**Purpose**: Trace the origin of each field in a record: schema rule source, ~.stem~ file, derivation expression, or frontmatter value.

**Parameters**:
- ~path~ (required, string): File path to explain.
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~rootline/explain~. Fields: path, stem_chain array, fields array (name, value, origin, source, expression), errors array (rule, field, message, source, severity).

**Source**: T002 — explain section

#####


### 9. rootline-analyze

**Purpose**: Infer schema from existing documents. Detect missing fields, type mismatches, enum values, governance gaps (domain/schema coverage, validation gaps).

**Parameters**:
- ~path~ (required, string): Directory to analyze.
- ~--incremental~ (optional, flag): Only report inferences not covered by existing ~.stem~ files.
- ~--where~ (optional, string): Filter records before analysis.
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~analyze~. Fields: path, incremental (boolean), categories array (name, inferences array with type, path, field, value, message, requires_agent), summary (total_inferences, agent_required, engine_resolved).

**Source**: T002 — analyze section

#####


### 10. rootline-fix-all

**Purpose**: Apply automatic fixes to all documents: correct enum values, add missing fields, migrate values. Requires ~--dry-run~ preview before execution.

**Parameters**:
- ~path~ (required, string): Directory to fix recursively.
- ~--dry-run~ (required, flag): Preview changes without modifying. Must use before actual fix.
- ~--where~ (optional, string): Filter records before fixing.
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~rootline/fix-batch~. Fields: path, dry_run (boolean), results array (path, fixed, fields_added, values_corrected, changes array with field, before, after, reason), summary (total, fixed, skipped).

**Source**: T002 — fix --all section

#####


### 11. rootline-migrate-diff

**Purpose**: Analyze schema changes between two ~.stem~ versions. Classify changes as breaking or non-breaking.

**Parameters**:
- ~path~ (required, string): Path to ~.stem~ file to analyze.
- ~--from~ (required, string): Previous ~.stem~ content or file path for comparison.
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~rootline/migrate-diff~. Fields: stem_path, changes array (kind, field, breaking, before, after, message), breaking_count, total_count.

**Source**: T002 — migrate --from (diff) section

#####


### 12. rootline-migrate-batch

**Purpose**: Analyze schema changes across multiple ~.stem~ files. Returns summary of breaking changes per file.

**Parameters**:
- ~path~ (required, string): Directory containing multiple ~.stem~ files.
- ~--from~ (required, string): Previous version reference (git tag, commit, or directory path).
- ~--where~ (optional, string): Filter ~.stem~ files before analysis.
- ~--output~ (optional, enum): json (default) or table.

**Result Contract**: JSON version 1, kind ~rootline/migrate-batch~. Fields: path, from (string), results array (stem_path, changes array), summary (stems_checked, total_changes, breaking_count).

**Source**: T002 — migrate --from (múltiples stems) section
