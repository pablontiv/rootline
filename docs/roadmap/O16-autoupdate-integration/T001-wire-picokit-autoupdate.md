---
estado: Completed
tipo: task
---
# T001: Wirear `picokit/autoupdate` en el entry point del CLI

**Outcome**: [O16 Wirear picokit/autoupdate en el CLI de rootline](README.md)
**Contribuye a**: INV1 (paridad con roadmapctl) e INV2 (skip en dev).

[[blocked_by:../T007-bump-picokit-to-v0-4-0.md]]

## Preserva

- INV1 del outcome: misma estructura de wiring que `roadmapctl/internal/cli/cli.go:40-50`.
- INV2 del outcome: `version == "dev"` no toca red ni disco. Verificable con test golden de T002.

## Contexto

El package `picokit/autoupdate` (v0.4.0) ya está disponible tras O04 de picokit y el bump T007 de este repo. La firma variadic permite construir el updater con dos args: `autoupdate.New("pablontiv/rootline", "rootline")`. El patrón ya validado en roadmapctl es:

1. Construir updater al inicio del entry point.
2. Setear `u.CurrentVersion = version` (la variable inyectada por ldflags en build).
3. Sync: `_ = u.ApplyStagedIfAvailable()` antes de `Execute()` — aplica una release descargada en el run anterior.
4. Async: `go func() { _ = u.FetchAndStage(version) }()` lanzado en una goroutine, esperado al final del comando vía `WaitGroup` para permitir downloads en background sin bloquear al usuario.

Cortocircuito automático del package: si `currentVersion == "dev"`, ni `ApplyStagedIfAvailable` ni `FetchAndStage` tocan red ni cache. Esto cubre desarrollo local sin necesidad de env var.

## Alcance

**In**:

1. Localizar el entry point: `cmd/rootline/main.go` (o donde `cobra.Execute()` se invoque para el binario `rootline`).

2. Agregar wiring siguiendo el patrón de `roadmapctl/internal/cli/cli.go:40-50`. Esquema:

   ```go
   import (
       "sync"
       "github.com/pablontiv/picokit/autoupdate"
   )

   func main() {
       u := autoupdate.New("pablontiv/rootline", "rootline")
       u.CurrentVersion = version  // ldflags-injected
       _ = u.ApplyStagedIfAvailable()

       var wg sync.WaitGroup
       wg.Add(1)
       go func() {
           defer wg.Done()
           _ = u.FetchAndStage(version)
       }()

       err := rootCmd.Execute()
       wg.Wait()
       if err != nil {
           os.Exit(1)
       }
   }
   ```

   Ajustar imports y nombres a lo que ya use el archivo. No introducir env var de skip — la firma variadic permite omitir el tercer arg.

3. Verificar que `version` esté inyectada por ldflags. Si no lo está hoy, revisar `goreleaser.yml` y `.github/workflows/`: añadir `-X main.version=<tag>` en el build de release. Si ya está, no tocar.

4. Build local con ldflags de prueba: `go build -ldflags "-X main.version=0.0.1-test" ./cmd/rootline/...`. Verificar que el binario corre sin errores.

5. Build con `version == "dev"` (default sin ldflags): verificar que no se intenta tocar red — observar logs/stderr.

**Out**:
- No escribir tests (eso es T002).
- No escribir docs (eso es T003).
- No tocar el package `picokit/autoupdate` ni otros consumidores.
- No agregar env var de opt-out — decisión deliberada del outcome.

## Estado inicial esperado

- T007 completada: `go.mod` apunta a `picokit v0.4.0`.
- `cmd/rootline/main.go` no menciona `autoupdate`.
- `version` posiblemente ya inyectada por ldflags (verificar; si no, parte de esta task es habilitarlo).

## Criterios de Aceptación

- `cmd/rootline/main.go` (o el entry point real) construye y usa `autoupdate.Updater`.
- Llamada `autoupdate.New("pablontiv/rootline", "rootline")` con dos args (sin tercer arg).
- `ApplyStagedIfAvailable()` se invoca sync antes de `Execute()`.
- `FetchAndStage(version)` se invoca async vía goroutine, esperada con `WaitGroup`.
- Build verde con `go build ./...`.
- Build con ldflags `-X main.version=0.0.1-test` produce un binario funcional.
- `rootline --help` ejecuta normal; `version == "dev"` no produce errores ni network calls.

## Fuente de verdad

- `/home/shared/rootline/cmd/rootline/main.go` (o donde resida `Execute()`)
- `/home/shared/roadmapctl/internal/cli/cli.go:40-50` — referencia del patrón
- `/home/shared/picokit/autoupdate/updater.go` — firmas (`New`, `ApplyStagedIfAvailable`, `FetchAndStage`)
