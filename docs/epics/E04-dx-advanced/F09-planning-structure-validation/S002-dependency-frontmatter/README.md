---
tipo: historia
cliente: Platform Owner
---
# S002: Dependency Wiki-Links

**Feature**: [F09 Planning Structure Validation](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Tasks declaran dependencias via wiki-links `[[blocks:TXXX-name]]` en el body; rootline graph las detecta y valida nativamente

## Antes / Despues

**Antes**: Dependencias entre tasks son texto libre en seccion `## Dependencias` del markdown, o campos `blocks:` en frontmatter YAML que rootline graph no lee. `rootline graph --check` no detecta dependencias entre tasks. `/roadmap loop` ejecuta tasks sin respetar orden de dependencias.

**Despues**: Tasks declaran dependencias con `[[blocks:TXXX-name]]` en el body. El `.stem` tiene seccion `links:` que define `blocks` como tipo de link valido. `rootline graph --check` detecta broken links y ciclos en dependencias. `rootline graph --format mermaid` genera grafo con edges de dependencia.

## Criterios de Aceptacion (semanticos)

- [ ] Un task con `[[blocks:T002-nonexistent]]` es reportado como broken link por `rootline graph --check`
- [ ] Un ciclo T001 blocks T002, T002 blocks T001 es detectado por `rootline graph --check`
- [ ] `rootline graph --format mermaid` muestra edges entre tasks con dependencias

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-add-blocks-schema-to-stem.md) | Agregar seccion links: al .stem de epics con blocks como tipo permitido |
| [T002](T002-validate-dependency-targets-exist.md) | Verificar que rootline graph --check valida targets de wiki-links |
| [T003](T003-integrate-deps-with-graph-check.md) | Migrar 5 tasks existentes de blocks frontmatter a wiki-links en body |

## Fuente de verdad

- `docs/epics/.stem` — schema actual (sin seccion links)
- `internal/rules/rules.go` — LinkSchema struct (ya implementado)
- `internal/extract/links.go` — ParseLinks (ya soporta wiki-links tipados)
- `internal/graph/graph.go` — Graph.Build y DetectCycles (ya funciona con wiki-links)
