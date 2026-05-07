# Especificación Técnica por Tipo de Task

## Descubrimiento de tipos

Los tipos válidos se descubren dinámicamente desde el schema del proyecto:

```bash
rootline describe <story-dir> --field schema.tipo
```

## Templates de especificación

Los templates de especificación técnica se definen **por proyecto** en `.claude/roadmap.local.md`.

Cada proyecto define sus propios tipos y la estructura YAML que debe acompañar a cada task de ese tipo. El skill roadmap debe leer este archivo para saber qué campos incluir en la especificación técnica.

### Estructura esperada en roadmap.local.md

```markdown
## Tipos de task

### <nombre-del-tipo>

\```yaml
campo1: # descripción
campo2: # descripción
\```

### <otro-tipo>

\```yaml
...
\```
```

### Si no existe roadmap.local.md o no define templates

1. Ejecutar `rootline describe <story-dir>` para obtener los tipos válidos del schema efectivo donde se creará el Task.
2. NO inferir una estructura compleja por stack/lenguaje.
3. Usar un bloque libre mínimo bajo `## Especificacion Tecnica` con `objetivo`, `archivos`, `cambios`, `verificacion`; o preguntar al usuario si el tipo requiere una plantilla estricta.
4. No presuponer categorías ni enums — descubrirlos del schema con rootline.

## Tipos que no requieren especificación técnica

- **Documentación**: describir contenido y estructura en prosa dentro de la sección Alcance del task
- **Agrupación** (feature, historia): su contenido se define por [epic-guide.md](epic-guide.md) y [story-guide.md](story-guide.md)
