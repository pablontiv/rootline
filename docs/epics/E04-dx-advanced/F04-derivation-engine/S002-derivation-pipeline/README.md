---
tipo: historia
cliente: Platform Owner
---
# S002: Derivation Pipeline

**Feature**: [F04 Derivation Engine](../README.md)
**Capacidad**: Campos derivados definidos en .stem se evaluan en el pipeline y aparecen en query results, y el comando explain traza el origen de cada campo

## Antes / Despues

**Antes**: El pipeline es Extraction → Parsing → Rule Loading → Validation → Query. El slot de derivacion esta reservado pero es no-op. El comando explain es un stub que imprime "not implemented yet". Query results solo contienen campos de frontmatter.

**Despues**: El pipeline incluye derivacion: para cada record, evalua las expresiones de .stem derive y agrega resultados al record como campos derivados (namespace `derived.*`). Query results incluyen campos derivados. `rootline explain file.md` muestra para cada campo su origen (.stem source, tipo, expresion si es derivado, valor).

## Criterios de Aceptacion (semanticos)

- [ ] .stem con `derive: { slug: "slugify(titulo)" }` produce campo derivado en query results
- [ ] Campos derivados son accesibles via `--field derived.slug`
- [ ] `rootline explain file.md` muestra origen de cada campo (schema, derive, validate)
- [ ] Campos derivados nunca se escriben a archivos fuente

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-derive-evaluator.md) | Evaluar expresiones de .stem derive contra records |
| [T002](T002-pipeline-integration.md) | Insertar derivacion en pipeline de query/tree/stats |
| [T003](T003-explain-command.md) | Implementar comando explain con tracing completo |

## Fuente de verdad

- `internal/derive/` — Evaluator, builtins (S001)
- `internal/rules/rules.go` — StemFile.Derive
- `cmd/rootline/explain.go` — stub existente
- `cmd/rootline/query.go`, `tree.go`, `stats.go` — pipeline consumers
