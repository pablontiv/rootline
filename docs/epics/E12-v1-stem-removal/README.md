---
estado: Completed
tipo: feature
---
# E12: V1 Stem Removal

**Metrica de exito**: `rootline validate` rechaza stems v1 con error; no existe codigo de migracion v1 en el binario
**Timeline**: 2026-Q1 — en curso

## Intencion

E11 depreco el formato v1 de `.stem` con warnings y tooling de migracion. E12 cierra el ciclo: eliminar soporte v1 por completo. Stems sin version o con version 1 producen error hard. Esto simplifica el engine eliminando branches condicionales y codigo de migracion que ya no se necesita.

## Postcondiciones

- P1: `rootline validate` con stem v1 produce error, no warning
- P2: No existe codigo de migracion v1→v2 en el binario (`--from-levels`, `--to-v2` eliminados)
- P3: `go test ./... -race` pasa verde sin stems v1 en tests (~179 migrados a v2)
- P4: Documentacion no referencia v1 como formato soportado

## Invariantes

- INV1: `go test ./... -race` pasa verde en cada commit
- INV2: Coverage ≥85% (`go test ./... -race -coverprofile=coverage.out && go tool cover -func=coverage.out | tail -1`)

## Out of Scope

- Cambios al formato v2 (schema, match, etc.)
- Migracion automatica transparente (v1 se rechaza, no se convierte)
- Soporte para v3 (research separada)

## Features

| ID | Nombre | Descripcion |
|----|--------|-------------|
| F01 | [Hard-reject v1 stems](F01-hard-reject-v1/) | Engine rechaza version 0/1 en parse time, tests migrados |
| F02 | [Remove v1 migration tooling](F02-remove-v1-migration-tooling/) | Eliminar archivos, flags CLI, y docs de migracion v1 |

## Orden de Ejecucion

| Feature | Depende de | Razon |
|---------|-----------|-------|
| F01 | — | Foundation: el rechazo habilita que el codigo de migracion sea dead code |
| F02 | F01 | Cleanup: solo se puede borrar codigo que ya es inalcanzable |

## Decision Log

| Fecha | Decision | Razon |
|-------|----------|-------|
| 2026-03-02 | Error hard (no warning ni auto-upgrade) | Simplifica engine, fuerza migracion explicita |
| 2026-03-02 | Epic separado de E11 | E11 Out of Scope excluye remocion; objetivo distinto |

## Gaps Activos

- Ninguno identificado
