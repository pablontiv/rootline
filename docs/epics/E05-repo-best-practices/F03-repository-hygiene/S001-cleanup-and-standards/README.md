---
tipo: historia
cliente: Platform Owner
---
# S001: Cleanup and Standards

**Feature**: [F03 Repository Hygiene](../README.md)
**Capacidad**: Artefactos inconsistentes eliminados, herramientas actualizadas, y estándares de contribución completos

## Antes / Despues

**Antes**: Binario `rootline` de 8.8MB está committeado al repo pese a estar en .gitignore. `pre-commit-golang` está archivado/unmaintained. No hay `.editorconfig` ni `CODEOWNERS`. `CONTRIBUTING.md` no menciona golangci-lint, pre-commit setup, ni conventional commits.

**Despues**: Binario eliminado del tracking. Pre-commit usa hooks mantenidos. `.editorconfig` define tabs/spaces/line endings. `.github/CODEOWNERS` define reviewers. `CONTRIBUTING.md` cubre setup completo incluyendo pre-commit, golangci-lint, y conventional commits.

## Criterios de Aceptacion (semanticos)

- [ ] `git ls-files rootline` no retorna nada (binario fuera del tracking)
- [ ] `pre-commit run --all-files` pasa con hooks actualizados (sin dependencias archivadas)
- [ ] `.editorconfig` y `.github/CODEOWNERS` existen
- [ ] `CONTRIBUTING.md` incluye instrucciones de pre-commit, golangci-lint y conventional commits

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-remove-committed-binary.md) | Eliminar binario rootline del tracking git |
| [T002](T002-replace-archived-precommit.md) | Reemplazar pre-commit-golang archivado con alternativa mantenida |
| [T003](T003-add-editorconfig-codeowners.md) | Crear .editorconfig y .github/CODEOWNERS |
| [T004](T004-update-contributing-guide.md) | Actualizar CONTRIBUTING.md con setup completo |

## Fuente de verdad

- `rootline` — binario a eliminar
- `.pre-commit-config.yaml` — hooks a actualizar
- `CONTRIBUTING.md` — guía a completar
- `.gitignore` — ya lista `/rootline`
