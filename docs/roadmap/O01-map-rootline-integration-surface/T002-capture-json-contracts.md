---
estado: Completed
tipo: task
---
# T002: Capture JSON output contracts for commands that Pi can consume.

**Outcome**: [O01 Map Rootline integration surface](README.md)
**Contribuye a**: CE1 del Outcome.

[[blocked_by:./T001-inventory-rootline-commands.md]]

## Preserva

- INV1: No Pi tool is implemented before its Rootline command contract and risk class are documented.
  - Verificar: Check downstream tasks reference this Outcome as source of truth.

## Contexto

Esta task forma parte de O01 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Capture JSON output contracts for commands that Pi can consume.

## Alcance

**In**:
1. For each candidate command, document version/kind shape or note missing contract.
2. Identify commands whose output should not be parsed until normalized.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T001-inventory-rootline-commands.md` está completada o su salida está disponible.

## Criterios de Aceptación

- For each candidate command, document version/kind shape or note missing contract.
- Identify commands whose output should not be parsed until normalized.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `cmd/rootline/*.go`
- `internal/* result/report types`

## Contratos JSON por Comando

### Comandos Read-Only (sin modificaciones)

#### `analyze`
- **version**: 1
- **kind**: `"analyze"`
- **Campos clave**: `path`, `incremental`, `categories[]`, `summary{total_inferences, agent_required, engine_resolved}`
- **Contrato estable**: Sí. Versión inyectada en cada categoría como `ReportInference` con `type`, `field`, `value`, `message`, `requires_agent`.

#### `describe`
- **version**: 1
- **kind**: `"rootline/describe"`
- **Campos clave**: `path`, `applies[]`, `scope`, `schema{}`, `validate[]`, `derive{}`, `aggregate{}`, `links{}`, `hints[]`
- **Contrato estable**: Sí. Schema contiene tipo de campo, valores enum, requerido, y fuente.

#### `explain`
- **version**: 1
- **kind**: `"rootline/explain"`
- **Campos clave**: `path`, `stem_chain[]`, `fields[]{name, value, origin, source, expression}`, `errors[]{rule, field, message, source, severity}`
- **Contrato estable**: Sí. Traza cada campo a su origen (frontmatter, schema, derived).

#### `graph`
- **version**: 1
- **kind**: `"rootline/graph"`
- **Campos clave**: `nodes[]`, `edges[]{source, target, type, line}`, `cycles[][]`, `broken_links[]{source, target, type, line, suggestions[]}`
- **Contrato estable**: Sí. Salida directa JSON o Mermaid/DOT sin cambios.

#### `query`
- **version**: 1
- **kind**: `"rootline/query"` o `"rootline/count"`
- **Campos clave**: `meta{count}`, `rows[]` (para query) o `count` (para --count)
- **Contrato estable**: Sí. Devuelve registros completos con frontmatter + derived.

#### `stats`
- **version**: 1
- **kind**: `"rootline/stats"`
- **Campos clave**: `by_estado{}`, `by_tipo{}`, `total`
- **Contrato estable**: Sí. Agregados simples de conteos por campo.

#### `tree`
- **version**: 1
- **kind**: `"rootline/tree"`
- **Campos clave**: `root{name, path, children[], completed, total, estado}`
- **Contrato estable**: Sí. Estructura jerárquica con conteos de completitud recursivos.

#### `trace`
- **version**: (ninguno - sin top-level version en TraceResult)
- **kind**: (ninguno)
- **Campos clave**: `nodes[]{path, depth, estado, via}` 
- **Contrato estable**: **NO** - falta `version` y `kind` en `TraceResult`. Devuelve `graph.TraceResult` sin versión.

#### `validate` (un archivo)
- **version**: 1
- **kind**: `"rootline/validate"`
- **Campos clave**: `path`, `valid`, `errors[]`, `warnings[]`
- **Contrato estable**: Sí.

#### `validate --all` (múltiples archivos)
- **version**: 1
- **kind**: `"rootline/validate-batch"`
- **Campos clave**: `results[]`, `drift_warnings[]`, `summary{total, valid, invalid, errors_count, warnings_count, drift_warnings_count}`
- **Contrato estable**: Sí.

### Comandos Mutantes (modifican documentos o .stem)

#### `apply`
- **version**: (ninguno - ApplyResult sin top-level)
- **kind**: (ninguno)
- **Campos clave**: `applied[]`, `skipped[]`, `dry_run`
- **Contrato estable**: **NO** - falta `version` y `kind`. Debería ser `"rootline/apply"` versión 1.

#### `fix --all`
- **version**: 1
- **kind**: `"rootline/fix-batch"`
- **Campos clave**: `results[]{path, fixed, fields_added, values_corrected, changes[]}`, `summary{total, fixed, skipped}`
- **Contrato estable**: Sí.

#### `fix` (un archivo)
- **Sin JSON**. Salida a stdout en formato texto (sin contrato de versión).

#### `init`
- **Sin JSON**. Salida a stdout en formato texto.

#### `migrate --from` (diff)
- **version**: 1
- **kind**: `"rootline/migrate-diff"`
- **Campos clave**: `stem_path`, `changes[]{kind, field, breaking, before, after, message}`, `breaking_count`, `total_count`
- **Contrato estable**: Sí.

#### `migrate --from` (múltiples stems)
- **version**: 1
- **kind**: `"rootline/migrate-batch"`
- **Campos clave**: `results[]`, `summary{stems_checked, total_changes, breaking_count}`
- **Contrato estable**: Sí.

#### `migrate --rename`
- **Sin JSON estable**. Devuelve `RenameResult` sin versión top-level.

#### `migrate --scaffold`
- **Sin JSON estable**. Devuelve `ScaffoldResult` sin versión top-level.

#### `migrate --split`
- **Sin JSON**. Salida a stdout en formato texto.

#### `new`
- **Sin JSON**. Salida a stdout en formato texto.

#### `set`
- **Sin JSON**. Salida a stdout en formato texto.

### Resumen: Comandos sin contrato estable

Estos comandos **NO** emiten JSON versionado y requieren normalización antes de parsear:

| Comando | Problema | Solución sugerida |
|---------|----------|-------------------|
| `trace` (JSON) | Falta `version` y `kind` top-level | Agregar a `TraceResult` |
| `apply` | Falta `version` y `kind` top-level | Cambiar a `ApplyResult{Version: 1, Kind: "rootline/apply", Applied, Skipped, DryRun}` |
| `migrate --rename` | Devuelve `RenameResult` sin versión | Envolver en estructura versionada |
| `migrate --scaffold` | Devuelve `ScaffoldResult` sin versión | Envolver en estructura versionada |
| `fix` (un archivo) | Salida a stdout texto, no JSON | Agregar flag `--output json` |
| `init` | Salida a stdout texto, no JSON | Agregar flag `--output json` con `InitResult{Version: 1, Kind: "rootline/init", ...}` |
| `new` | Salida a stdout texto, no JSON | Agregar flag `--output json` |
| `set` | Salida a stdout texto, no JSON | Agregar flag `--output json` |

### Nota sobre `--output json` global

El flag `--output json` (default) solo afecta a comandos que tienen soporte para `table` alternative. Comandos sin `renderTable()` ignoran la flag y siempre emiten texto.
