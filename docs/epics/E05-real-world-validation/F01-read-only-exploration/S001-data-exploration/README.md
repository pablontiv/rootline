# S001: Data Exploration

**Feature**: [F01 Read-Only Exploration](../README.md)
**Estado**: Completado
**Cliente**: Platform Owner
**Capacidad**: Rootline muestra stats, tree y schema de 114 archivos del homeserver sin errores

## Antes / Despues

**Antes**: Rootline solo se ha probado contra fixtures internos del repo. No hay evidencia de que funcione contra datos reales externos.

**Despues**: rootline muestra stats, tree y schema de 114 archivos del homeserver sin errores. Los conteos son coherentes con la realidad.

## Criterios de Aceptacion (semanticos)

- [x] `stats` produce JSON con `version: 1` y conteos coherentes
- [x] `tree` muestra jerarquia de 4 niveles con conteos completados/total
- [x] `describe` ejecuta sin panic en directorio con y sin frontmatter

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-run-stats-command.md) | Ejecutar stats con ambos formatos de output |
| [T002](T002-run-tree-command.md) | Ejecutar tree con ambos formatos de output |
| [T003](T003-run-describe-command.md) | Ejecutar describe en directorios con y sin tasks |
