---
estado: Specified
tipo: ci-cd
ejecutable_en: 1 sesion
---
# T005: Configurar MARKETPLACE_TOKEN secret y verificar pipeline

**Story**: [S002 Cross-Repo Sync Pipeline](README.md)

## Contexto

El workflow publish-marketplace.yml está listo pero requiere un Personal Access Token (PAT) configurado como secret `MARKETPLACE_TOKEN` en el repo rootline para autenticarse al pushear al repo agent-marketplace. Sin este token, el workflow falla en CI. La documentación paso a paso ya existe en `docs/marketplace-pipeline.md`.

## Alcance

**In**:
1. Crear Fine-Grained PAT en GitHub con scope `Contents: Read and write` para `pablontiv/agent-marketplace`
2. Agregar secret `MARKETPLACE_TOKEN` en rootline repo settings
3. Ejecutar `workflow_dispatch` de publish-marketplace para verificar el pipeline end-to-end
4. Confirmar que agent-marketplace recibe el commit del sync

**Out**: Rotación automática de tokens, alertas de expiración, CI changes

## Estado inicial esperado

- T001-T004 completados: workflow y documentación listos
- Repo `pablontiv/agent-marketplace` existe y tiene skills
- `docs/marketplace-pipeline.md` documenta el procedimiento

## Criterios de Aceptacion

- Secret `MARKETPLACE_TOKEN` existe en rootline repo settings (verificable en GitHub UI)
- `workflow_dispatch` de publish-marketplace ejecuta sin errores de autenticación
- Agent-marketplace recibe commit del sync automático
- Workflow logs muestran "Successfully pushed to agent-marketplace"

## Fuente de verdad

- `docs/marketplace-pipeline.md` (procedimiento paso a paso)
- `.github/workflows/publish-marketplace.yml` (workflow que consume el token)
