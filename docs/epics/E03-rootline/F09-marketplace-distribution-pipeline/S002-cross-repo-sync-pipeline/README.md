---
tipo: historia
cliente: Platform Owner
---
# S002: Cross-Repo Sync Pipeline

**Feature**: [F09 Agent Marketplace](../README.md)
**Capacidad**: Cada push a master que cambia `.claude/skills/` sincroniza automáticamente al repo agent-marketplace sin intervención manual

## Antes / Despues

**Antes**: Actualizar skills en el marketplace requiere copia manual de archivos, commit y push al repo destino. Propenso a olvidos y desincronización.

**Despues**: GitHub Actions workflow detecta cambios en `.claude/skills/`, sincroniza a `skills/rootline-*/` en agent-marketplace, bumps version, y pushea. No-op cuando skills no cambian. Re-sync manual disponible via workflow_dispatch.

## Criterios de Aceptacion (semanticos)

- [ ] Workflow publish-marketplace.yml funcional en CI
- [ ] Sync automático en push a master cuando .claude/ cambia
- [ ] No-op cuando skills no han cambiado (idempotencia)
- [ ] Validación pre-push de estructura del marketplace
- [ ] Documentación de setup de PAT secret

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-publish-marketplace-workflow.md) | Crear workflow publish-marketplace.yml |
| [T002](T002-idempotency-diff-guard.md) | Agregar idempotencia y diff-guard al workflow |
| [T003](T003-sync-validation.md) | Agregar validación pre-push del marketplace |
| [T004](T004-document-pat-setup.md) | Documentar setup de PAT secret y arquitectura |
| [T005](T005-configure-marketplace-token.md) | Configurar MARKETPLACE_TOKEN secret y verificar pipeline |

## Fuente de verdad

- `.github/workflows/auto-tag.yml` (patrón de workflow con triggers)
- `.github/workflows/ci.yml` (estructura de jobs)
