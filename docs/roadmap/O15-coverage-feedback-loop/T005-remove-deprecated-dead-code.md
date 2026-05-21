---
estado: Completed
tipo: task
---
# T005: Eliminar código deprecado/muerto que arrastra coverage

**Outcome**: [O15 Coverage feedback loop](README.md)
**Contribuye a**: subir el porcentaje de coverage sin agregar tests, removiendo statements del denominador

## Preserva

- INV1 del outcome: el gate no se relaja — quitar dead code es lo opuesto a relajarlo (sube la calidad sin cambiar el umbral)
  - Verificar: `go test ./...` y `rootline validate --all docs/roadmap/` siguen verdes después del borrado

## Contexto

Hay funciones que viven en el conteo de statements (denominador del %) sin servir al producto:

1. **`DetectMissingDomains` (`internal/infer/domain_coverage.go:9`)** — retorna `nil` siempre. Comentario en el archivo: "deprecated — domain: field was removed in O14 refactor". Es claro que se puede borrar. Hay un test `domain_coverage_test.go` creado en PR #36 que valida que retorna nil — también borrar.

2. **`cmd/rootline/apply.go`** — comando `apply` está marcado `**deprecated**` en CLAUDE.md ("emit a warning and direct users to `schema apply` and `repair apply`; remains functional for backward compatibility"). Evaluar:
   - ¿Hay tests, skills, scripts que aún lo invocan? → grep `rootline apply` en docs/, .claude/skills/, scripts/.
   - Si **sí** queda código vivo dependiente: NO borrar el comando, pero sí `renderApplyTable` si su único caller es `runApply` y `runApply` ya no llega a esa rama.
   - Si **no**: borrar `apply.go` completo + tests asociados.

3. **`cmd/rootline/trace.go`** — `runTrace` está en 0%. Verificar:
   - ¿Está documentado el comando en CLAUDE.md o README? grep `rootline trace`.
   - ¿Hay skills/scripts que lo invocan?
   - Si no se usa, borrar.

Cada función borrada ~0.1–0.3 puntos de coverage total sin escribir tests.

## Alcance

**In**:
1. Borrar `internal/infer/domain_coverage.go` + `domain_coverage_test.go` + cualquier call site (debería ser ninguno).
2. Evaluar `cmd/rootline/apply.go` y `renderApplyTable`: borrar si confirmados muertos, o documentar por qué quedan.
3. Evaluar `cmd/rootline/trace.go` y `runTrace`: borrar si confirmados muertos.
4. Actualizar CLAUDE.md si se borra `apply` o `trace` para reflejar los comandos vigentes.

**Out**:
- No tocar comandos activos (`validate`, `query`, `describe`, `tree`, `stats`, `graph`, `analyze`, `schema`, `repair`, `fix`, `init`, `new`, `set`, `migrate`, `hooks`, `completion`).
- No refactorizar lo que queda.

## Estado inicial esperado

- `go build ./...` y `go test ./...` verdes.
- `domain_coverage.go` existe con `func DetectMissingDomains(...) []Inference { return nil }`.

## Criterios de Aceptación

- `internal/infer/domain_coverage.go` y su test asociado eliminados.
- `apply.go` / `trace.go` evaluados explícitamente: cualquiera de los dos resultados es aceptable, pero la decisión debe quedar registrada (en el commit o un comentario).
- `go build ./...` y `go test ./... -race` verdes.
- `rootline validate --all docs/roadmap/` verde.
- `golangci-lint run` sin issues nuevos (incluyendo unused-import si quedan).

## Fuente de verdad

- `/home/shared/rootline/internal/infer/domain_coverage.go` + test
- `/home/shared/rootline/cmd/rootline/apply.go`, `trace.go`
- `/home/shared/rootline/CLAUDE.md` — sección de comandos
- `/home/shared/rootline/.claude/skills/rootline/` — invocaciones desde skills
