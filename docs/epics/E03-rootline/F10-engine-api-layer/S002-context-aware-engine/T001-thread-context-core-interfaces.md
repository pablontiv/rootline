---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T001: Add context.Context to core engine interfaces

**Story**: [S002 Context-Aware Engine](README.md)

[[blocks:T001-extract-fix-engine]]
[[blocks:T002-extract-migrate-split]]
[[blocks:T003-extract-doctor-engine]]

## Contexto

Actualmente 0 funciones en `internal/` aceptan `context.Context`. Esto impide cancelación limpia (Ctrl+C), timeouts por request en el MCP server, y manejo de requests concurrentes. Este task agrega `ctx context.Context` como primer parámetro a todas las funciones públicas de los paquetes core, y agrega checkpoints de cancelación en loops de I/O.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: internal/index, internal/rules, internal/derive, internal/graph, internal/query, internal/fix, internal/doctor
interfaces:
  - nombre: index.Scan
    metodos:
      - nombre: Scan
        input: "ctx context.Context, rootPath string, registry *extract.Registry, opts ...ScanOption"
        output: "[]*extract.Record, error"
  - nombre: rules.Validate
    metodos:
      - nombre: Validate
        input: "ctx context.Context, record *extract.Record, effective *StemFile"
        output: "[]ValidationError"
  - nombre: derive.DeriveAll
    metodos:
      - nombre: DeriveAll
        input: "ctx context.Context, records []*extract.Record, root string, resolver StemResolver"
        output: ""
  - nombre: derive.AggregateAll
    metodos:
      - nombre: AggregateAll
        input: "ctx context.Context, records []*extract.Record, root string, resolver StemResolver"
        output: ""
  - nombre: graph.Build
    metodos:
      - nombre: Build
        input: "ctx context.Context, records []*extract.Record"
        output: "*Graph"
  - nombre: query.ExecuteExpr
    metodos:
      - nombre: ExecuteExpr
        input: "ctx context.Context, records []*extract.Record, whereExpr string, q *Query"
        output: "any, error"
  - nombre: fix.ApplyProposals
    metodos:
      - nombre: ApplyProposals
        input: "ctx context.Context, report *proposal.Report, root string, records []*extract.Record"
        output: "error"
  - nombre: doctor.RunChecks
    metodos:
      - nombre: RunChecks
        input: "ctx context.Context, absRoot string"
        output: "*Result, error"
dependencias_externas: []
tests:
  - Scan retorna error cuando ctx está cancelado antes de completar walk
  - DeriveAll respeta cancelación entre iteraciones de records
  - Todos los tests existentes pasan con context.Background()
```

## Dependencias

- S001 debe estar completa (internal/fix y internal/doctor deben existir)

## Alcance

**In**:
1. Agregar `ctx context.Context` como primer parámetro en funciones públicas de: `index.Scan`, `rules.Validate`, `rules.ValidateDirectory`, `derive.DeriveAll`, `derive.DeriveAllSimple`, `derive.AggregateAll`, `derive.AggregateAllSimple`, `derive.DeriveRecord`, `graph.Build`, `query.ExecuteExpr`, `query.MatchRecord`, `fix.ApplyProposals`, `fix.ApplyFixes`, `doctor.RunChecks`
2. En `index.Scan`: agregar check `ctx.Err()` dentro del `WalkDir` callback (cada directorio)
3. En `derive.DeriveAll` y `AggregateAll`: agregar check `ctx.Err()` en el loop de records
4. En `doctor.RunChecks`: agregar check `ctx.Err()` entre cada check
5. Actualizar TODOS los call sites en tests a usar `context.Background()`
6. NO actualizar call sites en `cmd/rootline/` — eso es T002

**Out**: No agregar context a funciones helper internas (unexported). No implementar graceful shutdown del CLI. No agregar timeouts — solo propagación.

## Estado inicial esperado

- `internal/fix/` y `internal/doctor/` existen (S001 completada)
- Ninguna función en `internal/` usa context.Context
- Todos los tests pasan

## Criterios de Aceptacion

- `go test ./... -race` pasa (todos los tests actualizados con context.Background())
- `go vet ./...` limpio
- `grep -r "context.Context" internal/` muestra matches en index, rules, derive, graph, query, fix, doctor
- `index.Scan` tiene checkpoint de cancelación dentro del WalkDir
- `derive.DeriveAll` tiene checkpoint de cancelación en el loop

## Fuente de verdad

- `internal/index/index.go` — Scan()
- `internal/rules/validate.go` — Validate()
- `internal/rules/structural.go` — ValidateDirectory()
- `internal/derive/pipeline.go` — DeriveAll(), DeriveAllSimple()
- `internal/derive/aggregate.go` — AggregateAll(), AggregateAllSimple()
- `internal/derive/record.go` — DeriveRecord()
- `internal/graph/graph.go` — Build()
- `internal/query/expr_eval.go` — ExecuteExpr(), MatchRecord(), CompileWhere()
