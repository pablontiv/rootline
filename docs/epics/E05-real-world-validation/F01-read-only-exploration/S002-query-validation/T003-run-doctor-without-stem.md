---
estado: Completado
tipo: software-test
ejecutable_en: 1 sesion
---
# T003: Ejecutar doctor sin .stem files

**Story**: [S002 Query & Validation](README.md)

## Contexto

El comando `doctor` diagnostica la salud de un directorio de documentacion. Sin `.stem` files, debe reportar la ausencia de schema o reportar salud OK. No debe fallar con panic.

## Especificacion Tecnica

```yaml
tipo: software-test
proyecto: rootline
comando: doctor
target: /opt/homeserver/automation/docs/epics/
condicion: sin .stem files
```

## Alcance

**In**:
1. `rootline doctor /opt/homeserver/automation/docs/epics/ --output json`
2. `rootline doctor /opt/homeserver/automation/docs/epics/ --output table`
3. Verificar reporte coherente sobre ausencia de .stem

**Out**: Generacion de .stem, modificacion de archivos

## Criterios de Aceptacion

- Reporta ausencia de .stem files o salud OK
- Exit code 0
- Sin panics ni stack traces
- Output coherente en ambos formatos
