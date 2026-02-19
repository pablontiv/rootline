# F01: Project Foundation

**Epic**: [E03](../README.md)
**Objetivo**: Rootline existe como proyecto Go buildable, testeable, con CI automatizado
**Beneficio**: Habilita todo el desarrollo posterior con quality gates desde el dia 1
**Milestone**: `go build ./...` exitoso, `go test ./...` ejecuta, GitHub Actions green

## Scope

**In**: Go module init, directory structure, cobra CLI skeleton, GitHub Actions, linting
**Out**: Implementacion de comandos (solo stubs), logic de negocio

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Repository Scaffold](S001-repository-scaffold/) | Go module compilable con estructura de directorios y cobra skeleton |
| S002 | [CI Pipeline](S002-ci-pipeline/) | Cada push ejecuta build, test, lint automaticamente |

## Dependencias

- D11 (GitHub org/user) debe resolverse antes de `go mod init`

## Fuente de verdad

- `src/rootline/` — proyecto standalone
- `src/rootline/docs/intent/v0-rootline.md` — arquitectura y stack
