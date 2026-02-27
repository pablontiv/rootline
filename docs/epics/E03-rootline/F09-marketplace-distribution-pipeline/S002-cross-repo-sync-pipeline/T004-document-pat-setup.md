---
ejecutable_en: 1 sesion
estado: Completed
tipo: ci-cd
---
# T004: Documentar setup de PAT secret y arquitectura

**Story**: [S002 Cross-Repo Sync Pipeline](README.md)

## Contexto

El pipeline depende de un Personal Access Token (MARKETPLACE_TOKEN) con permisos de push al repo agent-marketplace. Este es un punto común de fallo — tokens expiran, scopes incorrectos, secrets mal nombrados. Necesita documentación explícita.

## Alcance

**In**:
1. Sección en docs/ explicando:
   - Como crear PAT con scope `repo` para agent-marketplace
   - Como agregar secret MARKETPLACE_TOKEN en rootline repo settings
   - Cadena de triggers: push → CI → publish-marketplace
   - Procedimiento de re-sync manual via workflow_dispatch
2. Diagrama de flujo del pipeline

**Out**: Automatización de rotación de tokens, alertas de expiración

## Estado inicial esperado

- S002/T001-T003 completados: pipeline funcional
- PAT configurado y funcionando

## Criterios de Aceptacion

- Documentación existe en docs/ con instrucciones paso a paso
- Incluye diagrama de flujo del pipeline
- Incluye troubleshooting: token expirado, scope incorrecto, secret no encontrado
- Un developer nuevo puede configurar el pipeline siguiendo solo la documentación

## Fuente de verdad

- GitHub docs (creating PATs, repository secrets)
- `.github/workflows/publish-marketplace.yml` (workflow documentado)
