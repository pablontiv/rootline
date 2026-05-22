---
tipo: outcome
---
# O16: Wirear picokit/autoupdate en el CLI de rootline

rootline es el único CLI del ecosistema pablontiv (roadmapctl, backscroll, rootline) que no ejecuta self-update del binario. roadmapctl lo tiene live desde O27/T001; backscroll lo cerró en O15/T002. rootline tiene release pipeline CI-driven (goreleaser + smoke tests + GitHub Releases) pero el binario instalado no se actualiza solo — el usuario depende de re-instalar manualmente vía `go install` o descarga.

Este outcome cierra esa asimetría. El patrón está validado en los otros dos consumidores: en arranque, sync `ApplyStagedIfAvailable()` aplica una release ya descargada (re-exec con atomic rename); luego en goroutine `FetchAndStage(version)` descarga la siguiente release a `~/.cache/rootline/staged/<version>/rootline` sin bloquear el comando del usuario. El próximo arranque la aplica.

**Decisión deliberada**: este outcome no expone una variable de entorno de opt-out. Único cortocircuito es `version == "dev"` (lo hace el package `autoupdate`). La firma variadic publicada en picokit v0.4.0 (O04/T002) permite llamar `autoupdate.New("pablontiv/rootline", "rootline")` con dos args.

Resultado observable cuando todas las tasks estén completadas: corriendo `rootline <cualquier-comando>` con `version != "dev"` descarga la próxima release en background; el siguiente arranque hace re-exec sobre el binario actualizado. La feature está documentada en `docs/auto-update.md` y mencionada en el README.

Invariantes:
- INV1: paridad de comportamiento con roadmapctl (mismo patrón staged async, misma estructura de cache `~/.cache/<binary>/staged/<version>/<binary>`).
- INV2: builds de desarrollo (`version == "dev"`) jamás tocan red ni disco.
- INV3: coverage del nuevo wiring ≥85% (gate pkcov ya activo en rootline).

Scope: este outcome wirea autoupdate en rootline. No modifica el package `autoupdate/` ni los otros consumidores.
