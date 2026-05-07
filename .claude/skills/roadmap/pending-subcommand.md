# /roadmap pending

Vista jerarquica filtrada: solo Features con trabajo pendiente.

## Procedimiento (workspace mode)

Si `<repos>` existe (workspace mode detectado en bootstrap):

1. Para cada repo en `<repos>`, ejecutar en paralelo:
   ```bash
   rootline tree <abs-roadmap-root>/ --where '<where-leaf> && <where-not-done>' --output json
   ```

2. Procesar los JSONs en memoria (NO parsear tablas):
   - `pending = total - completed` en cada nodo.
   - Repos con `pending == 0`: omitir del output.
   - Totales workspace: sumar `pending`, `completed` y `total` de los nodos raíz.
   - Orden de repos: orden de `<repos>`; desempate por `name` ascendente.

3. Renderizar output agrupado por repo:

   ```
   WORKSPACE PENDING
   │
   ├── backscroll [completed/total]
   │   (render tree desde JSON)
   │
   ├── rootline [completed/total]
   │   (render tree desde JSON)
   │
   └── homeserver [completed/total]
       (render tree desde JSON)

   TOTALES: X pendientes across Y repos
   ```

Si `--repo` fue procesado en bootstrap → ya se resolvió a single-repo, usar procedimiento de abajo.

## Procedimiento (single-repo)

1. Ejecutar:
   ```bash
   rootline tree <roadmap-root>/ --where '<where-leaf> && <where-not-done>' --output json
   ```

2. Calcular totales desde JSON y renderizar una vista jerarquica para el usuario.

El tree ya incluye conteos `completed/total` por nodo — NO ejecutar `rootline stats` por separado (es redundante).
NO parsear output `table` para tomar decisiones; usar JSON como fuente de verdad.
