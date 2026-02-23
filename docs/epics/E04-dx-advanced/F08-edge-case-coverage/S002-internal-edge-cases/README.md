---
tipo: historia
cliente: Platform Owner
---
# S002: Internal Edge Cases

**Feature**: [F08 Edge Case Coverage](../README.md)
**Capacidad**: Los packages internos (graph, rules/merge, query/expr, extract) tienen tests de edge cases que cubren ciclos complejos, null removal en merge, type mismatches en expresiones, y caracteres especiales en frontmatter

## Antes / Despues

**Antes**: Los tests internos cubren los happy paths principales. Faltan: ciclos de 4+ nodos, multiples ciclos disjuntos, null removal en merge, boolean literals en expr eval, Unicode en frontmatter, YAML block scalars.

**Despues**: Cada package tiene tests de edge cases. `go test ./internal/... -race` pasa con cobertura de escenarios extremos.

## Criterios de Aceptacion (semanticos)

- [ ] `go test ./internal/graph/ -run TestDetectCycles_FourNode -v` pasa
- [ ] `go test ./internal/rules/ -run TestMerge_NullRemoval -v` pasa
- [ ] `go test ./internal/query/ -run TestExprEval_TypeMismatch -v` pasa (no panic)
- [ ] `go test ./internal/extract/ -run TestExtract_Unicode -v` pasa
- [ ] `go test ./internal/... -race` pasa sin regresiones

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-graph-internal-edge-cases.md) | graph_test.go: ciclos 4 nodos, disjuntos, multiples broken links |
| [T002](T002-merge-edge-cases.md) | merge_test.go: null removal, herencia 4 niveles, override required |
| [T003](T003-expr-eval-edge-cases.md) | expr_eval_test.go: type mismatch, boolean literals, campo ausente |
| [T004](T004-extract-edge-cases.md) | extract_test.go: Unicode, YAML block scalar, sin newline final |
| [T005](T005-fix-ne-operator-zsh-escaping.md) | Fix != operator zsh escaping en help text y docs |
| [T006](T006-fix-resolve-target-basename-fallback.md) | Fix resolveTarget: fallback por basename para wiki-links cross-directory |

## Fuente de verdad

- `internal/graph/graph_test.go` — tests a extender
- `internal/rules/merge_test.go` — tests a extender
- `internal/query/expr_eval_test.go` — tests a extender
- `internal/extract/extract_test.go` — tests a extender
