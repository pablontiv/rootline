---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T002: Ejecutar tree contra datos del homeserver

**Story**: [S001 Data Exploration](README.md)

## Contexto

El comando `tree` muestra la jerarquia de documentacion con conteos de estado. El homeserver tiene 4 niveles de jerarquia (Epic→Feature→Story→Task). Tree debe representar esta estructura visualmente.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: tree
target: /opt/homeserver/automation/docs/epics/
```

## Alcance

**In**:
1. `rootline tree /opt/homeserver/automation/docs/epics/ --output json`
2. `rootline tree /opt/homeserver/automation/docs/epics/ --output table`
3. Verificar jerarquia de 4 niveles en output
4. Verificar conteos completados/total

**Out**: Modificacion de archivos, escritura a disco

## Criterios de Aceptacion

- Exit code 0 en ambos formatos
- JSON output contiene `"version": 1`
- Muestra jerarquia con al menos 4 niveles de profundidad
- Conteos completados/total son coherentes con la realidad
- Sin panics ni stack traces
