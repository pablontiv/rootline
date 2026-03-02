---
estado: Completed
tipo: refactor
ejecutable_en: rootline
---
# T004: Run `migrate --to-v2` on Rootline Repository

**Story**: [S001 V1 Deprecation Engine & Migration](README.md)
[[blocks:T001-implement-migrate-to-v2]]
**Contribuye a**: Dogfooding: `rootline validate --all` sin warnings de version en el repo
**Preserva**: INV1 (tests existentes pasan), INV2 (pipeline verde)

## Contexto

El propio repo rootline tiene stems con `version: 1` (`docs/.stem`, `docs/research/.stem`) y stems sin version field (~65 bajo `epics/`). Usar la herramienta `--to-v2` creada en T001 para migrar todo.

## Alcance

**In scope**:
- Ejecutar `rootline migrate --to-v2 docs/`
- Ejecutar `rootline migrate --to-v2 docs/epics/`
- Verificar con `rootline validate --all docs/` y `rootline validate --all docs/epics/`
- Verificar que no hay warnings `version-deprecated`

**Out of scope**: Migrar test fixtures (los 179 test stems permanecen en v1)

## Criterios de Aceptación

- [ ] `rootline validate --all docs/` no emite `version-deprecated` warnings
- [ ] `rootline validate --all docs/epics/` no emite `version-deprecated` warnings
- [ ] `go test ./... -race` pasa verde
