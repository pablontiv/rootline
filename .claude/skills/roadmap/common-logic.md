# Lógica Común — Materialización y Ejecución

Pre-requisito compartido por `plan` y `loop`. Leer este archivo cuando se van a crear o modificar archivos del roadmap.

> **Workspace mode**: En workspace mode, `<roadmap-root>` se reemplaza por
> `<abs-roadmap-root>` (path absoluto desde `<repos>` table). rootline acepta
> paths absolutos sin cambios. Los comandos `git` requieren `git -C <repo-path>`
> para operar en el repo correcto. En single-repo mode, `<repo-path>` = `.`
> (comportamiento sin cambios).

## Auto-numbering

Para cada nivel, usar `rootline describe` con el campo `schema.id.next`:

```bash
# Requiere .stem con id: {type: sequence, prefix: X, digits: N} en cada nivel

# Epics: próximo EXX
rootline describe <roadmap-root>/ --field schema.id.next

# Features: próximo FXX dentro del Epic
rootline describe <roadmap-root>/EXX-name/ --field schema.id.next

# Stories: próximo SXXX dentro del Feature
rootline describe <roadmap-root>/.../FXX-name/ --field schema.id.next

# Tasks: próximo TXXX dentro de la Story
rootline describe <roadmap-root>/.../SXXX-name/ --field schema.id.next
```

El comando retorna el próximo identificador directamente (ej: `"T004"`).

## Verificación de Padre

SIEMPRE verificar que el directorio padre existe antes de crear un artefacto:
- Verificar con `rootline describe <roadmap-root>/<path>/` que el directorio destino existe

Si no existe → informar al usuario y sugerir crearlo primero.

## Cascading Links

Después de crear un artefacto, actualizar la tabla en el README padre:
- Task creado → agregar fila en la tabla "Tasks" del Story README (solo Task + Descripcion, sin Estado)
- Story creada → agregar fila en la sección "Stories" del Feature README (sin Estado)

**Nota**: Las tablas NO incluyen columna Estado. El estado se lee del YAML frontmatter de cada Task y se deriva para Stories/Features en `/roadmap`.

---

## Comandos Rootline de Referencia

| Comando | Cuándo usarlo en el skill |
|---------|--------------------------|
| `rootline init <path>` | Bootstrap: inferir .stem schema de archivos existentes. `--dry-run` para preview |
| `rootline validate <path>` | Después de crear/editar archivos .md — verificar contra .stem |
| `rootline fix <path>` | Cuando validate falla — corregir automáticamente |
| `rootline describe <dir> --field schema.id.next` | Auto-numbering: obtener próximo ID en cualquier nivel |
| `rootline new <path>` | Scaffolding: crear archivo con frontmatter correcto según .stem |
| `rootline query <path> --where "expr"` | Discovery: buscar records por frontmatter (estado, tipo, etc.) |
| `rootline tree <path> --where "expr" --output table\|json` | Vista jerárquica con conteos completed/total por nodo (reemplaza stats) |
| `rootline graph <path> --where "expr" --check` | Grafo de dependencias filtrado |

**Nota**: `rootline stats` es redundante — `rootline tree` ya incluye completed/total. NO usar stats por separado.
