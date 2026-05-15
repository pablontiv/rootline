---
estado: Completed
tipo: task
---
# T007: Remove `type: section` from rootline

**Outcome**: [O14 Field-agnostic refactor](README.md)
**Contribuye a**: rootline no tiene tipos especiales que mezclan tipo de campo con su fuente de extracción

## Preserva

- INV1: Campos con `source: body.section["..."]` (del nuevo sistema) no se ven afectados
  - Verificar: `go test ./internal/rules/... ./internal/derive/...`
- INV2: `go test ./...` verde
  - Verificar: `cd /home/shared/rootline && go test ./...`

## Contexto

`type: section` era un tipo de campo que mezclaba el tipo de almacenamiento con la fuente de extracción. Con T001, `source: body.section["## Heading"]` + `type: string` cubre el mismo caso de forma ortogonal.

La rama `if field.Type == "section"` en Phase 1 de `internal/rules/validate.go` debe eliminarse. El parser puede ignorar o emitir warning deprecated para `type: section`. Los .stems existentes en el repo que usen `type: section` deben migrarse a `source: body.section[...]`.

## Alcance

**In**:
1. Eliminar `if field.Type == "section"` de Phase 1 Validate en `internal/rules/validate.go`
2. Hacer que el parser ignore `type: section` (o emita `deprecated` warning) en `internal/rules/rules.go`
3. Migrar cualquier .stem en el repo que use `type: section`

**Out**:
- No eliminar `source: body.section[...]` (ese es el reemplazo que debe funcionar)
- No tocar domain: (eso es T006, paralela)

## Estado inicial esperado

- T001 completada (source: body.section funciona como reemplazo)
- `type: section` existe como rama en validate.go

## Criterios de Aceptación

- `if field.Type == "section"` eliminado de validate.go Phase 1
- Parser ignora o advierte sobre `type: section` sin error fatal
- Ningún .stem activo en el repo usa `type: section`
- `go test ./internal/rules/...` verde

## Fuente de verdad

- `internal/rules/validate.go` — Phase 1 validate
- `internal/rules/rules.go` — parser de tipo
