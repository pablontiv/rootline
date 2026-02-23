---
tipo: historia
cliente: Platform Owner
---
# S003: Graph Command

**Feature**: [F05 Dependency Graph](../README.md)
**Capacidad**: Rootline construye un grafo dirigido de dependencias entre documentos, detecta ciclos, y genera diagramas en formato DOT o mermaid

## Antes / Despues

**Antes**: No hay forma de visualizar relaciones entre documentos. Si un documento bloquea a otro via `[[blocks:T003]]`, esa relacion es invisible excepto leyendo el archivo. No hay deteccion de dependencias circulares ni de links rotos.

**Despues**: `rootline graph docs/` construye un grafo dirigido desde los links extraidos y genera output en DOT (para Graphviz) o mermaid (para Markdown). `rootline graph --check docs/` detecta ciclos y links rotos (target no existe) y retorna exit code 1 si hay problemas. El grafo muestra estado de cada nodo (Pending, Completado, etc.).

## Criterios de Aceptacion (semanticos)

- [ ] `rootline graph docs/` genera diagrama con nodos y edges
- [ ] `rootline graph --check` detecta ciclos y retorna exit code 1
- [ ] Links rotos (target no existe) se reportan como errores
- [ ] Output soporta formato DOT y mermaid

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-graph-builder-cycle-detection.md) | Construir grafo dirigido con deteccion de ciclos |
| [T002](T002-graph-command-output.md) | Comando graph con output DOT y mermaid |

## Fuente de verdad

- `internal/extract/extract.go` — Record.Links
- `internal/index/index.go` — Scan (para resolver targets)
