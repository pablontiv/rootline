---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T003: Ejecutar describe contra datos del homeserver

**Story**: [S001 Data Exploration](README.md)

## Contexto

El comando `describe` muestra el schema de un directorio (campos detectados, tipos, valores). Sin `.stem` files, describe debe mostrar schema inferido de los archivos existentes o schema vacio. Ejecutar en directorio con tasks (tienen frontmatter) y en directorio raiz (mix de archivos con y sin frontmatter).

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: describe
target: /opt/homeserver/automation/docs/epics/
```

## Alcance

**In**:
1. `rootline describe` en subdirectorio que contiene tasks (ej: una Story)
2. `rootline describe` en directorio raiz de epics
3. Verificar comportamiento sin `.stem` files

**Out**: Generacion de .stem, modificacion de archivos

## Criterios de Aceptacion

- Exit code 0 en ambos escenarios
- Sin `.stem`: schema vacio o inferido, sin panic
- En directorio con tasks: detecta campos `estado`, `tipo`, `ejecutable_en`
- Sin panics ni stack traces
