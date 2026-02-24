---
estado: Completed
tipo: software-module
ejecutable_en: 1 sesion
---
# T002: Wire cobra cmd.Context() to all internal calls

**Story**: [S002 Context-Aware Engine](README.md)

[[blocks:T001-thread-context-core-interfaces]]

## Contexto

Después de T001, todas las funciones públicas de `internal/` aceptan `context.Context` pero `cmd/rootline/` todavía no pasa contexto. Cobra provee `cmd.Context()` que retorna un context que se cancela con señales OS (SIGINT/SIGTERM). Este task conecta ese contexto a todas las llamadas internas, habilitando cancelación limpia con Ctrl+C.

## Especificacion Tecnica

```yaml
tipo: software-module
proyecto: rootline
lenguaje: go
paquete: cmd/rootline
interfaces:
  - nombre: CLI context wiring
    metodos:
      - nombre: Cada RunE function
        input: "cmd *cobra.Command, args []string"
        output: "error"
dependencias_externas: []
tests:
  - Todos los tests CLI existentes pasan sin cambios
```

## Dependencias

- T001-thread-context-core-interfaces debe estar completa

## Alcance

**In**:
1. En cada archivo de comando (`validate.go`, `query.go`, `fix.go`, `migrate.go`, `doctor.go`, `tree.go`, `describe.go`, `graph.go`, `explain.go`, `stats.go`):
   - Obtener ctx con `ctx := cmd.Context()`
   - Pasar `ctx` a todas las llamadas a funciones de `internal/`
2. En `root.go`: configurar signal handling si no lo provee cobra por defecto
3. Actualizar `cmd/rootline/filter.go` para pasar context a `query.MatchRecord`
4. Verificar que todos los tests CLI siguen pasando

**Out**: No agregar timeouts configurables. No agregar signal handling custom más allá de lo necesario. No modificar lógica de ningún comando.

## Estado inicial esperado

- T001 completada: todas las funciones de internal/ aceptan ctx
- `cmd/rootline/` llama a funciones sin ctx
- Todos los tests pasan

## Criterios de Aceptacion

- `go test ./... -race` pasa
- `go vet ./...` limpio
- `grep -rn "context.Background\|context.TODO" cmd/rootline/` retorna 0 matches en código de producción (solo en tests)
- Cada llamada a `index.Scan`, `rules.Validate`, `derive.DeriveAll`, `graph.Build`, `query.ExecuteExpr`, `fix.ApplyProposals`, `doctor.RunChecks` pasa `ctx` como primer argumento

## Fuente de verdad

- `cmd/rootline/validate.go`
- `cmd/rootline/query.go`
- `cmd/rootline/fix.go`
- `cmd/rootline/migrate.go`
- `cmd/rootline/doctor.go`
- `cmd/rootline/tree.go`
- `cmd/rootline/graph.go`
- `cmd/rootline/explain.go`
- `cmd/rootline/describe.go`
- `cmd/rootline/stats.go`
- `cmd/rootline/filter.go`
