# S001: Repository Scaffold

**Feature**: [F01 Project Foundation](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Rootline compila como binario Go con estructura de directorios alineada a la arquitectura

## Antes / Despues

**Antes**: No existe codigo. El diseno de Rootline vive solo en documentos de intent y research. No hay Go module, no hay estructura de paquetes, no hay CLI ejecutable.

**Despues**: Go module inicializado con `go mod init`. Estructura de directorios (`internal/extract/`, `internal/rules/`, `internal/index/`, `internal/query/`, `cmd/rootline/`) creada. Cobra root command con subcommand stubs compila y ejecuta `rootline --help`.

## Criterios de Aceptacion (semanticos)

- [ ] `go build ./cmd/rootline/` produce binario sin errores
- [ ] `rootline --help` muestra los 7 subcomandos (query, validate, describe, tree, stats, serve, explain)
- [ ] Estructura de directorios refleja la arquitectura del intent document

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-init-go-module.md) | Inicializar Go module con estructura de directorios |
| [T002](T002-cobra-cli-skeleton.md) | Crear cobra root command con subcommand stubs |

## Fuente de verdad

- `src/rootline/docs/intent/v0-rootline.md` — arquitectura (seccion 3)
- `src/rootline/docs/research/I7-extractors-architecture.md` — estructura de paquetes
