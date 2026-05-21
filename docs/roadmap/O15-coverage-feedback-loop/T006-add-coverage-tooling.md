---
estado: Completed
tipo: task
---
# T006: Agregar tooling de coverage (recipes + per-package floors)

**Outcome**: [O15 Coverage feedback loop](README.md)
**Contribuye a**: hacer visible la cobertura localmente y declarar el piso por paquete (INV3)

## Preserva

- INV2 del outcome: el piso es 85% uniforme; ningún paquete recibe descuento
  - Verificar: el archivo `.coverage-floors.toml` declara `default = 85` y no contiene overrides bajo 85
- INV3 del outcome: el feedback ocurre antes del push
  - Verificar: `just coverage-check` corre localmente sin necesidad de red ni CI

## Contexto

Hoy `just test` corre `go test ./... -race` sin `-coverprofile`. El developer no puede medir cobertura antes de pushear; la primera alarma es CI fallando con coverage <85.

Esta task añade dos piezas:

1. **Recipes `coverage` y `coverage-check`** en `Justfile`. Reusan el patrón ya validado en CI antes de migrar a crossbeam (commit `410c18e`):
   ```
   COV=$(go tool cover -func=coverage.out | grep total | awk '{print substr($3,1,length($3)-1)}')
   if (( $(echo "$COV < 85" | bc -l) )); then ... fi
   ```

2. **`.coverage-floors.toml`** declarativo + **`scripts/check-coverage-floors.sh`** que lo lee. El TOML expresa el piso uniforme:
   ```toml
   default = 85
   packages = ["cmd/rootline", "internal/derive", ...]
   ```
   El script bash (~25 líneas) parsea `coverage.out`, calcula % por paquete (`go tool cover -func` + agrupación por path), compara contra `default`, y emite exit 1 con lista de paquetes en falta si alguno está por debajo.

`just coverage-check` invoca ambos checks en cascada: primero total, luego per-package.

## Alcance

**In**:
1. Editar `Justfile` añadiendo recipes `coverage` y `coverage-check`.
2. Crear `.coverage-floors.toml` con `default = 85` + lista de paquetes actuales.
3. Crear `scripts/check-coverage-floors.sh` con shebang `#!/usr/bin/env bash`, `set -euo pipefail`, lectura del TOML (parsing bash de líneas `key = value` y arrays simples — no se requiere `tomlq`), agrupación de coverage por paquete, exit 1 con mensaje claro.
4. Hacer el script ejecutable (`chmod +x`).

**Out**:
- No activar el pre-push hook (eso es T007).
- No subir coverage de paquetes (eso es T001-T004).
- No cambiar el threshold del CI (sigue en 85 via `coverage-threshold: 85` en ci.yml).
- No agregar coverage measurement al recipe `test` existente — coverage es separado para no slow-down el ciclo de testing rápido.

## Estado inicial esperado

- `Justfile` no contiene recipes `coverage` ni `coverage-check`.
- No existen `.coverage-floors.toml` ni `scripts/check-coverage-floors.sh`.
- `go test ./... -coverprofile=coverage.out && go tool cover -func=coverage.out | grep total` reporta ≥85% (T001–T005 ya completadas).

## Criterios de Aceptación

- `just coverage` imprime tabla per-package y línea final `Total: NN.N%`.
- `just coverage-check` exit 0 cuando todos los paquetes están ≥85 (en el estado post-Fase-A).
- `just coverage-check` exit 1 con mensaje accionable si simulo regresión (e.g. `git rm internal/fuzzy/fuzzy_test.go` → `just coverage-check` falla nombrando `internal/fuzzy`).
- `bash scripts/check-coverage-floors.sh coverage.out .coverage-floors.toml` funciona standalone.
- `shellcheck scripts/check-coverage-floors.sh` sin warnings (o con `# shellcheck disable=...` justificado).

## Fuente de verdad

- `/home/shared/rootline/Justfile` — recipes existentes (check, test, fmt, validate, fix-docs)
- `/home/shared/rootline/.coverage-floors.toml` (nuevo)
- `/home/shared/rootline/scripts/check-coverage-floors.sh` (nuevo)
- Commit `410c18e` para el patrón histórico de coverage check inline
