---
estado: Specified
tipo: task
---
# T002: Subir internal/fix a ≥85% coverage

**Outcome**: [O15 Coverage feedback loop](README.md)
**Contribuye a**: precondición para activar el gate per-package (INV2)

## Preserva

- INV2 del outcome: el piso es 85% uniforme
  - Verificar: `go test ./internal/fix/ -cover` reporta ≥ 85.0%

## Contexto

`internal/fix` está en 73.3% (medido `2026-05-21`). Tras los tests agregados en PR #36 (`repair_extra_test.go`, `schema_proposals_test.go`), las ramas que quedan sin cubrir son:

- `applyRepairSetSection` (repair.go:268) en non-dry-run: delega a `applySetSection` (fix.go:370) sin appendear a `result.Changed` — el branch "create" cuando heading no existe, el branch "replace"/"append" cuando sí existe, y la ruta de error `section not found` con modo distinto a "create".
- `applySetSection` completo: parsing del heading level, búsqueda del próximo heading del mismo nivel, replace vs append.
- `RewriteFrontmatter`: edge cases (frontmatter vacío, sin frontmatter previo, valores con caracteres especiales).
- `applyExtendEnum` y `addEnumValueToNode`: agregar valor cuando ya existe, agregar a enum nested, error de YAML malformado.

Existe `internal/fix/fix_test.go` y `repair_test.go` como referencia.

## Alcance

**In**:
1. Tests para `applySetSection` cubriendo modos `create`, `replace`, append-at-EOF.
2. Tests para `RewriteFrontmatter` cubriendo casos límite.
3. Tests para `applyExtendEnum` cubriendo enum-already-has-value y nested schema fields.
4. Tests para los branches de `applyRepairSetSection` en non-dry-run.

**Out**:
- No refactor de las funciones; solo agregar tests.
- No tocar `internal/fix/repair.go` (el dispatcher central) — ya tiene buena coverage tras PR #36.

## Estado inicial esperado

- `go test ./internal/fix/ -cover` reporta ~73% en master `ca345a3` o posterior.

## Criterios de Aceptación

- `go test ./internal/fix/ -cover` reporta `coverage: 85.0% of statements` o superior.
- `go test ./... -race` verde.
- Diff incluye solo archivos `_test.go` en `internal/fix/`.

## Fuente de verdad

- `/home/shared/rootline/internal/fix/fix.go` — applySetSection, RewriteFrontmatter, applyExtendEnum
- `/home/shared/rootline/internal/fix/repair.go` — applyRepairSetSection
- `/home/shared/rootline/internal/fix/repair_test.go`, `repair_extra_test.go`, `schema_proposals_test.go` — patrones existentes
