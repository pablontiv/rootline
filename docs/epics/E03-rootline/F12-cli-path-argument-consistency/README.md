---
estado: Specified
tipo: feature
---
# F12: CLI Path Argument Consistency

**Epic**: [E03 Rootline](../README.md)
**Objetivo**: `validate --all` y `fix --all` aceptan argumento posicional de directorio, alineados con `tree`/`query`/`stats`/`graph`.
**Satisface**: P1 (validate path), P2 (fix path), P3 (skills correctos)

## Postcondiciones

- P1: `rootline validate --all docs/epics/` usa `docs/epics/` como scan root
- P2: `rootline fix --all docs/epics/` usa `docs/epics/` como scan root
- P3: Todos los comandos documentados en skills funcionan correctamente

## Invariantes

- INV1: Sin argumentos, `--all` sigue usando CWD (backward compatible)
- INV2: Tests existentes siguen pasando

## Stories

| Story | Descripcion |
|-------|-------------|
| [S001](S001-validate-fix-path-support/) | Soporte de path posicional en validate --all y fix --all |
