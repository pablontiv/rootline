---
estado: In Progress
tipo: task
---
# T003: Define output truncation and optional TUI rendering behavior.

**Outcome**: [O02 Design Pi extension architecture](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T002-design-rootline-cli-runner.md]]

## Preserva

- INV1: The extension treats Rootline CLI JSON as the integration boundary; it does not import Go internal packages.
  - Verificar: Check architecture docs and implementation tasks.

## Contexto

Esta task forma parte de O02 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Define output truncation and optional TUI rendering behavior.

## Alcance

**In**:
1. Large outputs have documented truncation limits and full-output handoff behavior.
2. Tool results distinguish model-facing content from details.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T002-design-rootline-cli-runner.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Large outputs have documented truncation limits and full-output handoff behavior.
- Tool results distinguish model-facing content from details.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `Pi extension output truncation docs`

## Output Truncation and Rendering Design

Large Rootline outputs can exceed model context windows and terminal rendering capacity. This section defines truncation limits, handoff behavior for full output, content classification (model-facing vs detail), and error surface.

### 1. Truncation Limits by Tool Category

Truncation thresholds are applied at the JSON layer before rendering to prevent bloated responses to the model:

| Tool | Dimension | Limit | Rationale |
|------|-----------|-------|-----------|
| **query** | Rows | 100 rows (~50k chars) | Models reason over field names and counts; full datasets are encyclopedic |
| **query** | Total match count | Report: "Found 500 matching records, showing 100" | Allow statistics without enumeration |
| **validate-all** | Error items | 50 errors before summary mode | Display first 50 errors; aggregate counts for remainder |
| **validate-all** | Result files | 50 files before summary | Show per-file details for first 50; aggregate summary for rest |
| **tree** | Depth | 6 levels max before truncation | Terminal rendering becomes unreadable beyond depth 6 |
| **tree** | Children per node | 50 children before collapse to "N more…" | Prevent horizontal sprawl in terminal |
| **graph** | Nodes | 200 nodes before switch to summary mode | Interactive rendering slows for graphs >200 nodes |
| **graph** | Edges | 500 edges before switch to summary mode | Cycle detection and broken-link display require tractable edge counts |
| **stats** | Distinct values | 20 distinct values per aggregation | Enum cardinality beyond 20 is better served by filtering |
| **describe** | Schema fields | 100 fields before truncation (rare) | Prevent describe output from drowning in inherited fields |

### 2. Full-Output Handoff Behavior

When a result exceeds truncation limits, the JSON response signals truncation and provides guidance:

#### Structure: Truncation Indicator

Every tool result includes an optional `"truncated"` field:

```json
{
  "version": 1,
  "kind": "rootline/query",
  "truncated": true,
  "total": 500,
  "returned": 100,
  "data": {
    "meta": { "count": 100 },
    "rows": [ /* first 100 rows */ ]
  }
}
```

**Fields**:
- `truncated` (boolean): Present and true only when output exceeds limit.
- `total` (number): Total matching records before truncation.
- `returned` (number): Number of items actually returned.

#### Agent Handoff Strategies

When `truncated: true`, the agent has two options:

**Option A: Write full output to temp file**
```
Agent response: "Output truncated (500 records, showing 100). 
  Writing full results to /tmp/rootline-query-<uuid>.json"
  
Then invoke rootline directly and write:
  rootline query <same-args> --output json > /tmp/rootline-query-<uuid>.json
  
Return file path to user for offline inspection.
```

**Option B: Paginate with filters**
```
Agent response: "Output truncated. Use --where filters to narrow scope:
  rootline query <path> --where 'tipo=="task" and estado=="In Progress"'
  
Retry query with additional constraints until result fits.
```

**Recommended approach**: Use Option A for exploratory queries; use Option B for targeted analysis.

### 3. Model-Facing vs Detail Content

JSON responses distinguish content intended for model reasoning from display-only details. This prevents the model from processing raw YAML dumps or multi-kilobyte file snippets.

#### Classification Strategy

Use a field-level classification scheme:

- **Model-facing**: Error messages, field names, record paths, counts, enum values, type names, rule sources.
  - Include in JSON response without modification.
  - Suitable for model reasoning and decision-making.

- **Detail**: Full file content, raw YAML, patch previews over 200 chars, multiline markdown bodies, image binary data.
  - Either omit from JSON entirely, OR use `"_detail"` prefix convention to signal it is auxiliary.
  - Model should not reason over detail fields.

#### Naming Convention

