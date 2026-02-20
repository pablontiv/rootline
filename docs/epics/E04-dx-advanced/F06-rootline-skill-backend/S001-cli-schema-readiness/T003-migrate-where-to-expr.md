---
estado: Completado
tipo: software-module
ejecutable_en: 1 sesion
---
# T003: Migrar --where de parsing manual a expr-lang/expr

**Story**: [S001 CLI & Schema Readiness](README.md)

## Contexto

El flag `--where` de `rootline query` usa parsing manual con `strings.SplitN(expr, " ", 3)` y operadores custom (`eq`, `ne`, `in`, `contains`, `exists`). Este formato `"field op value"` no es estandar en CLIs y el operador `in` con valores comma-separated es fragil. El research I3 ya evaluo `expr-lang/expr` como motor de expresiones recomendado para rootline (zero deps transitivas, 70ns/op, non-Turing complete). Usarlo para `--where` resuelve el parsing y sienta la base compartida para el derivation engine (F04) cuando se reactive.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/query
interfaces:
  - nombre: CompileWhere
    metodos:
      - nombre: CompileWhere
        input: "whereExpr string"
        output: "*vm.Program, error"
  - nombre: BuildEnv
    metodos:
      - nombre: BuildEnv
        input: "rec *extract.Record"
        output: "map[string]any"
  - nombre: MatchRecord
    metodos:
      - nombre: MatchRecord
        input: "program *vm.Program, rec *extract.Record"
        output: "bool, error"
  - nombre: ExecuteExpr
    metodos:
      - nombre: ExecuteExpr
        input: "records []*extract.Record, whereExpr string, q *Query"
        output: "any, error"
dependencias_externas:
  - github.com/expr-lang/expr
tests:
  - "estado == 'Pending'" filtra correctamente
  - "estado in ['Pending', 'Especificado']" retorna ambos estados
  - "tipo == 'lxc' && estado == 'Pending'" combina con AND
  - "tipo == 'lxc' || tipo == 'vm'" combina con OR
  - "body contains 'Migration'" busca en body
  - Campo inexistente retorna false sin panic
  - Expresion invalida retorna error claro
```

## Dependencias

- T001 completado (StringArrayVar ya en query.go)
- T002 completado (enum tipo ampliado en .stem)

## Alcance

**In**:
1. Agregar dependencia `github.com/expr-lang/expr` a go.mod
2. Crear `internal/query/expr_eval.go` con CompileWhere, BuildEnv, MatchRecord, ExecuteExpr
3. Crear `internal/query/expr_eval_test.go` con tests unitarios
4. Modificar `cmd/rootline/query.go`: reemplazar parseWhereFlags/parseWhereExpr con ExecuteExpr
5. Actualizar tests CLI en `cmd/rootline/commands_test.go` a nueva sintaxis
6. Eliminar parseWhereExpr y parseWhereFlags
7. Actualizar help text del flag --where con ejemplos de nueva sintaxis

**Out**: Migrar query.Condition/Execute interno (se mantiene para uso programatico), cambiar API del paquete query, modificar e2e tests

## Estado inicial esperado

- `cmd/rootline/query.go` con parseWhereExpr usando SplitN
- `internal/query/query.go` con Condition/Execute funcional
- `go.mod` sin expr-lang/expr

## Criterios de Aceptacion

- `go test ./internal/query/ -race` pasa (tests expr + tests Condition existentes)
- `go test ./cmd/rootline/ -race` pasa (tests CLI actualizados)
- `go test ./internal/e2e/ -race` pasa (sin cambios)
- `go vet ./...` sin errores
- `rootline query docs/epics/ --where "estado == 'Pending'"` retorna tasks pendientes
- `rootline query docs/epics/ --where "estado in ['Pending', 'Especificado']"` retorna ambos
- `rootline query docs/epics/ --where "estado == 'Pending' && tipo == 'software-module'" --output table` filtra con AND

## Fuente de verdad

- `cmd/rootline/query.go` — CLI layer con parseWhereExpr a reemplazar
- `internal/query/query.go` — Query engine interno (no modificar)
- `internal/query/shortcuts.go` — Field shortcuts (no modificar)
- `docs/research/I3-derivation-pre-research.md` — Evaluacion de expr-lang/expr
