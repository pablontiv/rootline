---
estado: Specified
tipo: task
---
# T003: Subir cmd/rootline a ≥85% coverage

**Outcome**: [O15 Coverage feedback loop](README.md)
**Contribuye a**: precondición para activar el gate per-package (INV2)

## Preserva

- INV2 del outcome: el piso es 85% uniforme
  - Verificar: `go test ./cmd/rootline/ -cover` reporta ≥ 85.0%

## Contexto

`cmd/rootline` está en 80.6% (medido `2026-05-21`). Hay dos clases de huecos:

1. **CLI runE handlers en 0%**: `runTrace` (trace.go:40), `runRepairApply` (repair.go:52). Tests existentes no llaman estos comandos vía cobra; pattern correcto es `rootCmd.SetArgs([]string{"trace", "..."}); rootCmd.Execute()` con buffers para stdout/err. Ver `describe_test.go:executeDescribe` para el patrón canónico.

2. **Render helpers en 0%**: `renderApplyTable` (apply.go:135), `renderMigrateScaffoldTable` (migrate.go:365). Existen `render_tables_test.go` (creado en PR #36) que prueba `renderRepairTable`, `renderSchemaProposeTable`, `renderSchemaApplyTable` — extender con los dos faltantes siguiendo el mismo patrón (`newRenderTestCmd()` helper).

Posible solapamiento con T005: si `runTrace` o `renderApplyTable` son código muerto (T005 los evalúa), se borran allá y aquí no se cubren. Coordinar: ejecutar T005 antes que T003 si es posible, o asumir que ambas funciones siguen vivas y testearlas.

## Alcance

**In**:
1. Tests cobra para `runTrace` (varios subcomandos / inputs).
2. Tests cobra para `runRepairApply` con report válido + report con surface=schema + report con dry-run.
3. Tests para `renderApplyTable` (empty, full result, dry-run flag).
4. Tests para `renderMigrateScaffoldTable` (empty, con scaffolds).
5. Cubrir error paths de comandos cuyo runE haga `os.ReadFile` o `json.Unmarshal` que pueda fallar.

**Out**:
- No cambiar la firma o output de los comandos.
- No agregar nuevos subcomandos.
- No tocar paquetes internos (solo `cmd/rootline/`).

## Estado inicial esperado

- `go test ./cmd/rootline/ -cover` reporta ~80.6%.
- Existe `render_tables_test.go` con el patrón `newRenderTestCmd`.

## Criterios de Aceptación

- `go test ./cmd/rootline/ -cover` reporta `coverage: 85.0% of statements` o superior.
- Si T005 borra `runTrace` o `renderApplyTable`, la cobertura igual debe alcanzar 85 con menor denominador.
- `go test ./... -race` verde.

## Fuente de verdad

- `/home/shared/rootline/cmd/rootline/trace.go`, `repair.go`, `apply.go`, `migrate.go`
- `/home/shared/rootline/cmd/rootline/render_tables_test.go` — patrón render
- `/home/shared/rootline/cmd/rootline/describe_test.go` — patrón cobra
