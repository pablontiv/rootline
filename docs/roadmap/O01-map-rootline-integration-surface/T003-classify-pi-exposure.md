---
estado: In Progress
tipo: task
---
# T003: Classify each command as Pi tool, slash command, prompt, context rule, or unsupported.

**Outcome**: [O01 Map Rootline integration surface](README.md)
**Contribuye a**: CE2 del Outcome.

[[blocked_by:./T002-capture-json-contracts.md]]

## Preserva

- INV1: No Pi tool is implemented before its Rootline command contract and risk class are documented.
  - Verificar: Check downstream tasks reference this Outcome as source of truth.

## Contexto

Esta task forma parte de O01 y debe ejecutarse leyendo este archivo, el README del Outcome y las fuentes de verdad listadas abajo. Classify each command as Pi tool, slash command, prompt, context rule, or unsupported.

## Alcance

**In**:
1. Every command from T001 appears exactly once in the classification matrix.
2. Mutating commands include an explicit risk class.

**Out**:
- Implementar tareas dependientes no listadas en este archivo.
- Cambiar alcance de otros Outcomes sin actualizar el roadmap.

## Estado inicial esperado

- El repositorio está en estado consistente para leer fuentes de verdad y ejecutar validaciones focalizadas.
- La dependencia `T002-capture-json-contracts.md` está completada o su salida está disponible.

## Criterios de Aceptación

- Every command from T001 appears exactly once in the classification matrix.
- Mutating commands include an explicit risk class.
- `rootline validate --all docs/roadmap/` retorna exit 0 o solo warnings aceptados por la configuración del roadmap.

## Fuente de verdad

- `T001-inventory-rootline-commands.md`
- `T002-capture-json-contracts.md`

## Clasificación de Exposición Pi

### Matriz de Clasificación

| Comando | Clasificación | Razón | Clase de riesgo | Contrato JSON | Notas |
|---------|---------------|-------|-----------------|---------------|-------|
| `--version` | Prompt/context rule | Global flag, always available, informational | read-only | N/A | Embed in system prompt |
| `--help` | Prompt/context rule | Global flag, always available, informational | read-only | N/A | Embed in system prompt |
| `--output` | Prompt/context rule | Global flag controlling output format (json\|table) | read-only | N/A | Document supported formats per command |
| `--field` | Prompt/context rule | Global flag for dot-path extraction, repetible | read-only | N/A | Embed in system prompt with examples |
| `analyze` | Pi Tool | Stable JSON contract (version 1), no mutations, agent-safe | read-only | Estable | Direct MCP tool: `infer` |
| `describe` | Pi Tool | Stable JSON contract (version 1), no mutations, schema introspection | read-only | Estable | Direct MCP tool: `describe` |
| `explain` | Pi Tool | Stable JSON contract (version 1), field-level traceability | read-only | Estable | Direct MCP tool: `explain` |
| `graph` | Pi Tool | Stable JSON contract (version 1), cycle + broken link detection | read-only | Estable | Direct MCP tool: `graph` |
| `query` | Pi Tool | Stable JSON contract (version 1), filtering + sorting | read-only | Estable | Direct MCP tool: `query` |
| `stats` | Pi Tool | Stable JSON contract (version 1), simple aggregation | read-only | Estable | Direct MCP tool: `stats` |
| `trace` | Unsupported | Missing `version` and `kind` top-level in JSON | read-only | Inestable | Requires contract normalization before agent use |
| `tree` | Pi Tool | Stable JSON contract (version 1), hierarchy + completion metrics | read-only | Estable | Direct MCP tool: `tree` |
| `validate` | Pi Tool | Stable JSON contract (version 1) for both single and --all modes | read-only | Estable | Direct MCP tool: `validate` |
| `completion` | Slash command | Shell completion, text output, interactive workflow only | read-only | N/A | Not for agent parsing; interactive shell use |
| `apply` | Slash command | Mutates .stem and docs, missing `version`/`kind` in JSON, requires pre-phase scaffolding | mutates-stem / mutates-docs | Inestable | Requires contract fix + explicit guardrails before agent use |
| `fix` (un archivo) | Unsupported | No JSON output support, text-only, mutates docs | mutates-docs | Ninguno | Requires `--output json` flag implementation |
| `fix --all` | Pi Tool | Stable JSON contract (version 1), batch reporting, mutates docs with validation | mutates-docs | Estable | Agent-safe with `--dry-run` guard + apply wrapper |
| `init` | Unsupported | No JSON output support, text-only, mutates .stem | mutates-stem | Ninguno | Requires `--output json` flag implementation |
| `migrate --from` (diff) | Pi Tool | Stable JSON contract (version 1), breaking change classification | read-only | Estable | Direct MCP tool: `migrate-diff` |
| `migrate --from` (batch) | Pi Tool | Stable JSON contract (version 1), breaking change summary | read-only | Estable | Direct MCP tool: `migrate-batch` |
| `migrate --rename` | Unsupported | Missing `version`/`kind` in JSON, mutates .stem | mutates-stem | Inestable | Requires contract normalization + guardrails |
| `migrate --scaffold` | Unsupported | Missing `version`/`kind` in JSON, mutates .stem | mutates-stem | Inestable | Requires contract normalization + guardrails |
| `migrate --split` | Unsupported | No JSON output support, text-only, mutates .stem | mutates-stem | Ninguno | Requires `--output json` flag implementation |
| `new` | Unsupported | No JSON output support, text-only, mutates docs | mutates-docs | Ninguno | Requires `--output json` flag implementation |
| `set` | Unsupported | No JSON output support, text-only, mutates docs | mutates-docs | Ninguno | Requires `--output json` flag implementation |
| `hooks install` | External | Git integration, external system mutation (install hook) | external | N/A | Operate via system commands, not MCP |
| `hooks status` | External | Git integration, external system query | external | N/A | Operate via system commands, not MCP |
| `hooks uninstall` | External | Git integration, external system mutation (uninstall hook) | external | N/A | Operate via system commands, not MCP |
| `serve` | External | MCP server startup, external process management | external | N/A | Operate via system commands or process manager |

