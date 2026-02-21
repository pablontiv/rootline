---
estado: Completado
tipo: documentation
ejecutable_en: 1 sesion
---
# T005: Fix != operator zsh escaping en help text y docs

**Story**: [S002 Internal Edge Cases](README.md)

## Contexto

Zsh con `HIST_EXPAND` (habilitado por defecto) interpreta `!` dentro de comillas dobles como expansion de historial y lo escapea a `\!`. Cuando `rootline query --where "tags != nil"` se ejecuta desde zsh, el shell transforma la expresion a `tags \!= nil`, causando que el parser `expr-lang/expr` falle con `unrecognized character: U+005C '\'`.

El operador `!=` funciona correctamente en `expr-lang/expr` — el problema es exclusivamente de quoting en la invocacion shell. El help text embebido en `cmd/rootline/query.go` muestra ejemplos con `!=` dentro de comillas dobles, lo cual induce al error al copiar-pegar en zsh.

## Alcance

**In**:
1. `cmd/rootline/query.go` linea 21 — cambiar el ejemplo `rootline query --where "tags != nil"` a usar comillas simples: `rootline query --where 'tags != nil'`
2. `docs/epics/E04-dx-advanced/F08-edge-case-coverage/S002-internal-edge-cases/T003-expr-eval-edge-cases.md` linea 20 — el criterio de aceptacion referencia `estado != nil` como expresion. Agregar nota aclarando que al invocar desde zsh se deben usar comillas simples para expresiones con `!=`
3. Verificar que no haya otros archivos en `cmd/rootline/` o `.claude/skills/` con ejemplos de `--where` usando `!=` en comillas dobles

**Out**: Cambios al parser de expr, cambios al shell del usuario, fix de `docs/research/I3` (informativo, no ejecutable)

## Estado inicial esperado

- `cmd/rootline/query.go` existe con `whereExamples` const
- `T003-expr-eval-edge-cases.md` existe en S002
- `go build ./cmd/rootline/` compila sin errores

## Criterios de Aceptacion

- `go build ./cmd/rootline/` compila sin errores tras el cambio
- `go run ./cmd/rootline/ query --help` muestra ejemplos con comillas simples para expresiones que usan `!=`
- Ejecutar en zsh: `go run ./cmd/rootline/ query docs/epics/ --where 'tags != nil' --output table` no produce error de parser
- T003-expr-eval-edge-cases.md contiene nota sobre quoting en zsh

## Fuente de verdad

- `cmd/rootline/query.go` — help text con whereExamples
- `docs/epics/E04-dx-advanced/F08-edge-case-coverage/S002-internal-edge-cases/T003-expr-eval-edge-cases.md` — task de edge cases de expr eval
