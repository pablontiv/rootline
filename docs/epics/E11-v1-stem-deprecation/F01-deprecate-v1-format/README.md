---
estado: Specified
tipo: feature
---
# F01: Deprecate V1 Stem Format

**Epic**: [E11 V1 Stem Format Deprecation](../README.md)
**Objetivo**: Implementar detección, migración y documentación de la deprecación del formato v1 de `.stem`
**Satisface**: P1, P2, P3, P4, P5

## Milestone

El comando `rootline validate --all` emite warnings para stems v1, `rootline migrate --to-v2` migra stems existentes, `rootline init` genera v2 por defecto, y la documentación refleja el cambio.

## Invariantes (heredados de E11)

- INV1: Tests existentes siguen pasando
- INV2: `go test ./... -race` verde

## Stories

| Story | Descripción |
|-------|-------------|
| [S001](S001-deprecation-engine-and-migration/) | Motor de deprecación y migración v1→v2 |
