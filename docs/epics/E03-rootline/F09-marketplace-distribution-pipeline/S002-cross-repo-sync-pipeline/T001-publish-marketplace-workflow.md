---
estado: Specified
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T001: Crear workflow publish-marketplace.yml

**Story**: [S002 Cross-Repo Sync Pipeline](README.md)

## Contexto

El workflow de GitHub Actions es el núcleo de la automatización. Debe detectar cambios en `.claude/`, clonar el repo agent-marketplace destino, sincronizar skills a `skills/rootline-*/`, actualizar version en marketplace.json, y pushear. Usa MARKETPLACE_TOKEN (PAT) para autenticación cross-repo.

## Alcance

**In**:
1. `.github/workflows/publish-marketplace.yml`
2. Trigger: push a master con paths `.claude/**`
3. Checkout repo rootline y repo agent-marketplace
4. rsync de `.claude/skills/` a `skills/rootline-*/` (con prefijo rootline-)
5. Actualizar version en marketplace.json desde último git tag (svu current)
6. Commit y push al agent-marketplace repo

**Out**: Idempotencia (T002), validación (T003), binary bundling (S003)

## Estado inicial esperado

- S001 completado: repo agent-marketplace existe con estructura base
- Secret MARKETPLACE_TOKEN configurado en rootline repo
- `svu` disponible en CI (ya usado en auto-tag.yml)

## Criterios de Aceptacion

- Workflow se dispara en push a master que toca `.claude/`
- Skills se sincronizan correctamente a `skills/rootline-*/` en agent-marketplace
- Version en marketplace.json refleja último tag de rootline
- Commit message incluye SHA del commit fuente
- Push al agent-marketplace repo exitoso

## Fuente de verdad

- `.github/workflows/auto-tag.yml` (patrón de cross-repo trigger y svu)
- `.github/workflows/ci.yml` (versiones de actions pinneadas)
