# S003: Tree and Stats

**Feature**: [F04 Query and Presentation](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: `rootline tree` y `rootline stats` muestran vistas jerarquicas y resumenes de documentos

## Antes / Despues

**Antes**: `/roadmap view` reconstruye el arbol completo inline cada vez con Python/LLM. `/roadmap pending` usa regex Python para contar. Cada invocacion re-parsea todo. No hay vista persistente ni eficiente.

**Despues**: `rootline tree docs/epics/` muestra arbol jerarquico ASCII con conteos de completitud. `rootline stats` muestra resumen por tipo/estado. Ambos producen JSON para consumo programatico.

## Criterios de Aceptacion (semanticos)

- [ ] `rootline tree` muestra jerarquia con conteos de tasks completados/total
- [ ] `rootline stats` muestra tabla de resumen por tipo y estado
- [ ] Ambos comandos soportan output JSON

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-tree-command.md) | Implementar `rootline tree` |
| [T002](T002-stats-command.md) | Implementar `rootline stats` |

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` seccion 3 (Commands: tree, stats)