### Resumen de Clasificación

| Categoría | Cantidad | Ejemplos |
|-----------|----------|----------|
| **Pi Tool** | 11 | analyze, describe, explain, graph, query, stats, tree, validate, fix --all, migrate --from (diff), migrate --from (batch) |
| **Slash command** | 2 | apply, completion |
| **Prompt/context rule** | 4 | --version, --help, --output, --field |
| **Unsupported** | 9 | trace, fix (un archivo), init, migrate --rename, migrate --scaffold, migrate --split, new, set |
| **External** | 4 | hooks install, hooks status, hooks uninstall, serve |

### Matriz de Riesgo y Guardrails

**Pi Tools (11 comandos):**
- 9 read-only: analyze, describe, explain, graph, query, stats, tree, validate, migrate --from (both)
  - Riesgo: Ninguno (no mutations)
  - Guardrail: Ninguno requerido
- 1 mutating: fix --all
  - Riesgo: mutates-docs
  - Guardrail: --dry-run flag obligatorio para preview antes de apply
- 1 mutating (slash): apply
  - Riesgo: mutates-stem / mutates-docs (doble mutación)
  - Guardrail: Requiere contrato estable + pre-phase scaffolding + explicit human confirmation

**Slash Commands (2):**
- apply: For interactive workflows with explicit user control over schema/doc mutations
- completion: Shell integration, not for agent automation

**Unsupported (9 comandos):**
- Todos requieren contrato JSON estable antes de exposición a agentes
- Prioridad de implementación:
  1. trace: Add version/kind to TraceResult (easy, read-only)
  2. fix (single file), init, new, set: Add --output json support (medium, read-only contracts exist)
  3. migrate --rename, migrate --scaffold: Wrap in versioned structures (medium, requires new JSON envelopes)
  4. apply: Add version/kind to ApplyResult + guardrails (high priority, double mutation risk)
