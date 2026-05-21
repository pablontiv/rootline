---
estado: Completed
tipo: task
---
# T004: Subir internal/templates a ≥85% coverage

**Outcome**: [O15 Coverage feedback loop](README.md)
**Contribuye a**: precondición para activar el gate per-package (INV2)

## Preserva

- INV2 del outcome: el piso es 85% uniforme
  - Verificar: `go test ./internal/templates/ -cover` reporta ≥ 85.0%

## Contexto

`internal/templates` está en 83.9% (medido `2026-05-21`) — el más cercano al gate. La función con menor cobertura es `loadFromGit` (source.go:24, ~31%). La rama testeada actualmente es happy path; faltan:

- error de `git clone` (repo inexistente, sin internet, ref inválida).
- error al copiar `.stem` files (permisos, paths inválidos).
- timeout de la operación.
- `GIT_TERMINAL_PROMPT=0` honrado (sin prompt interactivo).

Estrategia: usar `t.TempDir()` para un destino, y un repo git local creado en el mismo test (no se necesita red — `git init`, agregar `.stem`, commit, luego clonar desde `file://`). Ver tests existentes en `internal/templates/templates_test.go` para el setup.

## Alcance

**In**:
1. Tests para `loadFromGit` con:
   - repo local válido (file://) que tiene .stem en root → success
   - destino sin permisos de escritura → error claro
   - ref inválida (`owner/repo@nonexistent-tag`) → error
2. Asegurar que el path de error de `Fetch` cubre ambas ramas (clone falla vs. copy falla).

**Out**:
- No mockear `git` — usar repos locales reales (el patrón existente lo hace así).
- No cambiar la firma de `Fetch` o `loadFromGit`.

## Estado inicial esperado

- `go test ./internal/templates/ -cover` reporta ~83.9%.
- Hay tests existentes en `internal/templates/templates_test.go`.

## Criterios de Aceptación

- `go test ./internal/templates/ -cover` reporta `coverage: 85.0% of statements` o superior.
- Tests no requieren red.
- `go test ./... -race` verde (incluyendo no race conditions en el uso temporal de directorios).

## Fuente de verdad

- `/home/shared/rootline/internal/templates/source.go` — loadFromGit
- `/home/shared/rootline/internal/templates/templates.go` — Fetch
- `/home/shared/rootline/internal/templates/templates_test.go` — patrones
