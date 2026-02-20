---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T001: Ejecutar query con multiples operadores contra datos del homeserver

**Story**: [S002 Query & Validation](README.md)

## Contexto

El query engine soporta operadores `eq`, `ne`, `in`, `contains`, `exists`, `and`. Solo se han probado en unit tests con datos sinteticos. El homeserver tiene campos `estado` (Completado, Pendiente, etc.), `tipo` (software-module, software-test, etc.), y `ejecutable_en`.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: query
target: /opt/homeserver/automation/docs/epics/
operadores: eq, ne, in, contains, exists, and
```

## Alcance

**In**:
1. `rootline query /opt/homeserver/automation/docs/epics/ --where "estado eq Completado" --output json`
2. `rootline query /opt/homeserver/automation/docs/epics/ --where "tipo eq lxc"`
3. `rootline query /opt/homeserver/automation/docs/epics/ --where "estado eq Pendiente" --count`
4. Combinacion con `and`: `--where "estado eq Completado" --where "tipo eq software-module"`
5. `--limit 5` para verificar paginacion
6. `--field estado` para verificar extraccion de campo especifico

**Out**: Modificacion de archivos, escritura a disco

## Criterios de Aceptacion

- Resultados no vacios para queries validos
- Conteos coherentes con la realidad
- JSON output con `"version": 1`
- `--limit` respeta el limite especificado
- `--field` extrae solo el campo solicitado
- Sin panics ni stack traces
- Exit code 0
