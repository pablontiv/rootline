# S003: File Scanner

**Feature**: [F02 Core Engine](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Rootline descubre y procesa archivos automaticamente respetando .gitignore y scope matching

## Antes / Despues

**Antes**: Cada consumidor reimplementa file discovery con 4 patrones glob distintos (Python glob, shell glob por ID, fuzzy name match, natural language). No hay mecanismo unificado.

**Despues**: Scanner recorre el arbol de directorios, respeta .gitignore, aplica scope.match del .stem efectivo, y delega a extractors registrados. Un solo mecanismo de discovery para todos los comandos.

## Criterios de Aceptacion (semanticos)

- [ ] Scanner encuentra todos los .md en un arbol de directorios
- [ ] Archivos en .gitignore son excluidos
- [ ] scope.match filtra archivos antes de extraccion

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-directory-scanner.md) | Walk del arbol de directorios con .gitignore |
| [T002](T002-scope-matching.md) | Aplicar scope.match del .stem efectivo |

## Fuente de verdad

- `src/rootline/docs/research/I7-extractors-architecture.md` seccion 6 (Pipeline Integration)
