# KEDB — Known Error Database Reactiva

**Fecha**: 2026-02-25
**Tipo**: Research
**Contexto**: CLI en Go que mantiene una Known Error Database cross-proyecto. Responde en tiempo real durante sesiones de Claude Code via hooks, y crea KEs automaticamente cuando un error se confirma como recurrente. Tres dominios: Backscroll (descubrimiento/busqueda), Rootline (estructura/validacion), kedb (lifecycle/orquestacion). Sin LLM embebida — Claude Code ya es el LLM.

---

## Documentos

| Documento | Contenido |
|-----------|-----------|
| [research.md](research.md) | Problema, evidencia, estado del arte (ITIL, SRE, bridge entities, error matching, daemons), escenarios aspiracionales con datos |
| [design.md](design.md) | Arquitectura de tres dominios, flujo operativo, decisiones tecnicas, schema .stem, operaciones CLI, integracion con Claude Code |
| [roadmap.md](roadmap.md) | Estructura jerarquica de implementacion, tasks, dependencias, futuro v2+ |

## Documentos relacionados

- [Backscroll — Session & Plan Search CLI](../backscroll-session-search-cli.md) — Event store + busqueda full-text sobre sesiones y planes de Claude Code. Backscroll provee Tier 2 search para KEDB.
- [Rootline — File-based database engine](https://github.com/pablontiv/rootline) — Structured store + validacion. Los KE records viven como archivos .md con .stem schema en /opt/kedb/, queryables via `rootline query`.
