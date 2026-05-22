---
estado: Completed
tipo: task
---
# T007: Bump picokit de v0.2.0 a v0.4.0

**Contribuye a**: cerrar el desfase con la librería antes de promover picokit a dependencia directa (T008) y de wirear autoupdate (O16).

**Dependencia externa** (no expresable como `blocked_by` por estar en otro repo): requiere que `picokit/O04-autoupdate-envdisable-optional-and-windows-fix/T003-release-v0-4-0` esté Completed y el tag `v0.4.0` publicado en GitHub.

## Contexto

`/home/shared/rootline/go.mod` declara `github.com/pablontiv/picokit v0.2.0 // indirect` (consumida vía `tool github.com/pablontiv/picokit/cmd/pkcov`). Las otras dos consumidoras (roadmapctl, backscroll) también están en v0.2.0; este bump es parte del esfuerzo de sincronización del ecosistema.

Rootline ya cumple coverage-spec v1.0 (88.9% total, 13/13 paquetes PASS) usando pkcov de v0.2.0. v0.4.0 trae coverage-spec v1.1 (auto-discovery + exclude) — puede simplificar `.coverage-floors.toml` opcionalmente, pero no es obligatorio.

Esta task es precondición para T006 (fuzzy migration), que promueve picokit a dependencia directa, y para O16 (autoupdate wiring), que necesita la firma variadic publicada en v0.4.0.

## Alcance

**In**:

1. `cd /home/shared/rootline && go get github.com/pablontiv/picokit@v0.4.0`.
2. `go mod tidy`. La dependencia sigue marcada `indirect` (solo usada por la directive `tool`); eso lo cambia T006.
3. Correr suite local:
   - `just check`
   - `just test`
   - `just coverage-check` con `pkcov` v0.4.0.
4. Si auto-discovery v1.1 cambia el comportamiento de `pkcov`, revisar `.coverage-floors.toml`. Mantener el gate en verde.
5. Push del commit con mensaje `chore(deps): bump picokit to v0.4.0`.

**Out**:
- No promover picokit a direct (eso es T006).
- No tocar imports del código de rootline.
- No wirear autoupdate (eso es O16).

## Estado inicial esperado

- `go.mod` declara `github.com/pablontiv/picokit v0.2.0 // indirect`.
- Tag picokit `v0.4.0` publicado (precondición).

## Criterios de Aceptación

- `go.mod` declara `github.com/pablontiv/picokit v0.4.0 // indirect` (todavía indirect).
- `go mod tidy` clean.
- `just check && just test && just coverage-check` exit 0.
- CI verde tras push.

## Fuente de verdad

- `/home/shared/rootline/go.mod`
- `/home/shared/rootline/go.sum`
- `/home/shared/rootline/.coverage-floors.toml`
