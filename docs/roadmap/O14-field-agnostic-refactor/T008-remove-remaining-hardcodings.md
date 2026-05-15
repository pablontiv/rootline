---
estado: In Progress
tipo: task
---
# T008: Remove remaining hardcodings in stats, trace, analyze, proposal, schema_gen, migrate

**Outcome**: [O14 Field-agnostic refactor](README.md)
**Contribuye a**: ningún subcomando de rootline hardcodea nombres de campo semánticos

[[blocked_by:./T006-remove-domain-from-engine.md]]

## Preserva

- INV1: `go test ./...` verde después de todos los cambios
  - Verificar: `cd /home/shared/rootline && go test ./...`
- INV2: Los subcomandos que siguen siendo útiles (stats, trace) funcionan sin conocer el schema
  - Verificar: `rootline stats /home/shared/rootline/docs/roadmap --output json`

## Contexto

Después de eliminar `domain:` (T006), quedan hardcodings en varios subcomandos:

- **stats.go**: `StatsResult.ByEstado` y `StatsResult.ByTipo` con fallbacks `"estado"`/`"tipo"`. Renombrar a `ByLifecycle map[string]int json:"by_lifecycle_state"` y `ByRecordType map[string]int json:"by_record_type"`. Eliminar fallbacks domain-based.
- **trace.go**: `EffectiveField("estado")` en línea ~96 y `TraceNode.Estado`. trace debe ser agnóstico — eliminar ambos.
- **analyze.go + subschema_detection.go**: `DetectSubSchemas(records, "tipo")` asume campo discriminador hardcodeado. Eliminar la funcionalidad de sub-schema detection basada en campo hardcodeado.
- **proposal.go**: Inferencia de `"estado"` del README desde hijos (líneas ~490-512) asume lifecycle field y valores semánticos. Eliminar esta lógica de inferencia.
- **schema_gen.go + migrate/aggregate.go**: Keywords hardcodeadas (`"completed"`, `"bloqueada"`, `"done"`, `"obsoleto"`, `"diferida"`) para clasificar valores de enum. Eliminar keyword matching y clasificación semántica de valores.

## Alcance

**In**:
1. stats.go: renombrar ByEstado→ByLifecycle (json:"by_lifecycle_state"), ByTipo→ByRecordType (json:"by_record_type"), eliminar fallbacks "estado"/"tipo"
2. trace.go: eliminar `EffectiveField("estado")` y `TraceNode.Estado`; trace no decora con estado
3. analyze.go + subschema_detection.go: eliminar `DetectSubSchemas` basado en campo discriminador hardcodeado; conservar funciones agnósticas de campo si las hay
4. proposal.go: eliminar inferencia de estado desde hijos (líneas ~490-512)
5. schema_gen.go + migrate/aggregate.go: eliminar keyword matching semántico para clasificar enum values

**Out**:
- No cambiar el output format de subcomandos más allá de los campos renombrados
- No modificar stats/trace si queda funcionalidad útil agnóstica de campo

## Estado inicial esperado

- T006 completada (domain: eliminado del engine)
- `go test ./...` pasa

## Criterios de Aceptación

- `stats.go`: `ByEstado`/`ByTipo` no existen; `ByLifecycle`/`ByRecordType` retornan distribución genérica
- `trace.go`: `TraceNode.Estado` eliminado; `EffectiveField("estado")` eliminado
- `analyze.go`: `DetectSubSchemas(records, "tipo")` eliminado
- `proposal.go`: inferencia de estado desde hijos (líneas ~490-512) eliminada
- `schema_gen.go`/`migrate/aggregate.go`: keyword matching semántico (`"completed"`, `"done"`, etc.) eliminado
- `go test ./...` verde

## Fuente de verdad

- `cmd/rootline/stats.go`
- `cmd/rootline/trace.go`
- `cmd/rootline/analyze.go`
- `internal/infer/subschema_detection.go`
- `cmd/rootline/proposal.go`
- `cmd/rootline/schema_gen.go`
- `internal/migrate/aggregate.go`
