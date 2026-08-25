---
tipo: adr
estado: accepted
fecha: "2026-08-25"
contexto: "El hook post-merge versionado ejecutaba instalación local y reparación de roadmap durante merges, produciendo cambios persistentes sin acción explícita del usuario."
decision: "Eliminar el hook post-merge mutante y mantener la instalación del checkout actual como una acción explícita mediante just install."
consecuencias: "Los merges y checkouts no reinstalarán binarios ni modificarán docs versionados; quienes necesiten instalar una build local deberán ejecutar just install manualmente."
---

# ADR 0003: Eliminar hook post-merge mutante

## Contexto

El repositorio incluía `.githooks/post-merge` con tres responsabilidades: sincronizar la skill local de Claude, delegar en `just install` para reconstruir el binario local y ejecutar `rootline fix --all docs/roadmap/` para propagar agregados. Ese hook puede ejecutarse como parte del ciclo normal de Git después de integrar cambios en un checkout donde los hooks fueron instalados.

La reproducción controlada de la incidencia verificó que un merge exitoso podía dejar `docs/roadmap/README.md` modificado y cambiar el contenido del binario instalado en `$HOME/.local/bin/rootline`, sin que la persona hubiese solicitado instalación ni reparación de documentación.

## Decisión

Eliminar `.githooks/post-merge` por completo. No se reemplaza por un hook informativo, warn-only ni por otro comando automático de post-merge.

La instalación de la build del checkout actual queda como acción explícita: ejecutar `just install`. La documentación activa y el comentario del `Justfile` deben describir esa operación como opt-in.

El hook `pre-push` conserva sus responsabilidades existentes, incluida la validación y la sincronización de la skill local, sin instalar builds de ramas no integradas.

## Alternativas descartadas

### Mantener un hook post-merge warn-only

Evitaría la mutación directa, pero conservaría una superficie de lifecycle innecesaria y podría volver a acumular responsabilidades implícitas. El requisito operativo es no agregar hooks ni comandos sustitutos.

### Mantener sincronización de skills en post-merge

Separaría una parte no binaria del hook, pero seguiría modificando `$HOME` durante un merge. La sincronización ya existe en `pre-push`, donde el costo y el momento son más explícitos para contribuyentes.

### Ejecutar `rootline fix --all` sin instalar binario

Eliminaría el cambio de binario, pero seguiría modificando archivos versionados durante merge. Las reparaciones de documentación deben ser una acción deliberada y revisable.

### Hacer que `just install` sea no mutante bajo hooks

Complica una receta cuya finalidad declarada es instalar en `$HOME/.local/bin`. La frontera más mantenible es no invocarla desde hooks de merge.

## Consecuencias

- Los merges no reinstalan automáticamente `rootline` en `$HOME/.local/bin`.
- Los merges no ejecutan reparaciones automáticas sobre `docs/roadmap/`.
- La build local puede quedar desactualizada después de hacer pull hasta que la persona ejecute `just install` explícitamente.
- La sincronización de skills continúa en `pre-push`; ese hook no instala builds locales de ramas no integradas.
- La ausencia del hook debe defenderse con comprobaciones de referencia para evitar reintroducir side effects de post-merge.
