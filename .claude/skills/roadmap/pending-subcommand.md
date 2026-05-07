# /roadmap pending

Vista jerarquica filtrada: solo Features con trabajo pendiente.

## Procedimiento (workspace mode)

Si `<repos>` existe (workspace mode detectado en bootstrap):

1. Para cada repo en `<repos>`, ejecutar en paralelo:
   ```bash
   rootline tree <abs-roadmap-root>/ --where '<where-leaf> && <where-not-done>' --output table
   ```

2. Presentar output agrupado por repo:

   ```
   WORKSPACE PENDING
   │
   ├── backscroll [completed/total]
   │   (tree output)
   │
   ├── rootline [completed/total]
   │   (tree output)
   │
   └── homeserver [completed/total]
       (tree output)

   TOTALES: X pendientes across Y repos
   ```

3. Repos con 0 pendientes: omitir del output (no mostrar repos vacíos).

Si `--repo` fue procesado en bootstrap → ya se resolvió a single-repo, usar procedimiento de abajo.

## Procedimiento (single-repo)

1. Ejecutar `rootline tree <roadmap-root>/ --where '<where-leaf> && <where-not-done>' --output table`

El tree ya incluye conteos `completed/total` por nodo — NO ejecutar `rootline stats` por separado (es redundante).

Presenta el output al usuario.
