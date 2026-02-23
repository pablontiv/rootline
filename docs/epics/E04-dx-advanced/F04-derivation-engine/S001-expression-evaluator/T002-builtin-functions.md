---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Implementar funciones builtin para expresiones de derivacion

**Story**: [S001 Expression Evaluator](README.md)

## Contexto

El evaluador base (T001) ejecuta expresiones pero sin funciones utiles. Se necesitan builtins para transformacion de texto (slugify, lower, upper, trim) y agregacion (count, any, all, len). Estas funciones se registran en el environment de expr-lang via expr.Function().

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/derive
interfaces:
  - nombre: BuiltinFunctions
    metodos:
      - nombre: slugify
        input: "string"
        output: "string"
      - nombre: count
        input: "collection []map[string]any, predicate string"
        output: "int"
      - nombre: any
        input: "collection []map[string]any, predicate string"
        output: "bool"
      - nombre: all
        input: "collection []map[string]any, predicate string"
        output: "bool"
dependencias_externas: []
tests:
  - slugify("Hello World!") retorna "hello-world"
  - lower("HELLO") retorna "hello"
  - upper("hello") retorna "HELLO"
  - trim("  hello  ") retorna "hello"
  - len("hello") retorna 5
  - count con predicate filtra correctamente
  - any con predicate retorna true si al menos uno matchea
  - all con predicate retorna true si todos matchean
```

## Dependencias

- T001 (Evaluator con Compile/Eval)

## Alcance

**In**:
1. Archivo `internal/derive/builtins.go`
2. Funciones de texto: slugify, lower, upper, trim
3. Funciones de inspeccion: len (string length o array length)
4. Funciones de agregacion: count, any, all (operan sobre listas de records/maps)
5. slugify: lowercase, replace spaces/special chars con hyphens, strip non-alphanumeric
6. Registrar funciones en environment de expr via expr.Function()
7. Tests unitarios por funcion

**Out**: Funciones de fecha, funciones matematicas avanzadas, funciones custom definidas por usuario

## Estado inicial esperado

- internal/derive/ con Evaluator funcional (T001)
- expr-lang/expr en go.mod

## Criterios de Aceptacion

- `Eval("slugify(titulo)", {titulo: "Hello World!"})` retorna "hello-world"
- `Eval("lower(x)", {x: "HELLO"})` retorna "hello"
- `Eval("len(items)", {items: [1,2,3]})` retorna 3
- `Eval("count(children, estado == 'Completado')", {children: [...]})` retorna conteo correcto
- `Eval("any(children, estado == 'Pending')", {children: [...]})` retorna bool correcto
- Todas las funciones son side-effect-free
- `go test ./internal/derive/ -race` pasa

## Fuente de verdad

- `internal/derive/derive.go` — Evaluator (T001)
- expr-lang/expr docs — Function registration API
