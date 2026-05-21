---
estado: Specified
tipo: task
---
# T004: Adoptar pkcov de picokit en lugar del script bash local

**Contribuye a**: consolidar el tooling de coverage en picokit; rootline pasa de mantener `scripts/check-coverage-floors.sh` (115 líneas bash) a invocar `pkcov check` desde el binario compartido.

## Preserva

- INV: comportamiento de `just coverage-check` post-swap es funcionalmente equivalente al actual
  - Verificar: `just coverage-check` post-swap reporta el mismo output PASS/FAIL/SKIP/TOTAL que pre-swap (TOTAL 88.9%, 13/13 PASS)

## Contexto

O15 está completamente implementada y verde (TOTAL 88.9%, todos los paquetes en sus pisos). El tooling local funciona. Este task es una **refactorización mecánica**: el cálculo se mueve de `scripts/check-coverage-floors.sh` al binario `pkcov` que vive en picokit.

Dependencia cross-repo: picokit `O03-coverage-tooling` (T001–T004) debe estar Completed y el tag publicado antes de ejecutar esta task. No se puede expresar como `blocked_by` porque blockers cruzan repos sólo en prose.

rootline ya consume `github.com/pablontiv/picokit/{fuzzy,pathsec}` desde O15 (en realidad desde antes). Bump del módulo a la versión que incluye `coverage` + `cmd/pkcov` debería ser un cambio en `go.mod` + `go mod tidy`.

Garantía de equivalencia: la suite de tests de picokit/coverage (T002 de O03) incluye un fixture copiado del coverage profile real de rootline; pasa por verificar que pkcov produce las mismas líneas PASS/FAIL/SKIP/TOTAL que el script bash. Si esa equivalencia se respeta, el swap es seguro.

## Alcance

**In**:

1. `go.mod`: bump de `github.com/pablontiv/picokit` al tag que incluye `coverage`/`pkcov`. `go mod tidy`. Verificar que `go build ./...` sigue verde.

2. `Justfile`: cambiar las recipes:
   ```
   coverage:
       go test ./... -coverprofile=coverage.out
       go run github.com/pablontiv/picokit/cmd/pkcov report

   coverage-check: coverage
       go run github.com/pablontiv/picokit/cmd/pkcov check
   ```
   (Usar `go run` evita necesidad de instalar `pkcov` globalmente; los flags default detectan `coverage.out` y `.coverage-floors.toml` automáticamente.)

3. Borrar `scripts/check-coverage-floors.sh`.

4. `CLAUDE.md`: actualizar sección "Build & Test Commands" (sigue mostrando `just coverage` / `just coverage-check`); actualizar sección "CI Workflows" — el `.coverage-floors.toml` y el gate ahora se respaldan por `pkcov` (de picokit), con link a `picokit/docs/coverage-spec.md`. Declarar conformance: "Rootline cumple coverage-spec v1.0".

5. `.githooks/pre-push`: sin cambio (sigue llamando `just coverage-check`).

6. Validación: `just coverage-check` post-cambio produce 13/13 PASS + TOTAL ≥85.

**Out**:

- No tocar el contenido de `.coverage-floors.toml` (la lista de paquetes y default=85 se preservan tal cual).
- No modificar tests (la cobertura no se mueve).
- No bumpear el threshold (sigue 85).
- No tocar CI workflow (`coverage-threshold: 85` en `ci.yml` ya vive en crossbeam y no se relaciona con pkcov).

## Estado inicial esperado

- O15 completa: cobertura 88.9% total, 13/13 PASS con script bash actual.
- picokit O03 T004 Completed: tag publicado con `coverage` + `pkcov`.
- `scripts/check-coverage-floors.sh` existe y funciona.

## Criterios de Aceptación

- `go.mod` muestra `github.com/pablontiv/picokit` en versión que incluye `coverage`/`pkcov`.
- `just coverage-check` exit 0 y output muestra mismo PASS/FAIL/SKIP/TOTAL que pre-swap.
- `scripts/check-coverage-floors.sh` no existe.
- `Justfile` recipes invocan `go run github.com/pablontiv/picokit/cmd/pkcov`.
- `CLAUDE.md` referencia `picokit/docs/coverage-spec.md` y declara conformance v1.0.
- `.githooks/pre-push` ejecuta `just coverage-check` (sin cambios; ya estaba).
- `go test ./... -race` verde.
- `golangci-lint run` sin issues nuevos.

## Fuente de verdad

- `/home/shared/rootline/Justfile`
- `/home/shared/rootline/scripts/check-coverage-floors.sh` (a borrar)
- `/home/shared/rootline/go.mod`
- `/home/shared/rootline/CLAUDE.md`
- `/home/shared/picokit/cmd/pkcov/` — binario destino
- `/home/shared/picokit/docs/coverage-spec.md` — spec referenciada
- docs/roadmap/O15-coverage-feedback-loop/ — implementación local previa
