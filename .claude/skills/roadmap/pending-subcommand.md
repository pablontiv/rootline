# /roadmap pending

Vista filtrada de trabajo pendiente. Muestra Outcomes con tasks pendientes y tasks directas pendientes.

## Workspace mode

Si `<repos>` existe:

1. Para cada repo, ejecutar en paralelo:
   ```bash
   rootline tree <abs-roadmap-root>/ --where '<where-leaf> && <where-not-done>' --output json
   ```
2. Procesar JSON en memoria:
   - `pending = total - completed` por nodo.
   - Omitir repos con `pending == 0`.
   - Sumar totales del workspace.
3. Renderizar agrupado por repo.

Si `--repo` ya fue resuelto en bootstrap, usar el procedimiento single-repo.

## Single-repo

```bash
rootline tree <roadmap-root>/ --where '<where-leaf> && <where-not-done>' --output json
```

Renderizar desde JSON. No parsear tablas. No ejecutar `rootline stats`.
