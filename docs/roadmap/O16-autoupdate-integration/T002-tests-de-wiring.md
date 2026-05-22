---
estado: Completed
tipo: task
---
# T002: Tests del wiring de autoupdate

**Outcome**: [O16 Wirear picokit/autoupdate en el CLI de rootline](README.md)
**Contribuye a**: INV2 (skip en dev) e INV3 (coverage ≥85% del nuevo wiring).

[[blocked_by:./T001-wire-picokit-autoupdate.md]]

## Preserva

- INV2 del outcome: `version == "dev"` no toca red ni disco. Test debe demostrarlo explícitamente.
- INV3 del outcome: coverage del nuevo wiring ≥85%. El package `picokit/autoupdate` ya está cubierto en su propio repo; este test cubre el wiring en rootline.

## Contexto

El package `picokit/autoupdate` ya tiene su propia suite. Lo que falta cubrir es el wiring local: la construcción del `Updater`, la asignación de `CurrentVersion`, la goroutine de fetch, y la espera con `WaitGroup`. roadmapctl tiene tests análogos como golden que se pueden copiar/adaptar (revisar `roadmapctl/internal/cli/*_test.go` o equivalente).

## Alcance

**In**:

1. Inspeccionar el patrón de tests del wiring en roadmapctl (`internal/cli/cli_test.go` u homólogo). Identificar:
   - Cómo testean que `version == "dev"` no produce side effects.
   - Cómo testean que la goroutine de fetch se lanza y termina.
   - Si usan mocks de HTTP o si dependen de `version == "dev"` para no llamar a la red.

2. Replicar el patrón en `cmd/rootline/main_test.go` (o el package donde viva el wiring tras T001):
   - Test `TestWiring_VersionDev_NoSideEffects` (o nombre análogo): construye el updater con `version == "dev"`, llama `ApplyStagedIfAvailable` y `FetchAndStage`, verifica que no se crea cache directory ni se hacen requests.
   - Test `TestWiring_NewWithTwoArgs`: verifica que `autoupdate.New("pablontiv/rootline", "rootline")` produce un updater con `EnvDisable == ""` (no env opt-out) y que `Repo`/`Binary` están bien seteados.

3. Si la lógica de wiring está en una función testeable (recomendado por T001), invocarla en los tests. Si está inline en `main()`, refactorear mínimamente para extraer la función pura.

4. Correr: `go test ./cmd/rootline/... -cover -count=1`. Verificar ≥85% en el archivo nuevo.

5. Correr `just coverage-check`. Verificar que pkcov no flagea el nuevo código.

**Out**:
- No cubrir el package `picokit/autoupdate` — eso es responsabilidad de picokit.
- No testear comportamiento real de red (descarga de releases) — mockear o evitar.
- No tocar otros tests del repo.

## Estado inicial esperado

- T001 completada: wiring en `cmd/rootline/main.go` u homólogo.
- No hay tests específicos del wiring todavía.

## Criterios de Aceptación

- Test verifica que `version == "dev"` no toca red ni cache.
- Test verifica que `autoupdate.New` con dos args produce el updater correcto (sin env opt-out).
- `go test ./cmd/rootline/...` exit 0.
- Coverage del archivo de wiring ≥85% según pkcov.
- `just coverage-check` exit 0.

## Fuente de verdad

- `/home/shared/roadmapctl/internal/cli/` — buscar `*_test.go` con tests análogos como golden
- `/home/shared/rootline/cmd/rootline/` — donde vivirán los tests nuevos
- `/home/shared/picokit/autoupdate/updater.go` — referencia de la firma para los tests
