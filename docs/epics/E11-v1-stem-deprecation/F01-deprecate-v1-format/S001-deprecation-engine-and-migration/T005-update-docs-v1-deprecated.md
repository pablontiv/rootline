---
estado: Specified
tipo: documentation
ejecutable_en: rootline
---
# T005: Update Documentation with V1 Deprecation

**Story**: [S001 V1 Deprecation Engine & Migration](README.md)
[[blocks:T001-implement-migrate-to-v2]]
[[blocks:T002-add-version-deprecated-stemhealth]]
**Contribuye a**: Docs: `docs/migrate.md` documenta `--to-v2`, `docs/levels.md` tiene banner de deprecación
**Preserva**: INV1 (tests existentes pasan), INV2 (pipeline verde)

## Contexto

La documentación debe reflejar que v1 está deprecado. Afecta 4 archivos de docs.

## Alcance

**In scope**:
- `docs/levels.md` — Agregar banner de deprecación v1 al inicio
- `docs/init.md` — Actualizar ejemplo flat-mode de `version: 1` a `version: 2`
- `docs/validate.md` — Actualizar lista de stem health checks de 7 a 8, mencionar `version-deprecated`
- `docs/migrate.md` — Agregar `--to-v2` al usage, tabla de flags, y nueva sección

**Out of scope**: Reescribir documentación completa de migrate

## Criterios de Aceptación

- [ ] `docs/levels.md` contiene "deprecated" o "Deprecated" en las primeras 10 líneas
- [ ] `docs/init.md` muestra `version: 2` en el ejemplo flat-mode (no `version: 1`)
- [ ] `docs/validate.md` menciona `version-deprecated` como stem health check
- [ ] `docs/migrate.md` documenta `--to-v2` flag
- [ ] `rootline validate --all docs/` pasa sin errores
