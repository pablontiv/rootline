---
estado: Pending
tipo: feature
---
# F04: Derivation Engine

**Epic**: [E04](../README.md)
**Objetivo**: Rootline evalua expresiones declaradas en `.stem` derive para computar campos derivados en query time, sin modificar archivos fuente
**Beneficio**: Campos computados (slug, completion_pct, is_blocked) eliminan duplicacion manual y habilitan state derivation. El comando explain traza el origen de cada campo.
**Milestone**: `derive: { slug: "slugify(titulo)" }` en .stem produce campo derivado en query results. `rootline explain file.md` muestra origen de cada campo (schema, derived, validated).

## Scope

**In**: Expression evaluator con expr-lang/expr, builtin functions, derivation pipeline integration, explain command
**Out**: Level 3-4 derivation (link traversal, recursive), custom expression languages, derive-based validation, writing derived values to files

## Stories

| ID | Nombre | Capacidad |
|----|--------|-----------|
| S001 | [Expression Evaluator](S001-expression-evaluator/) | Motor de evaluacion de expresiones con funciones builtin |
| S002 | [Derivation Pipeline](S002-derivation-pipeline/) | Campos derivados aparecen en query results y explain los traza |

## Dependencias

- F01-F03 completados (core estable)
- Derive field ya parsea en StemFile (internal/rules/rules.go)

## Fuente de verdad

- `internal/rules/rules.go` — StemFile.Derive (map[string]any, reservado)
- `cmd/rootline/explain.go` — stub existente
- `docs/research/I3-derivation-pre-research.md` — expression language evaluation
