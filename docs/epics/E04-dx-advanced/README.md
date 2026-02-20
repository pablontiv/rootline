# E04: Rootline DX & Advanced Capabilities

**Estado**: Activa
**Metrica de exito**: `rootline completion zsh` genera completions funcionales, `rootline init docs/` infiere schema, `rootline new` genera documentos validos, `rootline fix` repara errores, `rootline doctor` reporta salud, `rootline explain` traza derivacion, `rootline graph` muestra dependencias
**Timeline**: 2026-Q1 — en curso

## Intencion

Evolucionar Rootline de MVP funcional (E03: core engine + 5 comandos) a herramienta completa con DX pulida, automatizacion de validacion, y capacidades avanzadas de derivacion y grafos. Las 10 oportunidades seleccionadas del analisis I9 se organizan en 5 milestones independientes que cubren: experiencia CLI, ciclo de vida documental, evolucion de validacion, motor de derivacion, y grafo de dependencias.

## Features

| ID | Nombre | Descripcion |
|----|--------|-------------|
| F01 | [CLI Experience](F01-cli-experience/) | Shell completions, table output, doctor command |
| F02 | [Document Lifecycle](F02-document-lifecycle/) | Schema inference (init), scaffolding (new), auto-fix |
| F03 | [Validation Evolution](F03-validation-evolution/) | Progressive strictness (severity levels), git hooks |
| F04 | [Derivation Engine](F04-derivation-engine/) | Expression language (expr-lang), derive pipeline, explain command |
| F05 | [Dependency Graph](F05-dependency-graph/) | Wiki-link extraction, link schema/validation, graph command |
| F06 | [E05 Hardening](F06-e05-hardening/) | fix --all, describe hints, init mixed-content warnings |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | CLI polish, sin dependencias |
| F02 | — | Document lifecycle, parallelizable con F01 y F03 |
| F03 | — | Validation evolution, parallelizable con F01 y F02 |
| F04 | F01-F03 (core estable) | Derivation requiere pipeline maduro |
| F05 | F04 | Estado derivado por propagacion usa expression language |
| F06 | F02 | Hardening de fix/describe/init requiere comandos base |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-02-19 | expr-lang/expr como expression language | ~3MB, zero deps, non-Turing-complete, type-safe. Suficiente para niveles 1-2 (field transform + cross-record aggregation). Migrar a Starlark/CEL si se necesita nivel 3-4. |
| 2026-02-19 | 10 oportunidades de I9 seleccionadas | Basado en ratio impacto/esfuerzo. Shell completions, init, table output, new, fix, hooks, doctor, progressive strictness, expression language, graph/links. |

## Gaps Activos

- **Expression language**: Eleccion de expr-lang/expr puede necesitar reevaluacion si se necesitan niveles 3-4 de derivacion (link traversal, recursion)
- **Wiki-link format**: Formato exacto de links (`[[target]]` vs `[[type:target]]`) requiere decision al implementar F05

## Referencias

- [I9: Opportunity Areas](../../research/I9-opportunity-areas.md) — analisis completo de oportunidades
- [I3: Derivation Pre-research](../../research/I3-derivation-pre-research.md) — expression language evaluation
- [E03: Core Engine](../E03-rootline/README.md) — MVP base