Fields containing detail content use the `_detail` prefix. For example:

```json
{
  "version": 1,
  "kind": "rootline/validate",
  "path": "docs/task.md",
  "valid": false,
  "errors": [
    {
      "rule": "required",
      "field": "tipo",
      "message": "Missing required field 'tipo'",
      "source": ".stem schema"
    }
  ],
  "_detail": {
    "raw_frontmatter": "---\nested: 2025-05-09\n---",
    "body_preview": "Long markdown body (truncated to 200 chars)"
  }
}
```

#### Per-Tool Guidelines

**rootline-query**:
- Model-facing: path, frontmatter field names/values (up to 1000 chars per field), derived field names/values, counts.
- Detail: Full file body. Use `_detail.body_lines` for first 3 lines only (truncated).

**rootline-validate / rootline-validate-all**:
- Model-facing: Error/warning messages, field names, rule names, severity levels, source (.stem file paths).
- Detail: Raw YAML frontmatter. Use `_detail.frontmatter_preview` (first 500 chars) only if diagnosis requires it.

**rootline-describe**:
- Model-facing: Field names, types, enum values (up to 10 values), required rules, aggregate formula, validation rules.
- Detail: Full rule expressions over 200 chars. Use `_detail.rule_expression` prefix.

**rootline-graph**:
- Model-facing: Node names, node paths, edge types, cycle sequences, broken link suggestions.
- Detail: Full graph in DOT/Mermaid format. Use `_detail.dot` / `_detail.mermaid` for visualization.

**rootline-fix-all**:
- Model-facing: Field names, before/after values (truncated to 100 chars), reason, counts.
- Detail: Full patch content. Use `_detail.patch` only if user explicitly requests the full diff.

### 4. Table Rendering Constraints

When `--output table` is used, the CLI applies formatting rules:

| Constraint | Rule |
|-----------|------|
| **Column width** | Max 40 chars per column, wrap text at column boundary |
| **Max rows** | Display first 50 rows; append "... (N more rows)" footer |
| **Overflow fields** | Truncate with "…" at column end; include full value in JSON via `_detail` |
| **Terminal width** | Detect terminal width; scale columns proportionally; drop columns if space insufficient |
| **Multiline cells** | Display first line only + "(+N more lines)" indicator |

**Example**:
```
path                    tipo        estado
docs/task1.md          task        In Progress
docs/task2.md          task        Completed
docs/epic1.md          epic        Blocked
... (47 more rows)
```

### 5. RootlineError to Model Visibility Mapping

The `RootlineError` object from T002 defines how subprocess failures surface to the model:

#### Error Codes and Model Visibility

| Code | Visibility | Recommended Action |
|------|-----------|-------------------|
| `BINARY_NOT_FOUND` | **Terminal error** — Block further API calls until binary is installed | Suggest: "Install rootline: `just check` or system package" |
| `PROJECT_ROOT_NOT_FOUND` | **Terminal error** — No project context, cannot proceed | Suggest: "Run from a directory inside the Git repository" |
| `TIMEOUT` | **Transient warning** — Retry with increased timeout or `--where` filter | Suggest pagination (Option B above) |
| `ABORT` | **User action** — No error recovery needed | Log cancellation; allow user to retry |
| `PARSE_ERROR` | **Terminal error** — Binary produced invalid output; likely internal bug | Suggest: "Report Rootline bug with stderr output" |
| `ROOTLINE_ERROR` | **Varies** — Includes validation failures, missing files, invalid arguments | Inspect stderr for specific cause; surface as part of result |
| `INVALID_ARGUMENTS` | **Preventable error** — Argument validation failed before subprocess | Suggest: "Check tool parameter names and formats against schema" |

#### Model-Visible Error Object

When returning a `RootlineError` to the model, surface only essential fields:

```json
{
  "code": "TIMEOUT",
  "message": "Query exceeded 30s timeout on 1M+ records",
  "suggestion": "Use --where filter to narrow scope"
}
```

Omit `stderr` and raw `exitCode` from model responses; use stderr for internal diagnostics only.

#### Warnings as Soft Errors

Some non-zero rootline exits produce valid JSON + warnings (e.g., broken links in graph output). Classify as warnings:

```json
{
  "version": 1,
  "kind": "rootline/graph",
  "data": { /* graph nodes and edges */ },
  "warnings": [
    {
      "message": "Broken link: [[missing-file.md]] not found",
      "suggestion": "Create the file or update the link"
    }
  ]
}
```

Model sees the valid data + warnings; can reason over partial results.
