---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T003: Ejecutar new --dry-run en story existente del homeserver

**Story**: [S001 Schema Inference Preview](README.md)

## Contexto

El comando `new --dry-run` genera un preview de un nuevo documento con frontmatter basado en el schema del directorio. Ejecutar en una Story existente del homeserver para ver que frontmatter generaria para un nuevo Task.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: new
flags: --dry-run
target: subdirectorio Story del homeserver
archivo: T999-test.md
```

## Alcance

**In**:
1. `rootline new T999-test.md --dry-run` en directorio de una Story existente
2. Verificar que muestra documento con frontmatter
3. Verificar que NO crea archivo en disco

**Out**: Creacion real de archivos

## Criterios de Aceptacion

- Muestra documento con frontmatter coherente (campos del directorio)
- NO crea archivo T999-test.md en disco (verificar con `ls`)
- Exit code 0
- Sin panics ni stack traces
