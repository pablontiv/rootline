---
estado: Completed
tipo: task
---
# T003: Documentar el comportamiento de auto-update

**Outcome**: [O16 Wirear picokit/autoupdate en el CLI de rootline](README.md)
**Contribuye a**: cerrar el outcome con documentación visible para usuarios y mantenedores.

[[blocked_by:./T001-wire-picokit-autoupdate.md]]

## Contexto

roadmapctl tiene `/home/shared/roadmapctl/docs/auto-update.md` con la documentación canónica del patrón staged async. Sirve como golden — explica qué pasa en cada arranque, dónde vive la cache, qué hace el binary updater, comportamiento OS por OS (Linux/macOS/Windows), troubleshooting básico.

Rootline necesita el equivalente, ajustado a su contexto. Además conviene una sección breve en el README mencionando que el binario se auto-actualiza, con un link al doc detallado.

## Alcance

**In**:

1. Leer `/home/shared/roadmapctl/docs/auto-update.md` como referencia.

2. Crear `/home/shared/rootline/docs/auto-update.md`. Estructura sugerida:
   - **Resumen**: rootline se auto-actualiza en background. La próxima ejecución después de una release nueva re-ejecuta sobre el binario actualizado.
   - **Cómo funciona**: patrón staged async (run N descarga, run N+1 aplica).
   - **Ubicación de la cache**: `~/.cache/rootline/staged/<version>/rootline`.
   - **Cuándo NO se auto-actualiza**: build con `version == "dev"` (típicamente `go run` o `go build` sin ldflags) jamás toca red ni cache.
   - **Comportamiento por OS**: copiar de roadmapctl, ajustando nombre del binario.
   - **Troubleshooting**:
     - "El binario no se actualiza" → verificar `version != "dev"`, verificar permisos en `~/.cache/rootline/`, verificar conectividad a GitHub.
     - "Quiero forzar la actualización ahora" → borrar `~/.cache/rootline/staged/` y reiniciar el binario; en el segundo arranque hace re-exec.
   - **Cómo desactivar**: no hay opt-out por env var. Build local con `version == "dev"` (sin ldflags) es la única manera.

3. Agregar una sección al `/home/shared/rootline/README.md` (o equivalente, e.g. `Updating`), con dos o tres oraciones:
   - Auto-update activo en builds release.
   - Link a `docs/auto-update.md`.

4. Verificar links internos (no broken).

5. Push del commit con mensaje `docs: auto-update behavior`.

**Out**:
- No documentar el package `picokit/autoupdate` desde rootline — eso vive en picokit.
- No tocar otros docs.

## Estado inicial esperado

- T001 completada: el wiring existe y funciona.
- No existe `docs/auto-update.md` en rootline.
- README sin mención a auto-update.

## Criterios de Aceptación

- `docs/auto-update.md` creado, con secciones de resumen, mecanismo, cache, OS-specific, troubleshooting.
- README menciona auto-update y linkea al doc.
- Links internos válidos (no 404).
- `roadmapctl check --repo /home/shared/rootline --strict` exit 0.

## Fuente de verdad

- `/home/shared/roadmapctl/docs/auto-update.md` — golden a imitar
- `/home/shared/rootline/docs/auto-update.md` — destino
- `/home/shared/rootline/README.md` — destino de la mención breve
