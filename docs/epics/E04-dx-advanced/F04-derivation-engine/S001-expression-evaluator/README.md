# S001: Expression Evaluator

**Feature**: [F04 Derivation Engine](../README.md)
**Estado**: Specified
**Cliente**: Platform Owner
**Capacidad**: Rootline compila y evalua expresiones declarativas contra records usando expr-lang/expr con funciones builtin para transformacion de texto y agregacion

## Antes / Despues

**Antes**: `derive:` en .stem se parsea como map[string]any pero es completamente no-op. No existe motor de evaluacion. No hay funciones builtin. El campo derivado es un concepto de diseno sin implementacion.

**Despues**: Expresiones como `slugify(titulo)` evaluan contra el frontmatter de un record y retornan valores. Funciones builtin incluyen slugify, lower, upper, trim, len, count, any, all. El evaluador es puro (sin side effects) y siempre termina (expr-lang no es Turing-complete).

## Criterios de Aceptacion (semanticos)

- [ ] Expresion `lower(titulo)` evalua correctamente contra record con campo titulo
- [ ] Funciones builtin slugify, count, any, all funcionan
- [ ] Expresion con campo inexistente retorna error descriptivo
- [ ] El evaluador siempre termina (no hay loops infinitos posibles)

## Tasks

| Task | Descripcion |
|------|-------------|
| [T001](T001-integrate-expr-library.md) | Integrar expr-lang/expr y crear wrapper de evaluacion |
| [T002](T002-builtin-functions.md) | Implementar funciones builtin (slugify, count, any, all, etc.) |

## Fuente de verdad

- `github.com/expr-lang/expr` — expression language library
- `internal/rules/rules.go` — StemFile.Derive
- `internal/extract/extract.go` — Record.Frontmatter (environment para expressions)
