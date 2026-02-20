---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T001: Ejecutar stats contra datos del homeserver

**Story**: [S001 Data Exploration](README.md)

## Contexto

El comando `stats` produce metricas agregadas de un directorio de documentacion. Nunca se ha ejecutado contra datos externos. El homeserver tiene ~58 tasks con frontmatter y ~56 archivos sin frontmatter (READMEs).

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: stats
target: /opt/homeserver/automation/docs/epics/
```

## Alcance

**In**:
1. `rootline stats /opt/homeserver/automation/docs/epics/ --output json`
2. `rootline stats /opt/homeserver/automation/docs/epics/ --output table`
3. Verificar que JSON tiene `"version": 1`
4. Verificar que conteo de `estado` suma ~58 tasks

**Out**: Modificacion de archivos, escritura a disco

## Criterios de Aceptacion

- Exit code 0 en ambos formatos
- JSON output contiene `"version": 1`
- Conteo de campo `estado` es coherente con cantidad real de tasks (~58)
- Sin panics ni stack traces
- Table output es legible y formateado correctamente
