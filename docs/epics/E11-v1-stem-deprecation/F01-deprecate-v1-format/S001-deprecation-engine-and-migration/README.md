---
estado: Completed
tipo: historia
---
# S001: V1 Deprecation Engine & Migration

**Feature**: [F01 Deprecate V1 Stem Format](../README.md)
**Capacidad**: El sistema detecta stems v1 como obsoletos, ofrece migración automática, y genera nuevos schemas en v2.
**Cubre**: Milestone completo del Feature — detección + migración + init + dogfooding + docs

## Antes / Despues

**Antes**: v1 y v2 coexisten silenciosamente. `rootline init` genera v1 para schemas planos. No hay forma automática de migrar stems sin `levels:` a v2. El propio repo tiene stems en v1.

**Despues**: `rootline validate --all` advierte sobre stems v1. `rootline migrate --to-v2` migra bulk. `rootline init` genera v2. El repo practica lo que predica.

## Criterios de Aceptacion (semanticos)

- [ ] Detección: `rootline validate --all docs/` muestra `version-deprecated` para stems v1
- [ ] Migración: `rootline migrate --to-v2 <path>` convierte stems v1/v0 a v2
- [ ] Init: `rootline init --dry-run` genera `version: 2` (no `version: 1`)
- [ ] Dogfooding: `rootline validate --all` sin warnings de version en el repo
- [ ] Docs: `docs/migrate.md` documenta `--to-v2`, `docs/levels.md` tiene banner de deprecación

## Invariantes

- INV1: Tests existentes siguen pasando
  - Verificar: `go test ./... -race`
- INV2: El pipeline completo pasa verde
  - Verificar: `go test ./... -race && go vet ./...`

## Tasks

| Task | Descripción |
|------|-------------|
| [T001](T001-implement-migrate-to-v2.md) | Implementar `rootline migrate --to-v2` |
| [T002](T002-add-version-deprecated-stemhealth.md) | Agregar check `version-deprecated` en stem health |
| [T003](T003-update-init-to-v2.md) | Cambiar `rootline init` flat-mode a generar v2 |
| [T004](T004-run-migrate-to-v2-on-repo.md) | Ejecutar migración en el propio repo |
| [T005](T005-update-docs-v1-deprecated.md) | Actualizar documentación con deprecación v1 |
