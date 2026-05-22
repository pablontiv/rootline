---
estado: Completed
tipo: task
---
# T008: Migrar `internal/fuzzy` a `picokit/fuzzy` y borrar el duplicado

**Contribuye a**: eliminar duplicación de código no listada en ningún roadmap; promover picokit de `indirect` a dependencia directa.

[[blocked_by:./T007-bump-picokit-to-v0-4-0.md]]

## Contexto

`/home/shared/rootline/internal/fuzzy/fuzzy.go` define `Match`, `MatchN` y `Distance` con la misma semántica (case-insensitive, threshold `max(2, len(input)/3)`, Levenshtein) que `github.com/pablontiv/picokit/fuzzy`. El package interno está importado por 6 archivos:

- `internal/proposal/proposal.go`
- `internal/rules/validate.go`
- `internal/fix/fix.go`
- `internal/fix/fix_test.go`
- `internal/query/field_check.go`
- `internal/graph/graph.go`

picokit ya está en `go.mod` (indirect via `tool`). Esta task reemplaza los imports y borra el duplicado, promoviendo picokit a direct dep en el proceso (al usarse desde código aplicación, deja de ser solo tool).

No figura en ningún outcome existente; es una task suelta de higiene tras detectar el duplicado en el análisis de integración picokit.

## Alcance

**In**:

1. En cada uno de los 6 archivos importadores, reemplazar:
   ```go
   "github.com/pablontiv/rootline/internal/fuzzy"
   ```
   por:
   ```go
   "github.com/pablontiv/picokit/fuzzy"
   ```

   Las llamadas (`fuzzy.Match`, `fuzzy.MatchN`, `fuzzy.Distance`) mantienen la misma firma — verificado contra `picokit/fuzzy` v0.4.0. No es necesario tocar el cuerpo de las funciones.

2. Borrar:
   - `internal/fuzzy/fuzzy.go`
   - `internal/fuzzy/fuzzy_test.go`
   - El directorio `internal/fuzzy/` queda vacío y se elimina.

3. Si `.coverage-floors.toml` tiene una entrada `internal/fuzzy`, borrarla (el package ya no existe).

4. `go mod tidy`. picokit pasa de `indirect` a directo automáticamente al estar importado por código aplicación.

5. Suite local:
   - `just check`
   - `just test`
   - `just coverage-check`

6. Verificar que no quedan referencias: `grep -r "rootline/internal/fuzzy" .` debe dar cero matches dentro del repo.

7. Push del commit con mensaje `refactor: migrate internal/fuzzy to picokit/fuzzy`.

**Out**:
- No cambiar semántica de `Match`/`MatchN`/`Distance`.
- No tocar otros packages del repo más allá del cambio de imports.
- No incluir cambios de O16 (autoupdate).

## Estado inicial esperado

- T007 completada: `go.mod` apunta a `picokit v0.4.0 // indirect`.
- `internal/fuzzy/` existe con 2 archivos.
- 6 archivos del repo importan `github.com/pablontiv/rootline/internal/fuzzy`.

## Criterios de Aceptación

- 6 imports actualizados; cero referencias a `rootline/internal/fuzzy`.
- `internal/fuzzy/` borrado completamente.
- `go.mod` declara `github.com/pablontiv/picokit` sin el sufijo `// indirect`.
- `go mod tidy` no agrega líneas.
- `just check && just test && just coverage-check` exit 0.
- CI verde tras push.

## Fuente de verdad

- Archivos a modificar: `internal/proposal/proposal.go`, `internal/rules/validate.go`, `internal/fix/fix.go`, `internal/fix/fix_test.go`, `internal/query/field_check.go`, `internal/graph/graph.go`
- Archivos a borrar: `internal/fuzzy/fuzzy.go`, `internal/fuzzy/fuzzy_test.go`
- `/home/shared/rootline/go.mod`
- `/home/shared/rootline/.coverage-floors.toml`
