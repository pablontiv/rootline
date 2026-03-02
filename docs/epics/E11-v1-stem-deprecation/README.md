---
estado: Completed
---
# E11: V1 Stem Format Deprecation

**Intención**: Marcar formalmente el formato v1 de `.stem` como obsoleto, proporcionar herramienta CLI de migración bulk, y asegurar que todo nuevo schema nace en v2.

## Postcondiciones

- P1: `rootline validate --all` emite warning `version-deprecated` para todo `.stem` con version != 2
- P2: `rootline migrate --to-v2` existe y convierte version:0/1 → version:2 preservando formato
- P3: `rootline init` genera `version: 2` siempre
- P4: Todos los `.stem` del propio repo rootline son version: 2
- P5: Documentación refleja v1 como deprecado y v2 como estándar

## Invariantes

- INV1: Tests existentes (179 stems con `version: 1`) siguen pasando — backward compatibility no se rompe
- INV2: El pipeline `go test ./... -race` pasa verde antes y después

## Out of Scope

- Migrar los 179 test stems de version:1 a version:2 — siguen verificando backward compat
- Hacer que v1 sea un error (solo warning)
- Remover el código que parsea v1

## Features

| Feature | Descripción |
|---------|-------------|
| [F01](F01-deprecate-v1-format/) | Deprecar formato v1 con tooling y docs |
