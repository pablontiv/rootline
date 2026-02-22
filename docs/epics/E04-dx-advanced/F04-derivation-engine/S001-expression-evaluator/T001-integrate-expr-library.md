---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Integrar expr-lang/expr y crear wrapper de evaluacion

**Story**: [S001 Expression Evaluator](README.md)

## Contexto

expr-lang/expr es una libreria Go ligera (~3MB, zero deps) para evaluacion de expresiones. Es non-Turing-complete (siempre termina), type-safe, y sin side effects. Se usa en Grafana, Google, Uber. El wrapper debe compilar expresiones, crear un environment desde un Record, y evaluar retornando el resultado.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/derive
interfaces:
  - nombre: Evaluator
    metodos:
      - nombre: Compile
        input: "expression string"
        output: "*CompiledExpr, error"
      - nombre: Eval
        input: "compiled *CompiledExpr, env map[string]any"
        output: "any, error"
dependencias_externas:
  - github.com/expr-lang/expr
tests:
  - Compile expresion valida retorna sin error
  - Compile expresion invalida retorna error descriptivo
  - Eval "lower(titulo)" con env {titulo: "HELLO"} retorna "hello"
  - Eval con campo inexistente retorna error
  - Eval expresion aritmetica "a + b" funciona
```

## Dependencias

- Ninguna (nuevo paquete)

## Alcance

**In**:
1. `go get github.com/expr-lang/expr`
2. Paquete `internal/derive/derive.go`
3. Struct `Evaluator` con metodos Compile y Eval
4. Environment builder: convierte Record.Frontmatter a map para expr
5. Error wrapping con contexto (expresion, campo, error original)
6. Tests unitarios

**Out**: Funciones builtin (T002), integracion con pipeline (S002), caching de compilacion

## Estado inicial esperado

- go.mod existente con Go 1.24.4
- Paquete internal/derive/ no existe (crear)

## Criterios de Aceptacion

- `go get github.com/expr-lang/expr` se agrega a go.mod sin conflictos
- `Compile("1 + 2")` retorna sin error
- `Eval(compiled, map{})` con "1 + 2" retorna 3
- `Compile("invalid !!!")` retorna error descriptivo
- `Eval` con env {"titulo": "Hello"} y expresion `titulo` retorna "Hello"
- `go test ./internal/derive/ -race` pasa

## Fuente de verdad

- `github.com/expr-lang/expr` docs — API reference
- `internal/extract/extract.go` — Record struct (source of env data)
