---
estado: Pending
tipo: feature
---
# F03: Repository Hygiene

**Epic**: [E05](../README.md)
**Objetivo**: El repositorio está limpio de artefactos inconsistentes, tiene estándares de editor unificados, ownership definido, y guía de contribución completa
**Beneficio**: Reduce fricción para nuevos contributors, elimina el binario de 8.8MB del tracking, y reemplaza dependencia archivada
**Milestone**: Binario eliminado del tracking, pre-commit usa hooks mantenidos, `.editorconfig` y `CODEOWNERS` existen, CONTRIBUTING.md cubre setup completo

## Scope

**In**: Eliminar binario committeado, reemplazar pre-commit-golang archivado, crear .editorconfig y CODEOWNERS, actualizar CONTRIBUTING.md
**Out**: Rewrite de git history (BFG/filter-branch), branch protection rules (requiere admin access)

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| [S001](S001-cleanup-and-standards/) | Cleanup and Standards | Artefactos inconsistentes eliminados y estándares de contribución documentados |

## Dependencias

- Ninguna

## Fuente de verdad

- `rootline` — binario committeado a eliminar
- `.pre-commit-config.yaml` — hooks a actualizar
- `CONTRIBUTING.md` — guía a completar
