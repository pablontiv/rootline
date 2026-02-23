---
tipo: historia
cliente: Platform Owner
---
# S001: Body Text Scanning

**Feature**: [F10 Proposal-Based Fix Engine](../README.md)
**Capacidad**: Rootline puede detectar metadata estructurada en el body de archivos markdown (patron `**Key**: Value`), habilitando extraccion automatica a YAML frontmatter

## Antes / Despues

**Antes**: Rootline solo lee YAML frontmatter. Si un README tiene `**Estado**: Completada` en el body (formato comun en docs legacy), rootline no lo ve — reporta "required field missing" sin pista de que la informacion ya existe en el archivo.

**Despues**: `extract.ScanBodyFields()` parsea body text y detecta patrones bold-colon. El proposal engine usa esto para sugerir extraccion a frontmatter en vez de agregar defaults ciegos.

## Criterios de Aceptacion (semanticos)

- [ ] `ScanBodyFields("**Estado**: Completada")` retorna `{"estado": "Completada"}`
- [ ] Body sin patrones bold-colon retorna mapa vacio
- [ ] `go test ./internal/extract/ -run TestScanBody -v` pasa

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-scan-body-fields.md) | Implementar `ScanBodyFields()` en internal/extract con tests |

## Fuente de verdad

- `internal/extract/extract.go` — package a extender
- `internal/extract/extract_test.go` — tests existentes como patron
