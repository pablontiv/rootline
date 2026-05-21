---
estado: Specified
tipo: task
---
# T008: Documentar el coverage workflow en CLAUDE.md

**Outcome**: [O15 Coverage feedback loop](README.md)
**Contribuye a**: agentes y desarrolladores futuros conocen el workflow sin tener que descubrirlo via fallo

[[blocked_by:./T006-add-coverage-tooling.md]]
[[blocked_by:./T007-activate-prepush-coverage-gate.md]]

## Preserva

- INV3 del outcome: el feedback ocurre antes del push — documentar el flujo previene que devs lo descubran por error
  - Verificar: CLAUDE.md describe `just coverage` y el pre-push gate explícitamente

## Contexto

CLAUDE.md tiene dos secciones donde debe entrar el cambio:

- **`## Build & Test Commands`** (alrededor de la línea 13): listar el recipe `just coverage` y `just coverage-check`. Mencionar que `just test` no mide coverage (recipes separados intencionalmente para velocidad).
- **`## CI Workflows`** (alrededor de la línea 121): mencionar que el gate de 85% se duplica localmente vía pre-push, y que existe `.coverage-floors.toml` con piso uniforme.

Además, el pre-push hook ya tiene la regla de "si cambia CI/scripts/config, debe cambiar docs" — esta task lo satisface naturalmente porque sí actualiza docs.

## Alcance

**In**:
1. Agregar líneas a `## Build & Test Commands`:
   - `just coverage             # run tests with coverage, print per-package + total`
   - `just coverage-check       # like coverage, but fails if total < 85 or any package < 85`
2. Agregar a `## CI Workflows`:
   - Nota sobre `.coverage-floors.toml` con `default = 85` (mismo umbral que `coverage-threshold` en ci.yml).
   - Nota sobre el pre-push gate (`just coverage-check` corre automáticamente al push si cambia algún `.go`).
   - Recordatorio: NO bypassear con `--no-verify` salvo emergencia documentada.

**Out**:
- No cambiar `README.md` (CLAUDE.md es el punto de entrada para agentes en este repo).
- No documentar internals del script `scripts/check-coverage-floors.sh` — sólo cómo usar las recipes.

## Estado inicial esperado

- T006 y T007 completadas: las recipes y el hook existen y funcionan.
- CLAUDE.md no menciona `just coverage` ni `.coverage-floors.toml`.

## Criterios de Aceptación

- CLAUDE.md sección `## Build & Test Commands` incluye `just coverage` y `just coverage-check`.
- CLAUDE.md sección `## CI Workflows` incluye nota sobre `.coverage-floors.toml` y pre-push gate.
- `rootline validate /home/shared/rootline/CLAUDE.md` no aplica (no es un archivo del roadmap), pero el pre-push hook acepta el commit (porque CI/scripts no cambian).
- El commit que cierra esta task no toca código Go.

## Fuente de verdad

- `/home/shared/rootline/CLAUDE.md` — secciones existentes `## Build & Test Commands` y `## CI Workflows`
